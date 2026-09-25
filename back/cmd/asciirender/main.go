package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"

	"github.com/GioTld/video-ascii/internal/api"
	"github.com/GioTld/video-ascii/internal/ascii"
	"github.com/GioTld/video-ascii/internal/decode"
	"github.com/GioTld/video-ascii/internal/filter"
	"github.com/GioTld/video-ascii/internal/render"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "serve" {
		runServe(os.Args[2:])
		return
	}

	runCLI(os.Args[1:])
}

func runServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	port := fs.Int("port", 8080, "HTTP server port")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing serve flags: %v\n", err)
		os.Exit(1)
	}

	srv := api.NewServer()
	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Starting ASCII rendering HTTP server on http://localhost%s\n", addr)
	if err := http.ListenAndServe(addr, srv); err != nil {
		log.Fatalf("Server failure: %v", err)
	}
}

func runCLI(args []string) {
	fs := flag.NewFlagSet("asciirender", flag.ExitOnError)
	width := fs.Int("width", 80, "target output width in characters")
	height := fs.Int("height", 0, "target output height in characters (0 = auto)")
	ramp := fs.String("ramp", "", "custom character ramp (light to dark)")
	outputPath := fs.String("output", "", "output file path (default: stdout)")
	filterFlag := fs.String("filter", "", "comma-separated filters: brightness=1.2, contrast=1.2, sepia, invert, matrix, grayscale, edge")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage:\n  asciirender [opciones] <ruta-archivo>\n  asciirender serve [opciones]\n\nOpciones:\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if fs.NArg() < 1 {
		fs.Usage()
		os.Exit(1)
	}

	filterFn, err := filter.Parse(*filterFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing filters: %v\n", err)
		os.Exit(1)
	}

	inputPath := fs.Arg(0)

	if strings.ToLower(filepath.Ext(inputPath)) == ".mp4" {
		if err := runVideo(inputPath, *width, *height, *ramp, filterFn); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if err := runImage(inputPath, *width, *height, *ramp, *outputPath, filterFn); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runImage(path string, width, height int, ramp, outputPath string, filterFn filter.Func) error {
	frm, err := decode.DecodeFile(path)
	if err != nil {
		return fmt.Errorf("decode image: %w", err)
	}

	resized, err := frm.Resize(width, height, 0.5)
	if err != nil {
		return fmt.Errorf("resize: %w", err)
	}

	if filterFn != nil {
		resized = filterFn(resized)
	}

	conv, err := ascii.NewConverter(ramp)
	if err != nil {
		return fmt.Errorf("ascii converter: %w", err)
	}

	lines, err := conv.ConvertFrame(resized)
	if err != nil {
		return fmt.Errorf("convert to ascii: %w", err)
	}

	var out io.Writer = os.Stdout
	if outputPath != "" {
		f, err := os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("create output file: %w", err)
		}
		defer f.Close()
		out = f
	}

	return render.RenderImage(out, lines)
}

func runVideo(path string, width, height int, ramp string, filterFn filter.Func) error {
	meta, err := decode.ProbeVideo(path)
	if err != nil {
		return fmt.Errorf("probe video: %w", err)
	}

	conv, err := ascii.NewConverter(ramp)
	if err != nil {
		return fmt.Errorf("ascii converter: %w", err)
	}

	restoreTerminal, err := render.EnableRawMode()
	if err != nil {
		return fmt.Errorf("enable raw mode: %w", err)
	}
	defer restoreTerminal()

	cancel := make(chan struct{})
	var paused atomic.Bool

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sig
		close(cancel)
	}()

	go readKeys(cancel, &paused)

	rawFrames, errc := decode.DecodeVideo(path, cancel)

	asciiFrames := make(chan []string, 4)

	go func() {
		defer close(asciiFrames)
		for f := range rawFrames {
			resized, err := f.Resize(width, height, 0.5)
			if err != nil {
				continue
			}
			if filterFn != nil {
				resized = filterFn(resized)
			}
			lines, err := conv.ConvertFrame(resized)
			if err != nil {
				continue
			}
			select {
			case asciiFrames <- lines:
			case <-cancel:
				return
			}
		}
	}()

	opts := render.PlaybackOptions{
		FPS:    meta.FPS,
		Width:  width,
		Height: height,
	}

	if err := render.PlayVideo(os.Stdout, asciiFrames, opts, cancel, &paused); err != nil {
		return fmt.Errorf("playback: %w", err)
	}

	if err := <-errc; err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	return nil
}

func readKeys(cancel chan struct{}, paused *atomic.Bool) {
	buf := make([]byte, 1)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			return
		}
		switch buf[0] {
		case ' ':
			paused.Store(!paused.Load())
		case 'q', 'Q', 3:
			select {
			case <-cancel:
			default:
				close(cancel)
			}
			return
		}

		select {
		case <-cancel:
			return
		default:
		}
	}
}
