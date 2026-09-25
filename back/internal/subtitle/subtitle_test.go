package subtitle

import (
	"strings"
	"testing"
	"time"
)

func TestParseSRT(t *testing.T) {
	srtData := `1
00:00:01,000 --> 00:00:03,500
First subtitle line

2
00:00:05,200 --> 00:00:08,000
Second subtitle line
Second subtitle line part 2
`

	entries, err := ParseSRT(strings.NewReader(srtData))
	if err != nil {
		t.Fatalf("ParseSRT() unexpected error: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}

	if entries[0].Start != 1*time.Second || entries[0].End != 3500*time.Millisecond {
		t.Errorf("entry 0 times mismatch: got %v -> %v", entries[0].Start, entries[0].End)
	}

	if entries[0].Text != "First subtitle line" {
		t.Errorf("entry 0 text mismatch: got %q", entries[0].Text)
	}

	if entries[1].Start != 5200*time.Millisecond || entries[1].End != 8*time.Second {
		t.Errorf("entry 1 times mismatch: got %v -> %v", entries[1].Start, entries[1].End)
	}
}

func TestActiveAt(t *testing.T) {
	entries := []Entry{
		{
			Start: 1 * time.Second,
			End:   3 * time.Second,
			Text:  "Active Subtitle",
		},
	}

	tests := []struct {
		timestamp time.Duration
		wantText  string
	}{
		{500 * time.Millisecond, ""},
		{1 * time.Second, "Active Subtitle"},
		{2 * time.Second, "Active Subtitle"},
		{3 * time.Second, "Active Subtitle"},
		{4 * time.Second, ""},
	}

	for _, tt := range tests {
		t.Run(tt.timestamp.String(), func(t *testing.T) {
			got := ActiveAt(entries, tt.timestamp)
			if got != tt.wantText {
				t.Errorf("ActiveAt(%v) = %q, want %q", tt.timestamp, got, tt.wantText)
			}
		})
	}
}
