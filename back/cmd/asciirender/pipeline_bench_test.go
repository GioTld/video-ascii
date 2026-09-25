package main

import (
	"image"
	"image/color"
	"runtime"
	"testing"
	"time"

	"github.com/GioTld/video-ascii/internal/ascii"
	"github.com/GioTld/video-ascii/internal/filter"
	"github.com/GioTld/video-ascii/internal/frame"
	"github.com/GioTld/video-ascii/internal/render"
)

func create1080pImage() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 1920, 1080))
	for y := 0; y < 1080; y++ {
		for x := 0; x < 1920; x++ {
			img.SetRGBA(x, y, color.RGBA{
				R: uint8(x % 256),
				G: uint8(y % 256),
				B: uint8((x + y) % 256),
				A: 255,
			})
		}
	}
	return img
}

func BenchmarkResize_1080p(b *testing.B) {
	img := create1080pImage()
	f, _ := frame.New(img)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := f.Resize(80, 40, 0.5)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFilter_Chain(b *testing.B) {
	img := create1080pImage()
	f, _ := frame.New(img)
	rf, _ := f.Resize(80, 40, 0.5)

	chain := filter.Chain(
		filter.Brightness(1.2),
		filter.Contrast(1.2),
		filter.Sepia(),
	)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = chain(rf)
	}
}

func BenchmarkConvertFrame_Standard(b *testing.B) {
	img := create1080pImage()
	f, _ := frame.New(img)
	rf, _ := f.Resize(80, 40, 0.5)
	conv, _ := ascii.NewConverter("")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := conv.ConvertFrame(rf)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConvertFrame_HalfBlock(b *testing.B) {
	img := create1080pImage()
	f, _ := frame.New(img)
	rf, _ := f.Resize(80, 40, 0.5)
	conv, _ := ascii.NewConverter("")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := conv.ConvertFrameHD(rf, ascii.HDModeHalfBlock)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConvertFrame_Braille(b *testing.B) {
	img := create1080pImage()
	f, _ := frame.New(img)
	rf, _ := f.Resize(80, 40, 0.5)
	conv, _ := ascii.NewConverter("")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := conv.ConvertFrameHD(rf, ascii.HDModeBraille)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFormatANSI_24Bit(b *testing.B) {
	img := create1080pImage()
	f, _ := frame.New(img)
	rf, _ := f.Resize(80, 40, 0.5)
	conv, _ := ascii.NewConverter("")
	cf, _ := conv.ConvertFrame(rf)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = render.FormatFrameANSI(cf, render.ColorMode24Bit)
	}
}

func BenchmarkFormatANSI_256(b *testing.B) {
	img := create1080pImage()
	f, _ := frame.New(img)
	rf, _ := f.Resize(80, 40, 0.5)
	conv, _ := ascii.NewConverter("")
	cf, _ := conv.ConvertFrame(rf)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = render.FormatFrameANSI(cf, render.ColorMode256)
	}
}

func BenchmarkFullPipeline_PerFrame(b *testing.B) {
	img := create1080pImage()
	f, _ := frame.New(img)
	chain := filter.Chain(
		filter.Brightness(1.2),
		filter.Contrast(1.2),
	)
	conv, _ := ascii.NewConverter("")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rf, _ := f.Resize(80, 40, 0.5)
		filtered := chain(rf)
		cf, _ := conv.ConvertFrame(filtered)
		_ = render.FormatFrameANSI(cf, render.ColorMode24Bit)
	}
}

func TestContinuousPlaybackMetrics(t *testing.T) {
	img := create1080pImage()
	f, err := frame.New(img)
	if err != nil {
		t.Fatalf("frame.New failed: %v", err)
	}

	chain := filter.Chain(
		filter.Brightness(1.2),
		filter.Contrast(1.2),
	)
	conv, err := ascii.NewConverter("")
	if err != nil {
		t.Fatalf("NewConverter failed: %v", err)
	}

	const frameCount = 150 // 5 seconds at 30 fps
	start := time.Now()

	for i := 0; i < frameCount; i++ {
		rf, err := f.Resize(80, 40, 0.5)
		if err != nil {
			t.Fatalf("Resize failed: %v", err)
		}
		filtered := chain(rf)
		cf, err := conv.ConvertFrame(filtered)
		if err != nil {
			t.Fatalf("ConvertFrame failed: %v", err)
		}
		_ = render.FormatFrameANSI(cf, render.ColorMode24Bit)
	}

	totalDuration := time.Since(start)
	avgPerFrame := totalDuration / time.Duration(frameCount)
	frameBudget := 33333 * time.Microsecond
	estimatedCPUPercent := float64(avgPerFrame) / float64(frameBudget) * 100

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	heapAllocMB := float64(m.HeapAlloc) / (1024 * 1024)
	heapInuseMB := float64(m.HeapInuse) / (1024 * 1024)
	sysMB := float64(m.Sys) / (1024 * 1024)

	t.Logf("Continuous playback results for %d frames:", frameCount)
	t.Logf("  Avg time per frame: %v", avgPerFrame)
	t.Logf("  Estimated CPU load at 30 FPS: %.2f%% (requirement: < 20%%)", estimatedCPUPercent)
	t.Logf("  HeapAlloc: %.2f MB", heapAllocMB)
	t.Logf("  HeapInuse: %.2f MB (requirement: < 60 MB)", heapInuseMB)
	t.Logf("  Sys: %.2f MB", sysMB)

	if estimatedCPUPercent > 20.0 {
		t.Errorf("estimated CPU load %.2f%% exceeds 20%% requirement", estimatedCPUPercent)
	}
	if heapInuseMB > 60.0 {
		t.Errorf("heap in use %.2f MB exceeds 60 MB requirement", heapInuseMB)
	}
}

