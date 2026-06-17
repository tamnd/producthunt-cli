---
title: "ph"
description: "A command line for Product Hunt."
heroTitle: "product hunt, from the command line"
heroLead: "A command line for Product Hunt. One pure-Go binary, two planes that share one post id, output that pipes into the rest of your tools, and a resource-URI driver other programs can address."
heroPrimaryURL: "/getting-started/quick-start/"
heroPrimaryText: "Get started"
---

`ph` reads public Product Hunt data over plain HTTPS, shapes it into clean records, and gets out of your way.

```bash
ph feed                    # the keyless Atom feed of recent posts
ph post brainflow-2        # one post as a record
ph comments 1173164        # a post's comment thread (API plane)
ph serve --addr :7777      # the same operations over HTTP
```

It reads two planes that share one numeric post id and one record shape: the web plane on `www.producthunt.com` (the default, the keyless Atom feed at `/feed`) and the GraphQL API on `api.producthunt.com` (reliable, on when `PRODUCTHUNT_TOKEN`, or `PRODUCTHUNT_CLIENT_ID` and `PRODUCTHUNT_CLIENT_SECRET`, are set).
A feed read and an API read address the same record.
Output adapts to where it goes: an aligned table on your terminal, JSONL the moment you pipe it somewhere.

## Two ways to use it

- **As a command** for reading Product Hunt by hand or in a script. Start with the [quick start](/getting-started/quick-start/).
- **As a resource-URI driver** so a host like [ant](https://github.com/tamnd/ant) can address Product Hunt as `producthunt://` URIs and follow links across sites. See [resource URIs](/guides/resource-uris/).

Both are the same code: one operation, declared once, is a CLI command, an HTTP route, an MCP tool, and a URI dereference.

## Where to go next

- New here? Read the [introduction](/getting-started/introduction/), then the [quick start](/getting-started/quick-start/).
- Installing? See [installation](/getting-started/installation/).
- Doing a specific job? The [guides](/guides/) are task-first.
- Need every flag? The [CLI reference](/reference/cli/) is the full surface.
