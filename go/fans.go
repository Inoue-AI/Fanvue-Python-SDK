package fanvue

import (
	"context"
	"net/http"
)

// Fan is a follower or subscriber of the authenticated user. Nullable string
// fields use pointers so a JSON null is distinguishable from an empty string.
type Fan struct {
	UUID         string  `json:"uuid"`
	Handle       string  `json:"handle"`
	DisplayName  string  `json:"displayName"`
	Nickname     *string `json:"nickname"`
	IsTopSpender bool    `json:"isTopSpender"`
	AvatarURL    *string `json:"avatarUrl"`
	RegisteredAt string  `json:"registeredAt"`
}

// FansPage is one page of followers or subscribers.
type FansPage struct {
	Data       []Fan      `json:"data"`
	Pagination Pagination `json:"pagination"`
}

// ListFansParams configures a page request to GET /subscribers or
// GET /followers.
type ListFansParams struct {
	Page *int
	Size *int
}

// ListSubscribers returns one page of users subscribed to the authenticated
// user.
//
// GET /subscribers — scope: read:fan.
func (c *Client) ListSubscribers(ctx context.Context, p ListFansParams) (*FansPage, error) {
	return c.listFans(ctx, "/subscribers", p)
}

// ListFollowers returns one page of users following the authenticated user
// (excluding active subscribers).
//
// GET /followers — scope: read:fan.
func (c *Client) ListFollowers(ctx context.Context, p ListFansParams) (*FansPage, error) {
	return c.listFans(ctx, "/followers", p)
}

func (c *Client) listFans(ctx context.Context, path string, p ListFansParams) (*FansPage, error) {
	query := encodeQuery(map[string]any{"page": p.Page, "size": p.Size})
	out := &FansPage{}
	if err := c.doJSON(ctx, http.MethodGet, path, query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}
