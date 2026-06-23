// Package chat monitors Twitch chat via IRC and emits structured messages.
package chat

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/gempir/go-twitch-irc/v4"
)

// Message represents a single Twitch chat message.
type Message struct {
	User       string
	Channel    string
	Text       string
	Timestamp  time.Time
	EmoteCount int
}

// Monitor connects to Twitch IRC and streams chat messages.
type Monitor struct {
	username string
	token    string
	channels []string
	verbose  bool
	msgs     chan Message
	client   *twitch.Client
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

// New creates a Monitor for the given channels.
func New(token, username string, channels []string, verbose bool) *Monitor {
	return &Monitor{
		username: username,
		token:    token,
		channels: channels,
		verbose:  verbose,
		msgs:     make(chan Message, 1000),
	}
}

// Start connects to Twitch IRC and returns a channel of messages.
func (m *Monitor) Start(ctx context.Context) (<-chan Message, error) {
	ctx, cancel := context.WithCancel(ctx)
	m.cancel = cancel

	client := twitch.NewClient(m.username, "oauth:"+m.token)
	m.client = client

	for _, ch := range m.channels {
		client.Join(ch)
	}

	client.OnPrivateMessage(func(msg twitch.PrivateMessage) {
		emoteCount := 0
		for _, e := range msg.Emotes {
			emoteCount += e.Count
		}
		if m.verbose {
			slog.Debug("chat message",
				"channel", msg.Channel,
				"user", msg.User.Name,
				"text", msg.Message,
				"emotes", emoteCount,
			)
		}
		m.msgs <- Message{
			User:       msg.User.Name,
			Channel:    msg.Channel,
			Text:       msg.Message,
			Timestamp:  msg.Time,
			EmoteCount: emoteCount,
		}
	})

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		defer cancel()

		if err := client.Connect(); err != nil {
			slog.Error("irc connect failed", "err", err)
			close(m.msgs)
		}
	}()

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		<-ctx.Done()
		_ = client.Disconnect()
	}()

	return m.msgs, nil
}

// Stop disconnects from IRC and waits for the goroutines to finish.
func (m *Monitor) Stop() {
	if m.cancel != nil {
		m.cancel()
	}
	m.wg.Wait()
}
