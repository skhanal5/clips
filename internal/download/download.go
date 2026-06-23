// Package download fetches Twitch clips via yt-dlp.
package download

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Clip downloads a Twitch clip URL to the given directory and returns the file path.
func Clip(clipURL, clipID, outputDir string) (string, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("create output dir: %w", err)
	}

	outputPath := filepath.Join(outputDir, clipID+".mp4")
	cmd := exec.Command("yt-dlp", "-o", outputPath, clipURL)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("yt-dlp failed: %w", err)
	}

	return outputPath, nil
}
