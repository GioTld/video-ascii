package filter

import (
	"image/color"
	"testing"

	"github.com/GioTld/video-ascii/internal/frame"
)

func makeFrame(pixels [][]color.Color) *frame.ResizedFrame {
	return &frame.ResizedFrame{
		Width:  len(pixels[0]),
		Height: len(pixels),
		Pixels: pixels,
	}
}

func TestBrightness(t *testing.T) {
	tests := []struct {
		name   string
		factor float64
		input  color.RGBA
		wantR  uint8
		wantG  uint8
	}{
		{"double", 2.0, color.RGBA{R: 100, G: 50, B: 0, A: 255}, 200, 100},
		{"half", 0.5, color.RGBA{R: 100, G: 50, B: 0, A: 255}, 50, 25},
		{"clamp top", 2.0, color.RGBA{R: 200, G: 0, B: 0, A: 255}, 255, 0},
		{"zero factor", 0.0, color.RGBA{R: 200, G: 100, B: 50, A: 255}, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := makeFrame([][]color.Color{{tt.input}})
			out := Brightness(tt.factor)(f)
			c := out.Pixels[0][0].(color.RGBA)
			if c.R != tt.wantR || c.G != tt.wantG {
				t.Errorf("Brightness(%v) = {R:%d G:%d}, want {R:%d G:%d}", tt.factor, c.R, c.G, tt.wantR, tt.wantG)
			}
		})
	}
}

func TestContrast(t *testing.T) {
	// Un pixel en 128 no debe cambiar con ningún factor.
	f := makeFrame([][]color.Color{{color.RGBA{R: 128, G: 128, B: 128, A: 255}}})
	out := Contrast(3.0)(f)
	c := out.Pixels[0][0].(color.RGBA)
	if c.R != 128 {
		t.Errorf("Contrast: pixel en 128 debe ser invariante, got %d", c.R)
	}
}

func TestSepia(t *testing.T) {
	// Pixel naranja puro → sepia con R > G > B claramente.
	f := makeFrame([][]color.Color{{color.RGBA{R: 128, G: 64, B: 0, A: 255}}})
	out := Sepia()(f)
	c := out.Pixels[0][0].(color.RGBA)
	if c.R <= c.G || c.G <= c.B {
		t.Errorf("Sepia: expected R > G > B, got R=%d G=%d B=%d", c.R, c.G, c.B)
	}
}

func TestInvert(t *testing.T) {
	tests := []struct {
		input color.RGBA
		want  color.RGBA
	}{
		{color.RGBA{R: 0, G: 0, B: 0, A: 255}, color.RGBA{R: 255, G: 255, B: 255, A: 255}},
		{color.RGBA{R: 255, G: 255, B: 255, A: 255}, color.RGBA{R: 0, G: 0, B: 0, A: 255}},
		{color.RGBA{R: 100, G: 150, B: 200, A: 255}, color.RGBA{R: 155, G: 105, B: 55, A: 255}},
	}
	for _, tt := range tests {
		f := makeFrame([][]color.Color{{tt.input}})
		out := Invert()(f)
		c := out.Pixels[0][0].(color.RGBA)
		if c.R != tt.want.R || c.G != tt.want.G || c.B != tt.want.B {
			t.Errorf("Invert(%v) = %v, want %v", tt.input, c, tt.want)
		}
	}
}

func TestGrayscale(t *testing.T) {
	// Pixel ya gris → sin cambio.
	f := makeFrame([][]color.Color{{color.RGBA{R: 100, G: 100, B: 100, A: 255}}})
	out := Grayscale()(f)
	c := out.Pixels[0][0].(color.RGBA)
	if c.R != c.G || c.G != c.B {
		t.Errorf("Grayscale: R, G, B deben ser iguales, got R=%d G=%d B=%d", c.R, c.G, c.B)
	}
}

func TestMatrix(t *testing.T) {
	// Pixel blanco → verde puro normalizado.
	f := makeFrame([][]color.Color{{color.RGBA{R: 255, G: 255, B: 255, A: 255}}})
	out := Matrix()(f)
	c := out.Pixels[0][0].(color.RGBA)
	if c.R != 0 || c.G == 0 {
		t.Errorf("Matrix: expected R=0 and G>0, got R=%d G=%d", c.R, c.G)
	}
}

func TestEdge_BlackFrame(t *testing.T) {
	// Un frame completamente negro no tiene bordes → todo cero.
	pixels := make([][]color.Color, 5)
	for y := range pixels {
		pixels[y] = make([]color.Color, 5)
		for x := range pixels[y] {
			pixels[y][x] = color.RGBA{R: 0, G: 0, B: 0, A: 255}
		}
	}
	f := makeFrame(pixels)
	out := Edge()(f)
	for y := 0; y < out.Height; y++ {
		for x := 0; x < out.Width; x++ {
			c := out.Pixels[y][x].(color.RGBA)
			if c.R != 0 {
				t.Errorf("Edge en frame negro: esperado 0, got %d en (%d,%d)", c.R, x, y)
			}
		}
	}
}

func TestChain(t *testing.T) {
	// Encadenar Brightness(2.0) + Invert en pixel {100,0,0} = {200,0,0} → {55,255,255}.
	f := makeFrame([][]color.Color{{color.RGBA{R: 100, G: 0, B: 0, A: 255}}})
	out := Chain(Brightness(2.0), Invert())(f)
	c := out.Pixels[0][0].(color.RGBA)
	// R: clamp(200) → invert = 55
	if c.R != 55 || c.G != 255 {
		t.Errorf("Chain(Brightness+Invert) = {R:%d G:%d}, want {R:55 G:255}", c.R, c.G)
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		spec    string
		wantErr bool
	}{
		{
			name:    "empty spec returns nil",
			spec:    "",
			wantErr: false,
		},
		{
			name:    "single filter sepia",
			spec:    "sepia",
			wantErr: false,
		},
		{
			name:    "multiple filters with parameters",
			spec:    "brightness=1.5,contrast=1.1,invert,matrix,grayscale,edge",
			wantErr: false,
		},
		{
			name:    "unknown filter returns error",
			spec:    "unknown_filter",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, err := Parse(tt.spec)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Parse(%q) error = %v, wantErr %v", tt.spec, err, tt.wantErr)
			}
			if !tt.wantErr && tt.spec != "" && fn == nil {
				t.Fatalf("Parse(%q) returned nil func", tt.spec)
			}
		})
	}
}

