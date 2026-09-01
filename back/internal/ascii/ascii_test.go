package ascii

import (
	"image/color"
	"testing"

	"github.com/GioTld/video-ascii/internal/frame"
)

func TestPixelLuminance(t *testing.T) {
	tests := []struct {
		name    string
		c       color.Color
		wantMin float64
		wantMax float64
	}{
		{
			name:    "black pixel",
			c:       color.RGBA{R: 0, G: 0, B: 0, A: 255},
			wantMin: 0.0,
			wantMax: 0.1,
		},
		{
			name:    "white pixel",
			c:       color.RGBA{R: 255, G: 255, B: 255, A: 255},
			wantMin: 254.9,
			wantMax: 255.1,
		},
		{
			name:    "pure red",
			c:       color.RGBA{R: 255, G: 0, B: 0, A: 255},
			wantMin: 54.0,
			wantMax: 55.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lum := PixelLuminance(tt.c)
			if lum < tt.wantMin || lum > tt.wantMax {
				t.Errorf("PixelLuminance() = %v, want between %v and %v", lum, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestConverter_ConvertFrame(t *testing.T) {
	conv, err := NewConverter("")
	if err != nil {
		t.Fatalf("NewConverter() error = %v", err)
	}

	blackPixels := [][]color.Color{
		{color.RGBA{R: 0, G: 0, B: 0, A: 255}, color.RGBA{R: 0, G: 0, B: 0, A: 255}},
	}
	whitePixels := [][]color.Color{
		{color.RGBA{R: 255, G: 255, B: 255, A: 255}, color.RGBA{R: 255, G: 255, B: 255, A: 255}},
	}

	tests := []struct {
		name      string
		rf        *frame.ResizedFrame
		wantLines []string
		wantErr   bool
	}{
		{
			name:    "nil frame returns error",
			rf:      nil,
			wantErr: true,
		},
		{
			name: "black pixels map to first ramp char",
			rf: &frame.ResizedFrame{
				Width:  2,
				Height: 1,
				Pixels: blackPixels,
			},
			wantLines: []string{"  "},
			wantErr:   false,
		},
		{
			name: "white pixels map to last ramp char",
			rf: &frame.ResizedFrame{
				Width:  2,
				Height: 1,
				Pixels: whitePixels,
			},
			wantLines: []string{"@@"},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines, err := conv.ConvertFrame(tt.rf)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ConvertFrame() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if len(lines) != len(tt.wantLines) {
					t.Fatalf("got %d lines, want %d", len(lines), len(tt.wantLines))
				}
				for i := range lines {
					if lines[i] != tt.wantLines[i] {
						t.Errorf("line %d: got %q, want %q", i, lines[i], tt.wantLines[i])
					}
				}
			}
		})
	}
}
