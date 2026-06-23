package platform

import (
	"context"

	"github.com/skhanal5/clips/internal/auth"
	"github.com/skhanal5/clips/internal/chat"
	"github.com/skhanal5/clips/internal/clip"
)

type twitch struct {
	clientID string
	verbose  bool
}

func (t *twitch) Name() string { return "twitch" }

func (t *twitch) Authenticate(store *auth.Store) (*auth.Token, error) {
	return auth.EnsureToken(t.clientID, []string{"chat:read", "clips:edit"}, store)
}

func (t *twitch) StartChat(ctx context.Context, token *auth.Token, channels []string) (<-chan chat.Message, error) {
	m := chat.New(token.AccessToken, token.Username, channels, t.verbose)
	return m.Start(ctx)
}

func (t *twitch) CreateClip(_ context.Context, token *auth.Token, channel string) (*clip.Result, error) {
	svc := clip.NewService(t.clientID, token.AccessToken)
	return svc.CreateClip(channel)
}
