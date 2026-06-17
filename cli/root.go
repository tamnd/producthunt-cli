// Package cli assembles the ph command tree from the producthunt domain on top of
// the any-cli/kit framework.
package cli

import (
	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/producthunt-cli/producthunt"
)

// Build metadata, set via -ldflags at release time.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// builder holds the domain-global flags while the app is assembled, then folds them
// onto the resolved config in finalize, using the exact keys ClientFromConfig reads.
// There is no token flag: the API credentials are read only from PRODUCTHUNT_TOKEN,
// or PRODUCTHUNT_CLIENT_ID and PRODUCTHUNT_CLIENT_SECRET, in the environment.
type builder struct {
	userAgent string
	plane     string
	order     string
	topic     string
	featured  bool
	after     string
	before    string
	cacheTTL  string
	refresh   bool
}

// NewApp assembles the kit application from the producthunt domain. The domain's
// Register installs the client factory and every operation, so the binary and a host
// (ant, which blank-imports the package) share one source of truth. This package
// adds the domain-global flags and the version command; kit.Run turns the App into
// the CLI, plus the serve and mcp surfaces and the typed-error-to-exit-code mapping.
//
// To add a command, declare it in producthunt/domain.go with kit.Handle and it
// appears here automatically. Reach for app.AddCommand only for a verb that does not
// fit the emit-records shape, the way version does below.
func NewApp() *kit.App {
	b := &builder{}
	id := producthunt.Identity()
	id.Version = Version

	app := kit.New(id, kit.WithDefaults(producthunt.Defaults))
	app.GlobalFlags(b.globals)
	app.Finalize(b.finalize)

	producthunt.Domain{}.Register(app)
	app.AddCommand(newVersionCmd())
	return app
}

func (b *builder) globals(f *kit.FlagSet) {
	def := producthunt.DefaultConfig()
	f.StringVar(&b.userAgent, "user-agent", producthunt.DefaultUserAgent, "User-Agent sent with each request")
	f.StringVar(&b.plane, "plane", def.Plane, "which plane to read: web, api, or auto")
	f.StringVar(&b.order, "order", "", "list order: posts ranking|newest|votes|featured, topics followers|newest, collections followers|newest|featured")
	f.StringVar(&b.topic, "topic", "", "filter posts to a topic slug")
	f.BoolVar(&b.featured, "featured", false, "restrict to featured posts or collections")
	f.StringVar(&b.after, "after", "", "posts on or after this ISO date (postedAfter)")
	f.StringVar(&b.before, "before", "", "posts on or before this ISO date (postedBefore)")
	f.StringVar(&b.cacheTTL, "cache-ttl", producthunt.DefaultCacheTTL.String(), "how long a cached response stays fresh")
	f.BoolVar(&b.refresh, "refresh", false, "fetch fresh copies and rewrite the cache, ignoring any hit")
}

func (b *builder) finalize(c *kit.Config) {
	if c.Extra == nil {
		c.Extra = map[string]string{}
	}
	set := func(k, v string) {
		if v != "" {
			c.Extra[k] = v
		}
	}
	set("user-agent", b.userAgent)
	set("plane", b.plane)
	set("order", b.order)
	set("topic", b.topic)
	set("after", b.after)
	set("before", b.before)
	set("cache-ttl", b.cacheTTL)
	if b.featured {
		c.Extra["featured"] = "true"
	}
	if b.refresh {
		c.Extra["refresh"] = "true"
	}
}
