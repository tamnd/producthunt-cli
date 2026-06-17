package producthunt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// auth.go resolves the api-plane bearer token. A ready PRODUCTHUNT_TOKEN is used
// as-is. Otherwise a PRODUCTHUNT_CLIENT_ID and PRODUCTHUNT_CLIENT_SECRET pair is
// exchanged once for a public-scope token through the OAuth client-credentials
// grant, and the minted token is held in memory for the run. This is the sanctioned
// way to read public data with an application's own credentials; it is not a login,
// a password grant, or a credential the tool does not own. The token is never
// logged and never placed in a cache filename.

// token returns the bearer token for the api plane, minting one from client
// credentials on first use if no ready token is set. Absent credentials are
// ErrNeedKey, so a keyless api call fails fast with the right exit code.
func (c *Client) token(ctx context.Context) (string, error) {
	if c.cfg.Token != "" {
		return c.cfg.Token, nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.accessToken != "" {
		return c.accessToken, nil
	}
	if c.cfg.ClientID == "" || c.cfg.ClientSecret == "" {
		return "", ErrNeedKey
	}
	tok, err := c.mintToken(ctx)
	if err != nil {
		return "", err
	}
	c.accessToken = tok
	return tok, nil
}

// mintToken performs the client-credentials handshake. Bad credentials map to
// ErrBlocked (need-auth, exit 4); a transport failure is wrapped.
func (c *Client) mintToken(ctx context.Context) (string, error) {
	reqBody, err := json.Marshal(map[string]string{
		"client_id":     c.cfg.ClientID,
		"client_secret": c.cfg.ClientSecret,
		"grant_type":    "client_credentials",
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.OAuthURL, bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.cfg.UserAgent)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return "", ErrBlocked
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("oauth token: http %d", resp.StatusCode)
	}
	var out struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("decode oauth token: %w", err)
	}
	if out.AccessToken == "" {
		return "", ErrBlocked
	}
	return out.AccessToken, nil
}
