package fanvue

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

// CreatorPostsPage is one page of a managed creator's posts, returned by
// GET /creators/{creatorUserUuid}/posts. Each entry uses the same Post shape as
// the self-scoped /posts endpoints, so consumers can treat creator-scoped and
// self-scoped posts uniformly.
type CreatorPostsPage struct {
	Data       []Post     `json:"data"`
	Pagination Pagination `json:"pagination"`
}

// CreatorPostCreated is the response from POST /creators/{creatorUserUuid}/posts.
// The create endpoint returns a slimmer projection than the list/update/pin
// endpoints: it omits the collections, comment/like counts, pin state, and tip
// totals that are only populated once the post has accrued engagement. Nullable
// fields use pointers so a JSON null is distinguishable from a zero value.
type CreatorPostCreated struct {
	UUID             string       `json:"uuid"`
	Audience         PostAudience `json:"audience"`
	Text             *string      `json:"text"`
	MediaPreviewUUID *string      `json:"mediaPreviewUuid"`
	Price            *float64     `json:"price"`
	ExpiresAt        *string      `json:"expiresAt"`
	PublishAt        *string      `json:"publishAt"`
	PublishedAt      *string      `json:"publishedAt"`
	CreatedAt        string       `json:"createdAt"`
}

// GetCreatorAccount fetches the account view for a managed creator: profile,
// status, earnings totals, last payout, and fan counts. The response shape is
// identical to the self-scoped GET /users/account view, so it reuses Account.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// GET /creators/{creatorUserUuid}/account — scopes: read:creator.
func (c *Client) GetCreatorAccount(
	ctx context.Context, creatorUserUUID string,
) (*Account, error) {
	if creatorUserUUID == "" {
		return nil, errors.New("fanvue: GetCreatorAccount requires a non-empty creator user UUID")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) + "/account"
	out := &Account{}
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListCreatorPostsParams configures a page request to
// GET /creators/{creatorUserUuid}/posts. Pointers are used for optional fields
// so that "unset" is distinguishable from "zero". IncludeUnpublished, when
// non-nil, includes scheduled/unpublished posts in the result; it is sent as the
// Literal "true"/"false" string the API expects.
type ListCreatorPostsParams struct {
	Page               *int
	Size               *int
	IncludeUnpublished *bool
}

// GetCreatorPosts returns one page of a managed creator's posts.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// GET /creators/{creatorUserUuid}/posts — scopes: read:post, read:creator.
func (c *Client) GetCreatorPosts(
	ctx context.Context, creatorUserUUID string, p ListCreatorPostsParams,
) (*CreatorPostsPage, error) {
	if creatorUserUUID == "" {
		return nil, errors.New("fanvue: GetCreatorPosts requires a non-empty creator user UUID")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) + "/posts"
	query := encodeQuery(map[string]any{
		"page":               p.Page,
		"size":               p.Size,
		"includeUnpublished": boolToLiteralString(p.IncludeUnpublished),
	})
	out := &CreatorPostsPage{}
	if err := c.doJSON(ctx, http.MethodGet, path, query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateCreatorPost publishes a new post on behalf of a managed creator. The
// request body mirrors the self-scoped CreatePost exactly: Audience is required
// and validated client-side; all other fields are optional and omitted when
// unset.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// POST /creators/{creatorUserUuid}/posts — scopes: write:post, write:creator.
func (c *Client) CreateCreatorPost(
	ctx context.Context, creatorUserUUID string, p CreatePostParams,
) (*CreatorPostCreated, error) {
	if creatorUserUUID == "" {
		return nil, errors.New("fanvue: CreateCreatorPost requires a non-empty creator user UUID")
	}
	if p.Audience == "" {
		return nil, errors.New("fanvue: CreateCreatorPost requires an Audience")
	}
	switch p.Audience {
	case AudienceSubscribers, AudienceFollowersAndSubscribers:
	default:
		return nil, fmt.Errorf("fanvue: CreateCreatorPost invalid Audience %q", p.Audience)
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) + "/posts"
	out := &CreatorPostCreated{}
	if err := c.doJSON(ctx, http.MethodPost, path, nil, p, out); err != nil {
		return nil, err
	}
	return out, nil
}

// UpdateCreatorPost updates an existing post owned by a managed creator. The
// request body mirrors the self-scoped UpdatePost exactly: every field is
// optional, a nil pointer (or nil slice) is omitted so the field is left
// unchanged, and several fields support an explicit JSON null to CLEAR the value
// server-side (text, price, expiresAt, mediaPreviewUuid, collectionUuids; and
// publishAt=null publishes a scheduled post immediately). Callers needing the
// "set to null" semantics must use UpdatePostParams.RawBody, which is sent
// verbatim and takes precedence over every typed field.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// PATCH /creators/{creatorUserUuid}/posts/{uuid} — scopes: write:post, write:creator.
func (c *Client) UpdateCreatorPost(
	ctx context.Context, creatorUserUUID, postUUID string, p UpdatePostParams,
) (*Post, error) {
	if creatorUserUUID == "" {
		return nil, errors.New("fanvue: UpdateCreatorPost requires a non-empty creator user UUID")
	}
	if postUUID == "" {
		return nil, errors.New("fanvue: UpdateCreatorPost requires a non-empty post UUID")
	}
	if p.Audience != nil {
		switch *p.Audience {
		case AudienceSubscribers, AudienceFollowersAndSubscribers:
		default:
			return nil, fmt.Errorf("fanvue: UpdateCreatorPost invalid Audience %q", *p.Audience)
		}
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) +
		"/posts/" + url.PathEscape(postUUID)
	out := &Post{}
	if err := c.doJSON(ctx, http.MethodPatch, path, nil, postUpdateBody(p), out); err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteCreatorPost soft-deletes a post owned by a managed creator. The endpoint
// returns no body.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// DELETE /creators/{creatorUserUuid}/posts/{uuid} — scopes: write:post, write:creator.
func (c *Client) DeleteCreatorPost(
	ctx context.Context, creatorUserUUID, postUUID string,
) error {
	if creatorUserUUID == "" {
		return errors.New("fanvue: DeleteCreatorPost requires a non-empty creator user UUID")
	}
	if postUUID == "" {
		return errors.New("fanvue: DeleteCreatorPost requires a non-empty post UUID")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) +
		"/posts/" + url.PathEscape(postUUID)
	return c.doJSON(ctx, http.MethodDelete, path, nil, nil, nil)
}

// PinCreatorPost pins a managed creator's post so it appears at the top of their
// feed. Returns the updated post.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// POST /creators/{creatorUserUuid}/posts/{uuid}/pin — scopes: write:post, write:creator.
func (c *Client) PinCreatorPost(
	ctx context.Context, creatorUserUUID, postUUID string,
) (*Post, error) {
	if creatorUserUUID == "" {
		return nil, errors.New("fanvue: PinCreatorPost requires a non-empty creator user UUID")
	}
	if postUUID == "" {
		return nil, errors.New("fanvue: PinCreatorPost requires a non-empty post UUID")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) +
		"/posts/" + url.PathEscape(postUUID) + "/pin"
	out := &Post{}
	if err := c.doJSON(ctx, http.MethodPost, path, nil, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// UnpinCreatorPost unpins a managed creator's post so it returns to normal feed
// ordering. Returns the updated post.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// DELETE /creators/{creatorUserUuid}/posts/{uuid}/pin — scopes: write:post, write:creator.
func (c *Client) UnpinCreatorPost(
	ctx context.Context, creatorUserUUID, postUUID string,
) (*Post, error) {
	if creatorUserUUID == "" {
		return nil, errors.New("fanvue: UnpinCreatorPost requires a non-empty creator user UUID")
	}
	if postUUID == "" {
		return nil, errors.New("fanvue: UnpinCreatorPost requires a non-empty post UUID")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) +
		"/posts/" + url.PathEscape(postUUID) + "/pin"
	out := &Post{}
	if err := c.doJSON(ctx, http.MethodDelete, path, nil, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// RepostCreatorPost re-surfaces a managed creator's existing post by updating its
// publication date to now, moving it back to the top of the feed. Returns the
// updated post.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// POST /creators/{creatorUserUuid}/posts/{uuid}/repost — scopes: write:post, write:creator.
func (c *Client) RepostCreatorPost(
	ctx context.Context, creatorUserUUID, postUUID string,
) (*Post, error) {
	if creatorUserUUID == "" {
		return nil, errors.New("fanvue: RepostCreatorPost requires a non-empty creator user UUID")
	}
	if postUUID == "" {
		return nil, errors.New("fanvue: RepostCreatorPost requires a non-empty post UUID")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) +
		"/posts/" + url.PathEscape(postUUID) + "/repost"
	out := &Post{}
	if err := c.doJSON(ctx, http.MethodPost, path, nil, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetCreatorPostCommentsParams configures a page request to
// GET /creators/{creatorUserUuid}/posts/{uuid}/comments.
type GetCreatorPostCommentsParams struct {
	Page *int
	Size *int
}

// GetCreatorPostComments returns one page of comments on a managed creator's
// post. The entry shape is identical to the self-scoped post comments, so it
// reuses PostCommentsPage.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// GET /creators/{creatorUserUuid}/posts/{uuid}/comments — scopes: read:post, read:creator.
func (c *Client) GetCreatorPostComments(
	ctx context.Context, creatorUserUUID, postUUID string, p GetCreatorPostCommentsParams,
) (*PostCommentsPage, error) {
	if creatorUserUUID == "" {
		return nil, errors.New("fanvue: GetCreatorPostComments requires a non-empty creator user UUID")
	}
	if postUUID == "" {
		return nil, errors.New("fanvue: GetCreatorPostComments requires a non-empty post UUID")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) +
		"/posts/" + url.PathEscape(postUUID) + "/comments"
	query := encodeQuery(map[string]any{"page": p.Page, "size": p.Size})
	out := &PostCommentsPage{}
	if err := c.doJSON(ctx, http.MethodGet, path, query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateCreatorPostComment creates a comment on a managed creator's post. Text
// is required. Returns the created comment.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// POST /creators/{creatorUserUuid}/posts/{uuid}/comments — scopes: write:post, write:creator.
func (c *Client) CreateCreatorPostComment(
	ctx context.Context, creatorUserUUID, postUUID string, p CreatePostCommentParams,
) (*PostComment, error) {
	if creatorUserUUID == "" {
		return nil, errors.New("fanvue: CreateCreatorPostComment requires a non-empty creator user UUID")
	}
	if postUUID == "" {
		return nil, errors.New("fanvue: CreateCreatorPostComment requires a non-empty post UUID")
	}
	if p.Text == "" {
		return nil, errors.New("fanvue: CreateCreatorPostComment requires non-empty Text")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) +
		"/posts/" + url.PathEscape(postUUID) + "/comments"
	out := &PostComment{}
	if err := c.doJSON(ctx, http.MethodPost, path, nil, p, out); err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteCreatorPostComment deletes a comment from a managed creator's post. The
// endpoint returns no body.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// DELETE /creators/{creatorUserUuid}/posts/{uuid}/comments/{commentUuid} —
// scopes: write:post, write:creator.
func (c *Client) DeleteCreatorPostComment(
	ctx context.Context, creatorUserUUID, postUUID, commentUUID string,
) error {
	if creatorUserUUID == "" {
		return errors.New("fanvue: DeleteCreatorPostComment requires a non-empty creator user UUID")
	}
	if postUUID == "" {
		return errors.New("fanvue: DeleteCreatorPostComment requires a non-empty post UUID")
	}
	if commentUUID == "" {
		return errors.New("fanvue: DeleteCreatorPostComment requires a non-empty comment UUID")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) +
		"/posts/" + url.PathEscape(postUUID) +
		"/comments/" + url.PathEscape(commentUUID)
	return c.doJSON(ctx, http.MethodDelete, path, nil, nil, nil)
}
