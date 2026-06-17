package producthunt

import (
	"context"
	"strings"

	"github.com/tamnd/any-cli/kit/errs"
)

// ops.go holds the handler for every operation declared in domain.go. kit reflects
// each input struct into CLI flags, HTTP query params, and MCP tool arguments:
// kit:"arg" is a positional, kit:"flag,inherit" binds the shared --limit, and
// kit:"inject" receives the client newClient builds. The list-shaping flags (order,
// topic, featured, after, before) are domain-global and reach the client through its
// config, so they do not repeat on every input struct. The reference ops (id, url)
// take no client; they run offline.
//
// Each handler resolves the plane once and calls the matching method. The record
// type is identical either way, so the output, --fields, and the edges do not
// change with the plane; only which fields are filled changes, and omitempty carries
// the difference. A chosen plane that is walled or keyless returns its error and
// exits; no plane silently falls back to the other.

// planeFor resolves which plane an op runs on. webOK reports the op has a web
// method, apiOK that it has an API method. The rules follow the fleet model: API
// credentials prefer the api plane, none uses the web, --plane overrides, and a
// forced plane the op cannot serve is an auth error naming the credentials.
func (c *Client) planeFor(webOK, apiOK bool) (string, error) {
	switch c.cfg.Plane {
	case "web":
		if !webOK {
			return "", ErrNeedKey
		}
		return "web", nil
	case "api":
		if !apiOK || !c.cfg.hasKey() {
			return "", ErrNeedKey
		}
		return "api", nil
	default: // auto
		if apiOK && c.cfg.hasKey() {
			return "api", nil
		}
		if webOK {
			return "web", nil
		}
		return "", ErrNeedKey
	}
}

// webPostsOK reports whether the web plane (the Atom feed) can serve a posts
// request. The feed carries only the newest posts with no ranking, topic, or date
// filter, so any of those is an api-only request.
func (c *Client) webPostsOK() bool {
	switch strings.ToLower(c.cfg.Order) {
	case "", "newest":
	default:
		return false
	}
	return c.cfg.Topic == "" && !c.cfg.Featured && c.cfg.After == "" && c.cfg.Before == ""
}

// --- posts ---

type postsIn struct {
	Limit  int     `kit:"flag,inherit"`
	Client *Client `kit:"inject"`
}

func posts(ctx context.Context, in postsIn, emit func(*Post) error) error {
	plane, err := in.Client.planeFor(true, true)
	if err != nil {
		return mapErr(err)
	}
	n := limitOr(in.Limit, defaultLimit)
	var items []*Post
	if plane == "api" {
		items, err = in.Client.PostsAPI(ctx, n)
	} else {
		if !in.Client.webPostsOK() {
			return mapErr(ErrNeedKey)
		}
		items, err = in.Client.Feed(ctx, n)
	}
	if err != nil {
		return mapErr(err)
	}
	return emitAll(items, emit)
}

// --- post ---

type postIn struct {
	Ref    string  `kit:"arg" help:"a post id, slug, or a Product Hunt URL"`
	Client *Client `kit:"inject"`
}

func getPost(ctx context.Context, in postIn, emit func(*Post) error) error {
	plane, err := in.Client.planeFor(true, true)
	if err != nil {
		return mapErr(err)
	}
	var p *Post
	if plane == "api" {
		p, err = in.Client.PostAPI(ctx, in.Ref)
	} else {
		p, err = in.Client.FeedPost(ctx, in.Ref)
	}
	if err != nil {
		return mapErr(err)
	}
	return emit(p)
}

// --- topics (api-only) ---

type topicsIn struct {
	Limit  int     `kit:"flag,inherit"`
	Client *Client `kit:"inject"`
}

func topics(ctx context.Context, in topicsIn, emit func(*Topic) error) error {
	if _, err := in.Client.planeFor(false, true); err != nil {
		return mapErr(err)
	}
	items, err := in.Client.TopicsAPI(ctx, limitOr(in.Limit, defaultLimit))
	if err != nil {
		return mapErr(err)
	}
	return emitAll(items, emit)
}

// --- topic (api-only) ---

type topicIn struct {
	Ref    string  `kit:"arg" help:"a topic id or slug"`
	Client *Client `kit:"inject"`
}

func getTopic(ctx context.Context, in topicIn, emit func(*Topic) error) error {
	if _, err := in.Client.planeFor(false, true); err != nil {
		return mapErr(err)
	}
	t, err := in.Client.TopicAPI(ctx, in.Ref)
	if err != nil {
		return mapErr(err)
	}
	return emit(t)
}

// --- collections (api-only) ---

type collectionsIn struct {
	Limit  int     `kit:"flag,inherit"`
	Client *Client `kit:"inject"`
}

func collections(ctx context.Context, in collectionsIn, emit func(*Collection) error) error {
	if _, err := in.Client.planeFor(false, true); err != nil {
		return mapErr(err)
	}
	items, err := in.Client.CollectionsAPI(ctx, limitOr(in.Limit, defaultLimit))
	if err != nil {
		return mapErr(err)
	}
	return emitAll(items, emit)
}

// --- collection (api-only) ---

type collectionIn struct {
	Ref    string  `kit:"arg" help:"a collection id or slug"`
	Client *Client `kit:"inject"`
}

func getCollection(ctx context.Context, in collectionIn, emit func(*Collection) error) error {
	if _, err := in.Client.planeFor(false, true); err != nil {
		return mapErr(err)
	}
	col, err := in.Client.CollectionAPI(ctx, in.Ref)
	if err != nil {
		return mapErr(err)
	}
	return emit(col)
}

// --- user (api-only) ---

type userIn struct {
	Ref    string  `kit:"arg" help:"a user id or @username"`
	Client *Client `kit:"inject"`
}

func getUser(ctx context.Context, in userIn, emit func(*User) error) error {
	if _, err := in.Client.planeFor(false, true); err != nil {
		return mapErr(err)
	}
	u, err := in.Client.UserAPI(ctx, in.Ref)
	if err != nil {
		return mapErr(err)
	}
	return emit(u)
}

// --- comments (api-only) ---

type commentsIn struct {
	Ref    string  `kit:"arg" help:"a post id, slug, or a Product Hunt URL"`
	Limit  int     `kit:"flag,inherit"`
	Client *Client `kit:"inject"`
}

func comments(ctx context.Context, in commentsIn, emit func(*Comment) error) error {
	if _, err := in.Client.planeFor(false, true); err != nil {
		return mapErr(err)
	}
	cs, err := in.Client.CommentsAPI(ctx, in.Ref, limitOr(in.Limit, defaultLimit))
	if err != nil {
		return mapErr(err)
	}
	return emitAll(cs, emit)
}

// --- feed (web, crawl seed) ---

type feedIn struct {
	Limit  int     `kit:"flag,inherit"`
	Client *Client `kit:"inject"`
}

func feed(ctx context.Context, in feedIn, emit func(*Post) error) error {
	items, err := in.Client.Feed(ctx, limitOr(in.Limit, defaultLimit))
	if err != nil {
		return mapErr(err)
	}
	return emitAll(items, emit)
}

// --- reference tools (offline) ---

type refIn struct {
	Ref string `kit:"arg" help:"any Product Hunt URL, path, Atom id, @username, slug, or id"`
}

func classifyRef(_ context.Context, in refIn, emit func(*Ref) error) error {
	r := Classify(in.Ref)
	if r.Kind == "unknown" {
		return errs.Usage("unrecognized product hunt reference: %q", in.Ref)
	}
	return emit(&r)
}

type urlIn struct {
	Kind string `kit:"arg" help:"post, topic, collection, user, comments, posts, topics, collections, or feed"`
	ID   string `kit:"arg" help:"the id for that kind"`
}

func buildURL(_ context.Context, in urlIn, emit func(*Ref) error) error {
	u := URLFor(in.Kind, in.ID)
	if u == "" {
		return errs.Usage("product hunt cannot build a URL for %q/%q", in.Kind, in.ID)
	}
	return emit(&Ref{Input: in.Kind + "/" + in.ID, Kind: in.Kind, ID: in.ID, URL: u})
}

// --- helpers ---

// emitAll streams a slice of records through emit.
func emitAll[T any](items []*T, emit func(*T) error) error {
	for _, it := range items {
		if err := emit(it); err != nil {
			return err
		}
	}
	return nil
}

// limitOr returns the operator's --limit when set, else the command's own default
// fetch count.
func limitOr(limit, def int) int {
	if limit > 0 {
		return limit
	}
	return def
}
