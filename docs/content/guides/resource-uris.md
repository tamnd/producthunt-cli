---
title: "Resource URIs"
description: "Use ph as a database/sql-style driver so a host program can address Product Hunt as producthunt:// URIs."
weight: 20
---

`ph` is a command line, but the `producthunt` Go package is also a small driver
that makes Product Hunt addressable as a resource URI. A host program registers
it the way a program registers a database driver with `database/sql`, then
dereferences `producthunt://` URIs without knowing anything about how Product
Hunt is fetched.

The host that does this today is [ant](https://github.com/tamnd/ant), a single
binary that puts one URI namespace over a family of site tools. The examples
below use `ant`; any program that links the package gets the same behaviour.

## Mounting the driver

A host enables the driver with one blank import, exactly like `import _
"github.com/lib/pq"`:

```go
import _ "github.com/tamnd/producthunt-cli/producthunt"
```

The package's `init` registers a domain with the scheme `producthunt` for the
host `producthunt.com`. The standalone `ph` binary does not change.

## Addressing records

A URI is `scheme://authority/id`. The id is the canonical numeric post id shared
by both planes, or the human slug or username the API also accepts:

| URI | What it is |
| --- | --- |
| `producthunt://post/1173164` | a post (product launch) by its numeric id |
| `producthunt://post/brainflow-2` | the same kind of record, by its page slug |
| `producthunt://topic/artificial-intelligence` | a topic by its slug |
| `producthunt://collection/best-of-2026` | a collection by its slug |
| `producthunt://user/rrhoover` | a user by their username |
| `producthunt://comments/1173164` | a post's comment thread, by the post id |

```bash
ant get producthunt://post/1173164         # the post record
ant cat producthunt://post/1173164         # just the description body
ant url producthunt://post/1173164         # the addressable URL
ant resolve https://www.producthunt.com/products/brainflow-2
```

The last line resolves a pasted page link back to its
`producthunt://post/brainflow-2` URI offline, the same work `ph ref id` does.

## Walking the graph

Each record carries `kit:"link"` edges, so a host can follow the graph and write
it to disk. A post is the hub:

| Edge | Points at | Direction |
| --- | --- | --- |
| `comments_ref` | the post's comment thread | out to a list |
| `hunter_ref` | the user who hunted it | out to a user |
| `maker_refs` | each maker of the product | out to users |
| `topic_refs` | each topic it is filed under | out to topics |

The other records point back, so the graph is walkable from any seed:

| Record | Edge | Points at |
| --- | --- | --- |
| `topic` | `post_refs` | the posts in the topic |
| `collection` | `post_refs`, `topic_refs`, `curator_ref` | its posts, topics, and curator |
| `user` | `post_refs` | the posts the user made |
| `comment` | `user_ref` | the comment's author |

```bash
ant ls     producthunt://post/1173164            # the edges out of this post
ant export producthunt://post/1173164 --follow 2 --to ./data
```

`ant ls` reads the link fields off the record, so one post lists its comment
thread, its hunter, its makers, and its topics as URIs a host can follow next:

```
producthunt://comments/1173164
producthunt://user/rrhoover
producthunt://topic/artificial-intelligence
```

`ant export --follow` and `ant graph` walk those edges, across tools when a link
points at another site's scheme. A slice-valued edge (the makers and topics of a
post, the posts of a topic) expands to one URI per element, so the walk fans out
without any per-record special casing.

## Walking from the keyless seed

The feed is the one surface that reads with no account, so it is the natural seed
for a crawl. Each feed post carries a `comments_ref` and resolves to a full
`post`, and a post fans out to its topics, makers, and hunter, each of which is
itself an addressable URI:

```bash
ant export producthunt://feed --follow 2 --to ./data
```

With credentials set, the API plane fills the records the feed leaves sparse (the
feed carries no usernames or topics), so the same walk reaches the whole public
graph: from a post out to its topics and makers, from a topic back down to its
posts, and from a user back to the posts they made.

## Why this is the same code

The driver and the binary share one definition per operation. The `post` op
answers both `ph post` on the command line and `ant get
producthunt://post/...` through a host, from the same handler and the same
client. There is no second implementation to keep in step.
