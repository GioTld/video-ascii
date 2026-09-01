package ascii

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"github.com/GioTld/video-ascii/internal/frame"
)

// DefaultRamp es el caracter estandar para la transicion de claro a oscuro.
const DefaultRamp = " .:-=+*#%@"

// Converter mapea los colores de pixeles del frame a caracteres ASCII usando un character ramp.
type Converter struct {
	ramp []rune
}

// NewConverter inicializa el Converter con un character ramp.
// Si el ramp esta vacio, se usa DefaultRamp.
func NewConverter(ramp string) (*Converter, error) {
	if ramp == "" {
		ramp = DefaultRamp
	}
	runes := []rune(ramp)
	if len(runes) == 0 {
		return nil, fmt.Errorf("character ramp cannot be empty")
	}
	return &Converter{
		ramp: runes,
	}, nil
}

// PixelLuminance calcula la iluminacion relativa (0..255) de un color usando los pesos de BT.709.
func PixelLuminance(c color.Color) float64 {
	r, g, b, _ := c.RGBA()
	r8 := float64(r >> 8)
	g8 := float64(g >> 8)
	b8 := float64(b >> 8)
	return 0.2126*r8 + 0.7152*g8 + 0.0722*b8
}

// ConvertFrame transforma un ResizedFrame en un array de lineas de texto ASCII.
func (c *Converter) ConvertFrame(f *frame.ResizedFrame) ([]string, error) {
	if f == nil {
		return nil, fmt.Errorf("resized frame cannot be nil")
	}
	if f.Width <= 0 || f.Height <= 0 || len(f.Pixels) == 0 {
		return nil, fmt.Errorf("invalid resized frame dimensions")
	}
	maxIdx := float64(len(c.ramp) - 1)
	lines := make([]string, f.Height)
	for y := 0; y < f.Height; y++ {
		var sb strings.Builder
		sb.Grow(f.Width)
		for x := 0; x < f.Width; x++ {
			lum := PixelLuminance(f.Pixels[y][x])
			idx := int(math.Round((lum / 255.0) * maxIdx))
			if idx < 0 {
				idx = 0
			} else if idx > int(maxIdx) {
				idx = int(maxIdx)
			}
			sb.WriteRune(c.ramp[idx])
		}
		lines[y] = sb.String()
	}
	return lines, nil
}
