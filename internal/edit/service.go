package edit

import (
	"os"
	"os/exec"
)

// Config controls the video editing behavior.
type Config struct {
	Background  string // "blurred", "black", or "image"
	BgImagePath string // used when Background is "image"
	Title       string // optional overlay text
}

// Render processes a video file with the given config and writes the output.
func Render(inputPath, outputPath string, cfg Config) error {
	args := buildArgs(inputPath, outputPath, cfg)
	cmd := exec.Command("ffmpeg", args...)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
