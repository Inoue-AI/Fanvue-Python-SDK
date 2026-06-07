package fanvue

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

// PostAudience enumerates the audiences a post can target.
type PostAudience string

const (
	// AudienceSubscribers restricts a post to subscribers only.
	AudienceSubscribers PostAudience = "subscribers"
	// AudienceFollowersAndSubscribers exposes a post to followers and subscribers.
	AudienceFollowersAndSubscribers PostAudience = "followers-and-subscribers"
)

// Pagination is the standard page-based pagination envelope Fanvue returns on
// list endpoints.
type Pagination struct {
	Page    float64 `json:"page"`
	Size    float64 `json:"size"`
	HasMore bool    `json:"hasMore"`
}

// PostCollection is a content collection a post belongs to.
type PostCollection struct {
	UUID  string `json:"uuid"`
	Label string `json:"label"`
}

// PostTips holds the tip statistics for a post (amounts in cents).
type PostTips struct {
	Count      float64 `json:"count"`
	TotalGross float64 `json:"totalGross"`
	TotalNet   float64 `json:"totalNet"`
}

// Post is a content post owned by the authenticated user. Nullable fields use
// pointers so a JSON null is distinguishable from a zero value.
type Post struct {
	UUID             string           `json:"uuid"`
	Text             *string          `json:"text"`
	Audience         PostAudience     `json:"audience"`
	Collections      []PostCollection `json:"collections,omitempty"`
	CommentsCount    float64          `json:"commentsCount,omitempty"`
	LikesCount       float64          `json:"likesCount,omitempty"`
	IsPinned         bool             `json:"isPinned,omitempty"`
	MediaUUIDs       []string         `json:"mediaUuids,omitempty"`
	MediaPreviewUUID *string          `json:"mediaPreviewUuid"`
	Price            *float64         `json:"price"`
	ExpiresAt        *string          `json:"expiresAt"`
	PublishAt        *string          `json:"publishAt"`
	PublishedAt      *string          `json:"publishedAt"`
	CreatedAt        string           `json:"createdAt"`
	Tips             *PostTips        `json:"tips,omitempty"`
}

// PostsPage is one page of the authenticated user's posts.
type PostsPage struct {
	Data       []Post     `json:"data"`
	Pagination Pagination `json:"pagination"`
}

// ListPostsParams configures a page request to GET /posts. Pointers are used
// for optional fields so that "unset" is distinguishable from "zero".
type ListPostsParams struct {
	Page *int
	Size *int
}

// ListPosts returns one page of the authenticated user's posts.
//
// GET /posts — scope: read:post.
func (c *Client) ListPosts(ctx context.Context, p ListPostsParams) (*PostsPage, error) {
	query := encodeQuery(map[string]any{"page": p.Page, "size": p.Size})
	out := &PostsPage{}
	if err := c.doJSON(ctx, http.MethodGet, "/posts", query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetPost fetches a single post owned by the authenticated user by UUID.
//
// GET /posts/{uuid} — scope: read:post.
func (c *Client) GetPost(ctx context.Context, postUUID string) (*Post, error) {
	if postUUID == "" {
		return nil, errors.New("fanvue: GetPost requires a non-empty post UUID")
	}
	path := "/posts/" + url.PathEscape(postUUID)
	out := &Post{}
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreatePostParams is the request body for publishing a post. Audience is
// required; all other fields are optional and omitted when unset.
type CreatePostParams struct {
	Audience         PostAudience `json:"audience"`
	Text             *string      `json:"text,omitempty"`
	MediaUUIDs       []string     `json:"mediaUuids,omitempty"`
	MediaPreviewUUID *string      `json:"mediaPreviewUuid,omitempty"`
	CollectionUUIDs  []string     `json:"collectionUuids,omitempty"`
	Price            *float64     `json:"price,omitempty"`
	ExpiresAt        *string      `json:"expiresAt,omitempty"`
	PublishAt        *string      `json:"publishAt,omitempty"`
}

// CreatePost publishes a new post on behalf of the authenticated user. This is
// the primary publish operation the Inoue AI platform drives.
//
// POST /posts — scope: write:post.
func (c *Client) CreatePost(ctx context.Context, p CreatePostParams) (*Post, error) {
	if p.Audience == "" {
		return nil, errors.New("fanvue: CreatePost requires an Audience")
	}
	switch p.Audience {
	case AudienceSubscribers, AudienceFollowersAndSubscribers:
	default:
		return nil, fmt.Errorf("fanvue: CreatePost invalid Audience %q", p.Audience)
	}
	out := &Post{}
	if err := c.doJSON(ctx, http.MethodPost, "/posts", nil, p, out); err != nil {
		return nil, err
	}
	return out, nil
}
