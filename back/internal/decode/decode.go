package decode

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/GioTld/video-ascii/internal/frame"
	"github.com/GioTld/video-ascii/internal/subtitle"
)

// DecodeImage decodes an image from an io.Reader into a Frame.
func DecodeImage(r io.Reader) (*frame.Frame, error) {
	if r == nil {
		return nil, fmt.Errorf("reader cannot be nil")
	}

	img, _, err := image.Decode(r)
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	f, err := frame.New(img)
	if err != nil {
		return nil, fmt.Errorf("create frame: %w", err)
	}

	return f, nil
}

// DecodeFile reads an image file from disk and decodes it into a Frame.
func DecodeFile(path string) (*frame.Frame, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open image file: %w", err)
	}
	defer f.Close()

	return DecodeImage(f)
}

// VideoMeta holds timing information and stream capabilities extracted from an MP4 file.
type VideoMeta struct {
	FPS      float64
	Width    int
	Height   int
	HasAudio bool
}

// ProbeVideo uses ffprobe to extract frame rate, resolution, and audio presence from an MP4 file.
func ProbeVideo(path string) (VideoMeta, error) {
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_streams",
		path,
	)

	out, err := cmd.Output()
	if err != nil {
		return VideoMeta{}, fmt.Errorf("ffprobe: %w", err)
	}

	var result struct {
		Streams []struct {
			CodecType  string `json:"codec_type"`
			Width      int    `json:"width"`
			Height     int    `json:"height"`
			RFrameRate string `json:"r_frame_rate"`
		} `json:"streams"`
	}

	if err := json.Unmarshal(out, &result); err != nil {
		return VideoMeta{}, fmt.Errorf("parse ffprobe output: %w", err)
	}

	var videoStream *struct {
		CodecType  string `json:"codec_type"`
		Width      int    `json:"width"`
		Height     int    `json:"height"`
		RFrameRate string `json:"r_frame_rate"`
	}
	hasAudio := false

	for i := range result.Streams {
		st := &result.Streams[i]
		if st.CodecType == "video" && videoStream == nil {
			videoStream = st
		} else if st.CodecType == "audio" {
			hasAudio = true
		}
	}

	if videoStream == nil {
		return VideoMeta{}, fmt.Errorf("no video stream found in %q", path)
	}

	fps, err := parseRationalFPS(videoStream.RFrameRate)
	if err != nil {
		return VideoMeta{}, fmt.Errorf("parse frame rate %q: %w", videoStream.RFrameRate, err)
	}

	return VideoMeta{
		FPS:      fps,
		Width:    videoStream.Width,
		Height:   videoStream.Height,
		HasAudio: hasAudio,
	}, nil
}

func parseRationalFPS(r string) (float64, error) {
	parts := strings.SplitN(r, "/", 2)
	if len(parts) != 2 {
		return 0, fmt.Errorf("unexpected format %q", r)
	}
	num, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, err
	}
	den, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return 0, err
	}
	if den == 0 {
		return 0, fmt.Errorf("denominator is zero")
	}
	return num / den, nil
}

// DecodeVideo extracts frames from an MP4 file via ffmpeg starting from 0s and sends them over a channel.
func DecodeVideo(path string, cancel <-chan struct{}) (<-chan *frame.Frame, <-chan error) {
	return DecodeVideoFrom(path, 0, cancel)
}

// DecodeVideoFrom extracts frames from an MP4 file via ffmpeg starting at startSec timestamp.
func DecodeVideoFrom(path string, startSec float64, cancel <-chan struct{}) (<-chan *frame.Frame, <-chan error) {
	frames := make(chan *frame.Frame)
	errc := make(chan error, 1)

	go func() {
		defer close(frames)
		defer close(errc)

		args := []string{}
		if startSec > 0 {
			args = append(args, "-ss", fmt.Sprintf("%.2f", startSec))
		}
		args = append(args, "-i", path, "-f", "image2pipe", "-vcodec", "ppm", "pipe:1")

		cmd := exec.Command("ffmpeg", args...)

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			errc <- fmt.Errorf("create stdout pipe: %w", err)
			return
		}

		if err := cmd.Start(); err != nil {
			errc <- fmt.Errorf("start ffmpeg: %w", err)
			return
		}

		cmd.Stderr = nil
		reader := bufio.NewReaderSize(stdout, 1<<20)

		for {
			select {
			case <-cancel:
				cmd.Process.Kill() //nolint:errcheck
				return
			default:
			}

			img, err := readPPMFrame(reader)
			if err == io.EOF {
				break
			}
			if err != nil {
				errc <- fmt.Errorf("read ppm frame: %w", err)
				cmd.Process.Kill() //nolint:errcheck
				return
			}

			f, err := frame.New(img)
			if err != nil {
				errc <- fmt.Errorf("create frame: %w", err)
				cmd.Process.Kill() //nolint:errcheck
				return
			}

			select {
			case frames <- f:
			case <-cancel:
				cmd.Process.Kill() //nolint:errcheck
				return
			}
		}

		if err := cmd.Wait(); err != nil {
			if cancel != nil {
				select {
				case <-cancel:
					return
				default:
				}
			}
			errc <- fmt.Errorf("ffmpeg wait: %w", err)
		}
	}()

	return frames, errc
}

func readPPMFrame(r *bufio.Reader) (image.Image, error) {
	magic, err := r.ReadString('\n')
	if err == io.EOF && magic == "" {
		return nil, io.EOF
	}
	if err != nil {
		return nil, fmt.Errorf("read magic: %w", err)
	}

	magic = strings.TrimSpace(magic)
	if magic != "P6" {
		return nil, fmt.Errorf("expected PPM magic P6, got %q", magic)
	}

	dimLine, err := r.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("read dimensions: %w", err)
	}

	var w, h int
	if _, err := fmt.Sscanf(strings.TrimSpace(dimLine), "%d %d", &w, &h); err != nil {
		return nil, fmt.Errorf("parse dimensions %q: %w", dimLine, err)
	}

	maxLine, err := r.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("read maxval: %w", err)
	}
	if strings.TrimSpace(maxLine) != "255" {
		return nil, fmt.Errorf("unexpected maxval %q", maxLine)
	}

	buf := make([]byte, w*h*3)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, fmt.Errorf("read pixel data: %w", err)
	}

	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := (y*w + x) * 3
			img.SetNRGBA(x, y, struct{ R, G, B, A uint8 }{
				R: buf[i],
				G: buf[i+1],
				B: buf[i+2],
				A: 255,
			})
		}
	}

	return img, nil
}

// ExtractEmbeddedSubtitles attempts to extract the first embedded subtitle stream from a video using ffmpeg.
func ExtractEmbeddedSubtitles(path string) ([]byte, error) {
	cmd := exec.Command("ffmpeg",
		"-v", "quiet",
		"-i", path,
		"-map", "0:s:0",
		"-f", "srt",
		"pipe:1",
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("extract embedded subtitles: %w", err)
	}
	return out, nil
}

// LoadSubtitles resolves subtitle entries following the priority order:
// 1. Explicit subFlag file path.
// 2. Same-name .srt file in video's directory (e.g. video.mp4 -> video.srt).
// 3. Embedded subtitle stream extracted from video via ffmpeg.
func LoadSubtitles(videoPath string, subFlag string) ([]subtitle.Entry, error) {
	if subFlag != "" {
		f, err := os.Open(subFlag)
		if err != nil {
			return nil, fmt.Errorf("open subtitle file %q: %w", subFlag, err)
		}
		defer f.Close()
		return subtitle.ParseSRT(f)
	}

	ext := filepath.Ext(videoPath)
	autoSrtPath := strings.TrimSuffix(videoPath, ext) + ".srt"
	if _, err := os.Stat(autoSrtPath); err == nil {
		f, err := os.Open(autoSrtPath)
		if err == nil {
			defer f.Close()
			entries, parseErr := subtitle.ParseSRT(f)
			if parseErr == nil && len(entries) > 0 {
				return entries, nil
			}
		}
	}

	embeddedData, err := ExtractEmbeddedSubtitles(videoPath)
	if err == nil && len(embeddedData) > 0 {
		entries, parseErr := subtitle.ParseSRT(bytes.NewReader(embeddedData))
		if parseErr == nil && len(entries) > 0 {
			return entries, nil
		}
	}

	return nil, nil
}
