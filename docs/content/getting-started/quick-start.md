---
title: "Quick start"
description: "Fetch your first record with ph."
weight: 30
---

Once `ph` is on your `PATH`, read the keyless feed.
It needs no account and works from anywhere:

```bash
ph feed
```

By default you get an aligned table.
Ask for JSON when you want to pipe it:

```bash
$ ph feed -o json -n 1
[
  {
    "id": "1173164",
    "name": "BrainFlow",
    "tagline": "Turn your rambling thoughts into coherent notes",
    "slug": "brainflow-2",
    "website": "https://www.producthunt.com/r/p/1173164?app_id=339",
    "hunter_name": "Tristan Manchester",
    "url": "https://www.producthunt.com/products/brainflow-2",
    "comments_ref": "1173164"
  }
]
```

## Turn on the API for everything else

The feed is the newest launches only.
Topics, collections, users, comments, and any filtered or ordered post list read the GraphQL API, which is reliable from anywhere.
Set free credentials and the rest of the commands light up:

```bash
export PRODUCTHUNT_TOKEN=...                   # a developer token, or
export PRODUCTHUNT_CLIENT_ID=...               # an API application's
export PRODUCTHUNT_CLIENT_SECRET=...           # client id and secret

ph posts -o json | jq '.[].name'      # the ranked post stream
ph comments 1173164                    # a post's comment thread
```

The credentials come from Product Hunt's API dashboard and are read from the environment only, never a flag.
With nothing set, `ph` reads the web plane and the feed-backed commands still work.

## Read one post

`post` takes the numeric id or the page slug and returns the full record: tagline, vote and comment counts, the media, the topics, the makers, and the edges into its comment thread and hunter.

```bash
ph post brainflow-2                    # full record as a table
ph post brainflow-2 -o json | jq .tagline
```

Without credentials, `post` falls back to the feed: it answers for a recent launch and reports not found for anything older.

## Shape the output

The same flags work on every command:

```bash
ph posts --fields name,votes,comments         # keep only these columns
ph post brainflow-2 --template '{{.Name}}: {{.Tagline}}'
ph comments 1173164 -o jsonl | jq .body        # one object per line, into jq
```

`-o` takes `table`, `markdown`, `list`, `json`, `jsonl`, `csv`, `tsv`, `url`, or `raw`.
Left to `auto`, it prints a table to a terminal and JSONL into a pipe.
See [output formats](/reference/output/) for the full contract.

## Topics, collections, and users

```bash
ph topic artificial-intelligence      # a topic with its top posts
ph topics -n 20                        # list topics
ph collection best-of-2026             # a collection with its posts
ph user @rrhoover                      # a user and the posts they made
```

These read the API plane, so they need credentials set.

## Resolve any URL offline

The `ref` commands need no network.
They turn any Product Hunt URL, path, Atom id, `@username`, or bare id into a canonical id and back into an addressable URL:

```bash
ph ref id "https://www.producthunt.com/products/brainflow-2"
ph ref url post 1173164
```

## Serve it instead

The same operations are available over HTTP and to agents over MCP:

```bash
ph serve --addr :7777 &
curl -s 'localhost:7777/v1/post/brainflow-2'   # NDJSON, one record per line
ph mcp                                          # MCP over stdio
```

The [guides](/guides/) cover the common jobs in more depth.
