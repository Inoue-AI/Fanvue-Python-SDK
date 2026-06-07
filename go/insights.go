package fanvue

import (
	"context"
	"errors"
	"net/http"
	"net/url"
)

// EarningsSource enumerates the transaction sources accepted by the source
// filter on GET /insights/earnings and returned on each earnings line item.
type EarningsSource string

const (
	// EarningsSourceAll matches every earnings source (the default filter).
	EarningsSourceAll EarningsSource = "all"
	// EarningsSourceAffiliate is affiliate-program revenue.
	EarningsSourceAffiliate EarningsSource = "affiliate"
	// EarningsSourceAppStore is app-store purchase revenue.
	EarningsSourceAppStore EarningsSource = "appStore"
	// EarningsSourceCheckoutLink is revenue from a checkout link.
	EarningsSourceCheckoutLink EarningsSource = "checkoutLink"
	// EarningsSourceMediaLink is revenue from a media link.
	EarningsSourceMediaLink EarningsSource = "mediaLink"
	// EarningsSourceMessage is revenue from a paid message.
	EarningsSourceMessage EarningsSource = "message"
	// EarningsSourcePost is revenue from a pay-to-view post.
	EarningsSourcePost EarningsSource = "post"
	// EarningsSourceReferral is referral revenue.
	EarningsSourceReferral EarningsSource = "referral"
	// EarningsSourceRenewal is subscription-renewal revenue.
	EarningsSourceRenewal EarningsSource = "renewal"
	// EarningsSourceSubscription is new-subscription revenue.
	EarningsSourceSubscription EarningsSource = "subscription"
	// EarningsSourceTip is tip revenue.
	EarningsSourceTip EarningsSource = "tip"
	// EarningsSourceGiveaway is giveaway revenue.
	EarningsSourceGiveaway EarningsSource = "giveaway"
)

// SpendingSource enumerates the reversal-transaction sources accepted by the
// source filter on GET /insights/spending and returned on each line item.
type SpendingSource string

const (
	// SpendingSourceAll matches every reversal source (the default filter, valid
	// only as a request filter — line items carry "refund" or "chargeback").
	SpendingSourceAll SpendingSource = "all"
	// SpendingSourceRefund is a refunded transaction.
	SpendingSourceRefund SpendingSource = "refund"
	// SpendingSourceChargeback is a charged-back transaction.
	SpendingSourceChargeback SpendingSource = "chargeback"
)

// EarningsGranularity is the bucket size for the earnings-summary chart series.
type EarningsGranularity string

const (
	// GranularityDay buckets the earnings-summary series by day.
	GranularityDay EarningsGranularity = "day"
	// GranularityWeek buckets the earnings-summary series by week.
	GranularityWeek EarningsGranularity = "week"
)

// FanStatus is the relationship state of a fan to the authenticated creator, as
// reported by the fan-insights endpoints.
type FanStatus string

const (
	// FanStatusSubscriber is an active subscriber.
	FanStatusSubscriber FanStatus = "subscriber"
	// FanStatusExpired is a lapsed subscriber.
	FanStatusExpired FanStatus = "expired"
	// FanStatusFollower is a follower who has never subscribed.
	FanStatusFollower FanStatus = "follower"
	// FanStatusNotContactable is a fan who can no longer be contacted.
	FanStatusNotContactable FanStatus = "not_contactable"
)

// InsightsUser is the abbreviated fan profile attached to an earnings or
// spending line item. Nullable string fields use pointers so a JSON null is
// distinguishable from an empty string.
type InsightsUser struct {
	UUID         string  `json:"uuid"`
	Handle       string  `json:"handle"`
	DisplayName  string  `json:"displayName"`
	Nickname     *string `json:"nickname"`
	IsTopSpender bool    `json:"isTopSpender"`
}

// EarningsItem is a single invoice line from GET /insights/earnings (amounts in
// cents). MessageUUID/PostUUID are populated only for message/post sources;
// Currency and User are nil when unavailable.
type EarningsItem struct {
	Currency               *string        `json:"currency"`
	Date                   string         `json:"date"`
	Gross                  float64        `json:"gross"`
	MessageUUID            *string        `json:"messageUuid,omitempty"`
	Net                    float64        `json:"net"`
	PostUUID               *string        `json:"postUuid,omitempty"`
	Source                 EarningsSource `json:"source"`
	TransactionOrderID     string         `json:"transactionOrderId"`
	TransactionOrderStatus string         `json:"transactionOrderStatus"`
	User                   *InsightsUser  `json:"user"`
}

// EarningsPage is one cursor-paginated page of earnings line items. NextCursor
// is nil on the final page; pass it back as GetEarningsParams.Cursor to fetch
// the next page.
type EarningsPage struct {
	Data       []EarningsItem `json:"data"`
	NextCursor *string        `json:"nextCursor"`
}

// GetEarningsParams configures GET /insights/earnings. All fields are optional;
// nil pointers and an empty Source slice are omitted from the query string.
type GetEarningsParams struct {
	StartDate *string
	EndDate   *string
	Source    []EarningsSource
	Cursor    *string
	Size      *int
}

// GetEarnings returns cursor-paginated invoice data for the authenticated
// creator over the requested period.
//
// GET /insights/earnings — scope: read:insights.
func (c *Client) GetEarnings(ctx context.Context, p GetEarningsParams) (*EarningsPage, error) {
	query := encodeQuery(map[string]any{
		"startDate": p.StartDate,
		"endDate":   p.EndDate,
		"source":    earningsSourcesToStrings(p.Source),
		"cursor":    p.Cursor,
		"size":      p.Size,
	})
	out := &EarningsPage{}
	if err := c.doJSON(ctx, http.MethodGet, "/insights/earnings", query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// EarningsPercentile is the percentile bucket the authenticated creator falls
// into based on gross earnings over the last 30 days. Lower values rank higher
// (e.g. 0.01 means top 0.01% of earners).
type EarningsPercentile struct {
	Percentile float64 `json:"percentile"`
}

// GetEarningsPercentile returns the authenticated creator's earnings percentile.
//
// GET /insights/earnings/percentile — scope: read:insights.
func (c *Client) GetEarningsPercentile(ctx context.Context) (*EarningsPercentile, error) {
	out := &EarningsPercentile{}
	if err := c.doJSON(ctx, http.MethodGet, "/insights/earnings/percentile", nil, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GrossNet is a gross/net amount pair (in cents), reused throughout the
// earnings-summary breakdowns.
type GrossNet struct {
	Gross float64 `json:"gross"`
	Net   float64 `json:"net"`
}

// EarningsSummaryAverageByDayOfWeek holds the average earnings keyed by ISO day
// of week (1 = Monday through 7 = Sunday).
type EarningsSummaryAverageByDayOfWeek struct {
	Day1 float64 `json:"1"`
	Day2 float64 `json:"2"`
	Day3 float64 `json:"3"`
	Day4 float64 `json:"4"`
	Day5 float64 `json:"5"`
	Day6 float64 `json:"6"`
	Day7 float64 `json:"7"`
}

// EarningsSummaryBreakdownBySource is the gross/net earnings split by source
// channel.
type EarningsSummaryBreakdownBySource struct {
	Messages  GrossNet `json:"messages"`
	Other     GrossNet `json:"other"`
	Posts     GrossNet `json:"posts"`
	Referrals GrossNet `json:"referrals"`
	Renewals  GrossNet `json:"renewals"`
	Subs      GrossNet `json:"subs"`
	Tips      GrossNet `json:"tips"`
}

// EarningsSummaryEarningsByType is the gross/net earnings split by transaction
// type.
type EarningsSummaryEarningsByType struct {
	Messages GrossNet `json:"messages"`
	Renewals GrossNet `json:"renewals"`
	Subs     GrossNet `json:"subs"`
	Tips     GrossNet `json:"tips"`
}

// EarningsSummaryOverTimeItem is one point in the earnings-summary chart series.
type EarningsSummaryOverTimeItem struct {
	Gross       float64 `json:"gross"`
	Net         float64 `json:"net"`
	PeriodStart string  `json:"periodStart"`
}

// EarningsSummaryPeriod describes the resolved time window of an earnings
// summary. StartDate/EndDate are nil when the request did not bound the range.
type EarningsSummaryPeriod struct {
	EndDate     *string             `json:"endDate"`
	Granularity EarningsGranularity `json:"granularity"`
	StartDate   *string             `json:"startDate"`
	Timezone    string              `json:"timezone"`
}

// EarningsSummaryTotalsThisMonth holds this-month totals plus the
// month-over-month comparison. ChangePercentage fields are nil when the prior
// month had no earnings to compare against.
type EarningsSummaryTotalsThisMonth struct {
	Gross                 float64  `json:"gross"`
	GrossChangePercentage *float64 `json:"grossChangePercentage"`
	Net                   float64  `json:"net"`
	NetChangePercentage   *float64 `json:"netChangePercentage"`
	PreviousMonthGross    float64  `json:"previousMonthGross"`
	PreviousMonthNet      float64  `json:"previousMonthNet"`
}

// EarningsSummaryTotals holds all-time and this-month earnings totals.
type EarningsSummaryTotals struct {
	AllTime   GrossNet                       `json:"allTime"`
	ThisMonth EarningsSummaryTotalsThisMonth `json:"thisMonth"`
}

// EarningsSummary is the pre-aggregated earnings view from
// GET /insights/earnings/summary. AverageByHourOfDay is keyed by hour-of-day
// string ("0".."23") to its average earnings — a genuinely free-form numeric
// map that mirrors the Python SDK's dict[str, float].
type EarningsSummary struct {
	AverageByDayOfWeek EarningsSummaryAverageByDayOfWeek `json:"averageByDayOfWeek"`
	AverageByHourOfDay map[string]float64                `json:"averageByHourOfDay"`
	BreakdownBySource  EarningsSummaryBreakdownBySource  `json:"breakdownBySource"`
	EarningsByType     EarningsSummaryEarningsByType     `json:"earningsByType"`
	OverTime           []EarningsSummaryOverTimeItem     `json:"overTime"`
	Period             EarningsSummaryPeriod             `json:"period"`
	Totals             EarningsSummaryTotals             `json:"totals"`
}

// GetEarningsSummaryParams configures GET /insights/earnings/summary. All fields
// are optional; nil pointers are omitted from the query string.
type GetEarningsSummaryParams struct {
	StartDate   *string
	EndDate     *string
	Granularity *EarningsGranularity
	Timezone    *string
}

// GetEarningsSummary returns the pre-aggregated earnings metrics for the
// authenticated creator.
//
// GET /insights/earnings/summary — scope: read:insights.
func (c *Client) GetEarningsSummary(
	ctx context.Context, p GetEarningsSummaryParams,
) (*EarningsSummary, error) {
	query := encodeQuery(map[string]any{
		"startDate":   p.StartDate,
		"endDate":     p.EndDate,
		"granularity": granularityToString(p.Granularity),
		"timezone":    p.Timezone,
	})
	out := &EarningsSummary{}
	if err := c.doJSON(ctx, http.MethodGet, "/insights/earnings/summary", query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// SpendingItem is a single reversal-invoice line from GET /insights/spending
// (amounts in cents). Source is "refund" or "chargeback".
type SpendingItem struct {
	Currency    *string        `json:"currency"`
	Date        string         `json:"date"`
	Gross       float64        `json:"gross"`
	MessageUUID *string        `json:"messageUuid,omitempty"`
	Net         float64        `json:"net"`
	PostUUID    *string        `json:"postUuid,omitempty"`
	Source      SpendingSource `json:"source"`
	User        *InsightsUser  `json:"user"`
}

// SpendingPage is one cursor-paginated page of reversal line items. NextCursor
// is nil on the final page; pass it back as GetSpendingParams.Cursor to fetch
// the next page.
type SpendingPage struct {
	Data       []SpendingItem `json:"data"`
	NextCursor *string        `json:"nextCursor"`
}

// GetSpendingParams configures GET /insights/spending. All fields are optional;
// nil pointers and an empty Source slice are omitted from the query string.
type GetSpendingParams struct {
	StartDate *string
	EndDate   *string
	Source    []SpendingSource
	Cursor    *string
	Size      *int
}

// GetSpending returns cursor-paginated reversal (refund/chargeback) data for the
// authenticated creator over the requested period.
//
// GET /insights/spending — scope: read:insights.
func (c *Client) GetSpending(ctx context.Context, p GetSpendingParams) (*SpendingPage, error) {
	query := encodeQuery(map[string]any{
		"startDate": p.StartDate,
		"endDate":   p.EndDate,
		"source":    spendingSourcesToStrings(p.Source),
		"cursor":    p.Cursor,
		"size":      p.Size,
	})
	out := &SpendingPage{}
	if err := c.doJSON(ctx, http.MethodGet, "/insights/spending", query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// SubscribersItem is one daily subscriber-event bucket. Total is the cumulative
// net change (new - cancelled) from the start of the requested range.
type SubscribersItem struct {
	CancelledSubscribersCount float64 `json:"cancelledSubscribersCount"`
	Date                      string  `json:"date"`
	NewSubscribersCount       float64 `json:"newSubscribersCount"`
	Total                     float64 `json:"total"`
}

// SubscribersPage is one cursor-paginated page of subscriber-event buckets.
// NextCursor is nil on the final page; pass it back as GetSubscribersParams.Cursor
// to fetch the next page.
type SubscribersPage struct {
	Data       []SubscribersItem `json:"data"`
	NextCursor *string           `json:"nextCursor"`
}

// GetSubscribersParams configures GET /insights/subscribers. All fields are
// optional; nil pointers are omitted from the query string.
type GetSubscribersParams struct {
	StartDate *string
	EndDate   *string
	Cursor    *string
	Size      *int
}

// GetSubscribers returns cursor-paginated subscriber-event time-series data for
// the authenticated creator over the requested period.
//
// GET /insights/subscribers — scope: read:insights.
func (c *Client) GetSubscribers(
	ctx context.Context, p GetSubscribersParams,
) (*SubscribersPage, error) {
	query := encodeQuery(map[string]any{
		"startDate": p.StartDate,
		"endDate":   p.EndDate,
		"cursor":    p.Cursor,
		"size":      p.Size,
	})
	out := &SubscribersPage{}
	if err := c.doJSON(ctx, http.MethodGet, "/insights/subscribers", query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// TopSpenderUser is the fan profile attached to a top-spenders row. Nullable
// string fields use pointers so a JSON null is distinguishable from an empty
// string.
type TopSpenderUser struct {
	AvatarURL    *string `json:"avatarUrl"`
	DisplayName  string  `json:"displayName"`
	Handle       string  `json:"handle"`
	IsTopSpender bool    `json:"isTopSpender"`
	Nickname     *string `json:"nickname"`
	RegisteredAt string  `json:"registeredAt"`
	UUID         string  `json:"uuid"`
}

// TopSpendersItem is one top-spending fan with their spending totals (in cents)
// and message count over the requested period.
type TopSpendersItem struct {
	Gross    float64        `json:"gross"`
	Messages float64        `json:"messages"`
	Net      float64        `json:"net"`
	User     TopSpenderUser `json:"user"`
}

// TopSpendersPagination is the page/size envelope returned by
// GET /insights/top-spenders.
type TopSpendersPagination struct {
	HasMore bool    `json:"hasMore"`
	Page    float64 `json:"page"`
	Size    float64 `json:"size"`
}

// TopSpendersPage is one page of top-spending fans.
type TopSpendersPage struct {
	Data       []TopSpendersItem     `json:"data"`
	Pagination TopSpendersPagination `json:"pagination"`
}

// GetTopSpendersParams configures GET /insights/top-spenders. All fields are
// optional; nil pointers are omitted from the query string.
type GetTopSpendersParams struct {
	StartDate *string
	EndDate   *string
	Page      *int
	Size      *int
}

// GetTopSpenders returns a paginated list of the authenticated creator's
// top-spending fans.
//
// GET /insights/top-spenders — scope: read:insights, read:fan.
func (c *Client) GetTopSpenders(
	ctx context.Context, p GetTopSpendersParams,
) (*TopSpendersPage, error) {
	query := encodeQuery(map[string]any{
		"startDate": p.StartDate,
		"endDate":   p.EndDate,
		"page":      p.Page,
		"size":      p.Size,
	})
	out := &TopSpendersPage{}
	if err := c.doJSON(ctx, http.MethodGet, "/insights/top-spenders", query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// FanInsightsSpending holds a fan's aggregate spending breakdown (amounts in
// cents). Sources is keyed by the source label to its gross/total spend — a
// free-form map mirroring the Python SDK's dict[str, ...]. LastPurchaseAt is nil
// when the fan has never purchased.
type FanInsightsSpending struct {
	LastPurchaseAt   *string                      `json:"lastPurchaseAt"`
	MaxSinglePayment FanInsightsAmount            `json:"maxSinglePayment"`
	Sources          map[string]FanInsightsAmount `json:"sources"`
	Total            FanInsightsAmount            `json:"total"`
}

// FanInsightsAmount is a gross/total amount pair (in cents) used throughout the
// fan-insights spending breakdown.
type FanInsightsAmount struct {
	Gross float64 `json:"gross"`
	Total float64 `json:"total"`
}

// FanInsightsSubscription holds a fan's subscription lifecycle. CreatedAt and
// RenewsAt are nil when the fan is not (or never was) subscribed.
type FanInsightsSubscription struct {
	AutoRenewalEnabled bool    `json:"autoRenewalEnabled"`
	CreatedAt          *string `json:"createdAt"`
	RenewsAt           *string `json:"renewsAt"`
}

// FanInsights is the detailed per-fan insight payload returned by
// GET /insights/fans/{userUuid} and, keyed by fan UUID, by the bulk/batch
// endpoints.
type FanInsights struct {
	Spending     FanInsightsSpending     `json:"spending"`
	Status       FanStatus               `json:"status"`
	Subscription FanInsightsSubscription `json:"subscription"`
}

// GetFanInsights returns detailed insights about a single fan for the
// authenticated creator.
//
// GET /insights/fans/{userUuid} — scope: read:insights, read:fan.
func (c *Client) GetFanInsights(ctx context.Context, userUUID string) (*FanInsights, error) {
	if userUUID == "" {
		return nil, errors.New("fanvue: GetFanInsights requires a non-empty user UUID")
	}
	path := "/insights/fans/" + url.PathEscape(userUUID)
	out := &FanInsights{}
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// BulkFanInsightsError is a per-fan failure reported inside a 200 bulk-insights
// response so a single bad fan UUID never collapses the whole request.
type BulkFanInsightsError struct {
	// Code is "NOT_FOUND" or "INTERNAL".
	Code    string `json:"code"`
	FanUUID string `json:"fanUuid"`
	Message string `json:"message"`
}

// BulkFanInsights is the response from GET /insights/fans. Results is keyed by
// the requested fan UUID; a nil value indicates that fan failed (see Errors).
type BulkFanInsights struct {
	Errors  []BulkFanInsightsError  `json:"errors"`
	Results map[string]*FanInsights `json:"results"`
}

// GetBulkFanInsights returns detailed insights for multiple fans in a single
// request (capped at 20 fan UUIDs). fanUUIDs is sent as the comma-separated
// fanUuids query parameter, mirroring the Python SDK's single-string argument.
//
// GET /insights/fans — scope: read:insights, read:fan.
func (c *Client) GetBulkFanInsights(
	ctx context.Context, fanUUIDs string,
) (*BulkFanInsights, error) {
	if fanUUIDs == "" {
		return nil, errors.New("fanvue: GetBulkFanInsights requires a non-empty fanUuids value")
	}
	query := encodeQuery(map[string]any{"fanUuids": fanUUIDs})
	out := &BulkFanInsights{}
	if err := c.doJSON(ctx, http.MethodGet, "/insights/fans", query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// BatchFanInsightsResult is one entry in a batch-fan-insights response, keyed by
// the input fan UUID. On success the FanInsights fields are populated and Error
// is empty; on a per-fan failure Error is set to "forbidden", "not_found", or
// "internal" and the insight fields are zero. Error is the discriminator,
// mirroring the Python SDK's Option1 | Option2 union.
type BatchFanInsightsResult struct {
	Spending     *FanInsightsSpending     `json:"spending,omitempty"`
	Status       FanStatus                `json:"status,omitempty"`
	Subscription *FanInsightsSubscription `json:"subscription,omitempty"`
	// Error is "forbidden", "not_found", or "internal" when this fan failed,
	// otherwise empty.
	Error string `json:"error,omitempty"`
}

// BatchFanInsights returns detailed insights for up to 100 fans in a single
// POST request, keyed by the input fan UUID. The body is sent verbatim,
// mirroring the Python SDK's opaque Mapping[str, Any] body — typically
// {"fanUuids": ["...", "..."]}.
//
// POST /insights/fans/batch — scope: read:insights, read:fan.
func (c *Client) BatchFanInsights(
	ctx context.Context, body RawBody,
) (map[string]BatchFanInsightsResult, error) {
	if len(body) == 0 {
		return nil, errors.New("fanvue: BatchFanInsights requires a non-empty body")
	}
	out := map[string]BatchFanInsightsResult{}
	if err := c.doJSON(ctx, http.MethodPost, "/insights/fans/batch", nil, body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ExternalSocialPlatform enumerates the social platforms a tracking link can be
// attributed to.
type ExternalSocialPlatform string

const (
	// PlatformFacebook is Facebook.
	PlatformFacebook ExternalSocialPlatform = "facebook"
	// PlatformInstagram is Instagram.
	PlatformInstagram ExternalSocialPlatform = "instagram"
	// PlatformOther is any platform not otherwise enumerated.
	PlatformOther ExternalSocialPlatform = "other"
	// PlatformReddit is Reddit.
	PlatformReddit ExternalSocialPlatform = "reddit"
	// PlatformSnapchat is Snapchat.
	PlatformSnapchat ExternalSocialPlatform = "snapchat"
	// PlatformTikTok is TikTok.
	PlatformTikTok ExternalSocialPlatform = "tiktok"
	// PlatformTwitter is Twitter/X.
	PlatformTwitter ExternalSocialPlatform = "twitter"
	// PlatformYouTube is YouTube.
	PlatformYouTube ExternalSocialPlatform = "youtube"
)

// TrackingLinkEarnings holds a tracking link's attributed earnings (in cents).
type TrackingLinkEarnings struct {
	TotalGross float64 `json:"totalGross"`
	TotalNet   float64 `json:"totalNet"`
}

// TrackingLinkEngagement holds a tracking link's follower/subscriber engagement
// tallies.
type TrackingLinkEngagement struct {
	AcquiredFollowers   float64 `json:"acquiredFollowers"`
	AcquiredSubscribers float64 `json:"acquiredSubscribers"`
	TotalFollowers      float64 `json:"totalFollowers"`
	TotalSubscribers    float64 `json:"totalSubscribers"`
}

// TrackingLink is a single tracking link owned by the authenticated user.
// Earnings is nil when the link has no attributed earnings.
type TrackingLink struct {
	Clicks                 float64                `json:"clicks"`
	CreatedAt              string                 `json:"createdAt"`
	Earnings               *TrackingLinkEarnings  `json:"earnings"`
	Engagement             TrackingLinkEngagement `json:"engagement"`
	ExternalSocialPlatform ExternalSocialPlatform `json:"externalSocialPlatform"`
	LinkURL                string                 `json:"linkUrl"`
	Name                   string                 `json:"name"`
	UUID                   string                 `json:"uuid"`
}

// TrackingLinksPage is one cursor-paginated page of tracking links. NextCursor
// is nil on the final page; pass it back as ListTrackingLinksParams.Cursor to
// fetch the next page.
type TrackingLinksPage struct {
	Data       []TrackingLink `json:"data"`
	NextCursor *string        `json:"nextCursor"`
}

// ListTrackingLinksParams configures GET /tracking-links. All fields are
// optional; nil pointers are omitted from the query string.
type ListTrackingLinksParams struct {
	Limit         *int
	Cursor        *string
	CreatedAfter  *string
	CreatedBefore *string
}

// ListTrackingLinks returns one cursor-paginated page of the authenticated
// user's tracking links.
//
// GET /tracking-links — scope: read:tracking_links.
func (c *Client) ListTrackingLinks(
	ctx context.Context, p ListTrackingLinksParams,
) (*TrackingLinksPage, error) {
	query := encodeQuery(map[string]any{
		"limit":         p.Limit,
		"cursor":        p.Cursor,
		"createdAfter":  p.CreatedAfter,
		"createdBefore": p.CreatedBefore,
	})
	out := &TrackingLinksPage{}
	if err := c.doJSON(ctx, http.MethodGet, "/tracking-links", query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateTrackingLink creates a new tracking link for the authenticated user. The
// body is sent verbatim, mirroring the Python SDK's opaque Mapping[str, Any]
// body — typically {"name": "...", "externalSocialPlatform": "instagram"}. A nil
// body is allowed (the endpoint accepts an empty body) and sends no payload.
//
// POST /tracking-links — scope: write:tracking_links.
func (c *Client) CreateTrackingLink(
	ctx context.Context, body RawBody,
) (*TrackingLink, error) {
	out := &TrackingLink{}
	if err := c.doJSON(ctx, http.MethodPost, "/tracking-links", nil, createTrackingLinkBody(body), out); err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteTrackingLink soft-deletes a tracking link owned by the authenticated
// user. The endpoint returns no body.
//
// DELETE /tracking-links/{uuid} — scope: write:tracking_links.
func (c *Client) DeleteTrackingLink(ctx context.Context, uuid string) error {
	if uuid == "" {
		return errors.New("fanvue: DeleteTrackingLink requires a non-empty tracking-link UUID")
	}
	path := "/tracking-links/" + url.PathEscape(uuid)
	return c.doJSON(ctx, http.MethodDelete, path, nil, nil, nil)
}

// TrackingLinkUser is one user attributed to a tracking link. Status is nil when
// the user's relationship state is unknown. Nullable string fields use pointers
// so a JSON null is distinguishable from an empty string.
type TrackingLinkUser struct {
	AvatarURL    *string `json:"avatarUrl"`
	DisplayName  string  `json:"displayName"`
	Handle       string  `json:"handle"`
	IsTopSpender bool    `json:"isTopSpender"`
	Nickname     *string `json:"nickname"`
	RegisteredAt string  `json:"registeredAt"`
	// Status is "subscriber", "follower", "expired", or "deleted" when known.
	Status *string `json:"status"`
	UUID   string  `json:"uuid"`
}

// TrackingLinkUsersPage is one cursor-paginated page of users attributed to a
// tracking link. NextCursor is nil on the final page; pass it back as
// ListTrackingLinkUsersParams.Cursor to fetch the next page.
type TrackingLinkUsersPage struct {
	Data       []TrackingLinkUser `json:"data"`
	NextCursor *string            `json:"nextCursor"`
}

// ListTrackingLinkUsersParams configures GET /tracking-links/{uuid}/users. All
// fields are optional; nil pointers are omitted from the query string.
type ListTrackingLinkUsersParams struct {
	Limit  *int
	Cursor *string
}

// ListTrackingLinkUsers returns one cursor-paginated page of users associated
// with the tracking link identified by uuid.
//
// GET /tracking-links/{uuid}/users — scope: read:tracking_links.
func (c *Client) ListTrackingLinkUsers(
	ctx context.Context, uuid string, p ListTrackingLinkUsersParams,
) (*TrackingLinkUsersPage, error) {
	if uuid == "" {
		return nil, errors.New("fanvue: ListTrackingLinkUsers requires a non-empty tracking-link UUID")
	}
	path := "/tracking-links/" + url.PathEscape(uuid) + "/users"
	query := encodeQuery(map[string]any{
		"limit":  p.Limit,
		"cursor": p.Cursor,
	})
	out := &TrackingLinkUsersPage{}
	if err := c.doJSON(ctx, http.MethodGet, path, query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// UserTrackingMetadata is the custom tracking metadata captured on a user's most
// recent impression of a tracking link. Metadata is nil when no metadata was
// recorded; it is a free-form string map mirroring the Python SDK's
// dict[str, str] | None.
type UserTrackingMetadata struct {
	Metadata map[string]string `json:"metadata"`
}

// GetUserTrackingMetadata returns the custom tracking metadata from a user's
// most recent impression on the tracking link identified by uuid.
//
// GET /tracking-links/{uuid}/users/{userUuid}/metadata — scope: read:tracking_links.
func (c *Client) GetUserTrackingMetadata(
	ctx context.Context, uuid, userUUID string,
) (*UserTrackingMetadata, error) {
	if uuid == "" {
		return nil, errors.New("fanvue: GetUserTrackingMetadata requires a non-empty tracking-link UUID")
	}
	if userUUID == "" {
		return nil, errors.New("fanvue: GetUserTrackingMetadata requires a non-empty user UUID")
	}
	path := "/tracking-links/" + url.PathEscape(uuid) + "/users/" + url.PathEscape(userUUID) + "/metadata"
	out := &UserTrackingMetadata{}
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// earningsSourcesToStrings converts a slice of EarningsSource enums to the
// []string form encodeQuery serializes as repeated query keys, returning nil
// (so the parameter is omitted) when the input is empty.
func earningsSourcesToStrings(sources []EarningsSource) []string {
	if len(sources) == 0 {
		return nil
	}
	out := make([]string, len(sources))
	for i, s := range sources {
		out[i] = string(s)
	}
	return out
}

// spendingSourcesToStrings converts a slice of SpendingSource enums to the
// []string form encodeQuery serializes as repeated query keys, returning nil
// (so the parameter is omitted) when the input is empty.
func spendingSourcesToStrings(sources []SpendingSource) []string {
	if len(sources) == 0 {
		return nil
	}
	out := make([]string, len(sources))
	for i, s := range sources {
		out[i] = string(s)
	}
	return out
}

// granularityToString dereferences an optional EarningsGranularity into the
// *string form encodeQuery expects, returning nil when unset so the parameter
// is omitted.
func granularityToString(g *EarningsGranularity) *string {
	if g == nil {
		return nil
	}
	s := string(*g)
	return &s
}

// createTrackingLinkBody resolves the body sent for a CreateTrackingLink
// request. A nil RawBody yields a nil body (no payload sent), matching the
// Python SDK's optional Mapping[str, Any] body; otherwise it is sent verbatim.
func createTrackingLinkBody(body RawBody) any {
	if body == nil {
		return nil
	}
	return body
}
