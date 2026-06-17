---
title: "Configuration"
description: "Environment variables, the API credentials, defaults, and the data directory."
weight: 20
---

`ph` needs almost no configuration: it runs anonymously against the public Atom feed out of the box.
The settings below let you turn on the GraphQL API plane and tune politeness and storage.

## The API credentials

The API plane reads `api.producthunt.com` reliably from anywhere.
Turn it on with either a ready developer token or an API application's client id and secret:

```bash
export PRODUCTHUNT_TOKEN=...                   # a developer token, used directly
# or
export PRODUCTHUNT_CLIENT_ID=...               # an application's client id, and
export PRODUCTHUNT_CLIENT_SECRET=...           # its client secret
```

Both come free from Product Hunt's API dashboard.
When the client id and secret are set, `ph` mints a short-lived access token with the OAuth client-credentials grant on first use and reuses it for the rest of the run.
The credentials are read from the environment only, never a flag, so they stay out of your shell history and process list.

With nothing set, `ph` reads the web plane, which serves only the keyless Atom feed; the topic, collection, user, and comment surfaces report need-auth (exit 4) until credentials are present.

`--plane` decides which plane a command uses:

- `auto` (default): the API when credentials are set, the web otherwise.
- `web`: force the web plane. Only the feed-backed commands answer here.
- `api`: force the API plane; without credentials this exits 4 (needs auth).

## Defaults

| Setting | Default | Flag |
|---|---|---|
| Plane | `auto` | `--plane` |
| Delay between requests | 2s | `--rate` |
| Retry attempts on 429 or 5xx | 3 | `--retries` |
| Per-request timeout | 30s | `--timeout` |
| On-disk cache | under the data directory, fresh for 6h | `--cache-ttl`, `--no-cache`, `--refresh` |

A bare `--rate`, `--retries`, or `--timeout` shows `0` in `--help` because that is the unset value; `ph` fills the defaults above when you leave the flag off.
Pass a flag to override one, for example `--rate 5s` to slow a long run down further.

## The data directory

Caches and any record store live under one data directory, chosen in this order:

1. `--data-dir`
2. `PH_DATA_DIR`
3. `$XDG_DATA_HOME/ph`
4. `~/.local/share/ph`

## Environment variables

`ph` reads a small, fixed set of environment variables. Everything else is a flag.

| Variable | Effect |
|---|---|
| `PRODUCTHUNT_TOKEN` | a developer token that turns on the API plane |
| `PRODUCTHUNT_CLIENT_ID`, `PRODUCTHUNT_CLIENT_SECRET` | application credentials that mint a token for the API plane |
| `PH_DATA_DIR` | the data directory for caches and any record store |
| `XDG_DATA_HOME` | the base for `ph`'s data directory when `PH_DATA_DIR` is unset |
| `NO_COLOR` | when set, disables colored output |

The API credentials have no flag on purpose, so they stay out of your shell history and process list.
The pacing, plane, cache, and output settings are flags, not environment variables.

## Sending records to a store

`--db` tees every emitted record into a store as a side effect of reading, so a session fills a local database without a separate import step:

```bash
ph posts --db out.db                  # a SQLite file
ph comments 1173164 --db out.db       # adds to the same store
```

The bundled build stores records in SQLite, so `--db` takes a file path.
The records keep their JSON shape, so you query them with plain SQL afterwards.
