package producthunt

import (
	"os"
	"time"
)

// config.go holds the resolved settings a Client reads. domain.go's
// ClientFromConfig maps the framework's kit.Config onto this, so the standalone
// binary and a host pace, identify, and authenticate themselves the same way.
//
// The credentials are the GraphQL API ones, read only from the environment: a
// ready PRODUCTHUNT_TOKEN, or a PRODUCTHUNT_CLIENT_ID and PRODUCTHUNT_CLIENT_SECRET
// pair that mint a token through the client-credentials grant. Their presence turns
// on the api plane for the surfaces the API covers; their absence leaves the tool
// on the web plane (the Atom feed). There is no token flag, by design (see the spec
// section 10.1).

const (
	// Host is the public website, the web plane, and the host the URI driver claims.
	Host = "www.producthunt.com"
	// BaseURL is the root of the web plane.
	BaseURL = "https://" + Host
	// FeedURL is the one anonymous web surface that survives the Cloudflare wall.
	FeedURL = BaseURL + "/feed"
	// APIURL is the GraphQL endpoint of the opt-in api plane.
	APIURL = "https://api.producthunt.com/v2/api/graphql"
	// OAuthURL mints a public-scope token from an application's client credentials.
	OAuthURL = "https://api.producthunt.com/v2/oauth/token"

	// DefaultPlane is the automatic, environment-driven plane choice.
	DefaultPlane = "auto"

	// DefaultCacheTTL is how long a cached response stays fresh by default. Product
	// Hunt runs on a daily cadence, so a few hours is plenty.
	DefaultCacheTTL = 6 * time.Hour

	defaultLimit = 20 // a bare list command's fetch count
	apiMaxLimit  = 20 // a single GraphQL connection page the tool requests
)

// DefaultUserAgent identifies the client. It names a current browser, because the
// web plane is fronted by Cloudflare and an obviously scripted agent is turned away
// faster; it is still honest in that the tool does not forge a crawler identity it
// is reverse-DNS-checked against.
const DefaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) " +
	"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"

// Config is the resolved settings a Client reads.
type Config struct {
	UserAgent string
	Delay     time.Duration // minimum gap between requests
	Retries   int           // retries on 429/5xx
	Timeout   time.Duration // per-request timeout

	Plane        string // "web", "api", or "auto"
	Token        string // a ready bearer token, from PRODUCTHUNT_TOKEN only
	ClientID     string // an application id, from PRODUCTHUNT_CLIENT_ID only
	ClientSecret string // its secret, from PRODUCTHUNT_CLIENT_SECRET only

	// List shaping, shared by the read commands.
	Order    string // ranking, newest, votes, featured (per surface)
	Topic    string // a topic slug to filter posts
	Featured bool   // restrict to featured posts or collections
	After    string // postedAfter, an ISO date
	Before   string // postedBefore, an ISO date

	BaseURL  string // overridable for tests
	APIURL   string // overridable for tests
	OAuthURL string // overridable for tests

	CacheDir string
	NoCache  bool
	CacheTTL time.Duration
	Refresh  bool // refetch and rewrite the cache, ignoring any hit
}

// DefaultConfig returns the baseline settings and reads the credentials from the
// environment.
func DefaultConfig() Config {
	return Config{
		UserAgent:    DefaultUserAgent,
		Delay:        2 * time.Second,
		Retries:      3,
		Timeout:      30 * time.Second,
		Plane:        DefaultPlane,
		Token:        os.Getenv("PRODUCTHUNT_TOKEN"),
		ClientID:     os.Getenv("PRODUCTHUNT_CLIENT_ID"),
		ClientSecret: os.Getenv("PRODUCTHUNT_CLIENT_SECRET"),
		BaseURL:      BaseURL,
		APIURL:       APIURL,
		OAuthURL:     OAuthURL,
		CacheTTL:     DefaultCacheTTL,
	}
}

// hasKey reports whether API credentials are configured: a ready token, or a
// client id and secret pair to mint one.
func (c Config) hasKey() bool {
	return c.Token != "" || (c.ClientID != "" && c.ClientSecret != "")
}
