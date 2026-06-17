# ph

A command line for Product Hunt.

`ph` is a single pure-Go binary. It reads public Product Hunt data over plain
HTTPS, shapes it into clean records, and prints output that pipes into the rest
of your tools.

It reads two planes that share one numeric id and one record shape:

- The **web plane** reads `www.producthunt.com` and is the default. Every page
  is fronted by Cloudflare, so the one anonymous surface it reads is the Atom
  feed at `/feed`: the recent launches, no account needed. A walled page returns
  exit 4.
- The **API plane** reads `api.producthunt.com` over GraphQL and reads posts,
  topics, collections, users, and comments reliably from anywhere. It turns on
  when `PRODUCTHUNT_TOKEN`, or `PRODUCTHUNT_CLIENT_ID` and
  `PRODUCTHUNT_CLIENT_SECRET`, are set in the environment. The credentials are
  free from Product Hunt and are read from the environment only, never a flag.

Because both planes carry the same numeric post id and the same record fields, a
feed read and an API read address the same record. Set the credentials and every
command gets reliable, or leave them unset and the feed-backed commands still
work with no account.

The same package is also a [resource-URI driver](#use-it-as-a-resource-uri-driver),
so a host program like [ant](https://github.com/tamnd/ant) can address
Product Hunt as `producthunt://` URIs.

## Install

```bash
go install github.com/tamnd/producthunt-cli/cmd/ph@latest
```

Or grab a prebuilt binary from the [releases](https://github.com/tamnd/producthunt-cli/releases), or run
the container image:

```bash
docker run --rm ghcr.io/tamnd/ph:latest --help
```

## Usage

```bash
# free credentials, optional, make every command reliable
export PRODUCTHUNT_TOKEN=...                   # a developer token, or
export PRODUCTHUNT_CLIENT_ID=...               # an API application's
export PRODUCTHUNT_CLIENT_SECRET=...           # client id and secret

ph feed                              # the keyless Atom feed of recent posts
ph posts                             # the post stream (feed when keyless, API when keyed)
ph post brainflow-2                  # one post as a record (by slug or id)
ph post brainflow-2 -o json          # as JSON, ready for jq
ph comments 1173164                  # a post's comment thread (API plane)
ph topic artificial-intelligence     # one topic with its top posts (API plane)
ph topics                            # list topics (API plane)
ph collection best-of-2026           # one collection with its posts (API plane)
ph user @rrhoover                    # one user with the posts they made (API plane)
ph ref id <url>                      # resolve any Product Hunt URL to its id
ph --help                            # the whole command tree
```

Every command shares one output contract:
`-o table|markdown|json|jsonl|csv|tsv|url|raw`, `--fields` to pick columns,
`--template` for a custom line, and `-n` to limit. The default adapts to where
output goes (a color-aware table on a terminal, JSONL in a pipe), so the same
command reads well by hand and parses cleanly downstream.

Pick a plane with `--plane web|api|auto` (default `auto`: the API when
credentials are set, the web otherwise). The `posts` list reads the keyless feed
for the newest case and the API for anything filtered or ordered; the topic,
collection, user, and comment surfaces are API only. The `ref` commands are
offline and resolve URLs to ids and back with no network at all.

## Serve it

The same operations are available over HTTP and as an MCP tool set for agents,
with no extra code:

```bash
ph serve --addr :7777    # GET /v1/post/brainflow-2  returns NDJSON
ph mcp                   # speak MCP over stdio
```

## Use it as a resource-URI driver

`ph` registers a `producthunt` domain the way a program registers a database
driver with `database/sql`. A host enables it with one blank import:

```go
import _ "github.com/tamnd/producthunt-cli/producthunt"
```

Then [ant](https://github.com/tamnd/ant) (or any program that links the package)
dereferences `producthunt://` URIs without knowing anything about Product Hunt:

```bash
ant get producthunt://post/1173164    # fetch the record
ant cat producthunt://post/1173164    # just the description body
ant ls  producthunt://post/1173164    # the edges (comments, topics, makers, hunter)
ant url producthunt://post/1173164    # the addressable URL
```

## Attribution

`ph` reads public, read-only data only. It does not log in, store credentials,
or solve anti-bot challenges, and it is honest about a walled surface rather than
working around it. Every record keeps its `url`, so a downstream view can link
back to the source on Product Hunt.

## Development

```
cmd/ph/       thin main: hands cli.NewApp to kit.Run
cli/          assembles the kit App from the producthunt domain
producthunt/  the library: HTTP client, two-plane readers, data models, and domain.go (the driver)
docs/         tago documentation site
```

```bash
make build      # ./bin/ph
make test       # go test ./...
make vet        # go vet ./...
```

## Releasing

Push a version tag and GitHub Actions runs GoReleaser, which builds the
archives, Linux packages, the multi-arch GHCR image, checksums, SBOMs, and a
cosign signature:

```bash
git tag v0.1.0
git push --tags
```

The Homebrew and Scoop steps self-disable until their tokens exist, so the first
release works with no extra secrets.

## License

`ph` is an independent tool and is not affiliated with Product Hunt. Apache-2.0.
See [LICENSE](LICENSE).
