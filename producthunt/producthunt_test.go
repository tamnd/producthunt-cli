package producthunt

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tamnd/any-cli/kit/errs"
)

// testClient returns a client with no pacing, pointed at base for both planes and
// the OAuth endpoint, with the disk cache off.
func testClient(base string) *Client {
	cfg := DefaultConfig()
	cfg.Delay = 0
	cfg.Token = ""
	cfg.ClientID = ""
	cfg.ClientSecret = ""
	cfg.BaseURL = base
	cfg.APIURL = base
	cfg.OAuthURL = base
	cfg.NoCache = true
	return NewClient(cfg)
}

func TestGetSendsUserAgent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("request carried no User-Agent")
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	body, err := testClient(srv.URL).get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "ok" {
		t.Errorf("body = %q, want %q", body, "ok")
	}
}

func TestGetRetriesOn500(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte("recovered"))
	}))
	defer srv.Close()

	c := testClient(srv.URL)
	c.cfg.Retries = 5

	start := time.Now()
	body, err := c.get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "recovered" {
		t.Errorf("body = %q after retries", body)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
	if time.Since(start) < 500*time.Millisecond {
		t.Error("retries did not back off")
	}
}

// TestGetNetworkError checks that a 5xx that never recovers ends as ErrNetwork, so
// mapErr can report it as exit 8 rather than a bare generic failure.
func TestGetNetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := testClient(srv.URL)
	c.cfg.Retries = 1

	_, err := c.get(context.Background(), srv.URL)
	if !errors.Is(err, ErrNetwork) {
		t.Fatalf("err = %v, want ErrNetwork", err)
	}
	if code := errs.ExitCode(mapErr(err)); code != 8 {
		t.Errorf("mapErr exit code = %d, want 8", code)
	}
}

func TestWallDetection(t *testing.T) {
	cases := []struct {
		name    string
		handler http.HandlerFunc
		want    error
	}{
		{
			name: "403 is the wall",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusForbidden)
			},
			want: ErrBlocked,
		},
		{
			name: "503 is the wall",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
			},
			want: ErrBlocked,
		},
		{
			name: "cloudflare interstitial body is the wall",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`<html><head><title>Just a moment...</title></head><body><script src="https://challenges.cloudflare.com/turnstile/v0/api.js"></script></body></html>`))
			},
			want: ErrBlocked,
		},
		{
			name: "clean body passes",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`<feed></feed>`))
			},
			want: nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(tc.handler)
			defer srv.Close()
			c := testClient(srv.URL)
			c.cfg.Retries = 0
			_, err := c.get(context.Background(), srv.URL)
			if tc.want == nil {
				if err != nil {
					t.Fatalf("err = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.want.Error()) && err != tc.want {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

const feedFixture = `<?xml version="1.0" encoding="UTF-8"?>
<feed xml:lang="en-US" xmlns="http://www.w3.org/2005/Atom">
  <id>tag:www.producthunt.com,2005:/feed</id>
  <title>Product Hunt</title>
  <entry>
    <id>tag:www.producthunt.com,2005:Post/1173164</id>
    <published>2026-06-16T02:18:05-07:00</published>
    <updated>2026-06-17T09:57:28-07:00</updated>
    <link rel="alternate" type="text/html" href="https://www.producthunt.com/products/brainflow-2"/>
    <title>BrainFlow</title>
    <content type="html">&lt;p&gt;Turn your rambling thoughts into coherent notes&lt;/p&gt;&lt;p&gt;&lt;a href="https://www.producthunt.com/products/brainflow-2?utm_source=x"&gt;Discussion&lt;/a&gt; | &lt;a href="https://www.producthunt.com/r/p/1173164?app_id=339"&gt;Link&lt;/a&gt;&lt;/p&gt;</content>
    <author><name>Tristan Manchester</name></author>
  </entry>
  <entry>
    <id>tag:www.producthunt.com,2005:Post/1173801</id>
    <published>2026-06-16T15:51:24-07:00</published>
    <updated>2026-06-17T09:57:16-07:00</updated>
    <link rel="alternate" type="text/html" href="https://www.producthunt.com/products/infinite-the-growth-engineering-agent"/>
    <title>Infinite</title>
    <content type="html">&lt;p&gt;OS runtime unifying GA4, PostHog, + Stripe into a local db&lt;/p&gt;&lt;p&gt;&lt;a href="https://www.producthunt.com/r/p/1173801?app_id=339"&gt;Link&lt;/a&gt;&lt;/p&gt;</content>
    <author><name>RTK</name></author>
  </entry>
</feed>`

func feedServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/feed") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/atom+xml")
		_, _ = io.WriteString(w, feedFixture)
	}))
}

func TestFeedParse(t *testing.T) {
	srv := feedServer(t)
	defer srv.Close()

	posts, err := testClient(srv.URL).Feed(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) != 2 {
		t.Fatalf("got %d posts, want 2", len(posts))
	}
	p := posts[0]
	if p.ID != "1173164" {
		t.Errorf("ID = %q, want 1173164", p.ID)
	}
	if p.Name != "BrainFlow" {
		t.Errorf("Name = %q, want BrainFlow", p.Name)
	}
	if p.Tagline != "Turn your rambling thoughts into coherent notes" {
		t.Errorf("Tagline = %q", p.Tagline)
	}
	if p.Slug != "brainflow-2" {
		t.Errorf("Slug = %q, want brainflow-2", p.Slug)
	}
	if p.URL != "https://www.producthunt.com/products/brainflow-2" {
		t.Errorf("URL = %q", p.URL)
	}
	if p.Website != "https://www.producthunt.com/r/p/1173164?app_id=339" {
		t.Errorf("Website = %q", p.Website)
	}
	if p.HunterName != "Tristan Manchester" {
		t.Errorf("HunterName = %q", p.HunterName)
	}
	if p.CommentsRef != "1173164" {
		t.Errorf("CommentsRef = %q, want 1173164", p.CommentsRef)
	}
	// The feed carries no usernames, so the hunter-username edge stays empty.
	if p.Hunter != "" || p.HunterRef != "" {
		t.Errorf("feed post carried a username it cannot know: Hunter=%q HunterRef=%q", p.Hunter, p.HunterRef)
	}
}

func TestFeedLimit(t *testing.T) {
	srv := feedServer(t)
	defer srv.Close()
	posts, err := testClient(srv.URL).Feed(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) != 1 {
		t.Fatalf("got %d posts, want 1", len(posts))
	}
}

func TestFeedPost(t *testing.T) {
	srv := feedServer(t)
	defer srv.Close()
	c := testClient(srv.URL)

	p, err := c.FeedPost(context.Background(), "1173801")
	if err != nil {
		t.Fatalf("FeedPost by id: %v", err)
	}
	if p.Name != "Infinite" {
		t.Errorf("Name = %q, want Infinite", p.Name)
	}
	p, err = c.FeedPost(context.Background(), "brainflow-2")
	if err != nil {
		t.Fatalf("FeedPost by slug: %v", err)
	}
	if p.ID != "1173164" {
		t.Errorf("ID = %q, want 1173164", p.ID)
	}
	if _, err := c.FeedPost(context.Background(), "999999"); err != ErrNotFound {
		t.Errorf("FeedPost(missing) = %v, want ErrNotFound", err)
	}
}

func TestClassifyGraphQL(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		want   error
	}{
		{"invalid token", http.StatusUnauthorized, `{"data":null,"errors":[{"error":"invalid_oauth_token"}]}`, ErrBlocked},
		{"complexity cap", http.StatusOK, `{"errors":[{"message":"Query has complexity that is too high"}]}`, ErrRateLimited},
		{"429", http.StatusTooManyRequests, `{}`, ErrRateLimited},
		{"clean", http.StatusOK, `{"data":{"posts":{"edges":[]}}}`, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := classifyGraphQL(tc.status, []byte(tc.body))
			if err != tc.want {
				t.Errorf("classifyGraphQL = %v, want %v", err, tc.want)
			}
		})
	}
}

// graphqlServer answers POST with the given response body, after asserting the
// request carried a bearer token and a JSON query body.
func graphqlServer(t *testing.T, response string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if got := r.Header.Get("Authorization"); !strings.HasPrefix(got, "Bearer ") {
			t.Errorf("Authorization = %q, want a Bearer token", got)
		}
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal(body, &req); err != nil || req.Query == "" {
			t.Errorf("request body is not a graphql query: %s", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, response)
	}))
}

func keyedClient(base string) *Client {
	c := testClient(base)
	c.cfg.Token = "test-token"
	return c
}

func TestPostsAPIDecode(t *testing.T) {
	const resp = `{"data":{"posts":{"edges":[
		{"node":{"id":"1","name":"Alpha","tagline":"first","slug":"alpha","votesCount":120,"commentsCount":7,"featuredAt":"2026-06-16T00:00:00Z","url":"https://www.producthunt.com/products/alpha",
			"topics":{"edges":[{"node":{"id":"10","name":"AI","slug":"ai"}}]},
			"user":{"id":"99","username":"hunter1","name":"Hunter One"},
			"makers":[{"id":"99","username":"hunter1","name":"Hunter One"},{"id":"100","username":"maker2","name":"Maker Two"}]}}
	],"pageInfo":{"endCursor":"","hasNextPage":false}}}}`
	srv := graphqlServer(t, resp)
	defer srv.Close()

	posts, err := keyedClient(srv.URL).PostsAPI(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) != 1 {
		t.Fatalf("got %d posts, want 1", len(posts))
	}
	p := posts[0]
	if p.ID != "1" || p.Name != "Alpha" || p.Votes != 120 || p.Comments != 7 {
		t.Errorf("post fields wrong: %+v", p)
	}
	if !p.Featured || p.FeaturedAt == "" {
		t.Errorf("Featured not derived from featuredAt: %+v", p)
	}
	if p.Hunter != "hunter1" || p.HunterRef != "hunter1" {
		t.Errorf("hunter edge wrong: %q / %q", p.Hunter, p.HunterRef)
	}
	if len(p.TopicRefs) != 1 || p.TopicRefs[0] != "ai" {
		t.Errorf("TopicRefs = %v, want [ai]", p.TopicRefs)
	}
	if len(p.MakerRefs) != 2 || p.MakerRefs[1] != "maker2" {
		t.Errorf("MakerRefs = %v", p.MakerRefs)
	}
	if p.CommentsRef != "1" {
		t.Errorf("CommentsRef = %q, want 1", p.CommentsRef)
	}
}

func TestPostsAPIPaginates(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Header().Set("Content-Type", "application/json")
		if hits == 1 {
			_, _ = io.WriteString(w, `{"data":{"posts":{"edges":[{"node":{"id":"1","name":"A"}}],"pageInfo":{"endCursor":"CUR","hasNextPage":true}}}}`)
			return
		}
		_, _ = io.WriteString(w, `{"data":{"posts":{"edges":[{"node":{"id":"2","name":"B"}}],"pageInfo":{"endCursor":"","hasNextPage":false}}}}`)
	}))
	defer srv.Close()

	posts, err := keyedClient(srv.URL).PostsAPI(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) != 2 || posts[1].ID != "2" {
		t.Fatalf("pagination failed: got %d posts, hits=%d", len(posts), hits)
	}
}

func TestPostAPINotFound(t *testing.T) {
	srv := graphqlServer(t, `{"data":{"post":null}}`)
	defer srv.Close()
	if _, err := keyedClient(srv.URL).PostAPI(context.Background(), "999"); err != ErrNotFound {
		t.Errorf("PostAPI(missing) = %v, want ErrNotFound", err)
	}
}

func TestTopicAPIDecode(t *testing.T) {
	const resp = `{"data":{"topic":{"id":"10","name":"AI","slug":"ai","followersCount":5000,"postsCount":42,"url":"https://www.producthunt.com/topics/ai","posts":{"edges":[{"node":{"id":"1"}},{"node":{"id":"2"}}]}}}}`
	srv := graphqlServer(t, resp)
	defer srv.Close()
	tp, err := keyedClient(srv.URL).TopicAPI(context.Background(), "ai")
	if err != nil {
		t.Fatal(err)
	}
	if tp.Followers != 5000 || tp.PostsCount != 42 {
		t.Errorf("topic counts wrong: %+v", tp)
	}
	if len(tp.PostRefs) != 2 || tp.PostRefs[0] != "1" {
		t.Errorf("PostRefs = %v, want [1 2]", tp.PostRefs)
	}
}

func TestCommentsAPIDecode(t *testing.T) {
	const resp = `{"data":{"post":{"id":"1","comments":{"edges":[
		{"node":{"id":"c1","body":"top","votesCount":3,"user":{"id":"7","username":"alice","name":"Alice"},
			"replies":{"edges":[{"node":{"id":"c2","body":"reply","user":{"id":"8","username":"bob","name":"Bob"}}}]}}}
	],"pageInfo":{"endCursor":"","hasNextPage":false}}}}}`
	srv := graphqlServer(t, resp)
	defer srv.Close()
	cs, err := keyedClient(srv.URL).CommentsAPI(context.Background(), "1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) != 2 {
		t.Fatalf("got %d comments, want 2 (top + reply)", len(cs))
	}
	if cs[0].Post != "1" || cs[0].Author != "alice" || cs[0].Parent != "" {
		t.Errorf("top comment wrong: %+v", cs[0])
	}
	if cs[1].Parent != "c1" || cs[1].Author != "bob" {
		t.Errorf("reply wrong: %+v", cs[1])
	}
}

func TestCommentsAPINotFound(t *testing.T) {
	srv := graphqlServer(t, `{"data":{"post":null}}`)
	defer srv.Close()
	if _, err := keyedClient(srv.URL).CommentsAPI(context.Background(), "999", 0); err != ErrNotFound {
		t.Errorf("CommentsAPI(missing) = %v, want ErrNotFound", err)
	}
}

func TestPlaneFor(t *testing.T) {
	cases := []struct {
		plane      string
		key        bool
		webOK      bool
		apiOK      bool
		want       string
		wantErrNil bool
	}{
		{"auto", true, true, true, "api", true},
		{"auto", false, true, true, "web", true},
		{"auto", false, false, true, "", false}, // api-only, no key
		{"web", false, true, true, "web", true},
		{"web", true, false, true, "", false}, // forced web, no web method
		{"api", true, true, true, "api", true},
		{"api", false, true, true, "", false}, // forced api, no key
	}
	for _, tc := range cases {
		c := testClient("http://example.com")
		c.cfg.Plane = tc.plane
		if tc.key {
			c.cfg.Token = "tok"
		}
		got, err := c.planeFor(tc.webOK, tc.apiOK)
		if tc.wantErrNil != (err == nil) {
			t.Errorf("planeFor(%v) err = %v, wantErrNil=%v", tc, err, tc.wantErrNil)
		}
		if got != tc.want {
			t.Errorf("planeFor(%v) = %q, want %q", tc, got, tc.want)
		}
	}
}

func TestWebPostsOK(t *testing.T) {
	c := testClient("http://example.com")
	if !c.webPostsOK() {
		t.Error("default (no filter) should be web-servable")
	}
	c.cfg.Order = "newest"
	if !c.webPostsOK() {
		t.Error("newest order should be web-servable")
	}
	c.cfg.Order = "ranking"
	if c.webPostsOK() {
		t.Error("ranking order is api-only")
	}
	c.cfg.Order = ""
	c.cfg.Topic = "ai"
	if c.webPostsOK() {
		t.Error("a topic filter is api-only")
	}
}

func TestTokenEnvShortCircuit(t *testing.T) {
	// A ready token is used directly and never hits the OAuth endpoint.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("OAuth endpoint hit despite a ready token")
	}))
	defer srv.Close()
	c := testClient(srv.URL)
	c.cfg.Token = "ready-token"
	tok, err := c.token(context.Background())
	if err != nil || tok != "ready-token" {
		t.Errorf("token() = (%q, %v), want (ready-token, nil)", tok, err)
	}
}

func TestOAuthHandshake(t *testing.T) {
	var mints int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mints++
		body, _ := io.ReadAll(r.Body)
		var req map[string]string
		_ = json.Unmarshal(body, &req)
		if req["grant_type"] != "client_credentials" {
			t.Errorf("grant_type = %q, want client_credentials", req["grant_type"])
		}
		if req["client_id"] != "cid" || req["client_secret"] != "secret" {
			t.Errorf("credentials not forwarded: %v", req)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"access_token":"minted-token","token_type":"Bearer","scope":"public"}`)
	}))
	defer srv.Close()

	c := testClient(srv.URL)
	c.cfg.ClientID = "cid"
	c.cfg.ClientSecret = "secret"

	tok, err := c.token(context.Background())
	if err != nil || tok != "minted-token" {
		t.Fatalf("token() = (%q, %v), want (minted-token, nil)", tok, err)
	}
	// A second call uses the cached token, not a second handshake.
	if _, err := c.token(context.Background()); err != nil {
		t.Fatal(err)
	}
	if mints != 1 {
		t.Errorf("OAuth endpoint hit %d times, want 1 (token cached for the run)", mints)
	}
}

func TestTokenNeedsCredentials(t *testing.T) {
	c := testClient("http://example.com")
	if _, err := c.token(context.Background()); err != ErrNeedKey {
		t.Errorf("token() with no credentials = %v, want ErrNeedKey", err)
	}
}
