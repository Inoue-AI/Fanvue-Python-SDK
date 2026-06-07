package fanvue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

// RawBody is a pre-serialized JSON request body sent to the API verbatim. It
// exists so callers can express request shapes the typed params cannot — most
// importantly explicit JSON nulls (e.g. {"text":null}) that clear a field
// server-side, matching the Python SDK's opaque Mapping[str, Any] body. Marshal
// of a RawBody emits its bytes unchanged.
type RawBody = json.RawMessage

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

// UpdatePostParams is the PATCH /posts/{uuid} request body. Every field is
// optional: a nil pointer (or nil slice) is omitted so the field is left
// unchanged on the server.
//
// Several fields support an explicit JSON null to CLEAR the value
// server-side (text, price, expiresAt, mediaPreviewUuid, collectionUuids; and
// publishAt=null publishes a scheduled post immediately). A nil Go pointer
// cannot express that distinction, so callers needing the "set to null"
// semantics must use RawBody, which mirrors the Python SDK's opaque-body
// signature exactly. When RawBody is set it is sent verbatim and all typed
// fields are ignored.
type UpdatePostParams struct {
	Audience         *PostAudience `json:"audience,omitempty"`
	Text             *string       `json:"text,omitempty"`
	MediaUUIDs       []string      `json:"mediaUuids,omitempty"`
	MediaPreviewUUID *string       `json:"mediaPreviewUuid,omitempty"`
	CollectionUUIDs  []string      `json:"collectionUuids,omitempty"`
	Price            *float64      `json:"price,omitempty"`
	ExpiresAt        *string       `json:"expiresAt,omitempty"`
	PublishAt        *string       `json:"publishAt,omitempty"`

	// RawBody, when non-nil, is sent as the request body verbatim and takes
	// precedence over every typed field above. Use it to express explicit JSON
	// nulls (e.g. {"text":null}) that clear a field server-side, matching the
	// Fanvue "set to null to remove" semantics that nil pointers cannot encode.
	RawBody RawBody `json:"-"`
}

// UpdatePost updates an existing post owned by the authenticated user. Only the
// post owner can edit. All provided fields replace the corresponding values; a
// scheduled post with publishAt set to null is published immediately.
//
// PATCH /posts/{uuid} — scope: write:post.
func (c *Client) UpdatePost(
	ctx context.Context, postUUID string, p UpdatePostParams,
) (*Post, error) {
	if postUUID == "" {
		return nil, errors.New("fanvue: UpdatePost requires a non-empty post UUID")
	}
	if p.Audience != nil {
		switch *p.Audience {
		case AudienceSubscribers, AudienceFollowersAndSubscribers:
		default:
			return nil, fmt.Errorf("fanvue: UpdatePost invalid Audience %q", *p.Audience)
		}
	}
	path := "/posts/" + url.PathEscape(postUUID)
	out := &Post{}
	if err := c.doJSON(ctx, http.MethodPatch, path, nil, postUpdateBody(p), out); err != nil {
		return nil, err
	}
	return out, nil
}

// DeletePost soft-deletes a post owned by the authenticated user. Only the post
// owner can delete their own post. The endpoint returns no body.
//
// DELETE /posts/{uuid} — scope: write:post.
func (c *Client) DeletePost(ctx context.Context, postUUID string) error {
	if postUUID == "" {
		return errors.New("fanvue: DeletePost requires a non-empty post UUID")
	}
	path := "/posts/" + url.PathEscape(postUUID)
	return c.doJSON(ctx, http.MethodDelete, path, nil, nil, nil)
}

// PinPost pins a post so it appears at the top of the authenticated user's
// feed. Only the post owner can pin their own content. Returns the updated post.
//
// POST /posts/{uuid}/pin — scope: write:post.
func (c *Client) PinPost(ctx context.Context, postUUID string) (*Post, error) {
	if postUUID == "" {
		return nil, errors.New("fanvue: PinPost requires a non-empty post UUID")
	}
	path := "/posts/" + url.PathEscape(postUUID) + "/pin"
	out := &Post{}
	if err := c.doJSON(ctx, http.MethodPost, path, nil, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// UnpinPost unpins a post so it returns to normal feed ordering. Only the post
// owner can unpin their own content. Returns the updated post.
//
// DELETE /posts/{uuid}/pin — scope: write:post.
func (c *Client) UnpinPost(ctx context.Context, postUUID string) (*Post, error) {
	if postUUID == "" {
		return nil, errors.New("fanvue: UnpinPost requires a non-empty post UUID")
	}
	path := "/posts/" + url.PathEscape(postUUID) + "/pin"
	out := &Post{}
	if err := c.doJSON(ctx, http.MethodDelete, path, nil, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// RepostPost re-surfaces an existing post by updating its publication date to
// now, moving it back to the top of the feed. Only the post owner can repost
// their own content. Returns the updated post.
//
// POST /posts/{uuid}/repost — scope: write:post.
func (c *Client) RepostPost(ctx context.Context, postUUID string) (*Post, error) {
	if postUUID == "" {
		return nil, errors.New("fanvue: RepostPost requires a non-empty post UUID")
	}
	path := "/posts/" + url.PathEscape(postUUID) + "/repost"
	out := &Post{}
	if err := c.doJSON(ctx, http.MethodPost, path, nil, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// PostComment is a single comment on a post.
type PostComment struct {
	UUID      string  `json:"uuid"`
	Text      string  `json:"text"`
	CreatedAt string  `json:"createdAt"`
	UpdatedAt *string `json:"updatedAt"`
}

// PostCommentUser is the author of a post comment. Returned by
// GET /posts/{uuid}/comments.
type PostCommentUser struct {
	UUID         string  `json:"uuid"`
	Handle       string  `json:"handle"`
	DisplayName  string  `json:"displayName"`
	Nickname     *string `json:"nickname"`
	IsTopSpender bool    `json:"isTopSpender"`
}

// PostCommentEntry is one entry in a page of post comments, pairing the comment
// with its author. The author is nil when the account is no longer resolvable.
type PostCommentEntry struct {
	UUID      string           `json:"uuid"`
	Text      string           `json:"text"`
	CreatedAt string           `json:"createdAt"`
	UpdatedAt *string          `json:"updatedAt"`
	User      *PostCommentUser `json:"user"`
}

// PostCommentsPage is one page of comments on a post.
type PostCommentsPage struct {
	Data       []PostCommentEntry `json:"data"`
	Pagination Pagination         `json:"pagination"`
}

// CreatePostCommentParams is the POST /posts/{uuid}/comments request body. Text
// is required.
type CreatePostCommentParams struct {
	Text string `json:"text"`
}

// CreatePostComment creates a comment on a post. Only the post owner can create
// comments via this endpoint. Returns the created comment.
//
// POST /posts/{uuid}/comments — scope: write:post.
func (c *Client) CreatePostComment(
	ctx context.Context, postUUID string, p CreatePostCommentParams,
) (*PostComment, error) {
	if postUUID == "" {
		return nil, errors.New("fanvue: CreatePostComment requires a non-empty post UUID")
	}
	if p.Text == "" {
		return nil, errors.New("fanvue: CreatePostComment requires non-empty Text")
	}
	path := "/posts/" + url.PathEscape(postUUID) + "/comments"
	out := &PostComment{}
	if err := c.doJSON(ctx, http.MethodPost, path, nil, p, out); err != nil {
		return nil, err
	}
	return out, nil
}

// DeletePostComment deletes a comment from a post. The comment can be deleted
// by either the comment author or the post owner. The endpoint returns no body.
//
// DELETE /posts/{uuid}/comments/{commentUuid} — scope: write:post.
func (c *Client) DeletePostComment(
	ctx context.Context, postUUID, commentUUID string,
) error {
	if postUUID == "" {
		return errors.New("fanvue: DeletePostComment requires a non-empty post UUID")
	}
	if commentUUID == "" {
		return errors.New("fanvue: DeletePostComment requires a non-empty comment UUID")
	}
	path := "/posts/" + url.PathEscape(postUUID) +
		"/comments/" + url.PathEscape(commentUUID)
	return c.doJSON(ctx, http.MethodDelete, path, nil, nil, nil)
}

// GetPostCommentsParams configures a page request to
// GET /posts/{uuid}/comments.
type GetPostCommentsParams struct {
	Page *int
	Size *int
}

// GetPostComments returns one page of comments on a post. Only the post owner
// can view comments.
//
// GET /posts/{uuid}/comments — scope: read:post.
func (c *Client) GetPostComments(
	ctx context.Context, postUUID string, p GetPostCommentsParams,
) (*PostCommentsPage, error) {
	if postUUID == "" {
		return nil, errors.New("fanvue: GetPostComments requires a non-empty post UUID")
	}
	path := "/posts/" + url.PathEscape(postUUID) + "/comments"
	query := encodeQuery(map[string]any{"page": p.Page, "size": p.Size})
	out := &PostCommentsPage{}
	if err := c.doJSON(ctx, http.MethodGet, path, query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// PostLikeUser is the user who liked a post. Returned by
// GET /posts/{uuid}/likes.
type PostLikeUser struct {
	UUID         string  `json:"uuid"`
	Handle       string  `json:"handle"`
	DisplayName  string  `json:"displayName"`
	Nickname     *string `json:"nickname"`
	AvatarURL    *string `json:"avatarUrl"`
	IsTopSpender bool    `json:"isTopSpender"`
	RegisteredAt string  `json:"registeredAt"`
}

// PostLikeEntry is one like on a post. The user is nil when the account is no
// longer resolvable.
type PostLikeEntry struct {
	CreatedAt string        `json:"createdAt"`
	User      *PostLikeUser `json:"user"`
}

// PostLikesPage is one page of likes on a post.
type PostLikesPage struct {
	Data       []PostLikeEntry `json:"data"`
	Pagination Pagination      `json:"pagination"`
}

// GetPostLikesParams configures a page request to GET /posts/{uuid}/likes.
type GetPostLikesParams struct {
	Page *int
	Size *int
}

// GetPostLikes returns one page of likes received on a post.
//
// GET /posts/{uuid}/likes — scope: read:post.
func (c *Client) GetPostLikes(
	ctx context.Context, postUUID string, p GetPostLikesParams,
) (*PostLikesPage, error) {
	if postUUID == "" {
		return nil, errors.New("fanvue: GetPostLikes requires a non-empty post UUID")
	}
	path := "/posts/" + url.PathEscape(postUUID) + "/likes"
	query := encodeQuery(map[string]any{"page": p.Page, "size": p.Size})
	out := &PostLikesPage{}
	if err := c.doJSON(ctx, http.MethodGet, path, query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// PostTipUser is the user who tipped a post. Returned by
// GET /posts/{uuid}/tips.
type PostTipUser struct {
	UUID         string  `json:"uuid"`
	Handle       string  `json:"handle"`
	DisplayName  string  `json:"displayName"`
	Nickname     *string `json:"nickname"`
	AvatarURL    *string `json:"avatarUrl"`
	IsTopSpender bool    `json:"isTopSpender"`
	RegisteredAt string  `json:"registeredAt"`
}

// PostTipEntry is one tip on a post (amounts in cents). The user is nil when the
// account is no longer resolvable; CreatedAt is nil for legacy tips.
type PostTipEntry struct {
	CreatedAt *string      `json:"createdAt"`
	Gross     float64      `json:"gross"`
	Net       float64      `json:"net"`
	User      *PostTipUser `json:"user"`
}

// PostTipsPage is one page of tips on a post.
type PostTipsPage struct {
	Data       []PostTipEntry `json:"data"`
	Pagination Pagination     `json:"pagination"`
}

// GetPostTipsParams configures a page request to GET /posts/{uuid}/tips.
type GetPostTipsParams struct {
	Page *int
	Size *int
}

// GetPostTips returns one page of tips received on a post. Only the post owner
// can view tips.
//
// GET /posts/{uuid}/tips — scope: read:post.
func (c *Client) GetPostTips(
	ctx context.Context, postUUID string, p GetPostTipsParams,
) (*PostTipsPage, error) {
	if postUUID == "" {
		return nil, errors.New("fanvue: GetPostTips requires a non-empty post UUID")
	}
	path := "/posts/" + url.PathEscape(postUUID) + "/tips"
	query := encodeQuery(map[string]any{"page": p.Page, "size": p.Size})
	out := &PostTipsPage{}
	if err := c.doJSON(ctx, http.MethodGet, path, query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// postUpdateBody resolves the body sent for an UpdatePost request. When RawBody
// is supplied it is sent verbatim (preserving explicit JSON nulls); otherwise
// the typed params are marshalled normally.
func postUpdateBody(p UpdatePostParams) any {
	if p.RawBody != nil {
		return p.RawBody
	}
	return p
}
