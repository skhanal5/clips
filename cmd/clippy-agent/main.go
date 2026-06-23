// Command clippy-agent is the Twitch clip automation pipeline.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

	"github.com/skhanal5/clippy-agent/internal/auth"
	"github.com/skhanal5/clippy-agent/internal/chat"
	"github.com/skhanal5/clippy-agent/internal/config"
)

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		slog.Error("loading config", "err", err)
		os.Exit(1)
	}

	store := auth.NewStore("data/tokens/token.json")
	token, err := auth.EnsureToken(cfg.ClientID, []string{"chat:read", "clips:edit"}, store)
	if err != nil {
		slog.Error("authentication", "err", err)
		os.Exit(1)
	}

	monitor := chat.New(token.AccessToken, token.Username, cfg.Channels)
	msgs, err := monitor.Start(context.Background())
	if err != nil {
		slog.Error("starting chat monitor", "err", err)
		os.Exit(1)
	}

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
			slog.Info("chat", "channel", msg.Channel, "user", msg.User, "text", msg.Text)
		case <-sig:
			slog.Info("shutting down")
			monitor.Stop()
			return
		}
	}
}
