package render

import (
	"bytes"
	"image/color"
	"strings"
	"testing"
	"time"

	"github.com/GioTld/video-ascii/internal/ascii"
)

func TestRenderImage(t *testing.T) {
	sampleFrame := &ascii.CharFrame{
		Width:  2,
		Height: 1,
		Cells: [][]ascii.Cell{
			{
				{Char: 'A', Color: color.RGBA{R: 255, G: 0, B: 0, A: 255}},
				{Char: 'B', Color: color.RGBA{R: 0, G: 255, B: 0, A: 255}},
			},
		},
	}

	t.Run("renders without color", func(t *testing.T) {
		buf := new(bytes.Buffer)
		err := RenderImage(buf, sampleFrame, ColorModeNone)
		if err != nil {
			t.Fatalf("RenderImage() error = %v", err)
		}
		if buf.String() != "AB\n" {
			t.Errorf("got %q, want %q", buf.String(), "AB\n")
		}
	})

	t.Run("renders 24bit color with RLE", func(t *testing.T) {
		buf := new(bytes.Buffer)
		err := RenderImage(buf, sampleFrame, ColorMode24Bit)
		if err != nil {
			t.Fatalf("RenderImage() error = %v", err)
		}
		output := buf.String()
		if !strings.Contains(output, "\033[38;2;255;0;0mA") {
			t.Errorf("missing red 24bit escape code in %q", output)
		}
		if !strings.Contains(output, "\033[38;2;0;255;0mB") {
			t.Errorf("missing green 24bit escape code in %q", output)
		}
		if !strings.HasSuffix(output, "\033[0m\n") {
			t.Errorf("missing reset escape code at end in %q", output)
		}
	})

	t.Run("nil writer returns error", func(t *testing.T) {
		if err := RenderImage(nil, sampleFrame, ColorModeNone); err == nil {
			t.Error("expected error for nil writer, got nil")
		}
	})

	t.Run("nil frame returns error", func(t *testing.T) {
		buf := new(bytes.Buffer)
		if err := RenderImage(buf, nil, ColorModeNone); err == nil {
			t.Error("expected error for nil frame, got nil")
		}
	})
}

func TestRGBToANSI256(t *testing.T) {
	if code := RGBToANSI256(0, 0, 0); code != 16 {
		t.Errorf("RGBToANSI256(0,0,0) = %d, want 16", code)
	}
	if code := RGBToANSI256(255, 0, 0); code != 196 {
		t.Errorf("RGBToANSI256(255,0,0) = %d, want 196", code)
	}
}

func TestPlayVideo_CancelImmediately(t *testing.T) {
	frames := make(chan *ascii.CharFrame)
	cancel := make(chan struct{})
	close(cancel)

	buf := new(bytes.Buffer)
	opts := PlaybackOptions{FPS: 24}

	err := PlayVideo(buf, frames, opts, cancel, nil)
	if err != nil {
		t.Fatalf("PlayVideo() unexpected error: %v", err)
	}
}

func TestPlayVideo_ClosedChannel(t *testing.T) {
	frames := make(chan *ascii.CharFrame)
	cancel := make(chan struct{})
	close(frames)

	buf := new(bytes.Buffer)
	opts := PlaybackOptions{FPS: 24}

	done := make(chan error, 1)
	go func() {
		done <- PlayVideo(buf, frames, opts, cancel, nil)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("PlayVideo() unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("PlayVideo() did not return after channel was closed")
	}
}

func TestGetTerminalSize(t *testing.T) {
	w, h, err := GetTerminalSize()
	if err == nil {
		if w <= 0 || h <= 0 {
			t.Errorf("GetTerminalSize() returned invalid dimensions %dx%d", w, h)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	if got := formatDuration(75 * time.Second); got != "01:15" {
		t.Errorf("formatDuration(75s) = %q, want '01:15'", got)
	}
}


