package edit

// Render processes a video file with the given options and writes the output.
func Render(inputPath, outputPath string, opts ...Option) error {
	options := &Options{}
	for _, opt := range opts {
		opt(options)
	}

	cmd, err := buildFFmpegCommand(inputPath, outputPath, options)
	if err != nil {
		return err
	}

	return cmd.
		OverWriteOutput().
		ErrorToStdOut().
		Run()
}
