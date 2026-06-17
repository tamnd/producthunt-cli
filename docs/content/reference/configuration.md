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
| Requests | paced and retried on 429/5xx | `--rate`, `--retries` |
| Per-request timeout | 30s | `--timeout` |
| On-disk cache | under the data directory, fresh for 6h | `--cache-ttl`, `--no-cache`, `--refresh` |

## The data directory

Caches and any record store live under one data directory, chosen in this order:

1. `--data-dir`
2. `PH_DATA_DIR`
3. `$XDG_DATA_HOME/ph`
4. `~/.local/share/ph`

## Environment variables

Every flag has an environment fallback, prefixed `PH_` in upper case with dashes as underscores.
For example:

```bash
export PH_RATE=1s        # same as --rate 1s
export PH_PLANE=web      # same as --plane web
export PH_DATA_DIR=~/data/ph
```

Flags win over environment variables, which win over the built-in defaults.
The exception is the API credentials, which have no flag and are read only from `PRODUCTHUNT_TOKEN`, `PRODUCTHUNT_CLIENT_ID`, and `PRODUCTHUNT_CLIENT_SECRET`.

## Sending records to a store

`--db` tees every emitted record into a store as a side effect of reading, so a session fills a local database without a separate import step:

```bash
ph posts --db out.db                  # SQLite file
ph comments 1173164 --db 'postgres://...'
```
