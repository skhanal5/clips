// Package edit processes video clips into vertical short-form format using ffmpeg.
package edit

import (
	"fmt"
	"math"
	"strings"
)

var canvasWidth = 1080
var canvasHeight = 1920

var defaultFgWidth = 1080
var defaultFgHeight = 607

func buildArgs(inputPath, outputPath string, options *Options) []string {
	fgW := options.ForegroundSize.Width
	fgH := options.ForegroundSize.Height
	if fgW == 0 || fgH == 0 {
		fgW = defaultFgWidth
		fgH = defaultFgHeight
	}

	bgLabel := "b"
	fgLabel := "f"
	overlayX := (canvasWidth - fgW) / 2
	overlayY := (canvasHeight - fgH) / 2

	var bgFilter string
	switch options.Background {
	case BlackScreen:
		bgFilter = fmt.Sprintf("color=c=black:s=%dx%d:d=1[%s]", canvasWidth, canvasHeight, bgLabel)
	case BlurredVideo:
		bgFilter = fmt.Sprintf(
			"[0:v]scale=%d:%d:force_original_aspect_ratio=increase,boxblur=50,crop=%d:%d[%s]",
			canvasWidth, canvasHeight, canvasWidth, canvasHeight, bgLabel,
		)
	case StaticImage:
		if options.BgImagePath != "" {
			bgFilter = fmt.Sprintf(
				"[1:v]scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d[%s]",
				canvasWidth, canvasHeight, canvasWidth, canvasHeight, bgLabel,
			)
		}
	}

	fgFilter := fmt.Sprintf(
		"[0:v]scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d,format=yuv420p[%s]",
		fgW, fgH, fgW, fgH, fgLabel,
	)

	filterParts := []string{bgFilter, fgFilter}
	overlayFilter := fmt.Sprintf("[%s][%s]overlay=x=%d:y=%d", bgLabel, fgLabel, overlayX, overlayY)

	if options.Title != "" {
		filterParts = append(filterParts, buildTitleFilters(options.Title, fgH)...)
	}

	allFilters := append(filterParts, overlayFilter)
	filterComplex := strings.Join(allFilters, ";")

	args := []string{
		"-i", inputPath,
		"-filter_complex", filterComplex,
		"-map", "1:a",
		"-c:a", "copy",
		"-shortest",
		"-y",
		outputPath,
	}

	if options.Background == StaticImage && options.BgImagePath != "" {
		args = append([]string{"-i", options.BgImagePath, "-i", inputPath}, args[1:]...)
	}

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
		f := fmt.Sprintf(
			"drawtext=text='%s':fontfile=font/Montserrat-Bold.ttf:fontsize=72:fontcolor=white:x=(w-text_w)/2:y=%d:borderw=10:bordercolor=black",
			line, startY+i*lineHeight,
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
