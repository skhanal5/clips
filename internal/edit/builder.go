// Package edit processes video clips into vertical short-form format using ffmpeg.
package edit

import (
	"fmt"
	"math"
	"strings"

	ffmpeg_go "github.com/u2takey/ffmpeg-go"
)

// CanvasSize is the target output resolution (1080x1920 vertical).
var CanvasSize = Size{Width: 1080, Height: 1920}

func buildFFmpegCommand(inputPath, outputPath string, options *Options) (*ffmpeg_go.Stream, error) {
	input := ffmpeg_go.Input(inputPath)

	var bgStream *ffmpeg_go.Stream

	switch options.Background {
	case BlackScreen:
		bgStream = blackBackground()
	case BlurredVideo:
		bgStream = blurredBackground(inputPath)
	case StaticImage:
		if options.BgImagePath == "" {
			return nil, fmt.Errorf("BgImagePath must be set for StaticImage background")
		}
		bgStream = imageBackground(options.BgImagePath)
	default:
		return nil, fmt.Errorf("unsupported background type: %v", options.Background)
	}

	if options.Title != "" {
		filterArgs := buildTitleArgs(options.Title, options.ForegroundSize.Height)
		for _, args := range filterArgs {
			bgStream = bgStream.Filter("drawtext", args)
		}
	}

	fgStream := input.
		Filter("scale", ffmpeg_go.Args{
			fmt.Sprintf("%d:%d:force_original_aspect_ratio=increase", options.ForegroundSize.Width, options.ForegroundSize.Height),
		}).
		Filter("crop", ffmpeg_go.Args{
			fmt.Sprintf("%d:%d:(in_w-out_w)/2:(in_h-out_h)/2",
				options.ForegroundSize.Width,
				options.ForegroundSize.Height),
		}).
		Filter("format", ffmpeg_go.Args{"yuv420p"})

	x := (CanvasSize.Width - options.ForegroundSize.Width) / 2
	y := (CanvasSize.Height - options.ForegroundSize.Height) / 2

	video := ffmpeg_go.Filter(
		[]*ffmpeg_go.Stream{bgStream, fgStream},
		"overlay",
		ffmpeg_go.Args{
			fmt.Sprintf("x=%d", x),
			fmt.Sprintf("y=%d", y),
		},
	)

	output := ffmpeg_go.Output([]*ffmpeg_go.Stream{video}, outputPath,
		ffmpeg_go.KwArgs{
			"map":      "1:a",
			"c:a":      "copy",
			"shortest": "",
			"y":        "",
		})

	return output, nil
}

func blackBackground() *ffmpeg_go.Stream {
	return ffmpeg_go.Input(fmt.Sprintf("color=size=%dx%d:color=black", CanvasSize.Width, CanvasSize.Height), ffmpeg_go.KwArgs{
		"f": "lavfi",
	})
}

func blurredBackground(inputPath string) *ffmpeg_go.Stream {
	input := ffmpeg_go.Input(inputPath)

	blurred := input.
		Filter("scale", ffmpeg_go.Args{
			fmt.Sprintf("%d:%d:force_original_aspect_ratio=increase", CanvasSize.Width, CanvasSize.Height),
		}).
		Filter("boxblur", ffmpeg_go.Args{"50"}).
		Filter("crop", ffmpeg_go.Args{
			fmt.Sprintf("%d:%d", CanvasSize.Width, CanvasSize.Height),
		})

	return blurred
}

func imageBackground(imagePath string) *ffmpeg_go.Stream {
	input := ffmpeg_go.Input(imagePath)

	scaledAndCropped := input.
		Filter("scale", ffmpeg_go.Args{
			fmt.Sprintf("%d:%d:force_original_aspect_ratio=increase", CanvasSize.Width, CanvasSize.Height),
		}).
		Filter("crop", ffmpeg_go.Args{
			fmt.Sprintf("%d:%d", CanvasSize.Width, CanvasSize.Height),
		})

	return scaledAndCropped
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

func buildTitleArgs(title string, fgHeight int) []ffmpeg_go.Args {
	charactersPerLine := 20
	lineHeight := 80
	textPadding := 60
	safeTextAreaHeight := 2 * lineHeight

	rawY := float64((CanvasSize.Height-fgHeight)/2 - safeTextAreaHeight - textPadding)
	startY := int(math.Max(0, rawY))

	lines := splitTextIntoLines(title, charactersPerLine)
	var argsList []ffmpeg_go.Args
	for i, line := range lines {
		escapedLine := line
		args := []string{
			fmt.Sprintf("text='%s'", escapedLine),
			"fontfile=font/Montserrat-Bold.ttf",
			"fontsize=72",
			"fontcolor=white",
			"x=(w-text_w)/2",
			fmt.Sprintf("y=%d", startY+i*lineHeight),
			"borderw=10",
			"bordercolor=black",
		}
		argsList = append(argsList, ffmpeg_go.Args{strings.Join(args, ":")})
	}

	return argsList
}
