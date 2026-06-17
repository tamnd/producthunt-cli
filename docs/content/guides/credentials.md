---
title: "Set up the API plane"
description: "Get free credentials and turn on the reliable GraphQL reads."
weight: 5
---

`ph` reads the keyless Atom feed out of the box, which is the newest launches and nothing else.
The topic, collection, user, and comment surfaces, and any ordered or filtered post list, read the GraphQL API, which is reliable from anywhere.
This guide turns that plane on.

## Why the feed is not enough

`www.producthunt.com` is fronted by Cloudflare, so every page but the Atom feed at `/feed` returns a challenge that `ph` reports honestly with exit 4 rather than working around.
The feed is the one anonymous surface, so without credentials you can read `ph feed`, the newest `ph posts`, and a recent `ph post`.
Everything else needs the API.

## Get credentials

Product Hunt's API is free.
Sign in, open the [API dashboard](https://www.producthunt.com/v2/oauth/applications), and create an application.
You end up with two ways to authenticate, and `ph` accepts either:

- A **developer token**, used directly.
- An application's **client id and secret**, which `ph` exchanges for a short-lived access token itself.

## Set them in the environment

`ph` reads credentials from the environment only, never a flag, so they stay out of your shell history and process list:

```bash
export PRODUCTHUNT_TOKEN=...                   # the developer token, or
export PRODUCTHUNT_CLIENT_ID=...               # the application's client id, and
export PRODUCTHUNT_CLIENT_SECRET=...           # its client secret
```

With the client id and secret set, `ph` mints an access token with the OAuth client-credentials grant the first time it needs one and reuses it for the rest of the run.
There is nothing else to configure.
Put the exports in your shell profile to make every session reliable.

## Confirm it worked

A surface that needs the API answers once the credentials are set:

```bash
ph topics -n 5                        # exit 4 without credentials, records with them
ph comments 1173164 -o jsonl | jq .body
```

If you still get exit 4 with an invalid-token message, the credentials are stale.
Re-check the client id and secret in the dashboard; `ph` mints a fresh token from them on its own.

## Choosing a plane explicitly

`--plane` decides which plane a command uses:

- `auto` (default): the API when credentials are set, the web otherwise.
- `web`: force the web plane. Only the feed-backed commands answer here.
- `api`: force the API plane; without credentials this exits 4.

```bash
ph posts --plane web                  # the keyless feed, even with credentials set
ph posts --plane api --order votes    # the ranked stream from the API
```

See [configuration](/reference/configuration/) for the full list of settings and the environment variables `ph` reads.
