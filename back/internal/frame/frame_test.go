package frame

import (
	"image"
	"image/color"
	"testing"
)

func TestNewFrame(t *testing.T) {
	tests := []struct {
		name    string
		img     image.Image
		wantErr bool
	}{
		{
			name:    "nil image returns error",
			img:     nil,
			wantErr: true,
		},
		{
			name:    "valid image creates frame",
			img:     image.NewRGBA(image.Rect(0, 0, 100, 50)),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := New(tt.img)
			if (err != nil) != tt.wantErr {
				t.Fatalf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && (f.Width != 100 || f.Height != 50) {
				t.Errorf("got dimensions %dx%d, want 100x50", f.Width, f.Height)
			}
		})
	}
}

func TestFrameResize(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}

	f, err := New(img)
	if err != nil {
		t.Fatalf("failed to create frame: %v", err)
	}

	tests := []struct {
		name       string
		targetW    int
		targetH    int
		fontAspect float64
		wantW      int
		wantH      int
		expectErr  bool
	}{
		{
			name:       "invalid width returns error",
			targetW:    0,
			targetH:    10,
			fontAspect: 0.5,
			expectErr:  true,
		},
		{
			name:       "auto height calculation with default aspect ratio",
			targetW:    50,
			targetH:    0,
			fontAspect: 0.5,
			wantW:      50,
			wantH:      25,
			expectErr:  false,
		},
		{
			name:       "explicit width and height",
			targetW:    20,
			targetH:    10,
			fontAspect: 0.5,
			wantW:      20,
			wantH:      10,
			expectErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rf, err := f.Resize(tt.targetW, tt.targetH, tt.fontAspect)
			if (err != nil) != tt.expectErr {
				t.Fatalf("Resize() error = %v, expectErr %v", err, tt.expectErr)
			}
			if !tt.expectErr {
				if rf.Width != tt.wantW || rf.Height != tt.wantH {
					t.Errorf("got %dx%d, want %dx%d", rf.Width, rf.Height, tt.wantW, tt.wantH)
				}
				if len(rf.Pixels) != tt.wantH || len(rf.Pixels[0]) != tt.wantW {
					t.Errorf("pixel grid dimensions mismatch: got %dx%d", len(rf.Pixels[0]), len(rf.Pixels))
				}
			}
		})
	}
}
