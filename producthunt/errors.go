package producthunt

import "errors"

// Sentinel errors the library returns; domain.go's mapErr turns each into the kit
// error kind that carries the right exit code (see the spec section 4.5).
var (
	// ErrBlocked is the wall: a Cloudflare 403/503 or challenge interstitial on the
	// web plane, or a rejected/missing token (401, invalid_oauth_token) on the api
	// plane. It maps to need-auth (exit 4), with a remedy of reading the public Atom
	// feed or setting free Product Hunt API credentials.
	ErrBlocked = errors.New("blocked: Product Hunt's web pages are walled by Cloudflare here. The public Atom feed (ph feed) needs no account; for the full API set PRODUCTHUNT_TOKEN, or PRODUCTHUNT_CLIENT_ID and PRODUCTHUNT_CLIENT_SECRET")

	// ErrNeedKey is an api-only surface reached with no credentials. Maps to exit 4.
	ErrNeedKey = errors.New("this surface needs the Product Hunt API: set PRODUCTHUNT_TOKEN, or PRODUCTHUNT_CLIENT_ID and PRODUCTHUNT_CLIENT_SECRET (free credentials from https://www.producthunt.com/v2/oauth/applications)")

	// ErrNotFound is a missing entity (a null node for a single lookup). Maps to exit 6.
	ErrNotFound = errors.New("not found")

	// ErrRateLimited is a sustained 429 after retries, or the GraphQL complexity cap.
	// Maps to exit 5.
	ErrRateLimited = errors.New("rate limited by Product Hunt: slow down with --delay or try again later")

	// ErrUsage is a bad argument the library catches (an unrecognized reference, a
	// bad order). Maps to exit 2.
	ErrUsage = errors.New("usage")
)
