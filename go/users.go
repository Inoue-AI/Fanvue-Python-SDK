package fanvue

import (
	"context"
	"errors"
	"net/http"
	"net/url"
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

// AccountEarnings holds the authenticated user's lifetime and available
// earnings, plus the timestamp of their most recent payout. LastPayoutAt is a
// pointer so a JSON null (no payouts yet) is distinguishable from a value.
type AccountEarnings struct {
	AvailableBalance float64 `json:"availableBalance"`
	LastPayoutAt     *string `json:"lastPayoutAt"`
	Total            float64 `json:"total"`
}

// AccountFans holds the authenticated user's follower and subscriber tallies as
// surfaced on the account view.
type AccountFans struct {
	Followers   float64 `json:"followers"`
	Subscribers float64 `json:"subscribers"`
}

// AccountSummary is the account-status block of GET /users/account: earnings,
// fan counts, and the account standing.
type AccountSummary struct {
	Earnings AccountEarnings `json:"earnings"`
	Fans     AccountFans     `json:"fans"`
	// Status is one of "active" or "suspended".
	Status string `json:"status"`
}

// Account is the authenticated user's account view, as returned by
// GET /users/account: profile, status, earnings totals, last payout, and fan
// counts. Nullable string fields use pointers so a JSON null is distinguishable
// from an empty string.
type Account struct {
	Account     AccountSummary `json:"account"`
	AvatarURL   *string        `json:"avatarUrl"`
	BannerURL   *string        `json:"bannerUrl"`
	Bio         string         `json:"bio"`
	CreatedAt   string         `json:"createdAt"`
	DisplayName string         `json:"displayName"`
	Email       string         `json:"email"`
	Handle      string         `json:"handle"`
	IsCreator   bool           `json:"isCreator"`
	UpdatedAt   *string        `json:"updatedAt"`
	UUID        string         `json:"uuid"`
}

// GetAccount fetches the authenticated user's account view: profile, status,
// earnings totals, last payout, and fan counts.
//
// GET /users/account — scope: read:self.
func (c *Client) GetAccount(ctx context.Context) (*Account, error) {
	out := &Account{}
	if err := c.doJSON(ctx, http.MethodGet, "/users/account", nil, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// AppManagedCreatorSubscription is the per-creator app-subscription state
// returned to agency team members in GetAppCurrentUserSubscription. For
// non-agency users the managedCreators array is empty. Nullable string fields
// use pointers so a JSON null is distinguishable from an empty string.
type AppManagedCreatorSubscription struct {
	CancelAtPeriodEnd     bool    `json:"cancelAtPeriodEnd"`
	CurrentPeriodEnd      *string `json:"currentPeriodEnd"`
	HasActiveSubscription bool    `json:"hasActiveSubscription"`
	PlanName              *string `json:"planName"`
	PlanUUID              *string `json:"planUuid"`
	// Status is one of "active", "pending", "cancelled", or "none".
	Status   string `json:"status"`
	UserUUID string `json:"userUuid"`
}

// AppCurrentUserSubscription is the authenticated user's paid entitlement for a
// given installed app, as returned by GET /apps/{appUuid}/subscription/me. When
// the caller is an agency team member, ManagedCreators carries the subscription
// state for every creator they are assigned to; otherwise it is empty. Nullable
// string fields use pointers so a JSON null is distinguishable from an empty
// string.
type AppCurrentUserSubscription struct {
	AppUUID               string                          `json:"appUuid"`
	CancelAtPeriodEnd     bool                            `json:"cancelAtPeriodEnd"`
	CurrentPeriodEnd      *string                         `json:"currentPeriodEnd"`
	HasActiveSubscription bool                            `json:"hasActiveSubscription"`
	ManagedCreators       []AppManagedCreatorSubscription `json:"managedCreators"`
	PlanName              *string                         `json:"planName"`
	PlanUUID              *string                         `json:"planUuid"`
	// Status is one of "active", "pending", "cancelled", or "none".
	Status   string `json:"status"`
	UserUUID string `json:"userUuid"`
}

// GetAppCurrentUserSubscription returns the authenticated user's paid
// entitlement for the app identified by appUUID. The OAuth access token must
// belong to that app.
//
// GET /apps/{appUuid}/subscription/me — scope: read:self.
func (c *Client) GetAppCurrentUserSubscription(
	ctx context.Context, appUUID string,
) (*AppCurrentUserSubscription, error) {
	if appUUID == "" {
		return nil, errors.New("fanvue: GetAppCurrentUserSubscription requires a non-empty app UUID")
	}
	path := "/apps/" + url.PathEscape(appUUID) + "/subscription/me"
	out := &AppCurrentUserSubscription{}
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// AppPricingPlan is one pricing plan within an app's subscription-status view.
// Interval is a pointer so a JSON null (e.g. for one-time or free plans) is
// distinguishable from a value.
type AppPricingPlan struct {
	// BillingType is one of "free", "one_time", or "recurring".
	BillingType  string `json:"billingType"`
	CurrencyCode string `json:"currencyCode"`
	// Interval is one of "monthly" or "yearly" when present, otherwise nil.
	Interval *string `json:"interval"`
	Name     string  `json:"name"`
	Price    float64 `json:"price"`
	// Status is one of "pending_setup", "active", or "withdrawn".
	Status string `json:"status"`
	UUID   string `json:"uuid"`
}

// AppSubscriptionStatus is the pricing-plan lifecycle view for a developer-owned
// app, as returned by GET /apps/{appUuid}/subscription-status.
type AppSubscriptionStatus struct {
	AppName string `json:"appName"`
	AppUUID string `json:"appUuid"`
	// Availability is one of "complete" or "horizonUnavailable".
	Availability string `json:"availability"`
	// OverallStatus is one of "notConfigured", "pendingSetup", "active",
	// "withdrawn", "mixed", or "unavailable".
	OverallStatus string           `json:"overallStatus"`
	PricingPlans  []AppPricingPlan `json:"pricingPlans"`
}

// GetAppSubscriptionStatus returns the pricing-plan lifecycle states for the
// developer-owned app identified by appUUID.
//
// GET /apps/{appUuid}/subscription-status — scope: read:self.
func (c *Client) GetAppSubscriptionStatus(
	ctx context.Context, appUUID string,
) (*AppSubscriptionStatus, error) {
	if appUUID == "" {
		return nil, errors.New("fanvue: GetAppSubscriptionStatus requires a non-empty app UUID")
	}
	path := "/apps/" + url.PathEscape(appUUID) + "/subscription-status"
	out := &AppSubscriptionStatus{}
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}
