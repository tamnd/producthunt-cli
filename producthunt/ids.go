package producthunt

import (
	"net/url"
	"regexp"
	"strings"
)

// ids.go is the offline reference layer: Classify turns any Product Hunt URL, path,
// Atom id, @username, slug, or bare id into a canonical (kind, id), and URLFor
// builds an addressable URL for a (kind, id). Both are pure and never touch the
// network, so `ph ref id` and `ph ref url` (and a host's resolve/url) answer
// instantly.
//
// The numeric id is the canonical key the API and the Atom feed share; a page slug
// or a username is the human key the API also accepts (post(id:) or post(slug:)),
// so a pasted link and a feed entry resolve to the same record.
//
// The kinds:
//   - post: a product by numeric id or slug
//   - topic/collection/user: an entity by slug or username
//   - comments: a post's comment thread, addressed by the post id
//   - posts/topics/collections: the list surfaces
//   - feed: the keyless web seed

var (
	atomPostRE = regexp.MustCompile(`Post/(\d+)`)
	rpRE       = regexp.MustCompile(`^r/p/(\d+)`)
	numRE      = regexp.MustCompile(`^\d+$`)
	userRE     = regexp.MustCompile(`^@([A-Za-z0-9_.-]+)$`)
	slugRE     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*$`)
)

// Classify resolves a reference offline. It accepts a full Product Hunt URL, a
// path, an Atom id, a @username, or a bare id/slug.
func Classify(input string) Ref {
	in := strings.TrimSpace(input)
	r := Ref{Input: input, Kind: "unknown"}

	// An Atom id (tag:www.producthunt.com,2005:Post/<n>) carries the numeric id.
	if strings.HasPrefix(in, "tag:") {
		if m := atomPostRE.FindStringSubmatch(in); m != nil {
			r.Kind, r.ID = "post", m[1]
			r.URL = URLFor(r.Kind, r.ID)
			return r
		}
	}

	path := in
	wasURL := false
	if u, err := url.Parse(in); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
		path = u.Path
		wasURL = true
	}
	clean := strings.Trim(path, "/")

	switch {
	case strings.HasPrefix(clean, "products/"):
		r.Kind, r.ID = "post", strings.TrimPrefix(clean, "products/")
	case strings.HasPrefix(clean, "posts/"):
		r.Kind, r.ID = "post", strings.TrimPrefix(clean, "posts/")
	case rpRE.MatchString(clean):
		r.Kind, r.ID = "post", rpRE.FindStringSubmatch(clean)[1]
	case strings.HasPrefix(clean, "topics/"):
		r.Kind, r.ID = "topic", strings.TrimPrefix(clean, "topics/")
	case strings.HasPrefix(clean, "collections/"):
		r.Kind, r.ID = "collection", strings.TrimPrefix(clean, "collections/")
	case strings.HasPrefix(clean, "@"):
		r.Kind, r.ID = "user", strings.TrimPrefix(clean, "@")
	case userRE.MatchString(clean):
		r.Kind, r.ID = "user", userRE.FindStringSubmatch(clean)[1]
	case numRE.MatchString(clean):
		r.Kind, r.ID = "post", clean
	case !wasURL && slugRE.MatchString(clean):
		r.Kind, r.ID = "post", clean
	}

	// A trailing path segment may carry the id, never an empty string.
	r.ID = strings.Trim(r.ID, "/")
	if r.ID == "" && r.Kind != "feed" {
		r.Kind = "unknown"
	}

	if r.Kind != "unknown" {
		if wasURL {
			r.URL = in // the human page is more useful than a rebuilt URL
		} else {
			r.URL = URLFor(r.Kind, r.ID)
		}
	}
	return r
}

// URLFor builds an addressable URL for a (kind, id), or "" if it cannot. A page URL
// needs the human slug, so a slug or username rebuilds the page directly; a bare
// numeric post id cannot rebuild the discussion URL offline (it needs the slug), so
// it returns the stable /r/p/<n> redirect, the one addressable URL a numeric post
// id can build. A record's own url carries the discussion page after a fetch.
func URLFor(kind, id string) string {
	id = strings.Trim(id, "/")
	switch kind {
	case "post", "comments":
		if id == "" {
			return ""
		}
		if numRE.MatchString(id) {
			return BaseURL + "/r/p/" + id
		}
		return BaseURL + "/products/" + id
	case "topic":
		if id == "" {
			return ""
		}
		return BaseURL + "/topics/" + id
	case "collection":
		if id == "" {
			return ""
		}
		return BaseURL + "/collections/" + id
	case "user":
		if id == "" {
			return ""
		}
		return BaseURL + "/@" + id
	case "posts":
		return BaseURL + "/products"
	case "topics":
		return BaseURL + "/topics"
	case "collections":
		return BaseURL + "/collections"
	case "feed":
		return FeedURL
	default:
		return ""
	}
}
