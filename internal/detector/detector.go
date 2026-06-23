// Package detector monitors chat velocity and emits clip triggers on spike detection.
package detector

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/skhanal5/clippy-agent/internal/chat"
	"github.com/skhanal5/clippy-agent/internal/config"
)

// ClipTrigger is emitted when a channel's chat activity exceeds the threshold.
type ClipTrigger struct {
	Streamer  string
	Timestamp time.Time
	Score     float64
}

// Detector maintains per-channel rolling windows and evaluates chat activity.
type Detector struct {
	config   config.Thresholds
	channels map[string]*channelState
	mu       sync.Mutex
	out      chan ClipTrigger
}

type channelState struct {
	messages    []chat.Message
	lastTrigger time.Time
}

// New creates a Detector with the given thresholds.
func New(cfg config.Thresholds) *Detector {
	return &Detector{
		config:   cfg,
		channels: make(map[string]*channelState),
		out:      make(chan ClipTrigger, 10),
	}
}

// Feed adds a chat message to the detector for scoring.
func (d *Detector) Feed(msg chat.Message) {
	d.mu.Lock()
	defer d.mu.Unlock()

	state, ok := d.channels[msg.Channel]
	if !ok {
		state = &channelState{}
		d.channels[msg.Channel] = state
	}
	state.messages = append(state.messages, msg)
}

// Triggers returns a read-only channel of ClipTrigger events.
func (d *Detector) Triggers() <-chan ClipTrigger {
	return d.out
}

// Start runs the evaluation loop until the context is cancelled.
func (d *Detector) Start(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			d.evaluate()
		case <-ctx.Done():
			return
		}
	}
}

func (d *Detector) evaluate() {
	d.mu.Lock()
	defer d.mu.Unlock()

	windowDur := time.Duration(d.config.EvaluationWindow) * time.Second
	now := time.Now()
	cutoff := now.Add(-windowDur)

	for channel, state := range d.channels {
		var recent []chat.Message
		totalEmotes := 0
		users := make(map[string]struct{})

		for _, m := range state.messages {
			if m.Timestamp.After(cutoff) {
				recent = append(recent, m)
				users[m.User] = struct{}{}
				totalEmotes += m.EmoteCount
			}
		}
		state.messages = recent

		if len(recent) < 3 {
			continue
		}

		mps := float64(len(recent)) / windowDur.Seconds()
		uniqueUsers := len(users)

		score := d.computeScore(mps, uniqueUsers, totalEmotes)
		if score < 1.0 {
			continue
		}

		if now.Sub(state.lastTrigger) < time.Duration(d.config.CooldownSeconds)*time.Second {
			slog.Debug("trigger suppressed by cooldown", "channel", channel, "score", score)
			continue
		}
		state.lastTrigger = now

		select {
		case d.out <- ClipTrigger{Streamer: channel, Timestamp: now, Score: score}:
			slog.Info("clip trigger", "channel", channel, "score", score,
				"mps", mps, "users", uniqueUsers, "emotes", totalEmotes)
		default:
		}
	}
}

func (d *Detector) computeScore(mps float64, users, emotes int) float64 {
	vel := mps / float64(d.config.MessagesPerSecond)
	if vel > 1.0 {
		vel = 1.0
	}

	usr := float64(users) / float64(d.config.UniqueUsers)
	if usr > 1.0 {
		usr = 1.0
	}

	emt := float64(emotes) / float64(d.config.EmotesPerWindow)
	if emt > 1.0 {
		emt = 1.0
	}

	return (vel + usr + emt) / 3.0
}
