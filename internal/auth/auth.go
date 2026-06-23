package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type deviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

type userResponse struct {
	Data []struct {
		ID          string `json:"id"`
		Login       string `json:"login"`
		DisplayName string `json:"display_name"`
	} `json:"data"`
}

// DeviceCodeFlow runs the Twitch OAuth device code flow.
// It prints a verification URL and code, then polls until the user authorizes.
func DeviceCodeFlow(clientID string, scopes []string) (*Token, error) {
	resp, err := http.PostForm("https://id.twitch.tv/oauth2/device", url.Values{
		"client_id": {clientID},
		"scopes":    {strings.Join(scopes, " ")},
	})
	if err != nil {
		return nil, fmt.Errorf("device code request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var dc deviceCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&dc); err != nil {
		return nil, fmt.Errorf("decode device code: %w", err)
	}

	fmt.Printf("\nOpen %s and enter code: %s\n\n", dc.VerificationURI, dc.UserCode)

	interval := time.Duration(dc.Interval) * time.Second
	if interval < 5*time.Second {
		interval = 5 * time.Second
	}

	deadline := time.Now().Add(time.Duration(dc.ExpiresIn) * time.Second)

	for time.Now().Before(deadline) {
		time.Sleep(interval)

		resp, err := http.PostForm("https://id.twitch.tv/oauth2/token", url.Values{
			"client_id":   {clientID},
			"device_code": {dc.DeviceCode},
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		})
		if err != nil {
			return nil, fmt.Errorf("token poll: %w", err)
		}

		if resp.StatusCode == http.StatusOK {
			var tr tokenResponse
			if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
				_ = resp.Body.Close()
				return nil, fmt.Errorf("decode token: %w", err)
			}
			_ = resp.Body.Close()

			username, err := fetchUsername(clientID, tr.AccessToken)
			if err != nil {
				return nil, fmt.Errorf("fetch user: %w", err)
			}

			return &Token{
				AccessToken:  tr.AccessToken,
				RefreshToken: tr.RefreshToken,
				Expiry:       time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second),
				Username:     username,
			}, nil
		}

		var errResp struct {
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		_ = resp.Body.Close()

		switch errResp.Error {
		case "authorization_pending":
		case "slow_down":
			interval += 5 * time.Second
		case "access_denied":
			return nil, fmt.Errorf("access denied by user")
		default:
			return nil, fmt.Errorf("auth error: %s", errResp.ErrorDescription)
		}
	}

	return nil, fmt.Errorf("authorization timed out")
}

// RefreshToken exchanges a refresh token for a new access token.
func RefreshToken(clientID, refreshToken string) (*Token, error) {
	resp, err := http.PostForm("https://id.twitch.tv/oauth2/token", url.Values{
		"client_id":     {clientID},
		"refresh_token": {refreshToken},
		"grant_type":    {"refresh_token"},
	})
	if err != nil {
		return nil, fmt.Errorf("refresh request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("refresh failed: status %d", resp.StatusCode)
	}

	var tr tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return nil, fmt.Errorf("decode refresh: %w", err)
	}

	return &Token{
		AccessToken:  tr.AccessToken,
		RefreshToken: tr.RefreshToken,
		Expiry:       time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second),
	}, nil
}

func fetchUsername(clientID, accessToken string) (string, error) {
	req, _ := http.NewRequest("GET", "https://api.twitch.tv/helix/users", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Client-Id", clientID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	var ur userResponse
	if err := json.NewDecoder(resp.Body).Decode(&ur); err != nil {
		return "", err
	}
	if len(ur.Data) == 0 {
		return "", fmt.Errorf("no user data in response")
	}
	return ur.Data[0].Login, nil
}

// EnsureToken returns a valid token, using stored token, refresh, or device code flow.
func EnsureToken(clientID string, scopes []string, store *Store) (*Token, error) {
	token, err := store.Load()
	if err == nil && token.Valid() {
		return token, nil
	}

	if err == nil && token.RefreshToken != "" {
		newToken, err := RefreshToken(clientID, token.RefreshToken)
		if err == nil {
			newToken.Username = token.Username
			if saveErr := store.Save(newToken); saveErr != nil {
				return nil, saveErr
			}
			return newToken, nil
		}
	}

	token, err = DeviceCodeFlow(clientID, scopes)
	if err != nil {
		return nil, err
	}

	if saveErr := store.Save(token); saveErr != nil {
		return nil, saveErr
	}
	return token, nil
}
