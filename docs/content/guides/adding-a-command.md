---
title: "Add a command"
description: "Model a real Product Hunt record and expose it as a command, a route, and a tool at once."
weight: 10
---

Adding a surface to `ph` means modelling the record Product Hunt serves, then
declaring one operation over it. Every surface (the CLI command, the HTTP route,
the MCP tool, and the URI dereference) updates itself from that one declaration.

## 1. Model the record

In the `producthunt` package, add a struct for the thing you are fetching and a
client method that returns it. The `kit` struct tags decide how a host addresses
the record:

```go
type Item struct {
    ID    string `json:"id"    kit:"id"`                       // the URI id
    Title string `json:"title"`
    Body  string `json:"body"  kit:"body"`                     // what cat and Markdown print
    Owner string `json:"owner" kit:"link,kind=producthunt/user"` // an edge to another record
}

func (c *Client) GetItem(ctx context.Context, id string) (*Item, error) {
    data, err := c.postGraphQL(ctx, itemQuery, map[string]any{"id": id})
    if err != nil {
        return nil, err
    }
    // decode data into an Item ...
    return item, nil
}
```

- `kit:"id"` marks the field that becomes the URI id.
- `kit:"body"` marks the prose that `cat` and the Markdown export render.
- `kit:"link,kind=<scheme>/<type>"` marks an outbound edge. It can point at
  another producthunt type or at another site entirely, which is what lets a host
  walk the graph across tools. A slice-valued link field emits one edge per
  element.

## 2. Declare the operation

In `producthunt/domain.go`, add an input struct and a handler, then register it
in `Register`:

```go
type itemRef struct {
    Ref    string  `kit:"arg"`
    Client *Client `kit:"inject"`
}

func getItem(ctx context.Context, in itemRef, emit func(*Item) error) error {
    it, err := in.Client.GetItem(ctx, in.Ref)
    if err != nil {
        return mapErr(err)
    }
    return emit(it)
}

// inside Register(app):
kit.Handle(app, kit.OpMeta{Name: "item", Group: "read", Single: true,
    Summary: "Fetch an item by id", URIType: "item", Resolver: true,
    Args: []kit.Arg{{Name: "ref", Help: "item id or URL"}}}, getItem)
```

That is the whole change. `kit.Handle` reflects the input for flags and the
output for the record shape, so the operation immediately becomes:

```bash
ph item <id>                            # the command
curl 'localhost:7777/v1/item/<id>'      # the route, under serve
ant get producthunt://item/<id>         # the URI dereference, via a host
```

## Resolver ops and list ops

Two flags shape how a host treats an operation:

- **`Single: true`** with **`Resolver: true`** marks the canonical one-record
  fetch for a `URIType`. It answers `ant get`.
- **`List: true`** marks a member-lister for a parent resource. It answers
  `ant ls`. A list op should emit records that are themselves addressable, so
  every member is a URI a host can follow. The `comments` op does this: each
  comment carries its `post` and `user_ref` edges.

## Pick the right plane

A handler chooses its plane with `planeFor(webOK, apiOK)`, which honours
`--plane` and whether credentials are set. A surface that only the API serves
passes `planeFor(false, true)`, so it reports need-auth (exit 4) with a clear
message when no credentials are present. The `posts` op is the mixed case: it
serves the keyless feed for the newest, unfiltered list and the API for anything
ordered or filtered.

## Map errors to exit codes

Return the `errs` kinds from `mapErr` so every surface reports the same outcome
with the same exit code:

```go
case errors.Is(err, ErrNotFound):
    return errs.NotFound("%s", err.Error())
case errors.Is(err, ErrRateLimited):
    return errs.RateLimited("%s", err.Error())
case errors.Is(err, ErrBlocked), errors.Is(err, ErrNeedKey):
    return errs.NeedAuth("%s", err.Error())
```

See [output formats](/reference/output/) for how records render, and
[resource URIs](/guides/resource-uris/) for how a host addresses them.
