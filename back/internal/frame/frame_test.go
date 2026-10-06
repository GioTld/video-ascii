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

func TestFrameResizePackedRGBA(t *testing.T) {
	createImage := func(nrgba bool) image.Image {
		bounds := image.Rect(10, 20, 14, 22)
		colors := []color.RGBA{
			{R: 10, G: 20, B: 30, A: 40},
			{R: 30, G: 40, B: 50, A: 60},
			{R: 50, G: 60, B: 70, A: 80},
			{R: 70, G: 80, B: 90, A: 100},
			{R: 20, G: 30, B: 40, A: 50},
			{R: 40, G: 50, B: 60, A: 70},
			{R: 60, G: 70, B: 80, A: 90},
			{R: 80, G: 90, B: 100, A: 110},
		}
		if nrgba {
			img := image.NewNRGBA(bounds)
			for i, c := range colors {
				img.SetNRGBA(bounds.Min.X+i%4, bounds.Min.Y+i/4, color.NRGBA(c))
			}
			return img.SubImage(image.Rect(11, 20, 13, 22))
		}
		img := image.NewRGBA(bounds)
		for i, c := range colors {
			img.SetRGBA(bounds.Min.X+i%4, bounds.Min.Y+i/4, c)
		}
		return img.SubImage(image.Rect(11, 20, 13, 22))
	}

	tests := []struct {
		name   string
		image  image.Image
		width  int
		height int
		want   [][]color.RGBA
	}{
		{
			name:   "rgba subimage downscale",
			image:  createImage(false),
			width:  1,
			height: 1,
			want:   [][]color.RGBA{{{R: 45, G: 55, B: 65, A: 75}}},
		},
		{
			name:   "nrgba subimage downscale",
			image:  createImage(true),
			width:  1,
			height: 1,
			want:   [][]color.RGBA{{{R: 45, G: 55, B: 65, A: 75}}},
		},
		{
			name:   "rgba subimage upscale",
			image:  createImage(false),
			width:  4,
			height: 2,
			want: [][]color.RGBA{
				{{R: 30, G: 40, B: 50, A: 60}, {R: 30, G: 40, B: 50, A: 60}, {R: 50, G: 60, B: 70, A: 80}, {R: 50, G: 60, B: 70, A: 80}},
				{{R: 40, G: 50, B: 60, A: 70}, {R: 40, G: 50, B: 60, A: 70}, {R: 60, G: 70, B: 80, A: 90}, {R: 60, G: 70, B: 80, A: 90}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := New(tt.image)
			if err != nil {
				t.Fatal(err)
			}
			resized, err := f.Resize(tt.width, tt.height, 0.5)
			if err != nil {
				t.Fatal(err)
			}
			for y, row := range tt.want {
				for x, want := range row {
					got := resized.Pixels[y][x]
					if got != want {
						t.Errorf("pixel (%d, %d) = %+v, want %+v", x, y, got, want)
					}
				}
			}
		})
	}
}

func TestResizedFrame_Crop(t *testing.T) {
	pixels := make([][]color.RGBA, 10)
	for y := range pixels {
		pixels[y] = make([]color.RGBA, 10)
		for x := range pixels[y] {
			pixels[y][x] = color.RGBA{R: uint8(x * 10), G: uint8(y * 10), B: 0, A: 255}
		}
	}
	rf := &ResizedFrame{Width: 10, Height: 10, Pixels: pixels}
	cropped := rf.Crop(2, 2, 4, 4)
	if cropped.Width != 4 || cropped.Height != 4 {
		t.Fatalf("Crop dimensions = %dx%d, want 4x4", cropped.Width, cropped.Height)
	}
	// Pixel en (0,0) del recortado debe corresponder a (2,2) del original
	c := cropped.Pixels[0][0]
	if c.R != 20 || c.G != 20 {
		t.Errorf("Cropped (0,0) = {R:%d G:%d}, want {R:20 G:20}", c.R, c.G)
	}
}
