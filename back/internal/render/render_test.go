package render

import (
	"bytes"
	"testing"
	"time"
)

func TestRenderImage(t *testing.T) {
	tests := []struct {
		name       string
		lines      []string
		wantOutput string
		wantErr    bool
	}{
		{
			name:       "nil lines produces empty output",
			lines:      nil,
			wantOutput: "",
			wantErr:    false,
		},
		{
			name:       "renders lines with newlines",
			lines:      []string{"  ..", "@@##"},
			wantOutput: "  ..\n@@##\n",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			err := RenderImage(buf, tt.lines)
			if (err != nil) != tt.wantErr {
				t.Fatalf("RenderImage() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && buf.String() != tt.wantOutput {
				t.Errorf("got %q, want %q", buf.String(), tt.wantOutput)
			}
		})
	}

	t.Run("nil writer returns error", func(t *testing.T) {
		if err := RenderImage(nil, []string{"test"}); err == nil {
			t.Error("expected error for nil writer, got nil")
		}
	})
}

func TestPlayVideo_CancelImmediately(t *testing.T) {
	frames := make(chan []string)
	cancel := make(chan struct{})
	close(cancel)

	buf := new(bytes.Buffer)
	opts := PlaybackOptions{FPS: 24}

	err := PlayVideo(buf, frames, opts, cancel, nil)
	if err != nil {
		t.Fatalf("PlayVideo() unexpected error: %v", err)
	}
}

func TestPlayVideo_ClosedChannel(t *testing.T) {
	frames := make(chan []string)
	cancel := make(chan struct{})
	close(frames)

	buf := new(bytes.Buffer)
	opts := PlaybackOptions{FPS: 24}

	done := make(chan error, 1)
	go func() {
		done <- PlayVideo(buf, frames, opts, cancel, nil)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("PlayVideo() unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("PlayVideo() did not return after channel was closed")
	}
}
