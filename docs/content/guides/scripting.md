---
title: "Script and pipe ph"
description: "Shape records for jq, spreadsheets, and a local store, and serve them over HTTP."
weight: 30
---

Every `ph` command emits the same kind of record, and every command shares one output contract.
That makes `ph` a building block in a pipeline rather than a thing you read by eye.
This guide covers the moving parts.

## The default adapts to the destination

Left alone, `ph` prints a table to a terminal and JSONL into a pipe, so the same command reads well by hand and parses cleanly downstream:

```bash
ph feed                  # a table, because this is a terminal
ph feed | head           # JSONL, because this is a pipe
```

Reach for `-o` when you want something other than that default.

## Into jq

JSONL is one object per line, which is what `jq` wants:

```bash
ph posts -o jsonl | jq -r '.name'                    # just the names
ph posts -o jsonl | jq 'select(.votes > 100)'        # the popular ones
ph comments 1173164 -o jsonl | jq -r '.author'       # who commented
```

Use `-o json` instead when a tool wants a single array:

```bash
ph topics -o json | jq 'length'
```

## Narrow and reshape

Keep only the columns you care about, or write each line yourself:

```bash
ph posts --fields name,votes,comments         # three columns
ph posts --fields name,url -o tsv             # tab-separated, for cut and awk
ph post brainflow-2 --template '{{.Name}} got {{.Votes}} votes'
```

Template fields are the JSON keys, capitalised.
`--no-header` drops the header row in `table` and `csv` when a downstream tool expects bare rows.

## Into a spreadsheet

```bash
ph posts -n 100 -o csv > posts.csv            # open in any spreadsheet
ph topics -o csv > topics.csv
```

## Into a database

`--db` tees every emitted record into a store as a side effect of reading, so a session fills a local database without a separate import step:

```bash
ph posts -n 200 --db ph.db                    # a SQLite file
ph comments 1173164 --db ph.db                # adds to the same store
ph posts --db 'postgres://localhost/ph'       # or a Postgres URL
```

The records keep their shape, so you query them with plain SQL afterwards.

## Just the URLs

`-o url` prints the `url` column alone, which chains into other commands:

```bash
ph feed -o url                                # the page URLs of recent posts
ph feed -o url | while read u; do ph post "$u" -o json; done
```

## Serve the same operations

The operations are also an HTTP service and an MCP tool set, with no extra code.
The HTTP surface returns NDJSON, one record per line, so it pipes the same way:

```bash
ph serve --addr :7777 &
curl -s 'localhost:7777/v1/post/brainflow-2' | jq .tagline
curl -s 'localhost:7777/v1/comments/1173164'
ph mcp                                         # speak MCP over stdio to an agent
```

## Be a good client

`ph` paces requests and retries the transient failures, but a public site and a rate-limited API both push back under load.
Raise the delay between requests with `--rate 1s` for a long run, and lean on the on-disk cache: a repeated read is served from disk for six hours by default.
Use `--refresh` to rewrite the cache, `--no-cache` to bypass it, or `--cache-ttl` to change how long a hit stays fresh.
See [troubleshooting](/reference/troubleshooting/) for what the exit codes mean when a read does not come back.
