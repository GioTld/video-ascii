package render

import (
	"fmt"
	"io"
)

// RenderImage escribe un slice de lineas de texto ASCII a un io.Writer.
func RenderImage(w io.Writer, lines []string) error {
	if w == nil {
		return fmt.Errorf("writer cannot be nil")
	}
	for _, line := range lines {
		if _, err := fmt.Fprintln(w, line); err != nil {
			return fmt.Errorf("write ascii line: %w", err)
		}
	}
	return nil
}
