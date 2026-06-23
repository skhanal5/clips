package edit

import (
	"os/exec"
)

// Render processes a video file with the given options and writes the output.
func Render(inputPath, outputPath string, opts ...Option) error {
	options := &Options{}
	for _, opt := range opts {
		opt(options)
	}

	args := buildArgs(inputPath, outputPath, options)
	cmd := exec.Command("ffmpeg", args...)
	cmd.Stderr = nil
	return cmd.Run()
}
