// Command clippy-agent is the Twitch clip automation pipeline.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

	"github.com/skhanal5/clips/internal/auth"
	"github.com/skhanal5/clips/internal/chat"
	"github.com/skhanal5/clips/internal/clip"
	"github.com/skhanal5/clips/internal/config"
	"github.com/skhanal5/clips/internal/detector"
	"github.com/skhanal5/clips/internal/download"
	"github.com/skhanal5/clips/internal/edit"
)

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		slog.Error("loading config", "err", err)
		os.Exit(1)
	}
	if cfg.Verbose {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	store := auth.NewStore("data/tokens/token.json")
	token, err := auth.EnsureToken(cfg.ClientID, []string{"chat:read", "clips:edit"}, store)
	if err != nil {
		slog.Error("authentication", "err", err)
		os.Exit(1)
	}

	monitor := chat.New(token.AccessToken, token.Username, cfg.Channels, cfg.Verbose)
	msgs, err := monitor.Start(context.Background())
	if err != nil {
		slog.Error("starting chat monitor", "err", err)
		os.Exit(1)
	}

	det := detector.New(cfg.Thresholds)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go det.Start(ctx)

	clipSvc := clip.NewService(cfg.ClientID, token.AccessToken)

	slog.Info("connected", "channels", cfg.Channels)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	for {
		select {
		case msg, ok := <-msgs:
			if !ok {
				slog.Error("chat disconnected")
				return
			}
			det.Feed(msg)
		case trigger := <-det.Triggers():
			slog.Info("trigger", "streamer", trigger.Streamer, "score", trigger.Score)
			go handleTrigger(clipSvc, trigger.Streamer)
		case <-sig:
			slog.Info("shutting down")
			monitor.Stop()
			return
		}
	}
}

func handleTrigger(clipSvc *clip.Service, streamer string) {
	result, err := clipSvc.CreateClip(streamer)
	if err != nil {
		slog.Error("creating clip", "streamer", streamer, "err", err)
		return
	}
	slog.Info("clip created", "streamer", streamer, "clip_id", result.ID, "url", result.URL)

	path, err := download.Clip(result.URL, result.ID, "data/clips/raw")
	if err != nil {
		slog.Error("downloading clip", "clip_id", result.ID, "err", err)
		return
	}
	slog.Info("clip downloaded", "clip_id", result.ID, "path", path)

	outputPath := "data/clips/processed/" + result.ID + ".mp4"
	if err := edit.Render(path, outputPath, edit.Config{Background: "blurred"}); err != nil {
		slog.Error("editing clip", "clip_id", result.ID, "err", err)
		return
	}
	slog.Info("clip edited", "clip_id", result.ID, "path", outputPath)
}
