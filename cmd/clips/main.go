// Command clips monitors chat and creates clips during viral moments.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

	"github.com/skhanal5/clips/internal/auth"
	"github.com/skhanal5/clips/internal/config"
	"github.com/skhanal5/clips/internal/detector"
	"github.com/skhanal5/clips/internal/download"
	"github.com/skhanal5/clips/internal/edit"
	"github.com/skhanal5/clips/internal/platform"
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

	plat, err := platform.New(cfg.Platform, cfg.ClientID, cfg.Verbose)
	if err != nil {
		slog.Error("creating platform", "err", err)
		os.Exit(1)
	}

	store := auth.NewStore("data/tokens/token.json")
	token, err := plat.Authenticate(store)
	if err != nil {
		slog.Error("authentication", "err", err)
		os.Exit(1)
	}

	msgs, err := plat.StartChat(context.Background(), token, cfg.Channels)
	if err != nil {
		slog.Error("starting chat", "err", err)
		os.Exit(1)
	}

	det := detector.New(cfg.Thresholds)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go det.Start(ctx)

	slog.Info("connected", "platform", plat.Name(), "channels", cfg.Channels)

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
			slog.Info("trigger", "streamer", trigger.Streamer, "ratio", trigger.Ratio)
			go handleTrigger(plat, token, trigger.Streamer)
		case <-sig:
			slog.Info("shutting down")
			return
		}
	}
}

func handleTrigger(plat platform.Interface, token *auth.Token, streamer string) {
	result, err := plat.CreateClip(context.Background(), token, streamer)
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
