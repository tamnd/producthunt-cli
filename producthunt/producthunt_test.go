package producthunt_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tamnd/producthunt-cli/producthunt"
)

func newTestClient(t *testing.T, handler http.Handler) *producthunt.Client {
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)
	cfg := producthunt.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	return producthunt.NewClient(cfg)
}

func TestToday(t *testing.T) {
	atomXML := `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <entry>
    <id>https://www.producthunt.com/posts/test-app</id>
    <title>Test App — The best test app ever</title>
    <link href="https://www.producthunt.com/posts/test-app"/>
    <author><name>testuser</name></author>
    <published>2026-06-14T10:00:00Z</published>
  </entry>
</feed>`
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/atom+xml")
		_, _ = w.Write([]byte(atomXML))
	}))
	products, err := c.Today(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(products) != 1 {
		t.Fatalf("got %d products, want 1", len(products))
	}
	if products[0].Name != "Test App" {
		t.Errorf("got name %q, want %q", products[0].Name, "Test App")
	}
}

func TestTodayLimit(t *testing.T) {
	atomXML := `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <entry>
    <id>https://www.producthunt.com/posts/alpha</id>
    <title>Alpha — First product</title>
    <link href="https://www.producthunt.com/posts/alpha"/>
    <author><name>alice</name></author>
    <published>2026-06-14T10:00:00Z</published>
  </entry>
  <entry>
    <id>https://www.producthunt.com/posts/beta</id>
    <title>Beta — Second product</title>
    <link href="https://www.producthunt.com/posts/beta"/>
    <author><name>bob</name></author>
    <published>2026-06-14T10:01:00Z</published>
  </entry>
  <entry>
    <id>https://www.producthunt.com/posts/gamma</id>
    <title>Gamma — Third product</title>
    <link href="https://www.producthunt.com/posts/gamma"/>
    <author><name>carol</name></author>
    <published>2026-06-14T10:02:00Z</published>
  </entry>
</feed>`
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/atom+xml")
		_, _ = w.Write([]byte(atomXML))
	}))
	products, err := c.Today(context.Background(), 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(products) != 2 {
		t.Errorf("got %d products, want 2", len(products))
	}
}

func TestTodayError503(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	_, err := c.Today(context.Background(), 0)
	if err == nil {
		t.Error("expected error on 503, got nil")
	}
}
