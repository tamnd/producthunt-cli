---
title: "Introduction"
description: "What ph is and how it is put together."
weight: 10
---

A command line for Product Hunt.

`ph` is a single binary. It reads Product Hunt over plain HTTPS, shapes the
responses into clean records, and gets out of your way.

## Two planes, one record

Product Hunt is readable two ways, and `ph` reads both behind one record shape:

- The **web plane** reads `www.producthunt.com`. It is the default and needs no
  account. Every page is fronted by Cloudflare, so the one anonymous surface it
  reads is the Atom feed at `/feed`: the recent launches. A walled page returns
  exit 4.
- The **API plane** reads `api.producthunt.com` over GraphQL. It is reliable
  from anywhere and turns on when `PRODUCTHUNT_TOKEN`, or `PRODUCTHUNT_CLIENT_ID`
  and `PRODUCTHUNT_CLIENT_SECRET`, are set in the environment. The credentials
  are free from Product Hunt and are read from the environment only, never a
  flag.

Both planes carry the same numeric post id and the same fields, so a feed read
and an API read address the same record. `--plane web|api|auto` chooses; the
default `auto` uses the API when credentials are set and the web otherwise.

## How it is built

- A **library package** (`producthunt`) holds the HTTP client and the typed data
  models. It paces requests, sets an honest User-Agent, retries the transient
  failures any public site throws under load, and mints an access token from the
  client credentials when it needs one.
- A **domain** (`producthunt/domain.go`) declares each operation once on the
  [any-cli/kit](https://github.com/tamnd/any-cli) framework. That single
  declaration becomes a CLI command, an HTTP route, an MCP tool, and a
  resource-URI dereference.
- A thin **`cmd/ph`** hands the assembled app to `kit.Run`, which builds the
  command tree and the serve and mcp surfaces.

## One operation, four surfaces

Because an operation is surface-neutral, the same `post` you run on the command
line is also a route and a tool:

```bash
ph post brainflow-2                      # the command
ph serve --addr :7777                    # GET /v1/post/brainflow-2
ph mcp                                   # the post tool, over stdio
ant get producthunt://post/1173164       # the URI dereference (via a host)
```

## Scope

`ph` is a read-only client over data Product Hunt already serves publicly. It
does not log in, store credentials, or solve anti-bot challenges, and it is
honest about a walled surface rather than working around it. Every record keeps
its `url` so a downstream display can link back to the source. That narrow scope
keeps it a single small binary with no database, no daemon, and no setup.

Next: [install it](/getting-started/installation/), then take the
[quick start](/getting-started/quick-start/).
