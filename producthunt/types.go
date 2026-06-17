package producthunt

// types.go holds the exported records the commands emit. Their json tags name the
// fields a reader sees, kit:"id" marks the key the record store upserts on,
// kit:"body" marks the long-text field `ph cat` and the Markdown export print, and
// table:",truncate" keeps wide free text from blowing up a terminal table. Each
// record carries only fields a keyless Atom-feed read or a free public API token
// can fill: no private scope, no viewer state (whether you voted or follow), no
// member inbox, no maker dashboard metric.
//
// The kit:"link" edges connect the records into one graph a host walks for
// breadth-first crawls. The numeric id is the universal key, so a comment, a topic's
// post, a collection's post, and a feed entry all point at the same post record. A
// slice-valued kit:"link" field yields one edge per element, so the topic, maker,
// and post lists are walkable edges:
//
//	feed       --(Post)--> post        posts --(Post)--> post
//	post       --comments_ref--> comments --user_ref--> user
//	                                       --post-------> post
//	post       --hunter_ref--> user --post_refs-->> post
//	post       --maker_refs-->> user --post_refs-->> post
//	post       --topic_refs-->> topic --post_refs-->> post
//	topic      --post_refs-->> post
//	collection --curator_ref--> user      --topic_refs-->> topic
//	collection --post_refs-->> post
//	user       --post_refs-->> post
//
// A host that starts at the keyless feed can reach posts, from posts reach topics
// and users, from topics and users reach more posts, and so walk the public estate.
// No record is a dead leaf: every list record carries the edges back into the graph,
// and a person is a username and a display name, never a fabricated profile beyond
// what the API returns.

// Post is the core record: a product launch. The id is the numeric post id, the
// same id the website and the Atom feed use, so a web read and an API read address
// the same record.
type Post struct {
	ID            string        `json:"id" kit:"id"` // the numeric post id, e.g. "1173164"
	Name          string        `json:"name,omitempty" table:",truncate"`
	Tagline       string        `json:"tagline,omitempty" table:",truncate"`
	Slug          string        `json:"slug,omitempty"`
	Description   string        `json:"description,omitempty" table:",truncate" kit:"body"`
	Votes         int           `json:"votes,omitempty"`
	Comments      int           `json:"comments,omitempty"`
	ReviewsCount  int           `json:"reviews_count,omitempty" table:"-"`
	ReviewsRating float64       `json:"reviews_rating,omitempty" table:"-"`
	Featured      bool          `json:"featured,omitempty" table:"-"`
	FeaturedAt    string        `json:"featured_at,omitempty" table:"-"`
	CreatedAt     string        `json:"created_at,omitempty"`
	Website       string        `json:"website,omitempty" table:"-"` // the outbound product URL
	Thumbnail     string        `json:"thumbnail,omitempty" table:"-"`
	Media         []Media       `json:"media,omitempty" table:"-"`
	ProductLinks  []ProductLink `json:"product_links,omitempty" table:"-"`
	Topics        []TopicLink   `json:"topics,omitempty" table:"-"`
	Hunter        string        `json:"hunter,omitempty" table:"-"` // the hunter's username
	HunterName    string        `json:"hunter_name,omitempty" table:"-"`
	Makers        []Maker       `json:"makers,omitempty" table:"-"`
	URL           string        `json:"url"`                                                                   // the discussion page
	CommentsRef   string        `json:"comments_ref,omitempty" table:"-" kit:"link,kind=producthunt/comments"` // = ID
	HunterRef     string        `json:"hunter_ref,omitempty" table:"-" kit:"link,kind=producthunt/user"`       // hunter username
	MakerRefs     []string      `json:"maker_refs,omitempty" table:"-" kit:"link,kind=producthunt/user"`       // maker usernames
	TopicRefs     []string      `json:"topic_refs,omitempty" table:"-" kit:"link,kind=producthunt/topic"`      // topic slugs
}

// Topic is a subject tag a post can carry, emitted by topics and topic. PostRefs is
// its top posts, each a walkable post edge.
type Topic struct {
	ID          string   `json:"id" kit:"id"`
	Name        string   `json:"name,omitempty"`
	Slug        string   `json:"slug,omitempty"`
	Description string   `json:"description,omitempty" table:",truncate" kit:"body"`
	Followers   int      `json:"followers,omitempty"`
	PostsCount  int      `json:"posts_count,omitempty"`
	Image       string   `json:"image,omitempty" table:"-"`
	CreatedAt   string   `json:"created_at,omitempty" table:"-"`
	URL         string   `json:"url"`
	PostRefs    []string `json:"post_refs,omitempty" table:"-" kit:"link,kind=producthunt/post"` // its top posts
}

// Collection is a curated list of posts, emitted by collections and collection.
type Collection struct {
	ID          string      `json:"id" kit:"id"`
	Name        string      `json:"name,omitempty" table:",truncate"`
	Tagline     string      `json:"tagline,omitempty" table:",truncate"`
	Description string      `json:"description,omitempty" table:",truncate" kit:"body"`
	Slug        string      `json:"slug,omitempty"`
	Followers   int         `json:"followers,omitempty"`
	CoverImage  string      `json:"cover_image,omitempty" table:"-"`
	CreatedAt   string      `json:"created_at,omitempty" table:"-"`
	FeaturedAt  string      `json:"featured_at,omitempty" table:"-"`
	Curator     string      `json:"curator,omitempty" table:"-"` // the owner's username
	CuratorName string      `json:"curator_name,omitempty" table:"-"`
	Topics      []TopicLink `json:"topics,omitempty" table:"-"`
	URL         string      `json:"url"`
	CuratorRef  string      `json:"curator_ref,omitempty" table:"-" kit:"link,kind=producthunt/user"`
	TopicRefs   []string    `json:"topic_refs,omitempty" table:"-" kit:"link,kind=producthunt/topic"`
	PostRefs    []string    `json:"post_refs,omitempty" table:"-" kit:"link,kind=producthunt/post"`
}

// User is a Product Hunt member, emitted by user. PostRefs is the posts they made.
type User struct {
	ID        string   `json:"id" kit:"id"`
	Username  string   `json:"username,omitempty"`
	Name      string   `json:"name,omitempty"`
	Headline  string   `json:"headline,omitempty" table:",truncate"`
	Twitter   string   `json:"twitter,omitempty" table:"-"`
	Website   string   `json:"website,omitempty" table:"-"`
	Image     string   `json:"image,omitempty" table:"-"`
	Cover     string   `json:"cover,omitempty" table:"-"`
	CreatedAt string   `json:"created_at,omitempty" table:"-"`
	URL       string   `json:"url"`
	PostRefs  []string `json:"post_refs,omitempty" table:"-" kit:"link,kind=producthunt/post"` // posts they made
}

// Comment is one comment on a post, emitted by comments. Post is the edge back to
// the commented post; an author is a username and a display name, never a fabricated
// member node.
type Comment struct {
	ID         string `json:"id" kit:"id"`
	Post       string `json:"post,omitempty" table:"-" kit:"link,kind=producthunt/post"` // the post id
	Parent     string `json:"parent,omitempty" table:"-"`                                // parent comment id, for replies
	Body       string `json:"body,omitempty" table:",truncate" kit:"body"`
	Votes      int    `json:"votes,omitempty"`
	CreatedAt  string `json:"created_at,omitempty"`
	Author     string `json:"author,omitempty"` // username
	AuthorName string `json:"author_name,omitempty" table:"-"`
	URL        string `json:"url,omitempty" table:"-"`
	UserRef    string `json:"user_ref,omitempty" table:"-" kit:"link,kind=producthunt/user"`
}

// Media is one image or video in a post's gallery or its thumbnail.
type Media struct {
	Type     string `json:"type,omitempty"` // image, video
	URL      string `json:"url,omitempty"`
	VideoURL string `json:"video_url,omitempty"`
}

// ProductLink is one outbound link a post carries (website, app store, repo).
type ProductLink struct {
	Type string `json:"type,omitempty"` // website, app_store, play_store, github, ...
	URL  string `json:"url,omitempty"`
}

// TopicLink is the embedded reference to a topic inside a post or a collection.
type TopicLink struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	Slug string `json:"slug,omitempty"`
}

// Maker is the embedded reference to a person who made a post.
type Maker struct {
	ID       string `json:"id,omitempty"`
	Username string `json:"username,omitempty"`
	Name     string `json:"name,omitempty"`
}

// Ref is the result of `ph ref id`: the canonical (kind, id) a reference resolves
// to, plus the URL, all without touching the network.
type Ref struct {
	Input string `json:"input"`
	Kind  string `json:"kind"`
	ID    string `json:"id"`
	URL   string `json:"url"`
}
