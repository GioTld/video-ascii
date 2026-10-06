package api

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/GioTld/video-ascii/internal/ascii"
	"github.com/GioTld/video-ascii/internal/decode"
	"github.com/GioTld/video-ascii/internal/filter"
	"github.com/GioTld/video-ascii/internal/frame"
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

// ServeHTTP implements http.Handler and adds CORS headers for browser clients.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("/render/image", s.handleRenderImage)
	s.mux.HandleFunc("/render/video", s.handleRenderVideo)
	s.mux.HandleFunc("/upload/video", s.handleUploadVideo)
}

// ServeHTTP implements http.Handler and adds CORS headers for browser clients.
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

	hdMode := parseHDModeParam(r)
	fitMode := parseFitModeParam(r)
	resized, err := resizeForRender(frm, width, height, hdMode, fitMode)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to resize image: %v", err), http.StatusBadRequest)
		return
	}

	filterParam := r.URL.Query().Get("filter")
	filterFn, err := filter.Parse(filterParam)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid filter parameter: %v", err), http.StatusBadRequest)
		return
	}
	if filterFn != nil {
		resized = filterFn(resized)
	}

	ramp := r.URL.Query().Get("ramp")
	colorMode := parseColorModeParam(r)

	conv, err := ascii.NewConverter(ramp)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to initialize ascii converter: %v", err), http.StatusInternalServerError)
		return
	}

	charFrame, err := conv.ConvertFrameHD(resized, hdMode)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to convert frame to ascii: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if err := render.RenderImage(w, charFrame, colorMode); err != nil {
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

	var videoPath string
	if id := r.URL.Query().Get("id"); id != "" {
		p, ok := store.get(id)
		if !ok {
			http.Error(w, "upload not found or expired", http.StatusNotFound)
			return
		}
		videoPath = p
	} else {
		videoPath = r.URL.Query().Get("path")
		if videoPath == "" {
			http.Error(w, "missing 'id' or 'path' query parameter", http.StatusBadRequest)
			return
		}
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
	if height < 0 {
		http.Error(w, "invalid height parameter", http.StatusBadRequest)
		return
	}
	colorMode := parseColorModeParam(r)
	hdMode := parseHDModeParam(r)
	fitMode := parseFitModeParam(r)

	ramp := r.URL.Query().Get("ramp")
	filterParam := r.URL.Query().Get("filter")
	filterFn, err := filter.Parse(filterParam)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid filter parameter: %v", err), http.StatusBadRequest)
		return
	}

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

	conv, err := ascii.NewConverter(ramp)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to initialize ascii converter: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Video-FPS", strconv.FormatFloat(meta.FPS, 'f', 3, 64))

	cancel := make(chan struct{})
	go func() {
		<-r.Context().Done()
		close(cancel)
	}()

	decodeWidth, decodeHeight := width, height
	if hdMode == ascii.HDModeBraille {
		decodeWidth *= 2
		decodeHeight *= 4
	} else if hdMode == ascii.HDModeHalfBlock {
		decodeHeight *= 2
	}
	if decodeHeight == 0 {
		decodeHeight = meta.Height
	}
	decodeWidth = min(meta.Width, decodeWidth*4)
	decodeHeight = min(meta.Height, decodeHeight*4)
	rawFrames, _ := decode.DecodeVideoScaled(videoPath, decodeWidth, decodeHeight, cancel)

	frameDuration := time.Duration(float64(time.Second) / meta.FPS)
	var streamStart time.Time
	var frameIndex int64

	for rawFrame := range rawFrames {
		if r.Context().Err() != nil {
			return
		}
		var deadline time.Time
		if !streamStart.IsZero() {
			deadline = streamStart.Add(time.Duration(frameIndex) * frameDuration)
			frameIndex++
			if !time.Now().Before(deadline.Add(frameDuration)) {
				continue
			}
		}
		resized, err := resizeForRender(rawFrame, width, height, hdMode, fitMode)
		if err != nil {
			continue
		}
		if filterFn != nil {
			resized = filterFn(resized)
		}
		charFrame, err := conv.ConvertFrameHD(resized, hdMode)
		if err != nil {
			continue
		}

		lines := render.FormatFrameANSI(charFrame, colorMode)
		if streamStart.IsZero() {
			streamStart = time.Now()
			frameIndex = 1
		} else {
			if !time.Now().Before(deadline.Add(frameDuration)) {
				continue
			}
			if wait := time.Until(deadline); wait > 0 {
				select {
				case <-time.After(wait):
				case <-r.Context().Done():
					return
				}
			}
		}

		// Formato SSE: "data: <línea>\n" por cada línea, separado por "\n\n".
		fmt.Fprint(w, "event: frame\n")
		for _, line := range lines {
			fmt.Fprintf(w, "data: %s\n", line)
		}
		fmt.Fprint(w, "\n")
		flusher.Flush()
	}
}

func parseColorModeParam(r *http.Request) render.ColorMode {
	colorStr := r.URL.Query().Get("color")
	if colorStr == "true" || colorStr == "1" {
		modeStr := strings.ToLower(r.URL.Query().Get("color_mode"))
		if modeStr == "" {
			modeStr = strings.ToLower(r.URL.Query().Get("color-mode"))
		}
		if modeStr == "256" {
			return render.ColorMode256
		}
		return render.ColorMode24Bit
	}
	return render.ColorModeNone
}

func parseHDModeParam(r *http.Request) ascii.HDMode {
	switch r.URL.Query().Get("hd_mode") {
	case string(ascii.HDModeHalfBlock):
		return ascii.HDModeHalfBlock
	case string(ascii.HDModeBraille):
		return ascii.HDModeBraille
	default:
		return ascii.HDModeNone
	}
}

func parseFitModeParam(r *http.Request) frame.FitMode {
	if r.URL.Query().Get("fit_mode") == string(frame.FitModeCover) {
		return frame.FitModeCover
	}
	return frame.FitModeContain
}

func resizeForRender(src *frame.Frame, width, height int, hdMode ascii.HDMode, fitMode frame.FitMode) (*frame.ResizedFrame, error) {
	fontRatio := 0.5
	if hdMode == ascii.HDModeHalfBlock {
		fontRatio = 1.0
	} else if hdMode == ascii.HDModeBraille {
		fontRatio = 2.0
	}
	if hdMode == ascii.HDModeBraille {
		width *= 2
	}
	if hdMode == ascii.HDModeHalfBlock {
		height *= 2
	} else if hdMode == ascii.HDModeBraille {
		height *= 4
	}
	if height == 0 {
		return src.Resize(width, 0, fontRatio)
	}

	sourceAspect := float64(src.Height) / float64(src.Width) * fontRatio
	targetAspect := float64(height) / float64(width)
	var targetW, targetH int
	if fitMode == frame.FitModeCover {
		if sourceAspect > targetAspect {
			targetW = width
			targetH = int(math.Ceil(float64(width) * sourceAspect))
		} else {
			targetH = height
			targetW = int(math.Ceil(float64(height) / sourceAspect))
		}
	} else if sourceAspect > targetAspect {
		targetH = height
		targetW = int(math.Floor(float64(height) / sourceAspect))
	} else {
		targetW = width
		targetH = int(math.Floor(float64(width) * sourceAspect))
	}
	targetW = max(targetW, 1)
	targetH = max(targetH, 1)
	resized, err := src.Resize(targetW, targetH, fontRatio)
	if err != nil {
		return nil, err
	}
	if fitMode == frame.FitModeCover {
		return resized.Crop((targetW-width)/2, (targetH-height)/2, width, height), nil
	}
	return resized, nil
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
