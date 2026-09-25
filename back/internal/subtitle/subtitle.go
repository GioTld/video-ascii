package subtitle

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// Entry represents a single subtitle entry with start/end duration window and text.
type Entry struct {
	Start time.Duration
	End   time.Duration
	Text  string
}

// ParseSRT reads and parses subtitle data in SRT format from an io.Reader.
func ParseSRT(r io.Reader) ([]Entry, error) {
	if r == nil {
		return nil, fmt.Errorf("reader cannot be nil")
	}

	scanner := bufio.NewScanner(r)
	var entries []Entry

	var currentText []string
	var startDur, endDur time.Duration
	inTiming := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			if inTiming && len(currentText) > 0 {
				entries = append(entries, Entry{
					Start: startDur,
					End:   endDur,
					Text:  strings.Join(currentText, "\n"),
				})
				currentText = nil
				inTiming = false
			}
			continue
		}

		if strings.Contains(line, "-->") {
			parts := strings.Split(line, "-->")
			if len(parts) == 2 {
				s, err1 := parseSRTTime(strings.TrimSpace(parts[0]))
				e, err2 := parseSRTTime(strings.TrimSpace(parts[1]))
				if err1 == nil && err2 == nil {
					startDur = s
					endDur = e
					inTiming = true
					currentText = nil
					continue
				}
			}
		}

		if inTiming {
			currentText = append(currentText, line)
		}
	}

	if inTiming && len(currentText) > 0 {
		entries = append(entries, Entry{
			Start: startDur,
			End:   endDur,
			Text:  strings.Join(currentText, "\n"),
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan srt: %w", err)
	}

	return entries, nil
}

// parseSRTTime parses a timestamp string like "HH:MM:SS,mmm" or "HH:MM:SS.mmm" into time.Duration.
func parseSRTTime(s string) (time.Duration, error) {
	s = strings.ReplaceAll(s, ",", ".")
	parts := strings.Split(s, ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid time format %q", s)
	}

	h, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, err
	}
	m, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, err
	}

	secParts := strings.Split(parts[2], ".")
	sec, err := strconv.Atoi(secParts[0])
	if err != nil {
		return 0, err
	}

	ms := 0
	if len(secParts) > 1 {
		msStr := secParts[1]
		for len(msStr) < 3 {
			msStr += "0"
		}
		if len(msStr) > 3 {
			msStr = msStr[:3]
		}
		ms, _ = strconv.Atoi(msStr)
	}

	return time.Duration(h)*time.Hour +
		time.Duration(m)*time.Minute +
		time.Duration(sec)*time.Second +
		time.Duration(ms)*time.Millisecond, nil
}

// ActiveAt returns the active subtitle text at a given playback timestamp t, or empty string.
func ActiveAt(entries []Entry, t time.Duration) string {
	for _, e := range entries {
		if t >= e.Start && t <= e.End {
			return e.Text
		}
	}
	return ""
}
