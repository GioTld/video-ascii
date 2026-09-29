package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// uploadStore holds paths to temporary uploaded video files, keyed by a short ID.
// Files are removed automatically after a TTL.
type uploadStore struct {
	mu      sync.Mutex
	entries map[string]uploadEntry
}

type uploadEntry struct {
	path      string
	expiresAt time.Time
}

const uploadTTL = 10 * time.Minute

var store = &uploadStore{
	entries: make(map[string]uploadEntry),
}

// put registers a temp file path under id and schedules deletion after TTL.
func (s *uploadStore) put(id, path string) {
	s.mu.Lock()
	s.entries[id] = uploadEntry{path: path, expiresAt: time.Now().Add(uploadTTL)}
	s.mu.Unlock()

	time.AfterFunc(uploadTTL, func() {
		s.remove(id)
	})
}

// get returns the file path for id if it exists and has not expired.
func (s *uploadStore) get(id string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.entries[id]
	if !ok || time.Now().After(e.expiresAt) {
		return "", false
	}
	return e.path, true
}

// remove deletes the temp file and its store entry.
func (s *uploadStore) remove(id string) {
	s.mu.Lock()
	e, ok := s.entries[id]
	if ok {
		delete(s.entries, id)
	}
	s.mu.Unlock()
	if ok {
		os.Remove(e.path) //nolint:errcheck
	}
}

// handleUploadVideo accepts a multipart video file, writes it to a temp file,
// and returns a JSON body with an "id" field for use with /render/video?id=...
func (s *Server) handleUploadVideo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 500 MB max
	if err := r.ParseMultipartForm(500 << 20); err != nil {
		http.Error(w, "failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing 'file' form field", http.StatusBadRequest)
		return
	}
	defer file.Close()

	ext := filepath.Ext(header.Filename)
	if ext != ".mp4" {
		http.Error(w, "only .mp4 files are supported", http.StatusBadRequest)
		return
	}

	tmp, err := os.CreateTemp("", "video-ascii-upload-*"+ext)
	if err != nil {
		http.Error(w, "failed to create temp file", http.StatusInternalServerError)
		return
	}
	defer tmp.Close()

	if _, err := io.Copy(tmp, file); err != nil {
		os.Remove(tmp.Name()) //nolint:errcheck
		http.Error(w, fmt.Sprintf("failed to write upload: %v", err), http.StatusInternalServerError)
		return
	}

	id := filepath.Base(tmp.Name())
	store.put(id, tmp.Name())

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"id": id}) //nolint:errcheck
}
