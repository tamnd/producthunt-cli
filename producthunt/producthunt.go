// Package producthunt is the library behind the ph command: the HTTP client,
// request shaping, and the typed data models for Product Hunt.
//
// Data source: the public Atom feed at https://www.producthunt.com/feed.
// No authentication or API key is required.
package producthunt

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DefaultUserAgent identifies the client to Product Hunt.
const DefaultUserAgent = "ph/dev (+https://github.com/tamnd/producthunt-cli)"

// Config holds constructor parameters for Client.
type Config struct {
	BaseURL   string
	UserAgent string
	Rate      time.Duration
	Retries   int
	Timeout   time.Duration
}

// DefaultConfig returns sensible defaults: no rate limiting (one request per
// command), two retries on transient errors, and a 30-second timeout.
func DefaultConfig() Config {
	return Config{
		BaseURL:   "https://www.producthunt.com",
		UserAgent: DefaultUserAgent,
		Rate:      0,
		Retries:   2,
		Timeout:   30 * time.Second,
	}
}

// Client talks to Product Hunt over HTTP.
type Client struct {
	http      *http.Client
	userAgent string
	rate      time.Duration
	retries   int
	baseURL   string

	last time.Time
}

// NewClient returns a Client built from cfg.
func NewClient(cfg Config) *Client {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://www.producthunt.com"
	}
	return &Client{
		http:      &http.Client{Timeout: cfg.Timeout},
		userAgent: cfg.UserAgent,
		rate:      cfg.Rate,
		retries:   cfg.Retries,
		baseURL:   baseURL,
	}
}

// Get fetches url and returns the response body. It paces and retries
// according to the client's settings.
func (c *Client) Get(ctx context.Context, url string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, url)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", url, lastErr)
}

func (c *Client) do(ctx context.Context, url string) (body []byte, retry bool, err error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/atom+xml, application/xml, text/xml")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}

	b, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

func (c *Client) pace() {
	if c.rate <= 0 {
		return
	}
	if wait := c.rate - time.Since(c.last); wait > 0 {
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

// Today fetches the Product Hunt Atom feed and returns up to limit Product
// records. If limit is 0 or greater than the number of entries in the feed,
// all entries are returned.
func (c *Client) Today(ctx context.Context, limit int) ([]Product, error) {
	body, err := c.Get(ctx, c.baseURL+"/feed")
	if err != nil {
		return nil, err
	}
	var feed atomFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("parse feed: %w", err)
	}
	out := make([]Product, 0, len(feed.Entries))
	for i, e := range feed.Entries {
		name, tagline := splitTitle(e.Title)
		u := e.Link.Href
		if u == "" {
			u = e.ID
		}
		p := Product{
			Rank:      i + 1,
			Name:      name,
			Tagline:   tagline,
			Author:    e.Author.Name,
			Published: formatDate(e.Published),
			URL:       u,
		}
		out = append(out, p)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

// splitTitle parses "Product Name — Tagline" into its two parts.
// The separator is an em dash (U+2014) surrounded by spaces: " — ".
// Falls back to " - " if the em dash is not found.
func splitTitle(title string) (name, tagline string) {
	const emDashSep = " — "
	if i := strings.Index(title, emDashSep); i >= 0 {
		return strings.TrimSpace(title[:i]), strings.TrimSpace(title[i+len(emDashSep):])
	}
	const hyphenSep = " - "
	if i := strings.Index(title, hyphenSep); i >= 0 {
		return strings.TrimSpace(title[:i]), strings.TrimSpace(title[i+len(hyphenSep):])
	}
	return strings.TrimSpace(title), ""
}

// formatDate parses an RFC 3339 timestamp and returns the date portion
// formatted as "2006-01-02". Returns the input string unchanged on parse error.
func formatDate(published string) string {
	t, err := time.Parse(time.RFC3339, published)
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05Z07:00", published)
	}
	if err != nil {
		return published
	}
	return t.UTC().Format("2006-01-02")
}
