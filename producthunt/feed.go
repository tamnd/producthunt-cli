package producthunt

import (
	"context"
	"encoding/xml"
	"fmt"
	"regexp"
	"strings"
)

// feed.go reads the web plane: the Atom feed at /feed, the one anonymous surface
// that survives Cloudflare. It parses each entry into a Post with the fields the
// feed fills (id, name, tagline, slug, discussion URL, outbound link, dates, and
// the submitter's display name) and wires the comments edge. The feed carries no
// usernames, so the hunter-username and topic edges stay empty rather than be
// guessed; the api plane fills them.

// atomFeed is the wire shape of the Atom document.
type atomFeed struct {
	XMLName xml.Name    `xml:"feed"`
	Entries []atomEntry `xml:"entry"`
}

type atomEntry struct {
	ID        string     `xml:"id"`
	Title     string     `xml:"title"`
	Published string     `xml:"published"`
	Updated   string     `xml:"updated"`
	Links     []atomLink `xml:"link"`
	Content   string     `xml:"content"`
	Author    struct {
		Name string `xml:"name"`
	} `xml:"author"`
}

type atomLink struct {
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr"`
	Href string `xml:"href,attr"`
}

var (
	firstParaRE = regexp.MustCompile(`(?s)<p>(.*?)</p>`)
	rpHrefRE    = regexp.MustCompile(`href="([^"]*?/r/p/\d+[^"]*)"`)
	tagStripRE  = regexp.MustCompile(`<[^>]+>`)
)

// Feed fetches the Atom feed and returns up to limit recent posts, newest first as
// the feed orders them.
func (c *Client) Feed(ctx context.Context, limit int) ([]*Post, error) {
	body, err := c.get(ctx, strings.TrimRight(c.cfg.BaseURL, "/")+"/feed")
	if err != nil {
		return nil, err
	}
	var feed atomFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("parse feed: %w", err)
	}
	out := make([]*Post, 0, len(feed.Entries))
	for i := range feed.Entries {
		p := postFromEntry(&feed.Entries[i])
		if p == nil {
			continue
		}
		out = append(out, p)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

// FeedPost returns the one feed post whose id or slug matches ref, or ErrNotFound
// when it is not in the recent feed. This is the keyless fallback for `ph post`.
func (c *Client) FeedPost(ctx context.Context, ref string) (*Post, error) {
	want := Classify(ref)
	posts, err := c.Feed(ctx, 0)
	if err != nil {
		return nil, err
	}
	for _, p := range posts {
		if p.ID == want.ID || (want.ID != "" && p.Slug == want.ID) {
			return p, nil
		}
	}
	return nil, ErrNotFound
}

// postFromEntry maps one Atom entry onto a Post.
func postFromEntry(e *atomEntry) *Post {
	id := ""
	if m := atomPostRE.FindStringSubmatch(e.ID); m != nil {
		id = m[1]
	}
	if id == "" {
		return nil
	}
	p := &Post{
		ID:         id,
		Name:       squish(e.Title),
		CreatedAt:  e.Published,
		HunterName: squish(e.Author.Name),
	}
	// The alternate link is the clean discussion URL and carries the slug.
	for _, l := range e.Links {
		if l.Rel == "alternate" && l.Href != "" {
			p.URL = l.Href
			if r := Classify(l.Href); r.Kind == "post" && !numRE.MatchString(r.ID) {
				p.Slug = r.ID
			}
			break
		}
	}
	// The tagline is the first paragraph of the HTML content.
	if m := firstParaRE.FindStringSubmatch(e.Content); m != nil {
		p.Tagline = squish(tagStripRE.ReplaceAllString(m[1], ""))
	}
	// The /r/p/<n> anchor is the outbound product link.
	if m := rpHrefRE.FindStringSubmatch(e.Content); m != nil {
		p.Website = m[1]
	}
	// The comments edge is the post id; the feed carries no usernames or topics.
	p.CommentsRef = id
	return p
}
