package producthunt

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// api.go reads the api plane: the GraphQL endpoint at api.producthunt.com. Each
// query is a constant string sent with its variables through postGraphQL, which
// resolves the token and classifies the response. The decoders unwrap the Relay
// envelope (data.<field>.edges[].node for a list, data.<field> for a single
// lookup), map each node onto its record, and follow pageInfo.endCursor up to the
// caller's limit. The numeric counts are JSON numbers here, so the decoders read
// them directly. A null node for a single lookup is ErrNotFound.

// --- query strings ---

const postFields = `
fragment PostFields on Post {
  id
  name
  tagline
  slug
  description
  url
  votesCount
  commentsCount
  reviewsCount
  reviewsRating
  createdAt
  featuredAt
  website
  thumbnail { type url videoUrl }
  media { type url videoUrl }
  productLinks { type url }
  topics(first: 20) { edges { node { id name slug } } }
  user { id username name }
  makers { id username name }
}`

const topicFields = `
fragment TopicFields on Topic {
  id
  name
  slug
  description
  followersCount
  postsCount
  image
  url
  createdAt
  posts(first: 20) { edges { node { id } } }
}`

const collectionFields = `
fragment CollectionFields on Collection {
  id
  name
  tagline
  description
  slug
  followersCount
  url
  coverImage
  createdAt
  featuredAt
  user { id username name }
  topics(first: 20) { edges { node { id name slug } } }
  posts(first: 20) { edges { node { id } } }
}`

const commentFields = `
fragment CommentFields on Comment {
  id
  body
  createdAt
  votesCount
  url
  user { id username name }
  replies(first: 10) {
    edges { node { id body createdAt votesCount url user { id username name } } }
  }
}`

const postsQuery = `query Posts($order: PostsOrder, $topic: String, $featured: Boolean, $postedAfter: DateTime, $postedBefore: DateTime, $first: Int, $after: String) {
  posts(order: $order, topic: $topic, featured: $featured, postedAfter: $postedAfter, postedBefore: $postedBefore, first: $first, after: $after) {
    edges { node { ...PostFields } }
    pageInfo { endCursor hasNextPage }
  }
}` + postFields

const postQuery = `query Post($id: ID, $slug: String) {
  post(id: $id, slug: $slug) { ...PostFields }
}` + postFields

const topicsQuery = `query Topics($order: TopicsOrder, $first: Int, $after: String) {
  topics(order: $order, first: $first, after: $after) {
    edges { node { ...TopicFields } }
    pageInfo { endCursor hasNextPage }
  }
}` + topicFields

const topicQuery = `query Topic($id: ID, $slug: String) {
  topic(id: $id, slug: $slug) { ...TopicFields }
}` + topicFields

const collectionsQuery = `query Collections($order: CollectionsOrder, $featured: Boolean, $first: Int, $after: String) {
  collections(order: $order, featured: $featured, first: $first, after: $after) {
    edges { node { ...CollectionFields } }
    pageInfo { endCursor hasNextPage }
  }
}` + collectionFields

const collectionQuery = `query Collection($id: ID, $slug: String) {
  collection(id: $id, slug: $slug) { ...CollectionFields }
}` + collectionFields

const userQuery = `query User($id: ID, $username: String) {
  user(id: $id, username: $username) {
    id
    name
    username
    headline
    profileImage
    coverImage
    url
    twitterUsername
    websiteUrl
    createdAt
    madePosts(first: 20) { edges { node { id } } }
  }
}`

const commentsQuery = `query Comments($id: ID, $slug: String, $first: Int, $after: String) {
  post(id: $id, slug: $slug) {
    id
    comments(first: $first, after: $after) {
      edges { node { ...CommentFields } }
      pageInfo { endCursor hasNextPage }
    }
  }
}` + commentFields

// --- wire types ---

type gqlMedia struct {
	Type     string `json:"type"`
	URL      string `json:"url"`
	VideoURL string `json:"videoUrl"`
}

type gqlNamedNode struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	Username string `json:"username"`
}

type gqlProductLink struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

type gqlPageInfo struct {
	EndCursor   string `json:"endCursor"`
	HasNextPage bool   `json:"hasNextPage"`
}

type gqlNamedEdges struct {
	Edges []struct {
		Node gqlNamedNode `json:"node"`
	} `json:"edges"`
}

type gqlPostEdges struct {
	Edges []struct {
		Node struct {
			ID string `json:"id"`
		} `json:"node"`
	} `json:"edges"`
}

type gqlPost struct {
	ID            string           `json:"id"`
	Name          string           `json:"name"`
	Tagline       string           `json:"tagline"`
	Slug          string           `json:"slug"`
	Description   string           `json:"description"`
	URL           string           `json:"url"`
	VotesCount    int              `json:"votesCount"`
	CommentsCount int              `json:"commentsCount"`
	ReviewsCount  int              `json:"reviewsCount"`
	ReviewsRating float64          `json:"reviewsRating"`
	CreatedAt     string           `json:"createdAt"`
	FeaturedAt    string           `json:"featuredAt"`
	Website       string           `json:"website"`
	Thumbnail     gqlMedia         `json:"thumbnail"`
	Media         []gqlMedia       `json:"media"`
	ProductLinks  []gqlProductLink `json:"productLinks"`
	Topics        gqlNamedEdges    `json:"topics"`
	User          gqlNamedNode     `json:"user"`
	Makers        []gqlNamedNode   `json:"makers"`
}

func (g *gqlPost) toPost() *Post {
	p := &Post{
		ID:            g.ID,
		Name:          g.Name,
		Tagline:       g.Tagline,
		Slug:          g.Slug,
		Description:   g.Description,
		Votes:         g.VotesCount,
		Comments:      g.CommentsCount,
		ReviewsCount:  g.ReviewsCount,
		ReviewsRating: g.ReviewsRating,
		CreatedAt:     g.CreatedAt,
		FeaturedAt:    g.FeaturedAt,
		Featured:      g.FeaturedAt != "",
		Website:       g.Website,
		Thumbnail:     g.Thumbnail.URL,
		URL:           g.URL,
		Hunter:        g.User.Username,
		HunterName:    g.User.Name,
		CommentsRef:   g.ID,
	}
	if p.URL == "" {
		p.URL = URLFor("post", firstNonEmpty(g.Slug, g.ID))
	}
	for _, m := range g.Media {
		p.Media = append(p.Media, Media(m))
	}
	for _, l := range g.ProductLinks {
		p.ProductLinks = append(p.ProductLinks, ProductLink(l))
	}
	for _, e := range g.Topics.Edges {
		t := e.Node
		p.Topics = append(p.Topics, TopicLink{ID: t.ID, Name: t.Name, Slug: t.Slug})
		if t.Slug != "" {
			p.TopicRefs = append(p.TopicRefs, t.Slug)
		}
	}
	for _, m := range g.Makers {
		p.Makers = append(p.Makers, Maker{ID: m.ID, Username: m.Username, Name: m.Name})
		if m.Username != "" {
			p.MakerRefs = append(p.MakerRefs, m.Username)
		}
	}
	if g.User.Username != "" {
		p.HunterRef = g.User.Username
	}
	return p
}

type gqlTopic struct {
	ID             string       `json:"id"`
	Name           string       `json:"name"`
	Slug           string       `json:"slug"`
	Description    string       `json:"description"`
	FollowersCount int          `json:"followersCount"`
	PostsCount     int          `json:"postsCount"`
	Image          string       `json:"image"`
	URL            string       `json:"url"`
	CreatedAt      string       `json:"createdAt"`
	Posts          gqlPostEdges `json:"posts"`
}

func (g *gqlTopic) toTopic() *Topic {
	t := &Topic{
		ID:          g.ID,
		Name:        g.Name,
		Slug:        g.Slug,
		Description: g.Description,
		Followers:   g.FollowersCount,
		PostsCount:  g.PostsCount,
		Image:       g.Image,
		CreatedAt:   g.CreatedAt,
		URL:         g.URL,
	}
	if t.URL == "" {
		t.URL = URLFor("topic", firstNonEmpty(g.Slug, g.ID))
	}
	for _, e := range g.Posts.Edges {
		if e.Node.ID != "" {
			t.PostRefs = append(t.PostRefs, e.Node.ID)
		}
	}
	return t
}

type gqlCollection struct {
	ID             string        `json:"id"`
	Name           string        `json:"name"`
	Tagline        string        `json:"tagline"`
	Description    string        `json:"description"`
	Slug           string        `json:"slug"`
	FollowersCount int           `json:"followersCount"`
	URL            string        `json:"url"`
	CoverImage     string        `json:"coverImage"`
	CreatedAt      string        `json:"createdAt"`
	FeaturedAt     string        `json:"featuredAt"`
	User           gqlNamedNode  `json:"user"`
	Topics         gqlNamedEdges `json:"topics"`
	Posts          gqlPostEdges  `json:"posts"`
}

func (g *gqlCollection) toCollection() *Collection {
	col := &Collection{
		ID:          g.ID,
		Name:        g.Name,
		Tagline:     g.Tagline,
		Description: g.Description,
		Slug:        g.Slug,
		Followers:   g.FollowersCount,
		CoverImage:  g.CoverImage,
		CreatedAt:   g.CreatedAt,
		FeaturedAt:  g.FeaturedAt,
		Curator:     g.User.Username,
		CuratorName: g.User.Name,
		URL:         g.URL,
	}
	if col.URL == "" {
		col.URL = URLFor("collection", firstNonEmpty(g.Slug, g.ID))
	}
	if g.User.Username != "" {
		col.CuratorRef = g.User.Username
	}
	for _, e := range g.Topics.Edges {
		t := e.Node
		col.Topics = append(col.Topics, TopicLink{ID: t.ID, Name: t.Name, Slug: t.Slug})
		if t.Slug != "" {
			col.TopicRefs = append(col.TopicRefs, t.Slug)
		}
	}
	for _, e := range g.Posts.Edges {
		if e.Node.ID != "" {
			col.PostRefs = append(col.PostRefs, e.Node.ID)
		}
	}
	return col
}

type gqlUser struct {
	ID              string       `json:"id"`
	Name            string       `json:"name"`
	Username        string       `json:"username"`
	Headline        string       `json:"headline"`
	ProfileImage    string       `json:"profileImage"`
	CoverImage      string       `json:"coverImage"`
	URL             string       `json:"url"`
	TwitterUsername string       `json:"twitterUsername"`
	WebsiteURL      string       `json:"websiteUrl"`
	CreatedAt       string       `json:"createdAt"`
	MadePosts       gqlPostEdges `json:"madePosts"`
}

func (g *gqlUser) toUser() *User {
	u := &User{
		ID:        g.ID,
		Username:  g.Username,
		Name:      g.Name,
		Headline:  g.Headline,
		Twitter:   g.TwitterUsername,
		Website:   g.WebsiteURL,
		Image:     g.ProfileImage,
		Cover:     g.CoverImage,
		CreatedAt: g.CreatedAt,
		URL:       g.URL,
	}
	if u.URL == "" {
		u.URL = URLFor("user", firstNonEmpty(g.Username, g.ID))
	}
	for _, e := range g.MadePosts.Edges {
		if e.Node.ID != "" {
			u.PostRefs = append(u.PostRefs, e.Node.ID)
		}
	}
	return u
}

type gqlComment struct {
	ID         string       `json:"id"`
	Body       string       `json:"body"`
	CreatedAt  string       `json:"createdAt"`
	VotesCount int          `json:"votesCount"`
	URL        string       `json:"url"`
	User       gqlNamedNode `json:"user"`
	Replies    struct {
		Edges []struct {
			Node gqlComment `json:"node"`
		} `json:"edges"`
	} `json:"replies"`
}

func (g *gqlComment) toComment(postID, parent string) *Comment {
	c := &Comment{
		ID:         g.ID,
		Post:       postID,
		Parent:     parent,
		Body:       g.Body,
		Votes:      g.VotesCount,
		CreatedAt:  g.CreatedAt,
		Author:     g.User.Username,
		AuthorName: g.User.Name,
		URL:        g.URL,
	}
	if g.User.Username != "" {
		c.UserRef = g.User.Username
	}
	return c
}

// --- methods ---

// PostsAPI reads the ranked, filtered post stream, paging up to n.
func (c *Client) PostsAPI(ctx context.Context, n int) ([]*Post, error) {
	vars := map[string]any{"first": pageSize(n)}
	if c.cfg.Order != "" {
		o, ok := postsOrder(c.cfg.Order)
		if !ok {
			return nil, ErrUsage
		}
		vars["order"] = o
	}
	if c.cfg.Topic != "" {
		vars["topic"] = c.cfg.Topic
	}
	if c.cfg.Featured {
		vars["featured"] = true
	}
	if c.cfg.After != "" {
		vars["postedAfter"] = c.cfg.After
	}
	if c.cfg.Before != "" {
		vars["postedBefore"] = c.cfg.Before
	}
	var out []*Post
	for {
		data, err := c.postGraphQL(ctx, postsQuery, vars)
		if err != nil {
			return nil, err
		}
		var conn struct {
			Posts struct {
				Edges []struct {
					Node gqlPost `json:"node"`
				} `json:"edges"`
				PageInfo gqlPageInfo `json:"pageInfo"`
			} `json:"posts"`
		}
		if err := json.Unmarshal(data, &conn); err != nil {
			return nil, fmt.Errorf("decode posts: %w", err)
		}
		for i := range conn.Posts.Edges {
			out = append(out, conn.Posts.Edges[i].Node.toPost())
			if n > 0 && len(out) >= n {
				return out, nil
			}
		}
		if !conn.Posts.PageInfo.HasNextPage || conn.Posts.PageInfo.EndCursor == "" {
			break
		}
		vars["after"] = conn.Posts.PageInfo.EndCursor
	}
	return out, nil
}

// PostAPI reads one post by numeric id or slug.
func (c *Client) PostAPI(ctx context.Context, ref string) (*Post, error) {
	data, err := c.postGraphQL(ctx, postQuery, idOrSlugVars(ref, "slug"))
	if err != nil {
		return nil, err
	}
	var wrap struct {
		Post *gqlPost `json:"post"`
	}
	if err := json.Unmarshal(data, &wrap); err != nil {
		return nil, fmt.Errorf("decode post: %w", err)
	}
	if wrap.Post == nil {
		return nil, ErrNotFound
	}
	return wrap.Post.toPost(), nil
}

// TopicsAPI lists topics, paging up to n.
func (c *Client) TopicsAPI(ctx context.Context, n int) ([]*Topic, error) {
	vars := map[string]any{"first": pageSize(n)}
	if c.cfg.Order != "" {
		o, ok := topicsOrder(c.cfg.Order)
		if !ok {
			return nil, ErrUsage
		}
		vars["order"] = o
	}
	var out []*Topic
	for {
		data, err := c.postGraphQL(ctx, topicsQuery, vars)
		if err != nil {
			return nil, err
		}
		var conn struct {
			Topics struct {
				Edges []struct {
					Node gqlTopic `json:"node"`
				} `json:"edges"`
				PageInfo gqlPageInfo `json:"pageInfo"`
			} `json:"topics"`
		}
		if err := json.Unmarshal(data, &conn); err != nil {
			return nil, fmt.Errorf("decode topics: %w", err)
		}
		for i := range conn.Topics.Edges {
			out = append(out, conn.Topics.Edges[i].Node.toTopic())
			if n > 0 && len(out) >= n {
				return out, nil
			}
		}
		if !conn.Topics.PageInfo.HasNextPage || conn.Topics.PageInfo.EndCursor == "" {
			break
		}
		vars["after"] = conn.Topics.PageInfo.EndCursor
	}
	return out, nil
}

// TopicAPI reads one topic with its top posts.
func (c *Client) TopicAPI(ctx context.Context, ref string) (*Topic, error) {
	data, err := c.postGraphQL(ctx, topicQuery, idOrSlugVars(ref, "slug"))
	if err != nil {
		return nil, err
	}
	var wrap struct {
		Topic *gqlTopic `json:"topic"`
	}
	if err := json.Unmarshal(data, &wrap); err != nil {
		return nil, fmt.Errorf("decode topic: %w", err)
	}
	if wrap.Topic == nil {
		return nil, ErrNotFound
	}
	return wrap.Topic.toTopic(), nil
}

// CollectionsAPI lists collections, paging up to n.
func (c *Client) CollectionsAPI(ctx context.Context, n int) ([]*Collection, error) {
	vars := map[string]any{"first": pageSize(n)}
	if c.cfg.Order != "" {
		o, ok := collectionsOrder(c.cfg.Order)
		if !ok {
			return nil, ErrUsage
		}
		vars["order"] = o
	}
	if c.cfg.Featured {
		vars["featured"] = true
	}
	var out []*Collection
	for {
		data, err := c.postGraphQL(ctx, collectionsQuery, vars)
		if err != nil {
			return nil, err
		}
		var conn struct {
			Collections struct {
				Edges []struct {
					Node gqlCollection `json:"node"`
				} `json:"edges"`
				PageInfo gqlPageInfo `json:"pageInfo"`
			} `json:"collections"`
		}
		if err := json.Unmarshal(data, &conn); err != nil {
			return nil, fmt.Errorf("decode collections: %w", err)
		}
		for i := range conn.Collections.Edges {
			out = append(out, conn.Collections.Edges[i].Node.toCollection())
			if n > 0 && len(out) >= n {
				return out, nil
			}
		}
		if !conn.Collections.PageInfo.HasNextPage || conn.Collections.PageInfo.EndCursor == "" {
			break
		}
		vars["after"] = conn.Collections.PageInfo.EndCursor
	}
	return out, nil
}

// CollectionAPI reads one collection with its posts.
func (c *Client) CollectionAPI(ctx context.Context, ref string) (*Collection, error) {
	data, err := c.postGraphQL(ctx, collectionQuery, idOrSlugVars(ref, "slug"))
	if err != nil {
		return nil, err
	}
	var wrap struct {
		Collection *gqlCollection `json:"collection"`
	}
	if err := json.Unmarshal(data, &wrap); err != nil {
		return nil, fmt.Errorf("decode collection: %w", err)
	}
	if wrap.Collection == nil {
		return nil, ErrNotFound
	}
	return wrap.Collection.toCollection(), nil
}

// UserAPI reads one user with the posts they made.
func (c *Client) UserAPI(ctx context.Context, ref string) (*User, error) {
	data, err := c.postGraphQL(ctx, userQuery, idOrSlugVars(ref, "username"))
	if err != nil {
		return nil, err
	}
	var wrap struct {
		User *gqlUser `json:"user"`
	}
	if err := json.Unmarshal(data, &wrap); err != nil {
		return nil, fmt.Errorf("decode user: %w", err)
	}
	if wrap.User == nil {
		return nil, ErrNotFound
	}
	return wrap.User.toUser(), nil
}

// CommentsAPI reads a post's comment thread, paging up to n. Each top-level comment
// is followed by its replies, with the parent edge set.
func (c *Client) CommentsAPI(ctx context.Context, ref string, n int) ([]*Comment, error) {
	vars := idOrSlugVars(ref, "slug")
	vars["first"] = pageSize(n)
	var out []*Comment
	for {
		data, err := c.postGraphQL(ctx, commentsQuery, vars)
		if err != nil {
			return nil, err
		}
		var wrap struct {
			Post *struct {
				ID       string `json:"id"`
				Comments struct {
					Edges []struct {
						Node gqlComment `json:"node"`
					} `json:"edges"`
					PageInfo gqlPageInfo `json:"pageInfo"`
				} `json:"comments"`
			} `json:"post"`
		}
		if err := json.Unmarshal(data, &wrap); err != nil {
			return nil, fmt.Errorf("decode comments: %w", err)
		}
		if wrap.Post == nil {
			return nil, ErrNotFound
		}
		postID := wrap.Post.ID
		for i := range wrap.Post.Comments.Edges {
			node := &wrap.Post.Comments.Edges[i].Node
			out = append(out, node.toComment(postID, ""))
			for j := range node.Replies.Edges {
				reply := &node.Replies.Edges[j].Node
				out = append(out, reply.toComment(postID, node.ID))
			}
			if n > 0 && len(out) >= n {
				return out, nil
			}
		}
		if !wrap.Post.Comments.PageInfo.HasNextPage || wrap.Post.Comments.PageInfo.EndCursor == "" {
			break
		}
		vars["after"] = wrap.Post.Comments.PageInfo.EndCursor
	}
	return out, nil
}

// --- helpers ---

// idOrSlugVars routes a ref to the id variable when it is numeric, else to the
// human-key variable (slug or username) the API also accepts.
func idOrSlugVars(ref, humanKey string) map[string]any {
	r := Classify(ref)
	id := r.ID
	if id == "" {
		id = strings.TrimSpace(ref)
	}
	if numRE.MatchString(id) {
		return map[string]any{"id": id}
	}
	return map[string]any{humanKey: id}
}

// pageSize caps a single connection request to the connection maximum.
func pageSize(n int) int {
	if n <= 0 || n > apiMaxLimit {
		return apiMaxLimit
	}
	return n
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func postsOrder(s string) (string, bool) {
	switch strings.ToLower(s) {
	case "ranking":
		return "RANKING", true
	case "newest":
		return "NEWEST", true
	case "votes":
		return "VOTES", true
	case "featured":
		return "FEATURED_AT", true
	default:
		return "", false
	}
}

func topicsOrder(s string) (string, bool) {
	switch strings.ToLower(s) {
	case "followers":
		return "FOLLOWERS_COUNT", true
	case "newest":
		return "NEWEST", true
	default:
		return "", false
	}
}

func collectionsOrder(s string) (string, bool) {
	switch strings.ToLower(s) {
	case "followers":
		return "FOLLOWERS_COUNT", true
	case "newest":
		return "NEWEST", true
	case "featured":
		return "FEATURED_AT", true
	default:
		return "", false
	}
}
