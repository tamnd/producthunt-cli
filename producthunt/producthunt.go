// Package producthunt is the library behind the ph command line: the HTTP client
// for both planes, the offline reference layer, and the typed records read from
// public Product Hunt surfaces.
//
// Product Hunt has two planes. The web plane (www.producthunt.com) is the default,
// but every page is fronted by Cloudflare, so the one anonymous surface that
// survives is the Atom feed at /feed; a walled response becomes ErrBlocked before
// any parser runs. The api plane (api.producthunt.com/v2/api/graphql) is the opt-in
// upgrade, turned on by a PRODUCTHUNT_TOKEN or a PRODUCTHUNT_CLIENT_ID and
// PRODUCTHUNT_CLIENT_SECRET in the environment, and reads reliably from anywhere.
// The Client below fetches both, paces and retries politely, caches on disk, and
// turns a walled or rejected response into a typed error the exit-code mapping
// understands.
package producthunt

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Client reads public Product Hunt data over HTTP on both planes.
type Client struct {
	HTTP *http.Client
	cfg  Config

	mu          sync.Mutex
	last        time.Time
	accessToken string // a token minted from client credentials, cached for the run
}

// NewClient returns a Client configured from cfg, filling unset fields with their
// defaults.
func NewClient(cfg Config) *Client {
	if cfg.UserAgent == "" {
		cfg.UserAgent = DefaultUserAgent
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = BaseURL
	}
	if cfg.APIURL == "" {
		cfg.APIURL = APIURL
	}
	if cfg.OAuthURL == "" {
		cfg.OAuthURL = OAuthURL
	}
	return &Client{
		HTTP: &http.Client{Timeout: cfg.Timeout},
		cfg:  cfg,
	}
}

// get fetches a web-plane URL and returns the body. It serves from cache when
// fresh, paces and retries transient failures, and classifies a walled response as
// ErrBlocked.
func (c *Client) get(ctx context.Context, rawURL string) ([]byte, error) {
	if b := c.cacheGet(rawURL); b != nil {
		return b, nil
	}
	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, rawURL)
		if err == nil {
			c.cachePut(rawURL, body)
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	if errors.Is(lastErr, ErrRateLimited) {
		return nil, ErrRateLimited
	}
	return nil, fmt.Errorf("get %s: %w", rawURL, lastErr)
}

func (c *Client) do(ctx context.Context, rawURL string) (body []byte, retry bool, err error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)
	req.Header.Set("Accept", "application/atom+xml, application/xml, text/xml, text/html")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	switch {
	case resp.StatusCode == http.StatusForbidden, resp.StatusCode == http.StatusServiceUnavailable:
		return nil, false, ErrBlocked
	case resp.StatusCode == http.StatusNotFound:
		return nil, false, ErrNotFound
	case resp.StatusCode == http.StatusTooManyRequests:
		return nil, true, ErrRateLimited
	case resp.StatusCode >= 500:
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	case resp.StatusCode != http.StatusOK:
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, true, err
	}
	if isChallenge(b) {
		return nil, false, ErrBlocked
	}
	return b, false, nil
}

// postGraphQL sends a GraphQL query to the api plane and returns the data field of
// the response. It resolves the bearer token (from the env token or the
// client-credentials handshake) on first use, caches by the request body so the
// token never affects the cache, and classifies the response (a rejected/missing
// token, a rate limit, a missing node) before any decoder runs.
func (c *Client) postGraphQL(ctx context.Context, query string, vars map[string]any) (json.RawMessage, error) {
	tok, err := c.token(ctx)
	if err != nil {
		return nil, err
	}
	reqBody, err := json.Marshal(map[string]any{"query": query, "variables": vars})
	if err != nil {
		return nil, err
	}
	ckey := string(reqBody)
	if b := c.cacheGet(ckey); b != nil {
		return classifyGraphQL(http.StatusOK, b)
	}
	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, status, retry, err := c.doPost(ctx, c.cfg.APIURL, tok, reqBody)
		if err != nil {
			lastErr = err
			if retry {
				continue
			}
			return nil, err
		}
		if status >= 500 {
			lastErr = fmt.Errorf("http %d", status)
			continue
		}
		data, cerr := classifyGraphQL(status, body)
		if errors.Is(cerr, ErrRateLimited) {
			lastErr = cerr
			continue
		}
		if cerr != nil {
			return nil, cerr
		}
		c.cachePut(ckey, body)
		return data, nil
	}
	if errors.Is(lastErr, ErrRateLimited) {
		return nil, ErrRateLimited
	}
	return nil, fmt.Errorf("graphql: %w", lastErr)
}

// doPost performs one POST and returns the body, the status, and whether a
// transport-level failure is worth retrying.
func (c *Client) doPost(ctx context.Context, urlStr, token string, body []byte) (resp []byte, status int, retry bool, err error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlStr, bytes.NewReader(body))
	if err != nil {
		return nil, 0, false, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.cfg.UserAgent)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	r, err := c.HTTP.Do(req)
	if err != nil {
		return nil, 0, true, err
	}
	defer func() { _ = r.Body.Close() }()

	b, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, r.StatusCode, true, err
	}
	return b, r.StatusCode, false, nil
}

// classifyGraphQL turns a GraphQL response into its data field, or a typed error: a
// 401 or an invalid_oauth_token error is the rejected/missing token (ErrBlocked), a
// 429 or a complexity/throttle error is the cap (ErrRateLimited). A non-error
// envelope returns its data.
func classifyGraphQL(status int, body []byte) (json.RawMessage, error) {
	if status == http.StatusTooManyRequests {
		return nil, ErrRateLimited
	}
	var env struct {
		Data   json.RawMessage `json:"data"`
		Errors []struct {
			Message string `json:"message"`
			Error   string `json:"error"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		if status == http.StatusUnauthorized {
			return nil, ErrBlocked
		}
		return nil, fmt.Errorf("decode graphql response: %w", err)
	}
	for _, e := range env.Errors {
		kind := strings.ToLower(e.Error + " " + e.Message)
		switch {
		case strings.Contains(kind, "invalid_oauth_token"),
			strings.Contains(kind, "unauthorized"),
			strings.Contains(kind, "invalid token"):
			return nil, ErrBlocked
		case strings.Contains(kind, "rate"),
			strings.Contains(kind, "complexity"),
			strings.Contains(kind, "throttle"):
			return nil, ErrRateLimited
		}
	}
	if status == http.StatusUnauthorized {
		return nil, ErrBlocked
	}
	if len(env.Errors) > 0 {
		msg := env.Errors[0].Message
		if msg == "" {
			msg = env.Errors[0].Error
		}
		return nil, fmt.Errorf("graphql: %s", msg)
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("http %d", status)
	}
	return env.Data, nil
}

// pace blocks until at least Delay has passed since the previous request.
func (c *Client) pace() {
	if c.cfg.Delay <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if wait := c.cfg.Delay - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}

// challengeMarkers are byte signatures of a Cloudflare interstitial served with a
// 200 in place of the real page.
var challengeMarkers = [][]byte{
	[]byte("challenges.cloudflare.com"),
	[]byte("window._cf_chl_opt"),
	[]byte("just a moment..."),
	[]byte("enable javascript and cookies to continue"),
	[]byte("cf-browser-verification"),
}

// isChallenge reports whether a 200 body is a Cloudflare challenge rather than a
// real page, by looking for a known marker in the head of the body.
func isChallenge(body []byte) bool {
	head := body
	if len(head) > 8192 {
		head = head[:8192]
	}
	lower := bytes.ToLower(head)
	for _, m := range challengeMarkers {
		if bytes.Contains(lower, m) {
			return true
		}
	}
	return false
}

// squish collapses internal whitespace and trims, for text pulled out of HTML.
func squish(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
