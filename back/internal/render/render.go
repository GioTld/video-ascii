package render

import (
	"fmt"
	"io"
	"strings"
	"sync/atomic"
	"time"
)

// RenderImage writes a slice of ASCII text lines to an io.Writer.
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

// ClearScreen emite la secuencia ANSI para limpiar la pantalla y
// reposicionar el cursor en la esquina superior izquierda.
func ClearScreen(w io.Writer) error {
	_, err := fmt.Fprint(w, "\033[2J\033[H")
	return err
}

// HideCursor oculta el cursor del terminal durante la reproducción.
func HideCursor(w io.Writer) error {
	_, err := fmt.Fprint(w, "\033[?25l")
	return err
}

// ShowCursor restaura la visibilidad del cursor al finalizar la reproducción.
func ShowCursor(w io.Writer) error {
	_, err := fmt.Fprint(w, "\033[?25h")
	return err
}

// PlaybackOptions configura la reproducción en terminal.
type PlaybackOptions struct {
	FPS    float64
	Width  int
	Height int
}

// PlayVideo reproduce una secuencia de fotogramas ASCII en el terminal.
// Recibe fotogramas por el canal y los dibuja a la tasa de FPS indicada.
// paused puede ser nil; si no lo es, cuando está en true el loop espera
// sin consumir el canal hasta que cambie a false.
// Se detiene cuando el canal se cierra o cancel se señaliza.
func PlayVideo(w io.Writer, frames <-chan []string, opts PlaybackOptions, cancel <-chan struct{}, paused *atomic.Bool) error {
	if opts.FPS <= 0 {
		opts.FPS = 24
	}
	frameDuration := time.Duration(float64(time.Second) / opts.FPS)

	if err := HideCursor(w); err != nil {
		return err
	}
	defer ShowCursor(w) //nolint:errcheck

	if err := ClearScreen(w); err != nil {
		return err
	}

	var sb strings.Builder

	for {
		// Esperar mientras esté en pausa sin consumir fotogramas del canal.
		for paused != nil && paused.Load() {
			select {
			case <-cancel:
				return nil
			case <-time.After(50 * time.Millisecond):
			}
		}

		start := time.Now()

		select {
		case <-cancel:
			return nil
		case lines, ok := <-frames:
			if !ok {
				return nil
			}

			// Construir el fotograma completo en memoria antes de escribirlo
			// para minimizar el tiempo de escritura y reducir el parpadeo.
			sb.Reset()
			sb.WriteString("\033[H") // cursor home
			for _, line := range lines {
				sb.WriteString(line)
				sb.WriteByte('\n')
			}

			if _, err := fmt.Fprint(w, sb.String()); err != nil {
				return fmt.Errorf("write frame: %w", err)
			}

			// Dormir el tiempo restante para mantener el framerate.
			elapsed := time.Since(start)
			if sleep := frameDuration - elapsed; sleep > 0 {
				select {
				case <-time.After(sleep):
				case <-cancel:
					return nil
				}
			}
		}
	}
}
