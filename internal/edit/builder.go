// Package edit processes video clips into vertical short-form format using ffmpeg.
package edit

import (
	"fmt"
	"math"
	"strings"
)

const (
	canvasWidth  = 1080
	canvasHeight = 1920
	defaultFgW   = 1080
	defaultFgH   = 607
)

func buildArgs(inputPath, outputPath string, cfg Config) []string {
	fgW := defaultFgW
	fgH := defaultFgH

	var bgFilterTpl string
	switch cfg.Background {
	case "black":
		bgFilterTpl = "color=c=black:s=%dx%d:d=1[b]"
	case "blurred":
		bgFilterTpl = "[%s]scale=%d:%d:force_original_aspect_ratio=increase,boxblur=50,crop=%d:%d[b]"
	case "image":
		bgFilterTpl = "[%s]scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d[b]"
	}

	fgInput := "0:v"
	bgInput := "0:v"
	audioInput := "0:a"
	inputs := []string{"-i", inputPath}

	if cfg.Background == "image" && cfg.BgImagePath != "" {
		inputs = []string{"-i", cfg.BgImagePath, "-i", inputPath}
		fgInput = "1:v"
		audioInput = "1:a"
	}

	var bgFilter string
	if cfg.Background == "black" {
		bgFilter = fmt.Sprintf(bgFilterTpl, canvasWidth, canvasHeight)
	} else {
		bgFilter = fmt.Sprintf(bgFilterTpl, bgInput, canvasWidth, canvasHeight, canvasWidth, canvasHeight)
	}
	fgFilter := fmt.Sprintf(
		"[%s]scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d,format=yuv420p[f]",
		fgInput, fgW, fgH, fgW, fgH,
	)

	overlayX := (canvasWidth - fgW) / 2
	overlayY := (canvasHeight - fgH) / 2

	filterParts := []string{bgFilter, fgFilter}

	if cfg.Title != "" {
		filterParts = append(filterParts, buildTitleFilters(cfg.Title, fgH)...)
	}

	overlayFilter := fmt.Sprintf("[b][f]overlay=x=%d:y=%d", overlayX, overlayY)
	allFilters := append(filterParts, overlayFilter)
	filterComplex := strings.Join(allFilters, ";")

	args := append(inputs,
		"-filter_complex", filterComplex,
		"-map", audioInput,
		"-c:a", "copy",
		"-shortest",
		"-y",
		outputPath,
	)

	return args
}

func buildTitleFilters(title string, fgHeight int) []string {
	charactersPerLine := 20
	lineHeight := 80
	textPadding := 60
	safeTextAreaHeight := 2 * lineHeight

	rawY := float64((canvasHeight-fgHeight)/2 - safeTextAreaHeight - textPadding)
	startY := int(math.Max(0, rawY))

	lines := splitTextIntoLines(title, charactersPerLine)
	var filters []string
	for i, line := range lines {
		escaped := strings.ReplaceAll(line, "'", "'\\''")
		f := fmt.Sprintf(
			"drawtext=text='%s':fontfile=font/Montserrat-Bold.ttf:fontsize=72:fontcolor=white:x=(w-text_w)/2:y=%d:borderw=10:bordercolor=black",
			escaped, startY+i*lineHeight,
		)
		filters = append(filters, f)
	}
	return filters
}

func splitTextIntoLines(text string, maxWidth int) []string {
	uppercaseText := strings.ToUpper(text)
	words := strings.Fields(uppercaseText)
	var lines []string
	var currentLine string
	for _, word := range words {
		if len(currentLine)+len(word)+1 > maxWidth {
			if currentLine != "" {
				lines = append(lines, currentLine)
			}
			currentLine = word
		} else {
			if currentLine != "" {
				currentLine += " "
			}
			currentLine += word
		}
	}
	if currentLine != "" {
		lines = append(lines, currentLine)
	}
	return lines
}
