---
title: "CLI"
description: "Every command and subcommand, with the flags that matter."
weight: 10
---

```
ph <command> [arguments] [flags]
```

Run `ph <command> --help` for the full flag list on any command.

## Read commands

| Command | What it does |
|---|---|
| `posts` | List the post stream (the keyless Atom feed for the newest case, the API when keyed or filtered) |
| `post <ref>` | Show one post in full (API by id or slug, the feed when keyless and recent) |
| `comments <ref>` | List a post's comment thread (API plane) |
| `topics` | List topics (API plane; needs credentials) |
| `topic <ref>` | Show one topic with its top posts (API plane) |
| `collections` | List collections (API plane; needs credentials) |
| `collection <ref>` | Show one collection with its posts (API plane) |
| `user <ref>` | Show one user with the posts they made (API plane) |

A `<ref>` is a numeric post id, a page slug, an `@username` (for `user`), or a full Product Hunt URL.

## Crawl commands

| Command | What it does |
|---|---|
| `feed` | Read the public Atom feed of recent posts (the keyless web seed) |

## Ref commands (offline)

| Command | What it does |
|---|---|
| `ref id <ref>` | Resolve a URL, path, Atom id, `@username`, slug, or bare id to its canonical (kind, id) |
| `ref url <kind> <id>` | Build the addressable URL for a (kind, id) |

These need no network and answer instantly.

## Other commands

| Command | What it does |
|---|---|
| `serve [--addr]` | Serve the operations over HTTP as NDJSON |
| `mcp` | Run as an MCP server over stdio |
| `version` | Print the version and exit |

## Plane and filter flags

| Flag | Meaning |
|---|---|
| `--plane` | Which plane to read: `web`, `api`, or `auto` (default `auto`) |
| `--order` | List order: posts `ranking\|newest\|votes\|featured`, topics `followers\|newest`, collections `followers\|newest\|featured` |
| `--topic` | Filter posts to a topic slug (API plane) |
| `--featured` | Restrict to featured posts or collections (API plane) |
| `--after` | Posts on or after this ISO date (API plane) |
| `--before` | Posts on or before this ISO date (API plane) |

The API credentials are read from `PRODUCTHUNT_TOKEN`, or `PRODUCTHUNT_CLIENT_ID` and `PRODUCTHUNT_CLIENT_SECRET`, in the environment, never a flag.
The keyless feed serves only the newest, unfiltered post list; any `--order` other than `newest`, or any `--topic`, `--featured`, `--after`, or `--before`, needs the API plane.
See [configuration](/reference/configuration/).

## Global flags

These are shared by every operation, so they work the same on every command.

| Flag | Meaning |
|---|---|
| `-o, --output` | Output format: `auto`, `table`, `markdown`, `json`, `jsonl`, `csv`, `tsv`, `url`, `raw` |
| `--fields` | Comma-separated columns to keep |
| `--template` | Go text/template applied per record |
| `--no-header` | Omit the header row in `table` and `csv` |
| `-n, --limit` | Stop after N records (0 means no limit) |
| `--user-agent` | User-Agent sent with each request |
| `--rate` | Minimum delay between requests (default 2s) |
| `--retries` | Retry attempts on rate limit or 5xx (default 3) |
| `--timeout` | Per-request timeout (default 30s) |
| `--cache-ttl` | How long a cached response stays fresh (default 6h) |
| `--no-cache` | Bypass on-disk caches |
| `--refresh` | Fetch fresh copies and rewrite the cache, ignoring any hit |
| `--data-dir` | Override the data directory |
| `--db` | Tee every record into a SQLite store (a file path, e.g. `out.db`) |
| `-v, --verbose` | Increase verbosity (repeatable) |
| `-q, --quiet` | Suppress progress output |
| `--color` | `auto`, `always`, or `never` |

See [output formats](/reference/output/) for what `-o`, `--fields`, and `--template` produce, and [configuration](/reference/configuration/) for environment variables and defaults.
