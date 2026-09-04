package decode

import (
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/GioTld/video-ascii/internal/frame"
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

// VideoMeta holds timing information extracted from an MP4 file.
type VideoMeta struct {
	FPS    float64
	Width  int
	Height int
}

// ProbeVideo uses ffprobe to extract frame rate and resolution from an MP4 file.
func ProbeVideo(path string) (VideoMeta, error) {
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_streams",
		"-select_streams", "v:0",
		path,
	)

	out, err := cmd.Output()
	if err != nil {
		return VideoMeta{}, fmt.Errorf("ffprobe: %w", err)
	}

	var result struct {
		Streams []struct {
			Width      int    `json:"width"`
			Height     int    `json:"height"`
			RFrameRate string `json:"r_frame_rate"`
		} `json:"streams"`
	}

	if err := json.Unmarshal(out, &result); err != nil {
		return VideoMeta{}, fmt.Errorf("parse ffprobe output: %w", err)
	}

	if len(result.Streams) == 0 {
		return VideoMeta{}, fmt.Errorf("no video stream found in %q", path)
	}

	s := result.Streams[0]
	fps, err := parseRationalFPS(s.RFrameRate)
	if err != nil {
		return VideoMeta{}, fmt.Errorf("parse frame rate %q: %w", s.RFrameRate, err)
	}

	return VideoMeta{
		FPS:    fps,
		Width:  s.Width,
		Height: s.Height,
	}, nil
}

// parseRationalFPS convierte "num/den" (formato de ffprobe) a float64.
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
