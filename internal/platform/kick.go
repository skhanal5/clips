package platform

import (
	"context"
	"errors"

	"github.com/skhanal5/clips/internal/auth"
	"github.com/skhanal5/clips/internal/chat"
	"github.com/skhanal5/clips/internal/clip"
)

type kick struct{}

func (k *kick) Name() string { return "kick" }

func (k *kick) Authenticate(_ *auth.Store) (*auth.Token, error) {
	return nil, errors.New("kick auth not implemented")
}

func (k *kick) StartChat(_ context.Context, _ *auth.Token, _ []string) (<-chan chat.Message, error) {
	return nil, errors.New("kick chat not implemented")
}

func (k *kick) CreateClip(_ context.Context, _ *auth.Token, _ string) (*clip.Result, error) {
	return nil, errors.New("kick clip creation not implemented")
}
