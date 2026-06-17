---
title: "Troubleshooting"
description: "The handful of things that trip people up, and how to fix each one."
weight: 40
---

Most of these come down to network reality or how Product Hunt serves its data, not a bug.

## A web read returns exit 4 (blocked)

`www.producthunt.com` is fronted by Cloudflare, so every page but the Atom feed hits a challenge interstitial and `ph` reports it honestly with exit 4 rather than working around it.
The feed at `/feed` is the one anonymous surface.
Two ways forward:

- Set `PRODUCTHUNT_TOKEN`, or `PRODUCTHUNT_CLIENT_ID` and `PRODUCTHUNT_CLIENT_SECRET`, and let `--plane auto` use the GraphQL API, which is reliable from anywhere.
- Stay on the feed-backed commands (`feed`, the newest `posts`, and a recent `post`), which read with no account.

`ph` does not solve anti-bot challenges, forge sensors, or rotate proxies.
A wall is reported, not bypassed.

## A command needs credentials

The topic, collection, user, and comment surfaces are API only, and `--plane api` forces the API plane.
Either needs `PRODUCTHUNT_TOKEN`, or the client id and secret pair, set; without them you get exit 4 with a message naming the remedy.
Set the credentials, or stay on the feed-backed commands.
See [configuration](/reference/configuration/).

## The API rejects the token

If a GraphQL read returns exit 4 with an invalid-token message, the token has expired or the application's credentials are wrong.
With `PRODUCTHUNT_CLIENT_ID` and `PRODUCTHUNT_CLIENT_SECRET` set, `ph` mints a fresh token itself, so this usually means the client id or secret is stale.
Re-check them in Product Hunt's API dashboard.

## Requests start failing or returning 429

Product Hunt rate-limits like any public site, and the GraphQL API also caps query complexity.
`ph` already paces requests and retries the transient failures, but a hard limit still means backing off.
`ph` already waits two seconds between requests; raise that with `--rate` (for example `--rate 5s`) and retry later.
A burst of 429 or a complexity error is the API asking you to slow down, not a defect.

## Nothing is found for something you expected

The two planes differ.
The feed carries only the newest launches and leaves usernames and topics empty, so a keyless `post` answers for a recent launch and reports not found for anything older.
Set credentials and the API plane fills the fuller record and reaches the back catalogue.
Check that the slug or id is spelled the way the site uses it.

## Stale data

Reads are cached on disk and stay fresh for 6h by default.
Use `--refresh` to fetch fresh copies and rewrite the cache, `--no-cache` to bypass it entirely, or `--cache-ttl` to change how long a hit stays fresh.

## The binary is not on your PATH

`go install` puts the binary in `$(go env GOPATH)/bin` (usually `~/go/bin`), and a release archive leaves it wherever you unpacked it.
If your shell cannot find `ph`, add that directory to your `PATH`.
See [installation](/getting-started/installation/).

## Seeing what ph actually did

When something behaves unexpectedly, `-v` adds per-request detail so you can see the URLs it hit and the responses it got.
That is usually enough to tell a rate limit or a wall apart from a genuinely empty result.
