package decode

import (
	"bufio"
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

// DecodeVideo extrae fotogramas de un archivo MP4 via ffmpeg y los envía por un canal.
// El canal se cierra cuando ffmpeg termina o el contexto es cancelado.
// Usar cancel() para detener la extracción antes de que termine.
func DecodeVideo(path string, cancel <-chan struct{}) (<-chan *frame.Frame, <-chan error) {
	frames := make(chan *frame.Frame)
	errc := make(chan error, 1)

	go func() {
		defer close(frames)
		defer close(errc)

		// ffmpeg emite fotogramas en formato PPM (binario) por stdout.
		// PPM no necesita parsing de contenedor; cada fotograma es un bloque
		// "P6\n<w> <h>\n255\n<pixels>", lo que permite leer de forma streaming.
		cmd := exec.Command("ffmpeg",
			"-i", path,
			"-f", "image2pipe",
			"-vcodec", "ppm",
			"pipe:1",
		)

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			errc <- fmt.Errorf("create stdout pipe: %w", err)
			return
		}

		if err := cmd.Start(); err != nil {
			errc <- fmt.Errorf("start ffmpeg: %w", err)
			return
		}

		// stderr de ffmpeg se descarta; solo nos interesa el stream de fotogramas.
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
			// ffmpeg retorna código no cero cuando se mata el proceso; ignorar.
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

// readPPMFrame lee un fotograma PPM (P6) del reader.
// Retorna io.EOF cuando no hay más datos.
func readPPMFrame(r *bufio.Reader) (image.Image, error) {
	// Formato: "P6\n<width> <height>\n255\n<raw rgb bytes>"
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

	// 3 bytes por pixel (R, G, B).
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
