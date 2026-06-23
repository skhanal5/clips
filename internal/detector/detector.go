// Package detector monitors chat activity and emits clip triggers on spike detection.
package detector

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/skhanal5/clips/internal/chat"
	"github.com/skhanal5/clips/internal/config"
)

// ClipTrigger is emitted when a channel's chat activity spikes above the baseline ratio.
type ClipTrigger struct {
	Streamer  string
	Timestamp time.Time
	Ratio     float64
}

// Detector maintains messages per channel and evaluates short-window vs baseline rate.
type Detector struct {
	cfg      config.Thresholds
	channels map[string]*channelState
	mu       sync.Mutex
	out      chan ClipTrigger
}

type channelState struct {
	messages    []chat.Message
	lastTrigger time.Time
}

// New creates a Detector with the given threshold config.
func New(cfg config.Thresholds) *Detector {
	return &Detector{
		cfg:      cfg,
		channels: make(map[string]*channelState),
		out:      make(chan ClipTrigger, 10),
	}
}

// Feed adds a chat message to the detector.
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

	now := time.Now()
	shortCutoff := now.Add(-time.Duration(d.cfg.EvaluationSeconds) * time.Second)
	longCutoff := now.Add(-time.Duration(d.cfg.BaselineSeconds) * time.Second)

	for channel, state := range d.channels {
		var shortCount, longCount int
		keepIdx := 0

		for i, m := range state.messages {
			if m.Timestamp.After(longCutoff) {
				longCount++
				if m.Timestamp.After(shortCutoff) {
					shortCount++
				}
				if keepIdx != i {
					state.messages[keepIdx] = state.messages[i]
				}
				keepIdx++
			}
		}
		state.messages = state.messages[:keepIdx]

		if shortCount < 3 {
			continue
		}
		if longCount == shortCount {
			continue
		}

		shortDur := time.Duration(d.cfg.EvaluationSeconds) * time.Second
		longDur := time.Duration(d.cfg.BaselineSeconds) * time.Second
		shortRate := float64(shortCount) / shortDur.Seconds()
		longRate := float64(longCount) / longDur.Seconds()
		ratio := shortRate / longRate

		if ratio < d.cfg.TriggerRatio {
			slog.Debug("below trigger ratio", "channel", channel,
				"ratio", ratio, "short", shortCount, "long", longCount)
			continue
		}

		if now.Sub(state.lastTrigger) < time.Duration(d.cfg.CooldownSeconds)*time.Second {
			slog.Debug("trigger suppressed by cooldown", "channel", channel, "ratio", ratio)
			continue
		}
		state.lastTrigger = now

		select {
		case d.out <- ClipTrigger{Streamer: channel, Timestamp: now, Ratio: ratio}:
			slog.Info("clip trigger", "channel", channel, "ratio", ratio,
				"short_rate", shortRate, "long_rate", longRate)
		default:
		}
	}
}
