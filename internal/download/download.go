// Package download fetches Twitch clips via yt-dlp.
package download

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const maxRetries = 4

// Clip downloads a Twitch clip URL to the given directory and returns the file path.
func Clip(clipURL, clipID, outputDir string) (string, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("create output dir: %w", err)
	}

	outputPath := filepath.Join(outputDir, clipID+".mp4")

	var (
		lastErr error
		cmd     *exec.Cmd
		err     error
	)
	for attempt := range maxRetries {
		if attempt > 0 {
			backoff := time.Duration(3<<(attempt-1)) * time.Second
			slog.Debug("retrying download", "clip_id", clipID, "attempt", attempt+1, "backoff", backoff)
			time.Sleep(backoff)
		}

		cmd = exec.Command("yt-dlp", "-o", outputPath, clipURL)
		cmd.Stderr = os.Stderr
		if err = cmd.Run(); err == nil {
			return outputPath, nil
		}
		lastErr = err
	}

	return "", fmt.Errorf("yt-dlp failed after %d retries: %w", maxRetries, lastErr)
}
