package render

import (
	"bytes"
	"testing"
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
