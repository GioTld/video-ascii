package filter

import (
	"fmt"
	"image/color"
	"math"
	"strconv"
	"strings"

	"github.com/GioTld/video-ascii/internal/frame"
)

// Parse parses a comma-separated filter specification string
// (e.g. "brightness=1.2,contrast=1.2,sepia,invert,matrix,grayscale,edge")
// and returns a composed filter Func.
func Parse(spec string) (Func, error) {
	if spec == "" {
		return nil, nil
	}
	parts := strings.Split(spec, ",")
	var funcs []Func
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		kv := strings.SplitN(p, "=", 2)
		name := strings.ToLower(kv[0])
		val := ""
		if len(kv) > 1 {
			val = kv[1]
		}
		switch name {
		case "brightness":
			factor := 1.2
			if val != "" {
				if f, err := strconv.ParseFloat(val, 64); err == nil {
					factor = f
				}
			}
			funcs = append(funcs, Brightness(factor))
		case "contrast":
			factor := 1.2
			if val != "" {
				if f, err := strconv.ParseFloat(val, 64); err == nil {
					factor = f
				}
			}
			funcs = append(funcs, Contrast(factor))
		case "sepia":
			funcs = append(funcs, Sepia())
		case "invert":
			funcs = append(funcs, Invert())
		case "matrix":
			funcs = append(funcs, Matrix())
		case "grayscale", "gray":
			funcs = append(funcs, Grayscale())
		case "edge", "sobel":
			funcs = append(funcs, Edge())
		default:
			return nil, fmt.Errorf("unsupported filter: %s (opciones: brightness, contrast, sepia, invert, matrix, grayscale, edge)", name)
		}
	}
	return Chain(funcs...), nil
}

// Func es una transformacion pura sobre un ResizedFrame. Recibe el frame de entrada
// y devuelve un nuevo frame modificado. Las implementaciones no deben mutar el original.
type Func func(*frame.ResizedFrame) *frame.ResizedFrame

// Chain compone multiples filtros en orden: cada filtro recibe la salida del anterior.
func Chain(filters ...Func) Func {
	return func(f *frame.ResizedFrame) *frame.ResizedFrame {
		for _, fn := range filters {
			if fn != nil {
				f = fn(f)
			}
		}
		return f
	}
}

// clamp devuelve v limitado al rango [0, 255].
func clamp(v float64) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

// mapPixels aplica una función de transformación pixel-a-pixel sobre un ResizedFrame
// y devuelve un nuevo ResizedFrame con los píxeles transformados.
func mapPixels(f *frame.ResizedFrame, fn func(r, g, b, a uint8) color.RGBA) *frame.ResizedFrame {
	pixels := make([][]color.Color, f.Height)
	for y := 0; y < f.Height; y++ {
		pixels[y] = make([]color.Color, f.Width)
		for x := 0; x < f.Width; x++ {
			r32, g32, b32, a32 := f.Pixels[y][x].RGBA()
			r, g, b, a := uint8(r32>>8), uint8(g32>>8), uint8(b32>>8), uint8(a32>>8)
			pixels[y][x] = fn(r, g, b, a)
		}
	}
	return &frame.ResizedFrame{Width: f.Width, Height: f.Height, Pixels: pixels}
}

// Brightness multiplica la luminosidad de cada canal RGB por factor (neutro: 1.0).
func Brightness(factor float64) Func {
	return func(f *frame.ResizedFrame) *frame.ResizedFrame {
		return mapPixels(f, func(r, g, b, a uint8) color.RGBA {
			return color.RGBA{
				R: clamp(float64(r) * factor),
				G: clamp(float64(g) * factor),
				B: clamp(float64(b) * factor),
				A: a,
			}
		})
	}
}

// Contrast ajusta el contraste usando una curva S centrada en 128 (neutro: 1.0).
func Contrast(factor float64) Func {
	return func(f *frame.ResizedFrame) *frame.ResizedFrame {
		return mapPixels(f, func(r, g, b, a uint8) color.RGBA {
			adj := func(c uint8) uint8 {
				return clamp((float64(c)-128)*factor + 128)
			}
			return color.RGBA{R: adj(r), G: adj(g), B: adj(b), A: a}
		})
	}
}

// Sepia aplica la matriz de conversión sepia estándar ITU-R.
func Sepia() Func {
	return func(f *frame.ResizedFrame) *frame.ResizedFrame {
		return mapPixels(f, func(r, g, b, a uint8) color.RGBA {
			fr, fg, fb := float64(r), float64(g), float64(b)
			return color.RGBA{
				R: clamp(fr*0.393 + fg*0.769 + fb*0.189),
				G: clamp(fr*0.349 + fg*0.686 + fb*0.168),
				B: clamp(fr*0.272 + fg*0.534 + fb*0.131),
				A: a,
			}
		})
	}
}

// Invert invierte los canales RGB de cada pixel.
func Invert() Func {
	return func(f *frame.ResizedFrame) *frame.ResizedFrame {
		return mapPixels(f, func(r, g, b, a uint8) color.RGBA {
			return color.RGBA{R: 255 - r, G: 255 - g, B: 255 - b, A: a}
		})
	}
}

// Grayscale desatura completamente la imagen preservando la luminancia BT.709.
func Grayscale() Func {
	return func(f *frame.ResizedFrame) *frame.ResizedFrame {
		return mapPixels(f, func(r, g, b, a uint8) color.RGBA {
			lum := clamp(0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(b))
			return color.RGBA{R: lum, G: lum, B: lum, A: a}
		})
	}
}

// Matrix desatura la imagen y aplica una paleta de verde fosforescente (#00FF41)
// escalada por la luminancia, emulando el estilo de las pantallas de Matrix.
func Matrix() Func {
	return func(f *frame.ResizedFrame) *frame.ResizedFrame {
		return mapPixels(f, func(r, g, b, a uint8) color.RGBA {
			lum := (0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(b)) / 255.0
			return color.RGBA{
				R: clamp(lum * 0),
				G: clamp(lum * 255),
				B: clamp(lum * 65),
				A: a,
			}
		})
	}
}

// Edge aplica un filtro de detección de bordes Sobel 3×3 sobre la luminancia de
// los píxeles del ResizedFrame. El resultado es una imagen en escala de grises
// donde los bordes aparecen brillantes sobre fondo negro.
func Edge() Func {
	return func(f *frame.ResizedFrame) *frame.ResizedFrame {
		// Precalcular luminancias para evitar llamadas RGBA repetidas.
		lum := make([][]float64, f.Height)
		for y := 0; y < f.Height; y++ {
			lum[y] = make([]float64, f.Width)
			for x := 0; x < f.Width; x++ {
				r32, g32, b32, _ := f.Pixels[y][x].RGBA()
				r8 := float64(r32 >> 8)
				g8 := float64(g32 >> 8)
				b8 := float64(b32 >> 8)
				lum[y][x] = 0.2126*r8 + 0.7152*g8 + 0.0722*b8
			}
		}

		pixels := make([][]color.Color, f.Height)
		for y := 0; y < f.Height; y++ {
			pixels[y] = make([]color.Color, f.Width)
			for x := 0; x < f.Width; x++ {
				// Coordenadas clamp para bordes del frame.
				y0 := max(y-1, 0)
				y1 := min(y+1, f.Height-1)
				x0 := max(x-1, 0)
				x1 := min(x+1, f.Width-1)

				// Kernel Sobel horizontal (Gx).
				gx := -lum[y0][x0] + lum[y0][x1] +
					-2*lum[y][x0] + 2*lum[y][x1] +
					-lum[y1][x0] + lum[y1][x1]

				// Kernel Sobel vertical (Gy).
				gy := -lum[y0][x0] - 2*lum[y0][x] - lum[y0][x1] +
					lum[y1][x0] + 2*lum[y1][x] + lum[y1][x1]

				magnitude := clamp(math.Sqrt(gx*gx+gy*gy) / math.Sqrt2)
				pixels[y][x] = color.RGBA{R: magnitude, G: magnitude, B: magnitude, A: 255}
			}
		}
		return &frame.ResizedFrame{Width: f.Width, Height: f.Height, Pixels: pixels}
	}
}

// max devuelve el mayor de dos enteros.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// min devuelve el menor de dos enteros.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
