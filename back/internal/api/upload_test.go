package api

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleUploadVideo_MethodNotAllowed(t *testing.T) {
	srv := NewServer()
	req := httptest.NewRequest(http.MethodGet, "/upload/video", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("got %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleUploadVideo_MissingFile(t *testing.T) {
	srv := NewServer()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	w.Close()

	req := httptest.NewRequest(http.MethodPost, "/upload/video", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("got %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleUploadVideo_WrongExtension(t *testing.T) {
	srv := NewServer()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, _ := w.CreateFormFile("file", "clip.avi")
	part.Write([]byte("fake avi data")) //nolint:errcheck
	w.Close()

	req := httptest.NewRequest(http.MethodPost, "/upload/video", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("got %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleUploadVideo_ValidFile(t *testing.T) {
	srv := NewServer()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, _ := w.CreateFormFile("file", "clip.mp4")
	part.Write([]byte("fake mp4 data")) //nolint:errcheck
	w.Close()

	req := httptest.NewRequest(http.MethodPost, "/upload/video", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("got %d, want %d", rec.Code, http.StatusOK)
	}

	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["id"] == "" {
		t.Error("expected non-empty id in response")
	}
}

func TestHandleUploadVideo_StoreAndExpiry(t *testing.T) {
	id := "test-upload-id"
	store.put(id, "/tmp/nonexistent.mp4")

	path, ok := store.get(id)
	if !ok {
		t.Fatal("expected store to have entry")
	}
	if path != "/tmp/nonexistent.mp4" {
		t.Errorf("got path %q, want /tmp/nonexistent.mp4", path)
	}

	store.remove(id)
	_, ok = store.get(id)
	if ok {
		t.Error("expected entry to be removed")
	}
}

func TestCORSHeaders(t *testing.T) {
	srv := NewServer()
	req := httptest.NewRequest(http.MethodOptions, "/render/image", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Errorf("OPTIONS got %d, want %d", rec.Code, http.StatusNoContent)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("CORS origin = %q, want *", got)
	}
}
