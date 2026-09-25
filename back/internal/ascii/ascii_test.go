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
			cf, err := conv.ConvertFrame(tt.rf)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ConvertFrame() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				lines := cf.ToStrings()
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

func TestConvertFrameHD_HalfBlock(t *testing.T) {
	conv, _ := NewConverter("")
	// Frame 4×4: fila par=blanco, fila impar=negro → mitad de filas.
	pixels := [][]color.Color{
		{color.RGBA{R: 255, G: 255, B: 255, A: 255}, color.RGBA{R: 255, G: 255, B: 255, A: 255}},
		{color.RGBA{R: 0, G: 0, B: 0, A: 255}, color.RGBA{R: 0, G: 0, B: 0, A: 255}},
		{color.RGBA{R: 255, G: 0, B: 0, A: 255}, color.RGBA{R: 255, G: 0, B: 0, A: 255}},
		{color.RGBA{R: 0, G: 255, B: 0, A: 255}, color.RGBA{R: 0, G: 255, B: 0, A: 255}},
	}
	f := &frame.ResizedFrame{Width: 2, Height: 4, Pixels: pixels}
	cf, err := conv.ConvertFrameHD(f, HDModeHalfBlock)
	if err != nil {
		t.Fatalf("ConvertFrameHD HalfBlock: %v", err)
	}
	// Debe producir Height/2 = 2 filas.
	if cf.Height != 2 {
		t.Errorf("HalfBlock height = %d, want 2", cf.Height)
	}
	// Cada celda debe ser ▀.
	for y := 0; y < cf.Height; y++ {
		for x := 0; x < cf.Width; x++ {
			if cf.Cells[y][x].Char != '▀' {
				t.Errorf("HalfBlock cell [%d][%d] char = %q, want ▀", y, x, cf.Cells[y][x].Char)
			}
			if cf.Cells[y][x].BgColor == nil {
				t.Errorf("HalfBlock cell [%d][%d] BgColor should not be nil", y, x)
			}
		}
	}
}

func TestConvertFrameHD_Braille(t *testing.T) {
	conv, _ := NewConverter("")
	// Frame 8×4: suficiente para producir al menos una celda Braille.
	pixels := make([][]color.Color, 4)
	for y := range pixels {
		pixels[y] = make([]color.Color, 8)
		for x := range pixels[y] {
			pixels[y][x] = color.RGBA{R: 200, G: 200, B: 200, A: 255}
		}
	}
	f := &frame.ResizedFrame{Width: 8, Height: 4, Pixels: pixels}
	cf, err := conv.ConvertFrameHD(f, HDModeBraille)
	if err != nil {
		t.Fatalf("ConvertFrameHD Braille: %v", err)
	}
	// Braille agrupa 2×4 → Width/2=4, Height/4=1.
	if cf.Width != 4 || cf.Height != 1 {
		t.Errorf("Braille dimensions = %dx%d, want 4x1", cf.Width, cf.Height)
	}
	// Los caracteres deben estar en el rango Braille U+2800–U+28FF.
	for y := 0; y < cf.Height; y++ {
		for x := 0; x < cf.Width; x++ {
			ch := cf.Cells[y][x].Char
			if ch < 0x2800 || ch > 0x28FF {
				t.Errorf("Braille cell [%d][%d] char = U+%04X, out of Braille range", y, x, ch)
			}
		}
	}
}

func TestConvertFrameHD_NilError(t *testing.T) {
	conv, _ := NewConverter("")
	_, err := conv.ConvertFrameHD(nil, HDModeHalfBlock)
	if err == nil {
		t.Error("expected error for nil frame")
	}
}


