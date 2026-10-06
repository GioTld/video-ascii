package api

import (
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type slowFrameRecorder struct {
	*httptest.ResponseRecorder
	delay   time.Duration
	flushes int
}

func (w *slowFrameRecorder) Flush() {
	w.flushes++
	w.ResponseRecorder.Flush()
	time.Sleep(w.delay)
}

func TestRenderVideoDropsLateFrames(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg is not installed")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe is not installed")
	}

	path := filepath.Join(t.TempDir(), "pacing.mp4")
	cmd := exec.Command("ffmpeg", "-loglevel", "error", "-f", "lavfi", "-i", "testsrc=size=160x90:rate=10:duration=1.2", "-c:v", "mpeg4", "-y", path)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("create video: %v: %s", err, output)
	}

	request := httptest.NewRequest(http.MethodGet, "/render/video?path="+path+"&width=40&height=20", nil)
	response := &slowFrameRecorder{ResponseRecorder: httptest.NewRecorder(), delay: 180 * time.Millisecond}
	start := time.Now()
	NewServer().ServeHTTP(response, request)
	t.Logf("streamed %d of 12 frames in %s", response.flushes, time.Since(start))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %q", response.Code, response.Body.String())
	}
	if response.flushes < 2 || response.flushes >= 12 {
		t.Errorf("got %d streamed frames; expected late frames to be skipped", response.flushes)
	}
	if got := strings.Count(response.Body.String(), "event: frame\n"); got != response.flushes {
		t.Errorf("got %d frame events for %d flushes", got, response.flushes)
	}
}
