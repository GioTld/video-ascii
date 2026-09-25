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
	storage := make([]color.Color, targetHeight*targetWidth)
	for y := 0; y < targetHeight; y++ {
		pixels[y] = storage[y*targetWidth : (y+1)*targetWidth]
	}

	switch img := f.Image.(type) {
	case *image.RGBA:
		pix := img.Pix
		stride := img.Stride
		for y := 0; y < targetHeight; y++ {
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
					rowOff := (cy - bounds.Min.Y) * stride
					for cx := startX; cx < endX; cx++ {
						idx := rowOff + (cx-bounds.Min.X)*4
						rSum += uint64(pix[idx])
						gSum += uint64(pix[idx+1])
						bSum += uint64(pix[idx+2])
						aSum += uint64(pix[idx+3])
						count++
					}
				}
				if count == 0 {
					idx := (startY-bounds.Min.Y)*stride + (startX-bounds.Min.X)*4
					rSum = uint64(pix[idx])
					gSum = uint64(pix[idx+1])
					bSum = uint64(pix[idx+2])
					aSum = uint64(pix[idx+3])
					count = 1
				}
				pixels[y][x] = color.RGBA{
					R: uint8(rSum / count),
					G: uint8(gSum / count),
					B: uint8(bSum / count),
					A: uint8(aSum / count),
				}
			}
		}

	case *image.NRGBA:
		pix := img.Pix
		stride := img.Stride
		for y := 0; y < targetHeight; y++ {
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
					rowOff := (cy - bounds.Min.Y) * stride
					for cx := startX; cx < endX; cx++ {
						idx := rowOff + (cx-bounds.Min.X)*4
						rSum += uint64(pix[idx])
						gSum += uint64(pix[idx+1])
						bSum += uint64(pix[idx+2])
						aSum += uint64(pix[idx+3])
						count++
					}
				}
				if count == 0 {
					idx := (startY-bounds.Min.Y)*stride + (startX-bounds.Min.X)*4
					rSum = uint64(pix[idx])
					gSum = uint64(pix[idx+1])
					bSum = uint64(pix[idx+2])
					aSum = uint64(pix[idx+3])
					count = 1
				}
				pixels[y][x] = color.RGBA{
					R: uint8(rSum / count),
					G: uint8(gSum / count),
					B: uint8(bSum / count),
					A: uint8(aSum / count),
				}
			}
		}

	case *image.YCbCr:
		for y := 0; y < targetHeight; y++ {
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
				var rSum, gSum, bSum uint64
				var count uint64
				for cy := startY; cy < endY; cy++ {
					for cx := startX; cx < endX; cx++ {
						c := img.YCbCrAt(cx, cy)
						r, g, b := color.YCbCrToRGB(c.Y, c.Cb, c.Cr)
						rSum += uint64(r)
						gSum += uint64(g)
						bSum += uint64(b)
						count++
					}
				}
				if count == 0 {
					c := img.YCbCrAt(startX, startY)
					r, g, b := color.YCbCrToRGB(c.Y, c.Cb, c.Cr)
					rSum = uint64(r)
					gSum = uint64(g)
					bSum = uint64(b)
					count = 1
				}
				pixels[y][x] = color.RGBA{
					R: uint8(rSum / count),
					G: uint8(gSum / count),
					B: uint8(bSum / count),
					A: 255,
				}
			}
		}

	default:
		for y := 0; y < targetHeight; y++ {
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
				pixels[y][x] = color.RGBA{
					R: uint8(rSum / count),
					G: uint8(gSum / count),
					B: uint8(bSum / count),
					A: uint8(aSum / count),
				}
			}
		}
	}

	return &ResizedFrame{
		Width:  targetWidth,
		Height: targetHeight,
		Pixels: pixels,
	}, nil
}

// FitMode define cómo se ajusta una imagen a las dimensiones del contenedor de la terminal.
type FitMode string

const (
	// FitModeContain escala la imagen para que quepa completamente en la pantalla (Letterbox).
	FitModeContain FitMode = "contain"
	// FitModeCover escala la imagen para llenar toda la pantalla, recortando bordes sobrantes (Zoom to Fill).
	FitModeCover FitMode = "cover"
)

// Crop recorta un sub-rectángulo del ResizedFrame a partir de (x, y) de dimensiones w x h.
func (rf *ResizedFrame) Crop(x, y, w, h int) *ResizedFrame {
	if rf == nil {
		return nil
	}
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	if x+w > rf.Width {
		w = rf.Width - x
	}
	if y+h > rf.Height {
		h = rf.Height - y
	}
	if w <= 0 || h <= 0 {
		return rf
	}
	cropped := make([][]color.Color, h)
	cropStorage := make([]color.Color, h*w)
	for cy := 0; cy < h; cy++ {
		cropped[cy] = cropStorage[cy*w : (cy+1)*w]
		copy(cropped[cy], rf.Pixels[y+cy][x:x+w])
	}
	return &ResizedFrame{
		Width:  w,
		Height: h,
		Pixels: cropped,
	}
}
