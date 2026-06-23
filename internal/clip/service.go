// Package clip creates Twitch clips via the Helix API.
package clip

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Result holds the metadata for a created clip.
type Result struct {
	ID  string
	URL string
}

// Service creates Twitch clips for a given streamer.
type Service struct {
	clientID    string
	accessToken string
	httpClient  *http.Client
}

// NewService creates a clip service with the given Twitch credentials.
func NewService(clientID, accessToken string) *Service {
	return &Service{
		clientID:    clientID,
		accessToken: accessToken,
		httpClient:  &http.Client{Timeout: 10 * time.Second},
	}
}

type helixUserResponse struct {
	Data []struct {
		ID    string `json:"id"`
		Login string `json:"login"`
	} `json:"data"`
}

type createClipResponse struct {
	Data []struct {
		ID      string `json:"id"`
		EditURL string `json:"edit_url"`
	} `json:"data"`
}

func (s *Service) getBroadcasterID(channelName string) (string, error) {
	req, _ := http.NewRequest("GET", "https://api.twitch.tv/helix/users?login="+url.QueryEscape(channelName), nil)
	req.Header.Set("Authorization", "Bearer "+s.accessToken)
	req.Header.Set("Client-Id", s.clientID)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("user lookup: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("user lookup failed: status %d", resp.StatusCode)
	}

	var ur helixUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&ur); err != nil {
		return "", fmt.Errorf("decode user response: %w", err)
	}
	if len(ur.Data) == 0 {
		return "", fmt.Errorf("streamer %s not found", channelName)
	}
	return ur.Data[0].ID, nil
}

// CreateClip creates a Twitch clip for the given channel and returns its ID and URL.
func (s *Service) CreateClip(channelName string) (*Result, error) {
	broadcasterID, err := s.getBroadcasterID(channelName)
	if err != nil {
		return nil, fmt.Errorf("broadcaster lookup: %w", err)
	}

	req, _ := http.NewRequest("POST", "https://api.twitch.tv/helix/clips?broadcaster_id="+broadcasterID, nil)
	req.Header.Set("Authorization", "Bearer "+s.accessToken)
	req.Header.Set("Client-Id", s.clientID)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("create clip: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("create clip failed: status %d", resp.StatusCode)
	}

	var cr createClipResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return nil, fmt.Errorf("decode clip response: %w", err)
	}
	if len(cr.Data) == 0 {
		return nil, fmt.Errorf("no clip data in response")
	}

	clipID := cr.Data[0].ID
	return &Result{
		ID:  clipID,
		URL: "https://clips.twitch.tv/" + clipID,
	}, nil
}
