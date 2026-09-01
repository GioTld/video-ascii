package api

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/GioTld/video-ascii/internal/ascii"
	"github.com/GioTld/video-ascii/internal/decode"
	"github.com/GioTld/video-ascii/internal/render"
)

type Server struct {
	mux *http.ServeMux
}

func NewServer() *Server {
	s := &Server{
		mux: http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("/render/image", s.handleRenderImage)
}

func (s *Server) handleRenderImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	width := 80
	if wStr := r.URL.Query().Get("width"); wStr != "" {
		parsedW, err := strconv.Atoi(wStr)
		if err != nil || parsedW <= 0 {
			http.Error(w, "invalid width parameter", http.StatusBadRequest)
			return
		}
		width = parsedW
	}

	height := 0
	if hStr := r.URL.Query().Get("height"); hStr != "" {
		parsedH, err := strconv.Atoi(hStr)
		if err != nil || parsedH < 0 {
			http.Error(w, "invalid height parameter", http.StatusBadRequest)
			return
		}
		height = parsedH
	}

	var reader io.Reader

	if err := r.ParseMultipartForm(10 << 20); err == nil && r.MultipartForm != nil {
		file, _, err := r.FormFile("image")
		if err != nil {
			file, _, err = r.FormFile("file")
		}
		if err != nil {
			http.Error(w, "missing 'image' or 'file' form field", http.StatusBadRequest)
			return
		}
		defer file.Close()
		reader = file
	} else {
		reader = r.Body
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
