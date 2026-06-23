package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	fhttp "github.com/bogdanfinn/fhttp"
	"github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
	"github.com/gorilla/websocket"

	"github.com/skhanal5/clips/internal/auth"
	"github.com/skhanal5/clips/internal/chat"
	"github.com/skhanal5/clips/internal/clip"
)

const pusherAppKey = "eb1d5f2830810a534a6b"
const pusherURL = "wss://ws-us2.pusher.com/app/" + pusherAppKey + "?protocol=7&client=go&version=1.5.3&flash=false"

var kickClient = newKickClient()

func newKickClient() tls_client.HttpClient {
	c, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(30),
		tls_client.WithClientProfile(profiles.Chrome_124),
	}...)
	if err != nil {
		panic(fmt.Sprintf("tls client: %v", err))
	}
	return c
}

type kick struct{}

func (k *kick) Name() string { return "kick" }

func (k *kick) Authenticate(_ *auth.Store) (*auth.Token, error) {
	return &auth.Token{}, nil
}

func (k *kick) StartChat(ctx context.Context, _ *auth.Token, channels []string) (<-chan chat.Message, error) {
	var chatroomIDs []uint64
	for _, ch := range channels {
		id, err := resolveChatroomID(ch)
		if err != nil {
			return nil, fmt.Errorf("resolve %s: %w", ch, err)
		}
		chatroomIDs = append(chatroomIDs, id)
	}

	c, _, err := websocket.DefaultDialer.DialContext(ctx, pusherURL, nil)
	if err != nil {
		return nil, fmt.Errorf("pusher connect: %w", err)
	}

	_, _, err = c.ReadMessage()
	if err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("pusher handshake: %w", err)
	}

	out := make(chan chat.Message, 1000)

	for _, id := range chatroomIDs {
		sub := pusherEvent{Event: "pusher:subscribe", Channel: fmt.Sprintf("chatrooms.%d.v2", id)}
		data, _ := json.Marshal(sub)
		if err := c.WriteMessage(websocket.TextMessage, data); err != nil {
			_ = c.Close()
			return nil, fmt.Errorf("pusher subscribe: %w", err)
		}
	}

	go func() {
		defer func() { _ = c.Close() }()
		defer close(out)

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			_ = c.SetReadDeadline(time.Now().Add(3 * time.Minute))
			_, raw, err := c.ReadMessage()
			if err != nil {
				slog.Error("pusher read", "err", err)
				return
			}

			var evt pusherEvent
			if err := json.Unmarshal(raw, &evt); err != nil {
				slog.Debug("pusher unmarshal", "err", err)
				continue
			}

			switch evt.Event {
			case "pusher:ping":
				pong, _ := json.Marshal(pusherEvent{Event: "pusher:pong"})
				_ = c.WriteMessage(websocket.TextMessage, pong)
				continue
			case "App\\Events\\ChatMessageEvent":
			default:
				continue
			}

			var msg chatMessage
			if err := json.Unmarshal([]byte(evt.Data), &msg); err != nil {
				slog.Debug("push msg decode", "err", err)
				continue
			}

			out <- chat.Message{
				User:      msg.Sender.Username,
				Channel:   msg.Sender.Slug,
				Text:      msg.Content,
				Timestamp: time.Now(),
			}
		}
	}()

	return out, nil
}

func (k *kick) CreateClip(_ context.Context, _ *auth.Token, channel string) (*clip.Result, error) {
	clips, err := fetchClips(channel)
	if err != nil {
		return nil, fmt.Errorf("fetch clips: %w", err)
	}
	if len(clips) == 0 {
		return nil, fmt.Errorf("no clips found for %s", channel)
	}

	c := clips[0]
	return &clip.Result{
		ID:  fmt.Sprintf("%d", c.ID),
		URL: fmt.Sprintf("https://kick.com/%s?clip=%d", channel, c.ID),
	}, nil
}

type pusherEvent struct {
	Event   string `json:"event"`
	Data    string `json:"data,omitempty"`
	Channel string `json:"channel,omitempty"`
}

type chatMessage struct {
	ID         string     `json:"id"`
	ChatroomID uint64     `json:"chatroom_id"`
	Content    string     `json:"content"`
	Sender     chatSender `json:"sender"`
	Type       string     `json:"type"`
}

type chatSender struct {
	ID       uint64 `json:"id"`
	Username string `json:"username"`
	Slug     string `json:"slug"`
}

type channelResponse struct {
	Chatroom struct {
		ID uint64 `json:"id"`
	} `json:"chatroom"`
}

type kickClip struct {
	ID uint64 `json:"id"`
}

type clipsResponse struct {
	Clips []kickClip `json:"clips"`
}

func fetchClips(slug string) ([]kickClip, error) {
	req, _ := fhttp.NewRequest("GET", "https://kick.com/api/v2/channels/"+slug+"/clips?cursor=0&sort=recency", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "application/json")

	resp, err := kickClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("kick clips api: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("kick clips api: status %d", resp.StatusCode)
	}

	var cr clipsResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return nil, fmt.Errorf("decode clips: %w", err)
	}
	return cr.Clips, nil
}

func resolveChatroomID(slug string) (uint64, error) {
	req, _ := fhttp.NewRequest("GET", "https://kick.com/api/v2/channels/"+slug, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "application/json")

	resp, err := kickClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("kick api: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("kick api: status %d", resp.StatusCode)
	}

	var cr channelResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return 0, fmt.Errorf("decode channel: %w", err)
	}
	if cr.Chatroom.ID == 0 {
		return 0, fmt.Errorf("no chatroom for %s", slug)
	}
	return cr.Chatroom.ID, nil
}
