// Package auth handles Twitch OAuth device-code flow and token persistence.
package auth

import (
	"encoding/json"
	"os"
	"time"
)

// Token represents a Twitch OAuth token with its metadata.
type Token struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	Expiry       time.Time `json:"expiry"`
	Username     string    `json:"username"`
}

// Valid returns true if the token has not yet expired.
func (t *Token) Valid() bool {
	return time.Now().Before(t.Expiry)
}

// Store persists tokens to disk as JSON.
type Store struct {
	path string
}

// NewStore creates a Store that writes to the given file path.
func NewStore(path string) *Store {
	return &Store{path: path}
}

// Load reads a token from the store file.
func (s *Store) Load() (*Token, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil, err
	}
	var token Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}
	return &token, nil
}

// Save writes a token to the store file.
func (s *Store) Save(token *Token) error {
	dir := s.path[:len(s.path)-len("/token.json")]
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0600)
}
