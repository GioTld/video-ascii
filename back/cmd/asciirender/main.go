package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/GioTld/video-ascii/internal/api"
	"github.com/GioTld/video-ascii/internal/ascii"
	"github.com/GioTld/video-ascii/internal/decode"
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
	height := fs.Int("height", 0, "target output height in characters (0 for auto aspect ratio)")
	ramp := fs.String("ramp", "", "custom character ramp string (light to dark)")
	outputPath := fs.String("output", "", "output file path (default standard output)")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of asciirender:\n")
		fmt.Fprintf(os.Stderr, "  asciirender [options] <image-path>\n")
		fmt.Fprintf(os.Stderr, "  asciirender serve [options]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if fs.NArg() < 1 {
		fs.Usage()
		os.Exit(1)
	}

	imagePath := fs.Arg(0)

	frm, err := decode.DecodeFile(imagePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding image: %v\n", err)
		os.Exit(1)
	}

	resized, err := frm.Resize(*width, *height, 0.5)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resizing frame: %v\n", err)
		os.Exit(1)
	}

	conv, err := ascii.NewConverter(*ramp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating ascii converter: %v\n", err)
		os.Exit(1)
	}

	lines, err := conv.ConvertFrame(resized)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error converting frame to ascii: %v\n", err)
		os.Exit(1)
	}

	var out io.Writer = os.Stdout
	if *outputPath != "" {
		f, err := os.Create(*outputPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		out = f
	}

	if err := render.RenderImage(out, lines); err != nil {
		fmt.Fprintf(os.Stderr, "Error rendering output: %v\n", err)
		os.Exit(1)
	}
}
