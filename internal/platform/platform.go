// Package platform abstracts platform-specific auth, chat, and clip creation.
package platform

import (
	"context"
	"fmt"

	"github.com/skhanal5/clips/internal/auth"
	"github.com/skhanal5/clips/internal/chat"
	"github.com/skhanal5/clips/internal/clip"
)

// Interface represents a platform that can provide chat messages and create clips.
type Interface interface {
	Name() string
	Authenticate(store *auth.Store) (*auth.Token, error)
	StartChat(ctx context.Context, token *auth.Token, channels []string) (<-chan chat.Message, error)
	CreateClip(ctx context.Context, token *auth.Token, channel string) (*clip.Result, error)
}

// New creates a platform Interface for the given name.
func New(platformName, clientID string, verbose bool) (Interface, error) {
	switch platformName {
	case "twitch":
		return &twitch{clientID: clientID, verbose: verbose}, nil
	case "kick":
		return &kick{}, nil
	default:
		return nil, fmt.Errorf("unknown platform: %s", platformName)
	}
}
