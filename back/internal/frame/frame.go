package frame

import (
	"fmt"
	"image"
	"image/color"
)

// DefaultFontAspectRatio representa el ratio ancho-alto del estandar para caracteres de una terminal.
// La mayoria de fuentes monoespaciadas son el doble de alto y ancho.
const DefaultFontAspectRatio = 0.5

// Frame representa datos de pixeles y metadata para un solo frame de imagen.
type Frame struct {
	Image  image.Image
	Width  int
	Height int
}

// New crea un Frame wrapping una image.Image.
func New(img image.Image) (*Frame, error) {
	if img == nil {
		return nil, fmt.Errorf("frame image cannot be nil")
	}
	bounds := img.Bounds()
	return &Frame{
		Image:  img,
		Width:  bounds.Dx(),
		Height: bounds.Dy(),
	}, nil
}

// ResizedFrame contiene un grid 2D del color de los pixeles despues de resize a caracteres grid dimensions.
type ResizedFrame struct {
	Width  int
	Height int
	Pixels [][]color.Color
}

// Resize reduce el frame a un grid de targetWidth por targetHeight celdas.
// Si targetHeight es <= 0, se calcula automaticamente segun targetWidth,
// el ratio de la imagen y el ratio de la fuente.
func (f *Frame) Resize(targetWidth, targetHeight int, fontAspectRatio float64) (*ResizedFrame, error) {
	if targetWidth <= 0 {
		return nil, fmt.Errorf("target width must be positive, got %d", targetWidth)
	}
	if f.Width <= 0 || f.Height <= 0 {
		return nil, fmt.Errorf("invalid frame dimensions %dx%d", f.Width, f.Height)
	}
	if fontAspectRatio <= 0 {
		fontAspectRatio = DefaultFontAspectRatio
	}
	if targetHeight <= 0 {
		scale := float64(f.Height) / float64(f.Width)
		targetHeight = int(float64(targetWidth) * scale * fontAspectRatio)
		if targetHeight < 1 {
			targetHeight = 1
		}
	}
	bounds := f.Image.Bounds()
	minX, minY := bounds.Min.X, bounds.Min.Y
	pixels := make([][]color.Color, targetHeight)
	for y := 0; y < targetHeight; y++ {
		pixels[y] = make([]color.Color, targetWidth)
		startY := minY + (y * f.Height / targetHeight)
		endY := minY + ((y + 1) * f.Height / targetHeight)
		if endY <= startY {
			endY = startY + 1
		}
		for x := 0; x < targetWidth; x++ {
			startX := minX + (x * f.Width / targetWidth)
			endX := minX + ((x + 1) * f.Width / targetWidth)
			if endX <= startX {
				endX = startX + 1
			}
			var rSum, gSum, bSum, aSum uint64
			var count uint64
			for cy := startY; cy < endY; cy++ {
				for cx := startX; cx < endX; cx++ {
					r, g, b, a := f.Image.At(cx, cy).RGBA()
					rSum += uint64(r >> 8)
					gSum += uint64(g >> 8)
					bSum += uint64(b >> 8)
					aSum += uint64(a >> 8)
					count++
				}
			}
			if count == 0 {
				r, g, b, a := f.Image.At(startX, startY).RGBA()
				rSum = uint64(r >> 8)
				gSum = uint64(g >> 8)
				bSum = uint64(b >> 8)
				aSum = uint64(a >> 8)
				count = 1
			}
			avgR := uint8(rSum / count)
			avgG := uint8(gSum / count)
			avgB := uint8(bSum / count)
			avgA := uint8(aSum / count)
			pixels[y][x] = color.RGBA{R: avgR, G: avgG, B: avgB, A: avgA}
		}
	}
	return &ResizedFrame{
		Width:  targetWidth,
		Height: targetHeight,
		Pixels: pixels,
	}, nil
}
