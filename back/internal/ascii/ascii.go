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
	lut  [256]rune
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
	c := &Converter{
		ramp: runes,
	}
	maxIdx := float64(len(runes) - 1)
	for i := 0; i < 256; i++ {
		idx := int(math.Round((float64(i) / 255.0) * maxIdx))
		if idx < 0 {
			idx = 0
		} else if idx > int(maxIdx) {
			idx = int(maxIdx)
		}
		c.lut[i] = runes[idx]
	}
	return c, nil
}

// PixelLuminance calcula la iluminacion relativa (0..255) de un color usando los pesos de BT.709.
func PixelLuminance(c color.Color) float64 {
	r, g, b, _ := c.RGBA()
	r8 := float64(r >> 8)
	g8 := float64(g >> 8)
	b8 := float64(b >> 8)
	return 0.2126*r8 + 0.7152*g8 + 0.0722*b8
}

// Cell representa un caracter individual con su color RGBA correspondiente.
// BgColor es opcional (solo se usa en modos HD como HalfBlock) y
// representa el color de fondo ANSI de la celda.
type Cell struct {
	Char    rune
	Color   color.RGBA
	BgColor *color.RGBA
}

// CharFrame contiene la matriz 2D de celdas formateadas en el frame.
type CharFrame struct {
	Width  int
	Height int
	Cells  [][]Cell
}

// ToStrings convierte el CharFrame en una matriz de cadenas de texto ASCII sin color.
func (cf *CharFrame) ToStrings() []string {
	if cf == nil || cf.Height == 0 || cf.Width == 0 {
		return nil
	}
	lines := make([]string, cf.Height)
	for y := 0; y < cf.Height; y++ {
		var sb strings.Builder
		sb.Grow(cf.Width)
		for x := 0; x < cf.Width; x++ {
			sb.WriteRune(cf.Cells[y][x].Char)
		}
		lines[y] = sb.String()
	}
	return lines
}

// ConvertFrame transforma un ResizedFrame en un CharFrame con caracteres y colores por celda.
func (c *Converter) ConvertFrame(f *frame.ResizedFrame) (*CharFrame, error) {
	if f == nil {
		return nil, fmt.Errorf("resized frame cannot be nil")
	}
	if f.Width <= 0 || f.Height <= 0 || len(f.Pixels) == 0 {
		return nil, fmt.Errorf("invalid resized frame dimensions")
	}
	cells := make([][]Cell, f.Height)
	cellStorage := make([]Cell, f.Height*f.Width)
	for y := 0; y < f.Height; y++ {
		cells[y] = cellStorage[y*f.Width : (y+1)*f.Width]
		for x := 0; x < f.Width; x++ {
			rgba := toRGBA(f.Pixels[y][x])
			lumInt := (54*uint32(rgba.R) + 183*uint32(rgba.G) + 19*uint32(rgba.B)) >> 8
			if lumInt > 255 {
				lumInt = 255
			}
			cells[y][x] = Cell{
				Char:  c.lut[lumInt],
				Color: rgba,
			}
		}
	}
	return &CharFrame{
		Width:  f.Width,
		Height: f.Height,
		Cells:  cells,
	}, nil
}

// HDMode controla el modo de alta definición del renderizado.
type HDMode string

const (
	// HDModeNone es el modo estándar: un carácter ASCII por celda de terminal.
	HDModeNone HDMode = "none"
	// HDModeHalfBlock usa ▀ para codificar dos píxeles verticales por celda,
	// duplicando la resolución vertical efectiva.
	HDModeHalfBlock HDMode = "half"
	// HDModeBraille agrupa 2×4 píxeles en un único carácter Braille Unicode,
	// multiplicando la resolución por 8.
	HDModeBraille HDMode = "braille"
)

// toRGBA convierte un color.Color en color.RGBA con valores 0–255.
func toRGBA(c color.Color) color.RGBA {
	if rgba, ok := c.(color.RGBA); ok {
		return rgba
	}
	r, g, b, a := c.RGBA()
	return color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
}

// avgColor promedia una lista de colores RGBA.
func avgColor(colors []color.RGBA) color.RGBA {
	if len(colors) == 0 {
		return color.RGBA{}
	}
	var rSum, gSum, bSum, aSum int
	for _, c := range colors {
		rSum += int(c.R)
		gSum += int(c.G)
		bSum += int(c.B)
		aSum += int(c.A)
	}
	n := len(colors)
	return color.RGBA{R: uint8(rSum / n), G: uint8(gSum / n), B: uint8(bSum / n), A: uint8(aSum / n)}
}

// convertHalfBlock genera un CharFrame usando bloques ▀ para representar dos filas
// de píxeles por celda de terminal. El primer plano corresponde a la fila par (top)
// y el fondo a la fila impar (bottom).
func (c *Converter) convertHalfBlock(f *frame.ResizedFrame) *CharFrame {
	outH := f.Height / 2
	if outH < 1 {
		outH = 1
	}
	cells := make([][]Cell, outH)
	cellStorage := make([]Cell, outH*f.Width)
	bgStorage := make([]color.RGBA, outH*f.Width)
	for y := 0; y < outH; y++ {
		cells[y] = cellStorage[y*f.Width : (y+1)*f.Width]
		yTop := y * 2
		yBot := yTop + 1
		for x := 0; x < f.Width; x++ {
			fg := toRGBA(f.Pixels[yTop][x])
			bg := fg
			if yBot < f.Height {
				bg = toRGBA(f.Pixels[yBot][x])
			}
			idx := y*f.Width + x
			bgStorage[idx] = bg
			cells[y][x] = Cell{
				Char:    '▀',
				Color:   fg,
				BgColor: &bgStorage[idx],
			}
		}
	}
	return &CharFrame{Width: f.Width, Height: outH, Cells: cells}
}

// brailleOffsets mapea la posición (col 0–1, row 0–3) al bit de punto Braille.
// Estándar Unicode: puntos 1-4 en columna izquierda, 5-8 en derecha.
var brailleOffsets = [4][2]int{
	{0, 3}, // fila 0: puntos 1 (izq), 4 (der)
	{1, 4}, // fila 1: puntos 2 (izq), 5 (der)
	{2, 5}, // fila 2: puntos 3 (izq), 6 (der)
	{6, 7}, // fila 3: puntos 7 (izq), 8 (der)
}

// convertBraille genera un CharFrame usando caracteres Braille Unicode (U+2800–U+28FF).
// Cada celda agrupa una cuadrícula 2×4 de píxeles; los puntos encendidos se determinan
// por si la luminancia del píxel supera el umbral de 128.
func (c *Converter) convertBraille(f *frame.ResizedFrame) *CharFrame {
	outH := f.Height / 4
	outW := f.Width / 2
	if outH < 1 {
		outH = 1
	}
	if outW < 1 {
		outW = 1
	}
	cells := make([][]Cell, outH)
	cellStorage := make([]Cell, outH*outW)
	for cy := 0; cy < outH; cy++ {
		cells[cy] = cellStorage[cy*outW : (cy+1)*outW]
		for cx := 0; cx < outW; cx++ {
			var mask rune
			var lit [8]color.RGBA
			litCount := 0

			for row := 0; row < 4; row++ {
				for col := 0; col < 2; col++ {
					py := cy*4 + row
					px := cx*2 + col
					if py >= f.Height || px >= f.Width {
						continue
					}
					rgba := toRGBA(f.Pixels[py][px])
					lumInt := (54*uint32(rgba.R) + 183*uint32(rgba.G) + 19*uint32(rgba.B)) >> 8
					if lumInt > 128 {
						mask |= 1 << brailleOffsets[row][col]
						lit[litCount] = rgba
						litCount++
					}
				}
			}

			fg := color.RGBA{R: 180, G: 180, B: 180, A: 255}
			if litCount > 0 {
				var rSum, gSum, bSum, aSum int
				for i := 0; i < litCount; i++ {
					rSum += int(lit[i].R)
					gSum += int(lit[i].G)
					bSum += int(lit[i].B)
					aSum += int(lit[i].A)
				}
				fg = color.RGBA{
					R: uint8(rSum / litCount),
					G: uint8(gSum / litCount),
					B: uint8(bSum / litCount),
					A: uint8(aSum / litCount),
				}
			}
			cells[cy][cx] = Cell{
				Char:  0x2800 + mask,
				Color: fg,
			}
		}
	}
	return &CharFrame{Width: outW, Height: outH, Cells: cells}
}

// ConvertFrameHD convierte un ResizedFrame en un CharFrame usando el modo de alta
// definición especificado. Si mode es HDModeNone, delega a ConvertFrame.
func (c *Converter) ConvertFrameHD(f *frame.ResizedFrame, mode HDMode) (*CharFrame, error) {
	if f == nil {
		return nil, fmt.Errorf("resized frame cannot be nil")
	}
	if f.Width <= 0 || f.Height <= 0 || len(f.Pixels) == 0 {
		return nil, fmt.Errorf("invalid resized frame dimensions")
	}
	switch mode {
	case HDModeHalfBlock:
		return c.convertHalfBlock(f), nil
	case HDModeBraille:
		return c.convertBraille(f), nil
	default:
		return c.ConvertFrame(f)
	}
}


