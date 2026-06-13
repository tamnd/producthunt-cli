package producthunt_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tamnd/producthunt-cli/producthunt"
)

const sampleFeed = `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Product Hunt</title>
  <entry>
    <id>https://www.producthunt.com/posts/foobar</id>
    <title>FooBar &#x2014; The best tool ever</title>
    <link href="https://www.producthunt.com/posts/foobar"/>
    <author><name>alice</name></author>
    <published>2026-06-14T00:01:00Z</published>
    <updated>2026-06-14T10:00:00Z</updated>
    <content type="html">&lt;p&gt;Description&lt;/p&gt;</content>
  </entry>
  <entry>
    <id>https://www.producthunt.com/posts/bazqux</id>
    <title>BazQux &#x2014; Another great product</title>
    <link href="https://www.producthunt.com/posts/bazqux"/>
    <author><name>bob</name></author>
    <published>2026-06-14T00:02:00Z</published>
    <updated>2026-06-14T10:00:00Z</updated>
    <content type="html">&lt;p&gt;More description&lt;/p&gt;</content>
  </entry>
  <entry>
    <id>https://www.producthunt.com/posts/thirdie</id>
    <title>Thirdie &#x2014; Third one</title>
    <link href="https://www.producthunt.com/posts/thirdie"/>
    <author><name>carol</name></author>
    <published>2026-06-14T00:03:00Z</published>
    <updated>2026-06-14T10:00:00Z</updated>
    <content type="html">&lt;p&gt;Third&lt;/p&gt;</content>
  </entry>
</feed>`

// newTestClient returns a Client configured to fetch from the given test server.
// The test server must handle GET /feed.
func newTestClient(serverURL string) *producthunt.Client {
	cfg := producthunt.DefaultConfig()
	cfg.BaseURL = serverURL
	cfg.Retries = 0
	return producthunt.NewClient(cfg)
}

// feedHandler returns an http.HandlerFunc that serves sampleFeed on GET /feed.
func feedHandler(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/feed" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/atom+xml")
		_, _ = w.Write([]byte(body))
	}
}

// TestTodayParsesFeed checks that Today correctly parses the Atom feed.
func TestTodayParsesFeed(t *testing.T) {
	srv := httptest.NewServer(feedHandler(sampleFeed))
	defer srv.Close()

	c := newTestClient(srv.URL)
	products, err := c.Today(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(products) != 3 {
		t.Fatalf("got %d products, want 3", len(products))
	}
	p := products[0]
	if p.Name != "FooBar" {
		t.Errorf("Name = %q, want %q", p.Name, "FooBar")
	}
	if p.Tagline != "The best tool ever" {
		t.Errorf("Tagline = %q, want %q", p.Tagline, "The best tool ever")
	}
	if p.Author != "alice" {
		t.Errorf("Author = %q, want %q", p.Author, "alice")
	}
	if p.Published != "2026-06-14" {
		t.Errorf("Published = %q, want %q", p.Published, "2026-06-14")
	}
	if p.URL != "https://www.producthunt.com/posts/foobar" {
		t.Errorf("URL = %q, want producthunt URL", p.URL)
	}
	if p.Rank != 1 {
		t.Errorf("Rank = %d, want 1", p.Rank)
	}
	if products[1].Rank != 2 {
		t.Errorf("products[1].Rank = %d, want 2", products[1].Rank)
	}
}

// TestTodayRespectsLimit checks that Today returns at most limit records.
func TestTodayRespectsLimit(t *testing.T) {
	srv := httptest.NewServer(feedHandler(sampleFeed))
	defer srv.Close()

	c := newTestClient(srv.URL)
	products, err := c.Today(context.Background(), 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(products) != 2 {
		t.Errorf("got %d products, want 2", len(products))
	}
}

// TestTodayErrorOn503 checks that Today returns an error when the server responds with 503.
func TestTodayErrorOn503(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.Today(context.Background(), 0)
	if err == nil {
		t.Error("expected error on 503, got nil")
	}
}

// TestSplitTitleViaToday indirectly verifies title parsing by checking product
// names and taglines returned from a feed with various separator styles.
func TestSplitTitleViaToday(t *testing.T) {
	feed := `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <entry>
    <id>https://www.producthunt.com/posts/a</id>
    <title>FooBar &#x2014; The best tool</title>
    <link href="https://www.producthunt.com/posts/a"/>
    <author><name>x</name></author>
    <published>2026-06-14T00:00:00Z</published>
  </entry>
  <entry>
    <id>https://www.producthunt.com/posts/b</id>
    <title>Plain Product - A tagline here</title>
    <link href="https://www.producthunt.com/posts/b"/>
    <author><name>y</name></author>
    <published>2026-06-14T00:00:00Z</published>
  </entry>
  <entry>
    <id>https://www.producthunt.com/posts/c</id>
    <title>No separator at all</title>
    <link href="https://www.producthunt.com/posts/c"/>
    <author><name>z</name></author>
    <published>2026-06-14T00:00:00Z</published>
  </entry>
</feed>`

	srv := httptest.NewServer(feedHandler(feed))
	defer srv.Close()

	c := newTestClient(srv.URL)
	products, err := c.Today(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(products) != 3 {
		t.Fatalf("got %d products, want 3", len(products))
	}

	cases := []struct{ name, tagline string }{
		{"FooBar", "The best tool"},
		{"Plain Product", "A tagline here"},
		{"No separator at all", ""},
	}
	for i, tc := range cases {
		p := products[i]
		if p.Name != tc.name {
			t.Errorf("products[%d].Name = %q, want %q", i, p.Name, tc.name)
		}
		if p.Tagline != tc.tagline {
			t.Errorf("products[%d].Tagline = %q, want %q", i, p.Tagline, tc.tagline)
		}
	}
}
