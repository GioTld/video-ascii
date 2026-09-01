package decode

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"

	"github.com/GioTld/video-ascii/internal/frame"
)

// DecodeImage decodea una imagen desde un io.Reader a un Frame.
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

// DecodeFile lee un archivo de imagen desde el disco y decodea a un Frame.
func DecodeFile(path string) (*frame.Frame, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open image file: %w", err)
	}
	defer f.Close()
	return DecodeImage(f)
}
