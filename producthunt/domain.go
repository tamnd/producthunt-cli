package producthunt

import (
	"context"
	"errors"
	"time"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

// domain.go exposes producthunt as a kit Domain: a driver that a multi-domain host
// (ant) enables with a single blank import,
//
//	import _ "github.com/tamnd/producthunt-cli/producthunt"
//
// exactly as a database/sql program enables a driver with `import _
// "github.com/lib/pq"`. The init below registers it; the host then dereferences
// producthunt:// URIs by routing to the operations Register installs. The same
// Domain also builds the standalone ph binary (see cli.NewApp), so the binary and a
// host share one source of truth.
func init() { kit.Register(Domain{}) }

// Domain is the producthunt driver. It carries no state; the per-run client is
// built by the factory Register hands kit.
type Domain struct{}

// Info describes the scheme, the hostnames a pasted link is matched against, and the
// identity reused for the binary's help and version.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme:   "producthunt",
		Hosts:    []string{Host, "producthunt.com"},
		Identity: Identity(),
	}
}

// Identity is the fixed description of the Product Hunt CLI, shared by the domain and
// the standalone composition root so help and version read the same.
func Identity() kit.Identity {
	return kit.Identity{
		Binary: "ph",
		Short:  "Read public Product Hunt posts, topics, collections, users, and comments into structured records",
		Long: `ph reads public Product Hunt data over plain HTTPS in two planes. The
web plane reads www.producthunt.com and is the default, but every page is
fronted by Cloudflare, so the one anonymous surface it reads is the Atom
feed at /feed (recent launches, no account needed); a walled page returns
exit 4. The api plane reads api.producthunt.com over GraphQL and turns on
when PRODUCTHUNT_TOKEN, or PRODUCTHUNT_CLIENT_ID and PRODUCTHUNT_CLIENT_SECRET,
are set in the environment (free credentials, never a flag); it reads posts,
topics, collections, users, and comments reliably from anywhere. Both planes
share one numeric id and one record shape, so a feed read and an API read
address the same record. ph returns records as a table, JSON, JSONL, CSV,
TSV, or URLs, and serves the same operations over HTTP and MCP.

ph is an independent tool and is not affiliated with Product Hunt.`,
		Site: BaseURL,
		Repo: "https://github.com/tamnd/producthunt-cli",
	}
}

// Register installs the client factory and every operation onto app. The read group
// is the data; the crawl group holds the keyless feed, the seed a host walks from to
// reach the rest of the graph; the ref group is offline and never touches the
// network.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)
	app.CommandGroup("read", "Read public Product Hunt data")
	app.CommandGroup("crawl", "The keyless Atom feed (web plane)")
	app.CommandGroup("ref", "Resolve references to ids and URLs (offline)")

	kit.Handle(app, kit.OpMeta{
		Name: "posts", Group: "read", List: true,
		Summary: "List the post stream (API when keyed, the keyless Atom feed for the newest case)",
		URIType: "posts",
	}, posts)

	kit.Handle(app, kit.OpMeta{
		Name: "post", Group: "read", Single: true,
		Summary: "Show one post in full (API by id or slug, the feed when keyless and recent)",
		URIType: "post", Resolver: true,
		Args: []kit.Arg{{Name: "ref", Help: "a post id, slug, or a Product Hunt URL"}},
	}, getPost)

	kit.Handle(app, kit.OpMeta{
		Name: "topics", Group: "read", List: true,
		Summary: "List topics (API plane; needs Product Hunt credentials)",
		URIType: "topics",
	}, topics)

	kit.Handle(app, kit.OpMeta{
		Name: "topic", Group: "read", Single: true,
		Summary: "Show one topic with its top posts (API plane)",
		URIType: "topic",
		Args:    []kit.Arg{{Name: "ref", Help: "a topic id or slug"}},
	}, getTopic)

	kit.Handle(app, kit.OpMeta{
		Name: "collections", Group: "read", List: true,
		Summary: "List collections (API plane; needs Product Hunt credentials)",
		URIType: "collections",
	}, collections)

	kit.Handle(app, kit.OpMeta{
		Name: "collection", Group: "read", Single: true,
		Summary: "Show one collection with its posts (API plane)",
		URIType: "collection",
		Args:    []kit.Arg{{Name: "ref", Help: "a collection id or slug"}},
	}, getCollection)

	kit.Handle(app, kit.OpMeta{
		Name: "user", Group: "read", Single: true,
		Summary: "Show one user with the posts they made (API plane)",
		URIType: "user",
		Args:    []kit.Arg{{Name: "ref", Help: "a user id or @username"}},
	}, getUser)

	kit.Handle(app, kit.OpMeta{
		Name: "comments", Group: "read", List: true,
		Summary: "List a post's comment thread (API plane)",
		URIType: "comments",
		Args:    []kit.Arg{{Name: "ref", Help: "a post id, slug, or a Product Hunt URL"}},
	}, comments)

	kit.Handle(app, kit.OpMeta{
		Name: "feed", Group: "crawl", List: true,
		Summary: "Read the public Atom feed of recent posts (the keyless web seed)",
		URIType: "feed",
	}, feed)

	// Reference tools (offline).
	kit.Handle(app, kit.OpMeta{
		Name: "id", Parent: "ref", Single: true,
		Summary: "Classify a reference into its (kind, id)",
		Args:    []kit.Arg{{Name: "ref", Help: "any Product Hunt URL, path, Atom id, @username, slug, or id"}},
	}, classifyRef)

	kit.Handle(app, kit.OpMeta{
		Name: "url", Parent: "ref", Single: true,
		Summary: "Build the addressable URL for a (kind, id)",
		Args: []kit.Arg{
			{Name: "kind", Help: "post, topic, collection, user, comments, posts, topics, collections, or feed"},
			{Name: "id", Help: "the id for that kind"},
		},
	}, buildURL)
}

// newClient builds the client from the host-resolved config, so a host and the
// standalone binary pace, identify, and authenticate themselves the same way.
func newClient(_ context.Context, cfg kit.Config) (any, error) {
	return ClientFromConfig(cfg), nil
}

// ClientFromConfig maps the framework config onto a producthunt.Config and returns a
// client. The credentials are read from the environment in DefaultConfig, never from
// a flag.
func ClientFromConfig(cfg kit.Config) *Client {
	pc := DefaultConfig()
	if cfg.Rate > 0 {
		pc.Delay = cfg.Rate
	}
	if cfg.Retries >= 0 {
		pc.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		pc.Timeout = cfg.Timeout
	}
	if ua := cfg.Extra["user-agent"]; ua != "" {
		pc.UserAgent = ua
	} else if cfg.UserAgent != "" {
		pc.UserAgent = cfg.UserAgent
	}
	if v := cfg.Extra["plane"]; v != "" {
		pc.Plane = v
	}
	if v := cfg.Extra["order"]; v != "" {
		pc.Order = v
	}
	if v := cfg.Extra["topic"]; v != "" {
		pc.Topic = v
	}
	pc.Featured = cfg.Extra["featured"] == "true"
	if v := cfg.Extra["after"]; v != "" {
		pc.After = v
	}
	if v := cfg.Extra["before"]; v != "" {
		pc.Before = v
	}
	pc.CacheDir = cfg.CacheDir
	pc.NoCache = cfg.NoCache
	if ttl := cfg.Extra["cache-ttl"]; ttl != "" {
		if d, err := time.ParseDuration(ttl); err == nil {
			pc.CacheTTL = d
		}
	}
	pc.Refresh = cfg.Extra["refresh"] == "true"
	return NewClient(pc)
}

// Defaults seeds the framework baseline with producthunt's own values, so an unset
// --rate or --timeout uses the producthunt default rather than the generic kit one.
func Defaults(c *kit.Config) {
	def := DefaultConfig()
	c.Rate = def.Delay
	c.Retries = def.Retries
	c.Timeout = def.Timeout
	c.UserAgent = def.UserAgent
}

// Classify turns any accepted input into the canonical (type, id), so `ant resolve`
// and `ant url` touch no network.
func (Domain) Classify(input string) (uriType, id string, err error) {
	r := Classify(input)
	if r.Kind == "unknown" {
		return "", "", errs.Usage("unrecognized product hunt reference: %q", input)
	}
	return r.Kind, r.ID, nil
}

// Locate is the inverse: the addressable URL for a (type, id).
func (Domain) Locate(uriType, id string) (string, error) {
	u := URLFor(uriType, id)
	if u == "" {
		return "", errs.Usage("product hunt has no resource type %q", uriType)
	}
	return u, nil
}

// mapErr translates a library error into a kit error so the exit code matches the
// rest of the fleet: a missing entity reads as not found (exit 6), a throttle as
// rate limited (exit 5), the wall or a rejected/missing token as need-auth (exit 4),
// a missing credential on an opt-in surface as need-auth, and a caught bad argument
// as usage (exit 2).
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, ErrNotFound):
		return errs.NotFound("%s", err.Error())
	case errors.Is(err, ErrRateLimited):
		return errs.RateLimited("%s", err.Error())
	case errors.Is(err, ErrBlocked):
		return errs.NeedAuth("%s", err.Error())
	case errors.Is(err, ErrNeedKey):
		return errs.NeedAuth("%s", err.Error())
	case errors.Is(err, ErrUsage):
		return errs.Usage("%s", err.Error())
	default:
		return err
	}
}
