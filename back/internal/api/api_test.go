package api

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

func createTestPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	buf := new(bytes.Buffer)
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}
	if err := png.Encode(buf, img); err != nil {
		t.Fatalf("failed to encode PNG: %v", err)
	}
	return buf.Bytes()
}

func TestServer_RenderImage(t *testing.T) {
	srv := NewServer()
	pngData := createTestPNG(t, 20, 20)

	t.Run("GET method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/render/image", nil)
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("got status %d, want %d", rec.Code, http.StatusMethodNotAllowed)
		}
	})

	t.Run("invalid width query param", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/render/image?width=-5", bytes.NewReader(pngData))
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("got status %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("valid raw image body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/render/image?width=10", bytes.NewReader(pngData))
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d", rec.Code, http.StatusOK)
		}

		contentType := rec.Header().Get("Content-Type")
		if contentType != "text/plain; charset=utf-8" {
			t.Errorf("got Content-Type %q, want 'text/plain; charset=utf-8'", contentType)
		}

		if rec.Body.Len() == 0 {
			t.Error("response body is empty")
		}
	})

	t.Run("valid multipart form upload", func(t *testing.T) {
		body := new(bytes.Buffer)
		mw := multipart.NewWriter(body)
		fw, err := mw.CreateFormFile("image", "sample.png")
		if err != nil {
			t.Fatalf("failed to create form file: %v", err)
		}
		if _, err := fw.Write(pngData); err != nil {
			t.Fatalf("failed to write png data: %v", err)
		}
		mw.Close()

		req := httptest.NewRequest(http.MethodPost, "/render/image?width=10", body)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d", rec.Code, http.StatusOK)
		}

		if rec.Body.Len() == 0 {
			t.Error("response body is empty")
		}
	})
}
