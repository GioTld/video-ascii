package render

import (
	"fmt"
	"image/color"
	"io"
	"math"
	"strings"
	"sync/atomic"
	"time"

	"github.com/GioTld/video-ascii/internal/ascii"
	"github.com/GioTld/video-ascii/internal/frame"
	"github.com/GioTld/video-ascii/internal/subtitle"
)

// ColorMode define la modalidad de color ANSI para renderizar artefactos ASCII.
type ColorMode string

const (
	ColorModeNone  ColorMode = "none"
	ColorMode24Bit ColorMode = "24bit"
	ColorMode256   ColorMode = "256"
)

// RampPresets contiene las rampas de caracteres predefinidas ciclables con la tecla R.
var RampPresets = []string{
	" .:-=+*#%@",          // estándar (luz→oscuro)
	" ░▒▓█",               // bloques unicode
	" `^\",:;Il!i~+_-?][}{1)(|/tfjrxnuvczXYUJCLQ0OZmwqpdbkhao*#MW&8%B@$", // densa
	" .oO0@",              // minimalista
}

// RGBToANSI256 mapea un color RGB (0..255) al codigo de color de 8-bit de la paleta ANSI 256.
func RGBToANSI256(r, g, b uint8) int {
	if r == g && g == b {
		if r < 8 {
			return 16
		}
		if r > 248 {
			return 231
		}
		return 232 + int(math.Round(float64(r-8)/247.0*23.0))
	}

	r6 := int(math.Round(float64(r) / 255.0 * 5.0))
	g6 := int(math.Round(float64(g) / 255.0 * 5.0))
	b6 := int(math.Round(float64(b) / 255.0 * 5.0))
	return 16 + (36 * r6) + (6 * g6) + b6
}

// FormatFrameANSI convierte un CharFrame en lineas de texto aplicando las secuencias de color ANSI especificadas.
// Aplica RLE para evitar emitir secuencias de color duplicadas en celdas consecutivas con el mismo tono.
// Soporta BgColor en Cell para modos HD como HalfBlock.
func FormatFrameANSI(frame *ascii.CharFrame, mode ColorMode) []string {
	if frame == nil || frame.Height == 0 || frame.Width == 0 {
		return nil
	}

	if mode == "" || mode == ColorModeNone {
		return frame.ToStrings()
	}

	lines := make([]string, frame.Height)
	for y := 0; y < frame.Height; y++ {
		var sb strings.Builder
		var lastFg *color.RGBA
		var lastBg *color.RGBA

		for x := 0; x < frame.Width; x++ {
			cell := frame.Cells[y][x]

			switch mode {
			case ColorMode24Bit:
				if lastFg == nil || *lastFg != cell.Color {
					fmt.Fprintf(&sb, "\033[38;2;%d;%d;%dm", cell.Color.R, cell.Color.G, cell.Color.B)
					c := cell.Color
					lastFg = &c
				}
				if cell.BgColor != nil && (lastBg == nil || *lastBg != *cell.BgColor) {
					fmt.Fprintf(&sb, "\033[48;2;%d;%d;%dm", cell.BgColor.R, cell.BgColor.G, cell.BgColor.B)
					bg := *cell.BgColor
					lastBg = &bg
				} else if cell.BgColor == nil && lastBg != nil {
					sb.WriteString("\033[49m")
					lastBg = nil
				}
			case ColorMode256:
				if lastFg == nil || *lastFg != cell.Color {
					code := RGBToANSI256(cell.Color.R, cell.Color.G, cell.Color.B)
					fmt.Fprintf(&sb, "\033[38;5;%dm", code)
					c := cell.Color
					lastFg = &c
				}
				if cell.BgColor != nil && (lastBg == nil || *lastBg != *cell.BgColor) {
					code := RGBToANSI256(cell.BgColor.R, cell.BgColor.G, cell.BgColor.B)
					fmt.Fprintf(&sb, "\033[48;5;%dm", code)
					bg := *cell.BgColor
					lastBg = &bg
				} else if cell.BgColor == nil && lastBg != nil {
					sb.WriteString("\033[49m")
					lastBg = nil
				}
			}
			sb.WriteRune(cell.Char)
		}

		if lastFg != nil || lastBg != nil {
			sb.WriteString("\033[0m")
		}
		lines[y] = sb.String()
	}
	return lines
}

// RenderImage escribe un CharFrame a un io.Writer aplicando la modalidad de color indicada.
func RenderImage(w io.Writer, frame *ascii.CharFrame, mode ColorMode) error {
	if w == nil {
		return fmt.Errorf("writer cannot be nil")
	}
	if frame == nil {
		return fmt.Errorf("frame cannot be nil")
	}

	lines := FormatFrameANSI(frame, mode)
	for _, line := range lines {
		if _, err := fmt.Fprintln(w, line); err != nil {
			return fmt.Errorf("write ascii line: %w", err)
		}
	}
	return nil
}

// RenderLines escribe directamente una lista de líneas de texto plano o formateadas a un io.Writer.
func RenderLines(w io.Writer, lines []string) error {
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

// ClearScreen clears the terminal screen and moves cursor to top-left (1,1).
func ClearScreen(w io.Writer) error {
	_, err := fmt.Fprint(w, "\033[2J\033[H")
	return err
}

// HideCursor hides the terminal cursor during video playback.
func HideCursor(w io.Writer) error {
	_, err := fmt.Fprint(w, "\033[?25l")
	return err
}

// ShowCursor restores terminal cursor visibility after playback finishes.
func ShowCursor(w io.Writer) error {
	_, err := fmt.Fprint(w, "\033[?25h")
	return err
}

// PlaybackState agrupa el estado mutable de reproducción accesible concurrentemente
// desde readKeys y PlayVideo.
type PlaybackState struct {
	Paused       atomic.Bool
	SpeedMult    atomic.Value // float64 (0.25–4.0)
	ColorModeVal atomic.Value // ColorMode
	HDModeVal    atomic.Value // ascii.HDMode
	FitModeVal   atomic.Value // frame.FitMode
	RampIdx      atomic.Int32
}

// NewPlaybackState crea un PlaybackState con valores por defecto.
func NewPlaybackState(initialColor ColorMode, initialHD ascii.HDMode, initialFit frame.FitMode) *PlaybackState {
	ps := &PlaybackState{}
	ps.SpeedMult.Store(1.0)
	ps.ColorModeVal.Store(initialColor)
	ps.HDModeVal.Store(initialHD)
	ps.FitModeVal.Store(initialFit)
	ps.RampIdx.Store(0)
	return ps
}

// PlaybackOptions configures terminal video playback parameters.
type PlaybackOptions struct {
	FPS      float64
	Width    int
	Height   int
	Duration time.Duration
	Title    string
	State    *PlaybackState
	Clock    func() time.Duration
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	totalSeconds := int(d.Seconds())
	minutes := totalSeconds / 60
	seconds := totalSeconds % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

// PlayVideo plays ASCII video frames to an io.Writer at the target FPS rate.
// Reacts to PlaybackState for speed, color mode and ramp changes in real-time.
// Uses Master Clock function if provided in PlaybackOptions.
func PlayVideo(w io.Writer, frames <-chan *ascii.CharFrame, opts PlaybackOptions, cancel <-chan struct{}, subtitles []subtitle.Entry) error {
	if opts.FPS <= 0 {
		opts.FPS = 24
	}

	if err := HideCursor(w); err != nil {
		return err
	}
	defer ShowCursor(w) //nolint:errcheck

	if err := ClearScreen(w); err != nil {
		return err
	}

	var sb strings.Builder
	playbackStart := time.Now()
	var totalPausedDuration time.Duration
	var framesProcessed uint64

	for {
		// Pause loop.
		for opts.State != nil && opts.State.Paused.Load() {
			pauseBegin := time.Now()
			select {
			case <-cancel:
				return nil
			case <-time.After(50 * time.Millisecond):
			}
			totalPausedDuration += time.Since(pauseBegin)
		}

		start := time.Now()
		var elapsed time.Duration
		if opts.Clock != nil {
			elapsed = opts.Clock()
		} else {
			elapsed = start.Sub(playbackStart) - totalPausedDuration
		}

		// Compute effective frame duration from current speed multiplier.
		speedMult := 1.0
		if opts.State != nil {
			if v, ok := opts.State.SpeedMult.Load().(float64); ok && v > 0 {
				speedMult = v
			}
		}
		frameDuration := time.Duration(float64(time.Second) / (opts.FPS * speedMult))

		select {
		case <-cancel:
			return nil
		case frm, ok := <-frames:
			if !ok {
				return nil
			}

			// Frame Skipping / Catch-up: Si el canal de fotogramas acumuló retraso
			// respecto al Master Clock (audio/tiempo real), descartar fotogramas antiguos.
			if opts.Clock != nil {
				actualElapsed := opts.Clock()
				for len(frames) > 0 {
					targetElapsed := time.Duration(float64(framesProcessed) * float64(time.Second) / (opts.FPS * speedMult))
					if actualElapsed <= targetElapsed+frameDuration {
						break
					}
					nextFrm, ok := <-frames
					if !ok {
						break
					}
					frm = nextFrm
					framesProcessed++
				}
			}
			framesProcessed++

			// Resolve current color mode and ramp from state.
			colorMode := ColorModeNone
			rampLabel := RampPresets[0]
			isPaused := false
			if opts.State != nil {
				if cm, ok := opts.State.ColorModeVal.Load().(ColorMode); ok {
					colorMode = cm
				}
				idx := int(opts.State.RampIdx.Load()) % len(RampPresets)
				rampLabel = RampPresets[idx]
				isPaused = opts.State.Paused.Load()
			}

			lines := FormatFrameANSI(frm, colorMode)

			sb.Reset()

			curRow := 1
			// 1. TUI Header.
			fmt.Fprintf(&sb, "\033[%d;1H", curRow)
			curRow++

			statusBadge := "\033[1;32m[PLAYING]\033[0m"
			if isPaused {
				statusBadge = "\033[1;33m[PAUSED]\033[0m"
			}

			timeStr := formatDuration(elapsed)
			if opts.Duration > 0 {
				timeStr += " / " + formatDuration(opts.Duration)
			}

			titleStr := opts.Title
			if titleStr == "" {
				titleStr = "ASCII Video"
			}

			speedStr := fmt.Sprintf("%.2fx", speedMult)
			colorLabel := string(colorMode)

			fmt.Fprintf(&sb, "%s \033[1m%s\033[0m | %s | %.1f FPS | \033[36m%s\033[0m | \033[35m%s\033[0m\033[K",
				statusBadge, titleStr, timeStr, opts.FPS, speedStr, colorLabel)

			sepWidth := opts.Width
			if sepWidth <= 0 {
				sepWidth = 80
			}

			// Top Separator.
			fmt.Fprintf(&sb, "\033[%d;1H\033[90m%s\033[0m\033[K", curRow, strings.Repeat("─", sepWidth))
			curRow++

			// 2. Video Frame Lines.
			for _, line := range lines {
				fmt.Fprintf(&sb, "\033[%d;1H%s\033[K", curRow, line)
				curRow++
			}

			// 3. Subtitles Area.
			if len(subtitles) > 0 {
				subText := subtitle.ActiveAt(subtitles, elapsed)
				fmt.Fprintf(&sb, "\033[%d;1H\033[K", curRow)
				curRow++
				if subText != "" {
					subLines := strings.Split(subText, "\n")
					for _, sl := range subLines {
						pad := (sepWidth - len(sl)) / 2
						padStr := ""
						if pad > 0 {
							padStr = strings.Repeat(" ", pad)
						}
						fmt.Fprintf(&sb, "\033[%d;1H%s\033[1;97m%s\033[0m\033[K", curRow, padStr, sl)
						curRow++
					}
				}
			}

			// 4. TUI Footer.
			fmt.Fprintf(&sb, "\033[%d;1H\033[90m%s\033[0m\033[K", curRow, strings.Repeat("─", sepWidth))
			curRow++
			_ = rampLabel
			fmt.Fprintf(&sb, "\033[%d;1H\033[90m[Espacio] Pausa  [←][→] Seek  [+][-] Velocidad  [Z] Zoom  [C] Color  [R] Rampa  [H] HD  [Q] Salir\033[0m\033[K", curRow)

			if _, err := fmt.Fprint(w, sb.String()); err != nil {
				return fmt.Errorf("write frame: %w", err)
			}

			renderDuration := time.Since(start)
			if sleep := frameDuration - renderDuration; sleep > 0 {
				select {
				case <-time.After(sleep):
				case <-cancel:
					return nil
				}
			}
		}
	}
}
