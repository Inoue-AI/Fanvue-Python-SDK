package fanvue

import (
	"context"
	"net/http"
)

// FanCounts holds the follower/subscriber tallies for the authenticated user.
type FanCounts struct {
	FollowersCount   float64 `json:"followersCount"`
	SubscribersCount float64 `json:"subscribersCount"`
}

// ContentCounts holds the per-type content tallies for the authenticated user.
type ContentCounts struct {
	ImageCount         float64 `json:"imageCount"`
	VideoCount         float64 `json:"videoCount"`
	AudioCount         float64 `json:"audioCount"`
	PostCount          float64 `json:"postCount"`
	PayToViewPostCount float64 `json:"payToViewPostCount"`
}

// CurrentUser is the authenticated user's profile, as returned by
// GET /users/me. Nullable string fields use pointers so a JSON null is
// distinguishable from an empty string.
type CurrentUser struct {
	UUID          string         `json:"uuid"`
	Email         string         `json:"email"`
	Handle        string         `json:"handle"`
	Bio           string         `json:"bio"`
	DisplayName   string         `json:"displayName"`
	IsCreator     bool           `json:"isCreator"`
	CreatedAt     string         `json:"createdAt"`
	UpdatedAt     *string        `json:"updatedAt"`
	AvatarURL     *string        `json:"avatarUrl"`
	BannerURL     *string        `json:"bannerUrl"`
	LikesCount    *float64       `json:"likesCount,omitempty"`
	FanCounts     *FanCounts     `json:"fanCounts,omitempty"`
	ContentCounts *ContentCounts `json:"contentCounts,omitempty"`
}

// GetCurrentUser fetches the authenticated user's profile.
//
// GET /users/me — scope: read:self.
func (c *Client) GetCurrentUser(ctx context.Context) (*CurrentUser, error) {
	out := &CurrentUser{}
	if err := c.doJSON(ctx, http.MethodGet, "/users/me", nil, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}
