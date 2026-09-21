package api

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/GioTld/video-ascii/internal/ascii"
	"github.com/GioTld/video-ascii/internal/decode"
	"github.com/GioTld/video-ascii/internal/render"
)

// Server encapsulates HTTP handler routing for the ASCII renderer API.
type Server struct {
	mux *http.ServeMux
}

// NewServer creates and wires a new API Server instance.
func NewServer() *Server {
	s := &Server{
		mux: http.NewServeMux(),
	}
	s.routes()
	return s
}

// ServeHTTP implements the http.Handler interface.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("/render/image", s.handleRenderImage)
	s.mux.HandleFunc("/render/video", s.handleRenderVideo)
}

func (s *Server) handleRenderImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	width := parseIntParam(r, "width", 80)
	if width <= 0 {
		http.Error(w, "invalid width parameter", http.StatusBadRequest)
		return
	}

	height := parseIntParam(r, "height", 0)
	if height < 0 {
		http.Error(w, "invalid height parameter", http.StatusBadRequest)
		return
	}

	reader, cleanup, err := imageReaderFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if cleanup != nil {
		defer cleanup()
	}

	frm, err := decode.DecodeImage(reader)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to decode image: %v", err), http.StatusBadRequest)
		return
	}

	resized, err := frm.Resize(width, height, 0.5)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to resize image: %v", err), http.StatusBadRequest)
		return
	}

	conv, err := ascii.NewConverter("")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to initialize ascii converter: %v", err), http.StatusInternalServerError)
		return
	}

	lines, err := conv.ConvertFrame(resized)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to convert frame to ascii: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if err := render.RenderImage(w, lines); err != nil {
		http.Error(w, fmt.Sprintf("failed to render output: %v", err), http.StatusInternalServerError)
		return
	}
}

// handleRenderVideo acepta la ruta de un MP4 por query param y responde con
// Server-Sent Events: un evento por fotograma con el arte ASCII como dato.
// El cliente puede interrumpir la conexión para detener la extracción.
func (s *Server) handleRenderVideo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	videoPath := r.URL.Query().Get("path")
	if videoPath == "" {
		http.Error(w, "missing 'path' query parameter", http.StatusBadRequest)
		return
	}
	if strings.ToLower(filepath.Ext(videoPath)) != ".mp4" {
		http.Error(w, "only .mp4 files are supported", http.StatusBadRequest)
		return
	}

	width := parseIntParam(r, "width", 80)
	if width <= 0 {
		http.Error(w, "invalid width parameter", http.StatusBadRequest)
		return
	}
	height := parseIntParam(r, "height", 0)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	meta, err := decode.ProbeVideo(videoPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to probe video: %v", err), http.StatusBadRequest)
		return
	}

	conv, err := ascii.NewConverter("")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to initialize ascii converter: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Video-FPS", strconv.FormatFloat(meta.FPS, 'f', 3, 64))

	cancel := make(chan struct{})
	defer close(cancel)

	// Detectar cuando el cliente cierra la conexión.
	go func() {
		<-r.Context().Done()
		select {
		case <-cancel:
		default:
		}
	}()

	rawFrames, _ := decode.DecodeVideo(videoPath, cancel)

	frameDuration := time.Duration(float64(time.Second) / meta.FPS)

	for rawFrame := range rawFrames {
		resized, err := rawFrame.Resize(width, height, 0.5)
		if err != nil {
			continue
		}
		lines, err := conv.ConvertFrame(resized)
		if err != nil {
			continue
		}

		// Formato SSE: "data: <línea>\n" por cada línea, separado por "\n\n".
		start := time.Now()

		fmt.Fprint(w, "event: frame\n")
		for _, line := range lines {
			fmt.Fprintf(w, "data: %s\n", line)
		}
		fmt.Fprint(w, "\n")
		flusher.Flush()

		// Respetar el framerate del video también en el stream HTTP.
		if sleep := frameDuration - time.Since(start); sleep > 0 {
			select {
			case <-time.After(sleep):
			case <-r.Context().Done():
				return
			}
		}
	}
}

// parseIntParam lee un parámetro entero de la query string; si falta o es
// inválido devuelve el valor por defecto.
func parseIntParam(r *http.Request, key string, def int) int {
	s := r.URL.Query().Get(key)
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return -1 // inválido; el caller decide qué hacer
	}
	return v
}

// imageReaderFromRequest extrae un io.Reader con la imagen desde el cuerpo
// de la petición: primero intenta multipart form, luego cuerpo raw.
func imageReaderFromRequest(r *http.Request) (io.Reader, func(), error) {
	if err := r.ParseMultipartForm(10 << 20); err == nil && r.MultipartForm != nil {
		file, _, err := r.FormFile("image")
		if err != nil {
			file, _, err = r.FormFile("file")
		}
		if err != nil {
			return nil, nil, fmt.Errorf("missing 'image' or 'file' form field")
		}
		return file, func() { file.Close() }, nil
	}
	return r.Body, nil, nil
}
