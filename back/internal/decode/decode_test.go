package decode

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestDecodeImage(t *testing.T) {
	pngBuffer := new(bytes.Buffer)
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	img.Set(0, 0, color.RGBA{R: 255, G: 0, B: 0, A: 255})
	if err := png.Encode(pngBuffer, img); err != nil {
		t.Fatalf("failed to encode PNG: %v", err)
	}

	jpegBuffer := new(bytes.Buffer)
	if err := jpeg.Encode(jpegBuffer, img, nil); err != nil {
		t.Fatalf("failed to encode JPEG: %v", err)
	}

	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "valid PNG stream",
			data:    pngBuffer.Bytes(),
			wantErr: false,
		},
		{
			name:    "valid JPEG stream",
			data:    jpegBuffer.Bytes(),
			wantErr: false,
		},
		{
			name:    "invalid image data",
			data:    []byte("invalid image content"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.data)
			f, err := DecodeImage(r)
			if (err != nil) != tt.wantErr {
				t.Fatalf("DecodeImage() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && (f.Width != 10 || f.Height != 10) {
				t.Errorf("got dimensions %dx%d, want 10x10", f.Width, f.Height)
			}
		})
	}
}

func TestDecodeFile(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.png")

	img := image.NewRGBA(image.Rect(0, 0, 20, 20))
	f, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		t.Fatalf("failed to encode PNG: %v", err)
	}
	f.Close()

	frame, err := DecodeFile(filePath)
	if err != nil {
		t.Fatalf("DecodeFile() error = %v", err)
	}
	if frame.Width != 20 || frame.Height != 20 {
		t.Errorf("got dimensions %dx%d, want 20x20", frame.Width, frame.Height)
	}

	_, err = DecodeFile(filepath.Join(tempDir, "nonexistent.png"))
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}

func TestParseRationalFPS(t *testing.T) {
	tests := []struct {
		input   string
		want    float64
		wantErr bool
	}{
		{"24/1", 24.0, false},
		{"30000/1001", 29.97002997002997, false},
		{"25/1", 25.0, false},
		{"0/0", 0, true},
		{"abc/1", 0, true},
		{"notaration", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseRationalFPS(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseRationalFPS(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if !tt.wantErr && (got < tt.want-0.001 || got > tt.want+0.001) {
				t.Errorf("parseRationalFPS(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
