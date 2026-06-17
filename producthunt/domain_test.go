package producthunt

import (
	"testing"

	"github.com/tamnd/any-cli/kit"
)

// These tests are offline: they exercise the URI driver's pure string functions and
// the host wiring (mint, resolve), which need no network. The client's HTTP
// behaviour is covered in producthunt_test.go.

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "producthunt" {
		t.Errorf("Scheme = %q, want producthunt", info.Scheme)
	}
	if len(info.Hosts) == 0 || info.Hosts[0] != Host {
		t.Errorf("Hosts = %v, want first %s", info.Hosts, Host)
	}
	if info.Identity.Binary != "ph" {
		t.Errorf("Identity.Binary = %q, want ph", info.Identity.Binary)
	}
}

func TestClassify(t *testing.T) {
	cases := []struct{ in, kind, id string }{
		{"https://www.producthunt.com/products/brainflow-2", "post", "brainflow-2"},
		{"https://www.producthunt.com/posts/some-legacy-post", "post", "some-legacy-post"},
		{"https://www.producthunt.com/r/p/1173164?app_id=339", "post", "1173164"},
		{"tag:www.producthunt.com,2005:Post/1173164", "post", "1173164"},
		{"https://www.producthunt.com/topics/artificial-intelligence", "topic", "artificial-intelligence"},
		{"https://www.producthunt.com/collections/best-of-2026", "collection", "best-of-2026"},
		{"https://www.producthunt.com/@rrhoover", "user", "rrhoover"},
		{"@rrhoover", "user", "rrhoover"},
		{"1173164", "post", "1173164"},
		{"brainflow-2", "post", "brainflow-2"},
	}
	for _, tc := range cases {
		r := Classify(tc.in)
		if r.Kind != tc.kind || r.ID != tc.id {
			t.Errorf("Classify(%q) = (%q, %q), want (%q, %q)", tc.in, r.Kind, r.ID, tc.kind, tc.id)
		}
	}
}

func TestClassifyUnknown(t *testing.T) {
	for _, in := range []string{"", "not a url", "https://example.com/foo"} {
		if r := Classify(in); r.Kind != "unknown" {
			t.Errorf("Classify(%q).Kind = %q, want unknown", in, r.Kind)
		}
	}
}

func TestURLFor(t *testing.T) {
	cases := []struct{ kind, id, want string }{
		{"post", "brainflow-2", BaseURL + "/products/brainflow-2"},
		{"post", "1173164", BaseURL + "/r/p/1173164"},
		{"comments", "1173164", BaseURL + "/r/p/1173164"},
		{"topic", "artificial-intelligence", BaseURL + "/topics/artificial-intelligence"},
		{"collection", "best-of-2026", BaseURL + "/collections/best-of-2026"},
		{"user", "rrhoover", BaseURL + "/@rrhoover"},
		{"feed", "", FeedURL},
		{"posts", "", BaseURL + "/products"},
		{"topics", "", BaseURL + "/topics"},
		{"collections", "", BaseURL + "/collections"},
	}
	for _, tc := range cases {
		if got := URLFor(tc.kind, tc.id); got != tc.want {
			t.Errorf("URLFor(%q, %q) = %q, want %q", tc.kind, tc.id, got, tc.want)
		}
	}
	if got := URLFor("post", ""); got != "" {
		t.Errorf("URLFor(post, empty) = %q, want empty", got)
	}
	if got := URLFor("no_such_kind", "x"); got != "" {
		t.Errorf("URLFor for an unknown kind = %q, want empty", got)
	}
}

func TestDomainClassifyLocate(t *testing.T) {
	kind, id, err := Domain{}.Classify("1173164")
	if err != nil || kind != "post" || id != "1173164" {
		t.Fatalf("Domain.Classify = (%q, %q, %v)", kind, id, err)
	}
	got, err := Domain{}.Locate("post", "1173164")
	if err != nil || got != BaseURL+"/r/p/1173164" {
		t.Errorf("Domain.Locate = (%q, %v)", got, err)
	}
}

// TestHostWiring mounts the driver in a kit Host and checks the round trip: a record
// mints to its URI and a bare id resolves back to the same URI.
func TestHostWiring(t *testing.T) {
	h, err := kit.Open()
	if err != nil {
		t.Fatal(err)
	}

	post := &Post{ID: "1173164", Name: "BrainFlow", URL: "https://www.producthunt.com/products/brainflow-2"}
	u, err := h.Mint(post)
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}
	if want := "producthunt://post/1173164"; u.String() != want {
		t.Errorf("Mint = %q, want %q", u.String(), want)
	}

	got, err := h.ResolveOn("producthunt", "1173164")
	if err != nil || got.String() != "producthunt://post/1173164" {
		t.Errorf("ResolveOn = (%q, %v), want producthunt://post/1173164", got.String(), err)
	}
}
