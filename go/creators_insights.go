package fanvue

import (
	"context"
	"errors"
	"net/http"
	"net/url"
)

// This file implements the creator-scoped read surface an agency operates on
// behalf of a specific managed creator (identified by creatorUserUUID):
//
//   - insights: earnings, earnings summary, subscriber-event time series, and
//     top spenders, mirroring the self-scoped /insights/* endpoints;
//   - audience: creator subscribers, online subscribers, and followers;
//   - notifications: the creator's notifications feed;
//   - tracking links: list/create/delete plus per-link user listing and
//     per-user tracking metadata.
//
// Every method mirrors its Python counterpart (resources/creators.py) exactly in
// endpoint, verb, parameters, defaults, pagination, and error mapping. Where the
// creator-scoped response shape is byte-identical to its self-scoped counterpart
// the existing types are reused (EarningsPage, EarningsSummary, SubscribersPage,
// TopSpendersPage, FansPage, NotificationsPage, TrackingLinksPage, TrackingLink,
// TrackingLinkUsersPage, UserTrackingMetadata); only GET .../subscribers/online
// has a shape with no self-scoped twin, so OnlineSubscribers is introduced here.

// GetCreatorEarningsParams configures
// GET /creators/{creatorUserUuid}/insights/earnings. All fields are optional;
// nil pointers and an empty Source slice are omitted from the query string. It
// mirrors the self-scoped GetEarningsParams field-for-field.
type GetCreatorEarningsParams struct {
	StartDate *string
	EndDate   *string
	Source    []EarningsSource
	Cursor    *string
	Size      *int
}

// GetCreatorEarnings returns cursor-paginated invoice data for a managed creator
// over the requested period. The response shape is identical to the self-scoped
// GET /insights/earnings, so it reuses EarningsPage.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// GET /creators/{creatorUserUuid}/insights/earnings — scopes: read:insights, read:creator.
func (c *Client) GetCreatorEarnings(
	ctx context.Context, creatorUserUUID string, p GetCreatorEarningsParams,
) (*EarningsPage, error) {
	if creatorUserUUID == "" {
		return nil, errors.New("fanvue: GetCreatorEarnings requires a non-empty creator user UUID")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) + "/insights/earnings"
	query := encodeQuery(map[string]any{
		"startDate": p.StartDate,
		"endDate":   p.EndDate,
		"source":    earningsSourcesToStrings(p.Source),
		"cursor":    p.Cursor,
		"size":      p.Size,
	})
	out := &EarningsPage{}
	if err := c.doJSON(ctx, http.MethodGet, path, query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetCreatorEarningsSummaryParams configures
// GET /creators/{creatorUserUuid}/insights/earnings/summary. All fields are
// optional; nil pointers are omitted from the query string. It mirrors the
// self-scoped GetEarningsSummaryParams.
type GetCreatorEarningsSummaryParams struct {
	StartDate   *string
	EndDate     *string
	Granularity *EarningsGranularity
	Timezone    *string
}

// GetCreatorEarningsSummary returns the pre-aggregated earnings metrics for a
// managed creator. The response shape is identical to the self-scoped
// GET /insights/earnings/summary, so it reuses EarningsSummary.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// GET /creators/{creatorUserUuid}/insights/earnings/summary — scopes: read:insights, read:creator.
func (c *Client) GetCreatorEarningsSummary(
	ctx context.Context, creatorUserUUID string, p GetCreatorEarningsSummaryParams,
) (*EarningsSummary, error) {
	if creatorUserUUID == "" {
		return nil, errors.New("fanvue: GetCreatorEarningsSummary requires a non-empty creator user UUID")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) + "/insights/earnings/summary"
	query := encodeQuery(map[string]any{
		"startDate":   p.StartDate,
		"endDate":     p.EndDate,
		"granularity": granularityToString(p.Granularity),
		"timezone":    p.Timezone,
	})
	out := &EarningsSummary{}
	if err := c.doJSON(ctx, http.MethodGet, path, query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetCreatorSubscribersParams configures
// GET /creators/{creatorUserUuid}/insights/subscribers. All fields are optional;
// nil pointers are omitted from the query string. It mirrors the self-scoped
// GetSubscribersParams.
type GetCreatorSubscribersParams struct {
	StartDate *string
	EndDate   *string
	Cursor    *string
	Size      *int
}

// GetCreatorSubscribers returns cursor-paginated subscriber-event time-series
// data for a managed creator over the requested period. The response shape is
// identical to the self-scoped GET /insights/subscribers, so it reuses
// SubscribersPage.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// GET /creators/{creatorUserUuid}/insights/subscribers — scopes: read:insights, read:creator.
func (c *Client) GetCreatorSubscribers(
	ctx context.Context, creatorUserUUID string, p GetCreatorSubscribersParams,
) (*SubscribersPage, error) {
	if creatorUserUUID == "" {
		return nil, errors.New("fanvue: GetCreatorSubscribers requires a non-empty creator user UUID")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) + "/insights/subscribers"
	query := encodeQuery(map[string]any{
		"startDate": p.StartDate,
		"endDate":   p.EndDate,
		"cursor":    p.Cursor,
		"size":      p.Size,
	})
	out := &SubscribersPage{}
	if err := c.doJSON(ctx, http.MethodGet, path, query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetCreatorTopSpendersParams configures
// GET /creators/{creatorUserUuid}/insights/top-spenders. All fields are
// optional; nil pointers are omitted from the query string. It mirrors the
// self-scoped GetTopSpendersParams.
type GetCreatorTopSpendersParams struct {
	StartDate *string
	EndDate   *string
	Page      *int
	Size      *int
}

// GetCreatorTopSpenders returns a paginated list of a managed creator's
// top-spending fans. The response shape is identical to the self-scoped
// GET /insights/top-spenders, so it reuses TopSpendersPage.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// GET /creators/{creatorUserUuid}/insights/top-spenders — scopes: read:insights, read:fan, read:creator.
func (c *Client) GetCreatorTopSpenders(
	ctx context.Context, creatorUserUUID string, p GetCreatorTopSpendersParams,
) (*TopSpendersPage, error) {
	if creatorUserUUID == "" {
		return nil, errors.New("fanvue: GetCreatorTopSpenders requires a non-empty creator user UUID")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) + "/insights/top-spenders"
	query := encodeQuery(map[string]any{
		"startDate": p.StartDate,
		"endDate":   p.EndDate,
		"page":      p.Page,
		"size":      p.Size,
	})
	out := &TopSpendersPage{}
	if err := c.doJSON(ctx, http.MethodGet, path, query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListCreatorSubscribers returns one page of a managed creator's subscribers.
// The entry shape is identical to the self-scoped GET /subscribers, so it reuses
// FansPage. p mirrors the self-scoped ListFansParams (page/size).
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// GET /creators/{creatorUserUuid}/subscribers — scopes: read:fan, read:creator.
func (c *Client) ListCreatorSubscribers(
	ctx context.Context, creatorUserUUID string, p ListFansParams,
) (*FansPage, error) {
	if creatorUserUUID == "" {
		return nil, errors.New("fanvue: ListCreatorSubscribers requires a non-empty creator user UUID")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) + "/subscribers"
	query := encodeQuery(map[string]any{"page": p.Page, "size": p.Size})
	out := &FansPage{}
	if err := c.doJSON(ctx, http.MethodGet, path, query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// OnlineSubscriber is one online subscriber returned by
// GET /creators/{creatorUserUuid}/subscribers/online. LastSeenAt is nil when the
// subscriber's last-seen timestamp is unknown.
type OnlineSubscriber struct {
	LastSeenAt *string `json:"lastSeenAt"`
	UUID       string  `json:"uuid"`
}

// OnlineSubscribers is the response from
// GET /creators/{creatorUserUuid}/subscribers/online. Unlike the cursor- or
// page-paginated audience endpoints it returns a flat Count plus the matching
// subscriber rows, so it has no self-scoped counterpart to reuse.
type OnlineSubscribers struct {
	Count float64            `json:"count"`
	Data  []OnlineSubscriber `json:"data"`
}

// GetOnlineSubscribersParams configures
// GET /creators/{creatorUserUuid}/subscribers/online. Both fields are optional;
// unset pointers are omitted from the query string. SubscriberUUIDs is sent as
// the single subscriberUuids query parameter verbatim — mirroring the Python
// SDK's single-string argument (typically a comma-separated UUID list).
type GetOnlineSubscribersParams struct {
	Limit           *int
	SubscriberUUIDs *string
}

// GetOnlineSubscribers returns the currently-online subscribers of a managed
// creator, optionally filtered to a specific set of subscriber UUIDs.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// GET /creators/{creatorUserUuid}/subscribers/online — scopes: read:fan, read:creator.
func (c *Client) GetOnlineSubscribers(
	ctx context.Context, creatorUserUUID string, p GetOnlineSubscribersParams,
) (*OnlineSubscribers, error) {
	if creatorUserUUID == "" {
		return nil, errors.New("fanvue: GetOnlineSubscribers requires a non-empty creator user UUID")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) + "/subscribers/online"
	query := encodeQuery(map[string]any{
		"limit":           p.Limit,
		"subscriberUuids": p.SubscriberUUIDs,
	})
	out := &OnlineSubscribers{}
	if err := c.doJSON(ctx, http.MethodGet, path, query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListCreatorFollowers returns one page of a managed creator's followers. The
// entry shape is identical to the self-scoped GET /followers, so it reuses
// FansPage. p mirrors the self-scoped ListFansParams (page/size).
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// GET /creators/{creatorUserUuid}/followers — scopes: read:fan, read:creator.
func (c *Client) ListCreatorFollowers(
	ctx context.Context, creatorUserUUID string, p ListFansParams,
) (*FansPage, error) {
	if creatorUserUUID == "" {
		return nil, errors.New("fanvue: ListCreatorFollowers requires a non-empty creator user UUID")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) + "/followers"
	query := encodeQuery(map[string]any{"page": p.Page, "size": p.Size})
	out := &FansPage{}
	if err := c.doJSON(ctx, http.MethodGet, path, query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListCreatorNotifications returns one page of a managed creator's notifications
// feed. The entry shape is identical to the self-scoped GET /notifications, so it
// reuses NotificationsPage. p mirrors the self-scoped ListNotificationsParams
// (page/size/eventType).
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// GET /creators/{creatorUserUuid}/notifications — scopes: read:notification, read:creator.
func (c *Client) ListCreatorNotifications(
	ctx context.Context, creatorUserUUID string, p ListNotificationsParams,
) (*NotificationsPage, error) {
	if creatorUserUUID == "" {
		return nil, errors.New("fanvue: ListCreatorNotifications requires a non-empty creator user UUID")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) + "/notifications"
	query := encodeQuery(map[string]any{
		"page":      p.Page,
		"size":      p.Size,
		"eventType": p.EventType,
	})
	out := &NotificationsPage{}
	if err := c.doJSON(ctx, http.MethodGet, path, query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListCreatorTrackingLinksParams configures
// GET /creators/{creatorUserUuid}/tracking-links. All fields are optional; nil
// pointers are omitted from the query string. It mirrors the self-scoped
// ListTrackingLinksParams.
type ListCreatorTrackingLinksParams struct {
	Limit         *int
	Cursor        *string
	CreatedAfter  *string
	CreatedBefore *string
}

// ListCreatorTrackingLinks returns one cursor-paginated page of a managed
// creator's tracking links. The response shape is identical to the self-scoped
// GET /tracking-links, so it reuses TrackingLinksPage.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// GET /creators/{creatorUserUuid}/tracking-links — scopes: read:tracking_links, read:creator.
func (c *Client) ListCreatorTrackingLinks(
	ctx context.Context, creatorUserUUID string, p ListCreatorTrackingLinksParams,
) (*TrackingLinksPage, error) {
	if creatorUserUUID == "" {
		return nil, errors.New("fanvue: ListCreatorTrackingLinks requires a non-empty creator user UUID")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) + "/tracking-links"
	query := encodeQuery(map[string]any{
		"limit":         p.Limit,
		"cursor":        p.Cursor,
		"createdAfter":  p.CreatedAfter,
		"createdBefore": p.CreatedBefore,
	})
	out := &TrackingLinksPage{}
	if err := c.doJSON(ctx, http.MethodGet, path, query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateCreatorTrackingLink creates a new tracking link for a managed creator.
// Body is the opaque request payload (mirroring the Python SDK's optional
// Mapping[str, Any] body) — typically {"name": "...", "externalSocialPlatform":
// "instagram"}. A nil body is allowed (the endpoint accepts an empty body) and
// sends no payload. The response shape is identical to the self-scoped
// POST /tracking-links, so it reuses TrackingLink.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// POST /creators/{creatorUserUuid}/tracking-links — scopes: write:tracking_links, write:creator.
func (c *Client) CreateCreatorTrackingLink(
	ctx context.Context, creatorUserUUID string, body RawBody,
) (*TrackingLink, error) {
	if creatorUserUUID == "" {
		return nil, errors.New("fanvue: CreateCreatorTrackingLink requires a non-empty creator user UUID")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) + "/tracking-links"
	out := &TrackingLink{}
	if err := c.doJSON(ctx, http.MethodPost, path, nil, createTrackingLinkBody(body), out); err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteCreatorTrackingLink soft-deletes a tracking link owned by a managed
// creator. The endpoint returns no body.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// DELETE /creators/{creatorUserUuid}/tracking-links/{uuid} — scopes: write:tracking_links, write:creator.
func (c *Client) DeleteCreatorTrackingLink(
	ctx context.Context, creatorUserUUID, uuid string,
) error {
	if creatorUserUUID == "" {
		return errors.New("fanvue: DeleteCreatorTrackingLink requires a non-empty creator user UUID")
	}
	if uuid == "" {
		return errors.New("fanvue: DeleteCreatorTrackingLink requires a non-empty tracking-link UUID")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) +
		"/tracking-links/" + url.PathEscape(uuid)
	return c.doJSON(ctx, http.MethodDelete, path, nil, nil, nil)
}

// ListCreatorTrackingLinkUsersParams configures
// GET /creators/{creatorUserUuid}/tracking-links/{uuid}/users. Both fields are
// optional; nil pointers are omitted from the query string. It mirrors the
// self-scoped ListTrackingLinkUsersParams.
type ListCreatorTrackingLinkUsersParams struct {
	Limit  *int
	Cursor *string
}

// ListCreatorTrackingLinkUsers returns one cursor-paginated page of users
// associated with a managed creator's tracking link identified by uuid. The
// response shape is identical to the self-scoped GET /tracking-links/{uuid}/users,
// so it reuses TrackingLinkUsersPage.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// GET /creators/{creatorUserUuid}/tracking-links/{uuid}/users — scopes: read:tracking_links, read:creator.
func (c *Client) ListCreatorTrackingLinkUsers(
	ctx context.Context, creatorUserUUID, uuid string, p ListCreatorTrackingLinkUsersParams,
) (*TrackingLinkUsersPage, error) {
	if creatorUserUUID == "" {
		return nil, errors.New("fanvue: ListCreatorTrackingLinkUsers requires a non-empty creator user UUID")
	}
	if uuid == "" {
		return nil, errors.New("fanvue: ListCreatorTrackingLinkUsers requires a non-empty tracking-link UUID")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) +
		"/tracking-links/" + url.PathEscape(uuid) + "/users"
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

// GetCreatorUserTrackingMetadata returns the custom tracking metadata from a
// user's most recent impression on a managed creator's tracking link identified
// by uuid. The response shape is identical to the self-scoped
// GET /tracking-links/{uuid}/users/{userUuid}/metadata, so it reuses
// UserTrackingMetadata.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// GET /creators/{creatorUserUuid}/tracking-links/{uuid}/users/{userUuid}/metadata
// — scopes: read:tracking_links, read:creator.
func (c *Client) GetCreatorUserTrackingMetadata(
	ctx context.Context, creatorUserUUID, uuid, userUUID string,
) (*UserTrackingMetadata, error) {
	if creatorUserUUID == "" {
		return nil, errors.New("fanvue: GetCreatorUserTrackingMetadata requires a non-empty creator user UUID")
	}
	if uuid == "" {
		return nil, errors.New("fanvue: GetCreatorUserTrackingMetadata requires a non-empty tracking-link UUID")
	}
	if userUUID == "" {
		return nil, errors.New("fanvue: GetCreatorUserTrackingMetadata requires a non-empty user UUID")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) +
		"/tracking-links/" + url.PathEscape(uuid) +
		"/users/" + url.PathEscape(userUUID) + "/metadata"
	out := &UserTrackingMetadata{}
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}
