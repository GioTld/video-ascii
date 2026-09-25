package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/GioTld/video-ascii/internal/api"
	"github.com/GioTld/video-ascii/internal/ascii"
	"github.com/GioTld/video-ascii/internal/decode"
	"github.com/GioTld/video-ascii/internal/filter"
	"github.com/GioTld/video-ascii/internal/frame"
	"github.com/GioTld/video-ascii/internal/render"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "serve" {
		runServe(os.Args[2:])
		return
	}

	runCLI(os.Args[1:])
}

func runServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	port := fs.Int("port", 8080, "HTTP server port")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing serve flags: %v\n", err)
		os.Exit(1)
	}

	srv := api.NewServer()
	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Starting ASCII rendering HTTP server on http://localhost%s\n", addr)
	if err := http.ListenAndServe(addr, srv); err != nil {
		log.Fatalf("Server failure: %v", err)
	}
}

func parseFilters(spec string) (filter.Func, error) {
	return filter.Parse(spec)
}

func getHDMultiplier(mode ascii.HDMode) (fontRatio float64, termHReserve int) {
	switch mode {
	case ascii.HDModeHalfBlock:
		return 1.0, 4
	case ascii.HDModeBraille:
		return 2.0, 4
	default:
		return 0.5, 4
	}
}

func calcScaledDimensions(srcW, srcH, maxW, maxH int, fontRatio float64, fitMode frame.FitMode) (targetW, targetH int, cropX, cropY int) {
	if srcW <= 0 || srcH <= 0 || maxW <= 0 || maxH <= 0 {
		return maxW, maxH, 0, 0
	}
	srcAspect := (float64(srcH) / float64(srcW)) * fontRatio
	termAspect := float64(maxH) / float64(maxW)

	if fitMode == frame.FitModeCover {
		if srcAspect > termAspect {
			targetW = maxW
			targetH = int(float64(maxW) * srcAspect)
			cropY = (targetH - maxH) / 2
		} else {
			targetH = maxH
			targetW = int(float64(maxH) / srcAspect)
			cropX = (targetW - maxW) / 2
		}
	} else {
		calcH := int(float64(maxW) * srcAspect)
		if calcH <= maxH {
			targetW = maxW
			targetH = calcH
		} else {
			targetH = maxH
			targetW = int(float64(maxH) / srcAspect)
		}
	}
	return targetW, targetH, cropX, cropY
}

func runCLI(args []string) {
	fs := flag.NewFlagSet("asciirender", flag.ExitOnError)
	width := fs.Int("width", 0, "target output width in characters (0 = auto-fit terminal)")
	height := fs.Int("height", 0, "target output height in characters (0 = auto)")
	ramp := fs.String("ramp", "", "custom character ramp (light to dark)")
	colorFlag := fs.Bool("color", false, "enable ANSI color output")
	colorModeFlag := fs.String("color-mode", "24bit", "ANSI color mode: 24bit, 256, or none")
	hdModeFlag := fs.String("hd-mode", "none", "HD rendering mode: none, half, braille")
	fitModeFlag := fs.String("fit-mode", "contain", "Fitting mode: contain (letterbox) or cover (zoom to fill)")
	filterFlag := fs.String("filter", "", "comma-separated filters: brightness=1.2, contrast=1.2, sepia, invert, matrix, grayscale, edge")
	audioFlag := fs.Bool("audio", true, "enable audio playback for videos via ffplay")
	outputPath := fs.String("output", "", "output file path (default: stdout)")
	subPath := fs.String("sub", "", "subtitle file path (.srt)")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage:\n  asciirender [opciones] <ruta-archivo>\n  asciirender serve [opciones]\n\nOpciones:\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if fs.NArg() < 1 {
		fs.Usage()
		os.Exit(1)
	}

	colorMode := render.ColorModeNone
	if *colorFlag {
		switch strings.ToLower(*colorModeFlag) {
		case "256":
			colorMode = render.ColorMode256
		default:
			colorMode = render.ColorMode24Bit
		}
	}

	hdMode := ascii.HDMode(strings.ToLower(*hdModeFlag))
	fitMode := frame.FitMode(strings.ToLower(*fitModeFlag))
	if fitMode != frame.FitModeCover {
		fitMode = frame.FitModeContain
	}

	filterFn, err := parseFilters(*filterFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing filters: %v\n", err)
		os.Exit(1)
	}

	inputPath := fs.Arg(0)

	if strings.ToLower(filepath.Ext(inputPath)) == ".mp4" {
		if err := runVideo(inputPath, *width, *height, *ramp, *subPath, colorMode, hdMode, fitMode, filterFn, *audioFlag); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if err := runImage(inputPath, *width, *height, *ramp, *outputPath, colorMode, hdMode, fitMode, filterFn); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runImage(path string, width, height int, ramp, outputPath string, colorMode render.ColorMode, hdMode ascii.HDMode, fitMode frame.FitMode, filterFn filter.Func) error {
	frm, err := decode.DecodeFile(path)
	if err != nil {
		return fmt.Errorf("decode image: %w", err)
	}

	if outputPath != "" {
		fontRatio, _ := getHDMultiplier(hdMode)
		resized, err := frm.Resize(width, height, fontRatio)
		if err != nil {
			return fmt.Errorf("resize: %w", err)
		}
		if filterFn != nil {
			resized = filterFn(resized)
		}
		conv, err := ascii.NewConverter(ramp)
		if err != nil {
			return fmt.Errorf("ascii converter: %w", err)
		}
		charFrame, err := conv.ConvertFrameHD(resized, hdMode)
		if err != nil {
			return fmt.Errorf("convert to ascii: %w", err)
		}
		f, err := os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("create output file: %w", err)
		}
		defer f.Close()
		return render.RenderImage(f, charFrame, colorMode)
	}

	// Interactive TUI Viewer Mode for static images
	restoreTerminal, err := render.EnableRawMode()
	if err == nil {
		defer restoreTerminal()
	}

	if err := render.HideCursor(os.Stdout); err == nil {
		defer render.ShowCursor(os.Stdout) //nolint:errcheck
	}

	curColorMode := colorMode
	curHDMode := hdMode
	curFitMode := fitMode
	curRampIdx := 0
	curFilterIdx := 0

	filterPresets := []struct {
		name string
		fn   filter.Func
	}{
		{"normal", filterFn},
		{"matrix", filter.Matrix()},
		{"sepia", filter.Sepia()},
		{"invert", filter.Invert()},
		{"edge", filter.Edge()},
		{"grayscale", filter.Grayscale()},
	}

	hdModes := []ascii.HDMode{
		ascii.HDModeNone,
		ascii.HDModeHalfBlock,
		ascii.HDModeBraille,
	}

	hdIdx := 0
	for i, m := range hdModes {
		if m == curHDMode {
			hdIdx = i
			break
		}
	}

	renderTUI := func() {
		_ = render.ClearScreen(os.Stdout)
		termW, termH, err := render.GetTerminalSize()
		if err != nil || termW <= 0 {
			termW, termH = 80, 24
		}

		activeHD := hdModes[hdIdx]
		fontRatio, reserveH := getHDMultiplier(activeHD)

		maxW := termW
		if activeHD == ascii.HDModeBraille {
			maxW = termW * 2
		}
		maxH := (termH - reserveH)
		if activeHD == ascii.HDModeHalfBlock {
			maxH = (termH - reserveH) * 2
		} else if activeHD == ascii.HDModeBraille {
			maxH = (termH - reserveH) * 4
		}
		if maxH < 5 {
			maxH = 5
		}

		targetW, targetH, cropX, cropY := calcScaledDimensions(frm.Width, frm.Height, maxW, maxH, fontRatio, curFitMode)
		if width > 0 {
			targetW = width
		}
		if height > 0 {
			targetH = height
		}

		resized, err := frm.Resize(targetW, targetH, fontRatio)
		if err != nil {
			return
		}

		if curFitMode == frame.FitModeCover && (cropX > 0 || cropY > 0) {
			cropW := maxW
			cropH := maxH
			resized = resized.Crop(cropX, cropY, cropW, cropH)
		}

		activeFilter := filterPresets[curFilterIdx].fn
		if activeFilter != nil {
			resized = activeFilter(resized)
		}

		currentRamp := render.RampPresets[curRampIdx%len(render.RampPresets)]
		if ramp != "" && curRampIdx == 0 {
			currentRamp = ramp
		}

		conv, err := ascii.NewConverter(currentRamp)
		if err != nil {
			conv, _ = ascii.NewConverter("")
		}

		charFrame, err := conv.ConvertFrameHD(resized, activeHD)
		if err != nil {
			return
		}

		lines := render.FormatFrameANSI(charFrame, curColorMode)

		var sb strings.Builder

		curRow := 1
		titleStr := filepath.Base(path)
		dimStr := fmt.Sprintf("%dx%d", frm.Width, frm.Height)
		filterName := filterPresets[curFilterIdx].name

		fmt.Fprintf(&sb, "\033[%d;1H\033[1;34m[IMAGE Viewer]\033[0m \033[1m%s\033[0m | %s | Fit: \033[32m%s\033[0m | Color: \033[35m%s\033[0m | HD: \033[36m%s\033[0m | Filtro: \033[33m%s\033[0m\033[K",
			curRow, titleStr, dimStr, string(curFitMode), string(curColorMode), string(activeHD), filterName)
		curRow++

		sepWidth := charFrame.Width
		if sepWidth <= 0 {
			sepWidth = termW
		}
		fmt.Fprintf(&sb, "\033[%d;1H\033[90m%s\033[0m\033[K", curRow, strings.Repeat("─", sepWidth))
		curRow++

		for _, l := range lines {
			fmt.Fprintf(&sb, "\033[%d;1H%s\033[K", curRow, l)
			curRow++
		}

		fmt.Fprintf(&sb, "\033[%d;1H\033[90m%s\033[0m\033[K", curRow, strings.Repeat("─", sepWidth))
		curRow++
		fmt.Fprintf(&sb, "\033[%d;1H\033[90m[Z] Zoom/Fit  [C] Color  [R] Rampa  [F] Filtros  [H] HD  [Q / Esc] Salir\033[0m\033[K", curRow)

		fmt.Fprint(os.Stdout, sb.String())
	}

	renderTUI()

	buf := make([]byte, 3)
	for {
		n, err := os.Stdin.Read(buf[:1])
		if err != nil || n == 0 {
			return nil
		}
		b := buf[0]
		switch b {
		case 'q', 'Q', 3, 13, 27:
			return nil
		case 'z', 'Z', 'm', 'M':
			if curFitMode == frame.FitModeContain {
				curFitMode = frame.FitModeCover
			} else {
				curFitMode = frame.FitModeContain
			}
			renderTUI()
		case 'c', 'C':
			switch curColorMode {
			case render.ColorMode24Bit:
				curColorMode = render.ColorMode256
			case render.ColorMode256:
				curColorMode = render.ColorModeNone
			case render.ColorModeNone:
				curColorMode = render.ColorMode24Bit
			}
			renderTUI()
		case 'r', 'R':
			curRampIdx++
			renderTUI()
		case 'f', 'F':
			curFilterIdx = (curFilterIdx + 1) % len(filterPresets)
			renderTUI()
		case 'h', 'H':
			hdIdx = (hdIdx + 1) % len(hdModes)
			renderTUI()
		}
	}
}

type audioController struct {
	cmd       *exec.Cmd
	path      string
	startTime time.Time
	startSec  float64
	speed     float64
	pauseAcc  time.Duration
	pauseMark time.Time
}

func buildAtempoFilter(speed float64) string {
	if math.Abs(speed-1.0) <= 0.01 {
		return ""
	}
	var filters []string
	rem := speed
	for rem > 2.0 {
		filters = append(filters, "atempo=2.0")
		rem /= 2.0
	}
	for rem < 0.5 {
		filters = append(filters, "atempo=0.5")
		rem /= 0.5
	}
	if math.Abs(rem-1.0) > 0.01 {
		filters = append(filters, fmt.Sprintf("atempo=%.2f", rem))
	}
	return strings.Join(filters, ",")
}

func startAudioController(path string, startSec float64, speed float64) *audioController {
	if ffplayPath, err := exec.LookPath("ffplay"); err == nil && ffplayPath != "" {
		args := []string{"-nodisp", "-autoexit", "-loglevel", "quiet"}
		if startSec > 0 {
			args = append(args, "-ss", fmt.Sprintf("%.2f", startSec))
		}
		if filter := buildAtempoFilter(speed); filter != "" {
			args = append(args, "-af", filter)
		}
		args = append(args, path)
		cmd := exec.Command("ffplay", args...)
		if err := cmd.Start(); err == nil {
			return &audioController{
				cmd:       cmd,
				path:      path,
				startTime: time.Now(),
				startSec:  startSec,
				speed:     speed,
			}
		}
	}
	return nil
}

func (ac *audioController) Elapsed() time.Duration {
	if ac == nil || ac.cmd == nil {
		return 0
	}
	now := time.Now()
	pa := ac.pauseAcc
	if !ac.pauseMark.IsZero() {
		pa += now.Sub(ac.pauseMark)
	}
	realElapsed := now.Sub(ac.startTime) - pa
	scaledSec := ac.startSec + (realElapsed.Seconds() * ac.speed)
	return time.Duration(scaledSec * float64(time.Second))
}

func (ac *audioController) SetSpeed(newSpeed float64) {
	if ac == nil {
		return
	}
	curSec := ac.Elapsed().Seconds()
	if ac.cmd != nil && ac.cmd.Process != nil {
		_ = ac.cmd.Process.Kill()
	}
	next := startAudioController(ac.path, curSec, newSpeed)
	if next != nil {
		ac.cmd = next.cmd
		ac.startTime = next.startTime
		ac.startSec = next.startSec
		ac.speed = next.speed
		ac.pauseAcc = 0
		ac.pauseMark = time.Time{}
	}
}

func (ac *audioController) Pause(pause bool) {
	if ac == nil || ac.cmd == nil || ac.cmd.Process == nil {
		return
	}
	if pause {
		ac.pauseMark = time.Now()
		_ = syscall.Kill(ac.cmd.Process.Pid, syscall.SIGSTOP)
	} else {
		if !ac.pauseMark.IsZero() {
			ac.pauseAcc += time.Since(ac.pauseMark)
			ac.pauseMark = time.Time{}
		}
		_ = syscall.Kill(ac.cmd.Process.Pid, syscall.SIGCONT)
	}
}

func (ac *audioController) Stop() {
	if ac != nil && ac.cmd != nil && ac.cmd.Process != nil {
		_ = ac.cmd.Process.Kill()
	}
}

func runVideo(path string, width, height int, ramp, subFlag string, colorMode render.ColorMode, hdMode ascii.HDMode, fitMode frame.FitMode, filterFn filter.Func, playAudio bool) error {
	meta, err := decode.ProbeVideo(path)
	if err != nil {
		return fmt.Errorf("probe video: %w", err)
	}

	subtitles, err := decode.LoadSubtitles(path, subFlag)
	if err != nil {
		log.Printf("Warning: failed to load subtitles: %v\n", err)
	}

	restoreTerminal, err := render.EnableRawMode()
	if err != nil {
		return fmt.Errorf("enable raw mode: %w", err)
	}
	defer restoreTerminal()

	cancel := make(chan struct{})
	state := render.NewPlaybackState(colorMode, hdMode, fitMode)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sig
		close(cancel)
	}()

	var ac *audioController
	var clockFn func() time.Duration

	if playAudio && meta.HasAudio {
		ac = startAudioController(path, 0, 1.0)
		if ac != nil {
			clockFn = ac.Elapsed
			defer ac.Stop()
		}
	}

	go readKeys(cancel, state, ac)

	rawFrames, errc := decode.DecodeVideo(path, cancel)
	asciiFrames := make(chan *ascii.CharFrame, 4)

	go func() {
		defer close(asciiFrames)
		for f := range rawFrames {
			curHD := hdMode
			if hVal, ok := state.HDModeVal.Load().(ascii.HDMode); ok {
				curHD = hVal
			}
			curFit := fitMode
			if fVal, ok := state.FitModeVal.Load().(frame.FitMode); ok {
				curFit = fVal
			}

			fontRatio, reserveH := getHDMultiplier(curHD)

			termW, termH, err := render.GetTerminalSize()
			if err != nil || termW <= 0 {
				termW, termH = 80, 24
			}

			maxW := termW
			if curHD == ascii.HDModeBraille {
				maxW = termW * 2
			}
			maxH := (termH - reserveH)
			if curHD == ascii.HDModeHalfBlock {
				maxH = (termH - reserveH) * 2
			} else if curHD == ascii.HDModeBraille {
				maxH = (termH - reserveH) * 4
			}
			if maxH < 5 {
				maxH = 5
			}

			targetW, targetH, cropX, cropY := calcScaledDimensions(meta.Width, meta.Height, maxW, maxH, fontRatio, curFit)
			if width > 0 {
				targetW = width
			}
			if height > 0 {
				targetH = height
			}

			resized, err := f.Resize(targetW, targetH, fontRatio)
			if err != nil {
				continue
			}

			if curFit == frame.FitModeCover && (cropX > 0 || cropY > 0) {
				cropW := maxW
				cropH := maxH
				resized = resized.Crop(cropX, cropY, cropW, cropH)
			}

			if filterFn != nil {
				resized = filterFn(resized)
			}

			rampIdx := int(state.RampIdx.Load()) % len(render.RampPresets)
			currentRamp := render.RampPresets[rampIdx]
			if ramp != "" && rampIdx == 0 {
				currentRamp = ramp
			}

			conv, err := ascii.NewConverter(currentRamp)
			if err != nil {
				conv, _ = ascii.NewConverter("")
			}

			charFrame, err := conv.ConvertFrameHD(resized, curHD)
			if err != nil {
				continue
			}
			select {
			case asciiFrames <- charFrame:
			case <-cancel:
				return
			}
		}
	}()

	opts := render.PlaybackOptions{
		FPS:   meta.FPS,
		Width: width,
		Height: height,
		Title: filepath.Base(path),
		State: state,
		Clock: clockFn,
	}

	if err := render.PlayVideo(os.Stdout, asciiFrames, opts, cancel, subtitles); err != nil {
		return fmt.Errorf("playback: %w", err)
	}

	if err := <-errc; err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	return nil
}

func readKeys(cancel chan struct{}, state *render.PlaybackState, ac *audioController) {
	buf := make([]byte, 3)
	for {
		n, err := os.Stdin.Read(buf[:1])
		if err != nil || n == 0 {
			return
		}
		b := buf[0]
		switch b {
		case ' ':
			newPaused := !state.Paused.Load()
			state.Paused.Store(newPaused)
			if ac != nil {
				ac.Pause(newPaused)
			}
		case 'q', 'Q', 3:
			if ac != nil {
				ac.Stop()
			}
			select {
			case <-cancel:
			default:
				close(cancel)
			}
			return
		case 'z', 'Z', 'm', 'M':
			if currentFit, ok := state.FitModeVal.Load().(frame.FitMode); ok {
				nextFit := frame.FitModeContain
				if currentFit == frame.FitModeContain {
					nextFit = frame.FitModeCover
				}
				state.FitModeVal.Store(nextFit)
			}
		case '+', '=':
			if v, ok := state.SpeedMult.Load().(float64); ok {
				newV := v + 0.25
				if newV > 4.0 {
					newV = 4.0
				}
				state.SpeedMult.Store(newV)
				if ac != nil {
					ac.SetSpeed(newV)
				}
			}
		case '-', '_':
			if v, ok := state.SpeedMult.Load().(float64); ok {
				newV := v - 0.25
				if newV < 0.25 {
					newV = 0.25
				}
				state.SpeedMult.Store(newV)
				if ac != nil {
					ac.SetSpeed(newV)
				}
			}
		case 'c', 'C':
			if cm, ok := state.ColorModeVal.Load().(render.ColorMode); ok {
				next := render.ColorMode24Bit
				switch cm {
				case render.ColorMode24Bit:
					next = render.ColorMode256
				case render.ColorMode256:
					next = render.ColorModeNone
				case render.ColorModeNone:
					next = render.ColorMode24Bit
				}
				state.ColorModeVal.Store(next)
			}
		case 'r', 'R':
			state.RampIdx.Add(1)
		case 'h', 'H':
			if currentHD, ok := state.HDModeVal.Load().(ascii.HDMode); ok {
				nextHD := ascii.HDModeNone
				switch currentHD {
				case ascii.HDModeNone:
					nextHD = ascii.HDModeHalfBlock
				case ascii.HDModeHalfBlock:
					nextHD = ascii.HDModeBraille
				case ascii.HDModeBraille:
					nextHD = ascii.HDModeNone
				}
				state.HDModeVal.Store(nextHD)
			}
		case '\x1b':
			n2, _ := os.Stdin.Read(buf[1:3])
			if n2 == 2 && buf[1] == '[' {
				switch buf[2] {
				case 'C':
					if v, ok := state.SpeedMult.Load().(float64); ok {
						newV := v + 0.5
						if newV > 4.0 {
							newV = 4.0
						}
						state.SpeedMult.Store(newV)
						if ac != nil {
							ac.SetSpeed(newV)
						}
					}
				case 'D':
					if v, ok := state.SpeedMult.Load().(float64); ok {
						newV := v - 0.5
						if newV < 0.25 {
							newV = 0.25
						}
						state.SpeedMult.Store(newV)
						if ac != nil {
							ac.SetSpeed(newV)
						}
					}
				}
			}
		}

		select {
		case <-cancel:
			return
		default:
		}
	}
}
