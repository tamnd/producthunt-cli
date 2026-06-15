package producthunt

import (
	"context"

	"github.com/tamnd/any-cli/kit"
)

func init() { kit.Register(Domain{}) }

// Domain is the producthunt kit driver. It carries no state; the per-run
// Client is built by the factory Register hands kit.
type Domain struct{}

// Info describes the scheme, the hostnames a pasted link is matched against,
// and the identity reused for the binary's help and version.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme: "producthunt",
		Hosts:  []string{Host},
		Identity: kit.Identity{
			Binary: "producthunt",
			Short:  "A command line for Product Hunt.",
			Long: `A command line for Product Hunt.

Browse today's top product launches. No API key required.`,
			Site: "https://" + Host,
			Repo: "https://github.com/tamnd/producthunt-cli",
		},
	}
}

// Register installs the client factory and every operation onto app.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	kit.Handle(app, kit.OpMeta{Name: "today", Group: "products", List: true,
		URIType: "product", Summary: "Today's Product Hunt launches"}, todayCmd)
}

func newClient(_ context.Context, cfg kit.Config) (any, error) {
	c := NewClient(DefaultConfig())
	if cfg.UserAgent != "" {
		c.userAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		c.rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		c.retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		c.http.Timeout = cfg.Timeout
	}
	return c, nil
}

type todayIn struct {
	Limit  int     `kit:"flag,inherit" help:"max results"`
	Client *Client `kit:"inject"`
}

func todayCmd(ctx context.Context, in todayIn, emit func(*Product) error) error {
	products, err := in.Client.Today(ctx, in.Limit)
	if err != nil {
		return err
	}
	for i := range products {
		if err := emit(&products[i]); err != nil {
			return err
		}
	}
	return nil
}
