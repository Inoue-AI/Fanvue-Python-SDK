package fanvue

import (
	"context"
	"errors"
	"net/http"
	"net/url"
)

// TeamMemberRole enumerates the per-creator access roles a team member can hold
// within an agency, as returned in a team member's creatorAccess list.
type TeamMemberRole string

const (
	// TeamMemberRoleAdmin grants full administrative access to the creator.
	TeamMemberRoleAdmin TeamMemberRole = "ADMIN"
	// TeamMemberRoleChatter grants chatting-only access to the creator.
	TeamMemberRoleChatter TeamMemberRole = "CHATTER"
)

// CreateAgencyInviteParams is the POST /agencies/invites request body. The
// endpoint invites a new team member to the authenticated user's agency by
// email; the typed fields cover the documented properties while RawBody allows
// sending an opaque body verbatim, mirroring the Python SDK's Mapping[str, Any]
// signature.
//
// RawBody, when non-nil, is sent verbatim and takes precedence over the typed
// fields.
type CreateAgencyInviteParams struct {
	// Email is the email address of the existing account to invite. Required
	// when RawBody is not supplied.
	Email string `json:"email,omitempty"`
	// IsAdmin, when set, grants the invited member agency-admin access.
	IsAdmin *bool `json:"isAdmin,omitempty"`
	// Nickname is an optional display nickname for the invited member.
	Nickname *string `json:"nickname,omitempty"`
	// RawBody is sent verbatim as the request body when non-nil.
	RawBody RawBody `json:"-"`
}

// CreateAgencyInviteResult is the response from POST /agencies/invites.
type CreateAgencyInviteResult struct {
	InviteUUID string `json:"inviteUuid"`
	Message    string `json:"message"`
	Success    bool   `json:"success"`
}

// CreateAgencyInvite invites a new team member to the authenticated user's
// agency by email. The invited user must have an existing account, must not be a
// creator, and must not already be invited to this agency.
//
// POST /agencies/invites — scope: write:agency (requires agency admin access).
func (c *Client) CreateAgencyInvite(
	ctx context.Context, p CreateAgencyInviteParams,
) (*CreateAgencyInviteResult, error) {
	if p.RawBody == nil && p.Email == "" {
		return nil, errors.New("fanvue: CreateAgencyInvite requires an Email or RawBody")
	}
	out := &CreateAgencyInviteResult{}
	if err := c.doJSON(ctx, http.MethodPost, "/agencies/invites", nil, agencyBody(p, p.RawBody), out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateCreatorInviteParams is the POST /agencies/creator-invites request body.
// The endpoint invites a creator to connect to the authenticated user's agency
// by email; the typed field covers the documented property while RawBody allows
// sending an opaque body verbatim, mirroring the Python SDK's
// Mapping[str, Any] signature.
//
// RawBody, when non-nil, is sent verbatim and takes precedence over the typed
// fields.
type CreateCreatorInviteParams struct {
	// Email is the email address of the existing creator account to invite.
	// Required when RawBody is not supplied.
	Email string `json:"email,omitempty"`
	// RawBody is sent verbatim as the request body when non-nil.
	RawBody RawBody `json:"-"`
}

// CreateCreatorInviteResult is the response from POST /agencies/creator-invites.
type CreateCreatorInviteResult struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
}

// CreateCreatorInvite invites a creator to connect to the authenticated user's
// agency by email. The invited user must have an existing Fanvue creator
// account; an email is sent to the creator with a link to accept the invitation.
//
// POST /agencies/creator-invites — scope: write:agency (requires agency admin
// access).
func (c *Client) CreateCreatorInvite(
	ctx context.Context, p CreateCreatorInviteParams,
) (*CreateCreatorInviteResult, error) {
	if p.RawBody == nil && p.Email == "" {
		return nil, errors.New("fanvue: CreateCreatorInvite requires an Email or RawBody")
	}
	out := &CreateCreatorInviteResult{}
	if err := c.doJSON(ctx, http.MethodPost, "/agencies/creator-invites", nil, agencyBody(p, p.RawBody), out); err != nil {
		return nil, err
	}
	return out, nil
}

// ChatterLeaderboardEntry is a single chatter's performance metrics over the
// requested period. AvatarURL and AvgResponseMs are nil when unavailable.
// Monetary amounts (Revenue) are in minor currency units (cents).
type ChatterLeaderboardEntry struct {
	ActiveHours   float64  `json:"activeHours"`
	AvatarURL     *string  `json:"avatarUrl"`
	AvgResponseMs *float64 `json:"avgResponseMs"`
	ChatterName   string   `json:"chatterName"`
	ChatterUUID   string   `json:"chatterUuid"`
	EPH           float64  `json:"eph"`
	GoldenRatio   float64  `json:"goldenRatio"`
	Messages      int64    `json:"messages"`
	PPVsSent      int64    `json:"ppvsSent"`
	PPVsUnlocked  int64    `json:"ppvsUnlocked"`
	Revenue       int64    `json:"revenue"`
	UnlockRatio   float64  `json:"unlockRatio"`
}

// ChatterLeaderboard is the response from GET
// /agencies/insights/chatter-leaderboard: per-chatter performance rows sorted by
// revenue descending. The endpoint returns a bare {data: [...]} envelope with no
// pagination.
type ChatterLeaderboard struct {
	Data []ChatterLeaderboardEntry `json:"data"`
}

// GetChatterLeaderboardParams configures a request to GET
// /agencies/insights/chatter-leaderboard. All fields are optional and omitted
// from the query string when unset. ChatterUUIDs is a single query value
// (comma-separated when filtering multiple chatters), matching the Python SDK's
// str | None parameter.
type GetChatterLeaderboardParams struct {
	StartDate    *string
	EndDate      *string
	ChatterUUIDs *string
}

// GetChatterLeaderboard returns per-chatter performance metrics for the
// authenticated user's agency over the requested period. Stats are sourced from
// a daily aggregation table refreshed at most once per day, so very recent
// activity may not be reflected.
//
// GET /agencies/insights/chatter-leaderboard — scope: read:agency (requires
// agency admin access).
func (c *Client) GetChatterLeaderboard(
	ctx context.Context, p GetChatterLeaderboardParams,
) (*ChatterLeaderboard, error) {
	query := encodeQuery(map[string]any{
		"startDate":    p.StartDate,
		"endDate":      p.EndDate,
		"chatterUuids": p.ChatterUUIDs,
	})
	out := &ChatterLeaderboard{}
	if err := c.doJSON(ctx, http.MethodGet, "/agencies/insights/chatter-leaderboard", query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// AgencyChatLastMessage is a compact summary of the most recent message in an
// agency chat row, as returned by GET /agencies/chats. Nullable fields use
// pointers so a JSON null is distinguishable from a zero value.
type AgencyChatLastMessage struct {
	HasMedia     *bool   `json:"hasMedia"`
	MediaType    *string `json:"mediaType"`
	SenderUUID   string  `json:"senderUuid"`
	SentAt       *string `json:"sentAt"`
	SentByUserID *string `json:"sentByUserId"`
	Text         *string `json:"text"`
	Type         string  `json:"type"`
	UUID         string  `json:"uuid"`
}

// AgencyChatUser is the chat counterpart summary returned in an agency chat row.
type AgencyChatUser struct {
	AvatarURL    *string `json:"avatarUrl"`
	DisplayName  string  `json:"displayName"`
	Handle       string  `json:"handle"`
	IsTopSpender bool    `json:"isTopSpender"`
	Nickname     *string `json:"nickname"`
	RegisteredAt string  `json:"registeredAt"`
	UUID         string  `json:"uuid"`
}

// AgencyChat is one chat row in the agency-wide chats stream. Each row is tagged
// with CreatorUUID so consumers can group results by creator without an
// additional join. IsMuted is per-creator: true only when the creator that owns
// the row has muted the counterpart.
type AgencyChat struct {
	CreatedAt           *string                `json:"createdAt"`
	CreatorUUID         string                 `json:"creatorUuid"`
	IsMuted             bool                   `json:"isMuted"`
	IsRead              bool                   `json:"isRead"`
	LastMessage         *AgencyChatLastMessage `json:"lastMessage"`
	LastMessageAt       *string                `json:"lastMessageAt"`
	UnreadMessagesCount float64                `json:"unreadMessagesCount"`
	User                AgencyChatUser         `json:"user"`
}

// AgencyChatsPage is one page of the agency-wide chats stream.
type AgencyChatsPage struct {
	Data       []AgencyChat `json:"data"`
	Pagination Pagination   `json:"pagination"`
}

// ListAgencyChatsParams configures a page request to GET /agencies/chats.
type ListAgencyChatsParams struct {
	Page *int
	Size *int
}

// ListAgencyChats returns a single paginated stream of chats across every
// creator the authenticated agency manages, sorted by most recent activity.
//
// GET /agencies/chats — scopes: read:agency, read:chat (requires agency admin
// access).
func (c *Client) ListAgencyChats(
	ctx context.Context, p ListAgencyChatsParams,
) (*AgencyChatsPage, error) {
	query := encodeQuery(map[string]any{"page": p.Page, "size": p.Size})
	out := &AgencyChatsPage{}
	if err := c.doJSON(ctx, http.MethodGet, "/agencies/chats", query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// AgencyEarningsByDay is one per-creator-per-day earnings row from GET
// /agencies/earnings. Each row is bucketed by paid_at (UTC) and tagged with
// CreatorUUID and Currency. Gross and Net are in minor currency units (cents).
// Currency is nil when the row has no resolvable currency.
type AgencyEarningsByDay struct {
	CreatorUUID string  `json:"creatorUuid"`
	Currency    *string `json:"currency"`
	Date        string  `json:"date"`
	Gross       int64   `json:"gross"`
	Net         int64   `json:"net"`
}

// AgencyEarningsByDayPage is one page of per-creator-per-day earnings rows.
type AgencyEarningsByDayPage struct {
	Data       []AgencyEarningsByDay `json:"data"`
	Pagination Pagination            `json:"pagination"`
}

// ListAgencyEarningsByDayParams configures a page request to GET
// /agencies/earnings. StartDate and EndDate are required; CreatorUUIDs filters
// to specific creators and is sent as repeated query keys.
type ListAgencyEarningsByDayParams struct {
	StartDate    string
	EndDate      string
	Page         *int
	Size         *int
	CreatorUUIDs []string
}

// ListAgencyEarningsByDay returns a single paginated stream of
// per-creator-per-day earnings rows across every creator the authenticated
// agency manages, sorted by most recent day first. Only PAID earning invoices
// that count towards creator insights are included.
//
// GET /agencies/earnings — scopes: read:agency, read:creator (requires agency
// admin access).
func (c *Client) ListAgencyEarningsByDay(
	ctx context.Context, p ListAgencyEarningsByDayParams,
) (*AgencyEarningsByDayPage, error) {
	if p.StartDate == "" {
		return nil, errors.New("fanvue: ListAgencyEarningsByDay requires a non-empty StartDate")
	}
	if p.EndDate == "" {
		return nil, errors.New("fanvue: ListAgencyEarningsByDay requires a non-empty EndDate")
	}
	query := encodeQuery(map[string]any{
		"page":         p.Page,
		"size":         p.Size,
		"startDate":    p.StartDate,
		"endDate":      p.EndDate,
		"creatorUuids": creatorUUIDsParam(p.CreatorUUIDs),
	})
	out := &AgencyEarningsByDayPage{}
	if err := c.doJSON(ctx, http.MethodGet, "/agencies/earnings", query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// AgencySubscriber is one active-subscription row from GET /agencies/subscribers.
// Each row is tagged with CreatorUUID. ExpiresAt and Nickname are nil when not
// set; AvatarURL is nil when the subscriber has no avatar.
type AgencySubscriber struct {
	AvatarURL    *string `json:"avatarUrl"`
	CreatorUUID  string  `json:"creatorUuid"`
	DisplayName  string  `json:"displayName"`
	ExpiresAt    *string `json:"expiresAt"`
	Handle       string  `json:"handle"`
	IsTopSpender bool    `json:"isTopSpender"`
	Nickname     *string `json:"nickname"`
	RegisteredAt string  `json:"registeredAt"`
	SubscribedAt string  `json:"subscribedAt"`
	UUID         string  `json:"uuid"`
}

// AgencySubscribersPage is one page of active-subscription rows across all
// agency creators.
type AgencySubscribersPage struct {
	Data       []AgencySubscriber `json:"data"`
	Pagination Pagination         `json:"pagination"`
}

// ListAgencySubscribersParams configures a page request to GET
// /agencies/subscribers. CreatorUUIDs filters to specific creators and is sent
// as repeated query keys.
type ListAgencySubscribersParams struct {
	Page         *int
	Size         *int
	CreatorUUIDs []string
}

// ListAgencySubscribers returns a single paginated stream of active
// subscriptions across every creator the authenticated agency manages, sorted
// by most recent subscription first.
//
// GET /agencies/subscribers — scopes: read:agency, read:creator (requires
// agency admin access).
func (c *Client) ListAgencySubscribers(
	ctx context.Context, p ListAgencySubscribersParams,
) (*AgencySubscribersPage, error) {
	query := encodeQuery(map[string]any{
		"page":         p.Page,
		"size":         p.Size,
		"creatorUuids": creatorUUIDsParam(p.CreatorUUIDs),
	})
	out := &AgencySubscribersPage{}
	if err := c.doJSON(ctx, http.MethodGet, "/agencies/subscribers", query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// AgencySubscribersHistoryEntry is one per-creator-per-day subscriber-event row
// from GET /agencies/subscribers-history. NewSubscribersCount and
// CancelledSubscribersCount are per-day deltas; Total is the cumulative net
// change for the creator from the beginning of the requested range.
type AgencySubscribersHistoryEntry struct {
	CancelledSubscribersCount int64  `json:"cancelledSubscribersCount"`
	CreatorUUID               string `json:"creatorUuid"`
	Date                      string `json:"date"`
	NewSubscribersCount       int64  `json:"newSubscribersCount"`
	Total                     int64  `json:"total"`
}

// AgencySubscribersHistoryPage is one page of per-creator-per-day
// subscriber-event rows.
type AgencySubscribersHistoryPage struct {
	Data       []AgencySubscribersHistoryEntry `json:"data"`
	Pagination Pagination                      `json:"pagination"`
}

// ListAgencySubscribersHistoryParams configures a page request to GET
// /agencies/subscribers-history. StartDate and EndDate are required;
// CreatorUUIDs filters to specific creators and is sent as repeated query keys.
type ListAgencySubscribersHistoryParams struct {
	StartDate    string
	EndDate      string
	Page         *int
	Size         *int
	CreatorUUIDs []string
}

// ListAgencySubscribersHistory returns a single paginated stream of
// per-creator-per-day subscriber-event rows across every creator the
// authenticated agency manages, sorted by most recent day first. This endpoint
// is an analytics time series, not a real-time audience snapshot.
//
// GET /agencies/subscribers-history — scopes: read:agency, read:creator
// (requires agency admin access).
func (c *Client) ListAgencySubscribersHistory(
	ctx context.Context, p ListAgencySubscribersHistoryParams,
) (*AgencySubscribersHistoryPage, error) {
	if p.StartDate == "" {
		return nil, errors.New("fanvue: ListAgencySubscribersHistory requires a non-empty StartDate")
	}
	if p.EndDate == "" {
		return nil, errors.New("fanvue: ListAgencySubscribersHistory requires a non-empty EndDate")
	}
	query := encodeQuery(map[string]any{
		"page":         p.Page,
		"size":         p.Size,
		"startDate":    p.StartDate,
		"endDate":      p.EndDate,
		"creatorUuids": creatorUUIDsParam(p.CreatorUUIDs),
	})
	out := &AgencySubscribersHistoryPage{}
	if err := c.doJSON(ctx, http.MethodGet, "/agencies/subscribers-history", query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// TeamMemberCreatorAccess is a single creator-access grant on a team member: the
// creator UUID and the role the member holds for that creator.
type TeamMemberCreatorAccess struct {
	Role TeamMemberRole `json:"role"`
	UUID string         `json:"uuid"`
}

// TeamMember is a single member of the authenticated user's agency, including
// their per-creator access grants. Nickname is nil when unset.
type TeamMember struct {
	CreatorAccess []TeamMemberCreatorAccess `json:"creatorAccess"`
	DisplayName   string                    `json:"displayName"`
	Email         string                    `json:"email"`
	IsAdmin       bool                      `json:"isAdmin"`
	Nickname      *string                   `json:"nickname"`
	UUID          string                    `json:"uuid"`
}

// ListTeamMembers returns all team members in the authenticated user's agency.
// The endpoint returns a bare array, not a paginated envelope.
//
// GET /agencies/team-members — scope: read:agency (requires agency admin
// access).
func (c *Client) ListTeamMembers(ctx context.Context) ([]TeamMember, error) {
	out := []TeamMember{}
	if err := c.doJSON(ctx, http.MethodGet, "/agencies/team-members", nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// UpdateTeamMemberParams is the PUT /agencies/team-members/{userId} request
// body. The typed fields cover the documented properties (admin status,
// nickname) while RawBody allows sending an opaque body verbatim, mirroring the
// Python SDK's Mapping[str, Any] signature.
//
// RawBody, when non-nil, is sent verbatim and takes precedence over the typed
// fields.
type UpdateTeamMemberParams struct {
	// IsAdmin, when set, toggles the member's agency-admin access.
	IsAdmin *bool `json:"isAdmin,omitempty"`
	// Nickname, when set, updates the member's display nickname.
	Nickname *string `json:"nickname,omitempty"`
	// RawBody is sent verbatim as the request body when non-nil.
	RawBody RawBody `json:"-"`
}

// UpdatedTeamMember is the response from PUT
// /agencies/team-members/{userId}: the team member's updated properties.
type UpdatedTeamMember struct {
	CreatorAccess []TeamMemberCreatorAccess `json:"creatorAccess"`
	DisplayName   string                    `json:"displayName"`
	Email         string                    `json:"email"`
	IsAdmin       bool                      `json:"isAdmin"`
	Nickname      *string                   `json:"nickname"`
	UUID          string                    `json:"uuid"`
}

// UpdateTeamMember updates a team member's properties (such as admin status or
// nickname) within the authenticated user's agency. userID identifies the team
// member to update.
//
// PUT /agencies/team-members/{userId} — scope: write:agency (requires agency
// admin access).
func (c *Client) UpdateTeamMember(
	ctx context.Context, userID string, p UpdateTeamMemberParams,
) (*UpdatedTeamMember, error) {
	if userID == "" {
		return nil, errors.New("fanvue: UpdateTeamMember requires a non-empty user ID")
	}
	path := "/agencies/team-members/" + url.PathEscape(userID)
	out := &UpdatedTeamMember{}
	if err := c.doJSON(ctx, http.MethodPut, path, nil, agencyBody(p, p.RawBody), out); err != nil {
		return nil, err
	}
	return out, nil
}

// AgencyCreator is one creator associated with the authenticated agency user's
// organization, as returned by GET /creators. Role is nil when the API does not
// report a role for the creator.
type AgencyCreator struct {
	AvatarURL    *string `json:"avatarUrl"`
	DisplayName  string  `json:"displayName"`
	Handle       string  `json:"handle"`
	IsTopSpender bool    `json:"isTopSpender"`
	Nickname     *string `json:"nickname"`
	RegisteredAt string  `json:"registeredAt"`
	Role         *string `json:"role,omitempty"`
	UUID         string  `json:"uuid"`
}

// AgencyCreatorsPage is one page of creators associated with the authenticated
// agency.
type AgencyCreatorsPage struct {
	Data       []AgencyCreator `json:"data"`
	Pagination Pagination      `json:"pagination"`
}

// ListCreatorsParams configures a page request to GET /creators.
type ListCreatorsParams struct {
	Page *int
	Size *int
}

// ListCreators returns a paginated list of creators associated with the
// authenticated agency user's organization.
//
// GET /creators — scope: read:creator (agency endpoint; operates at the agency
// level for the agency you belong to).
func (c *Client) ListCreators(
	ctx context.Context, p ListCreatorsParams,
) (*AgencyCreatorsPage, error) {
	query := encodeQuery(map[string]any{"page": p.Page, "size": p.Size})
	out := &AgencyCreatorsPage{}
	if err := c.doJSON(ctx, http.MethodGet, "/creators", query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// agencyBody resolves the body sent for an agencies write request. When raw is
// supplied it is sent verbatim (preserving explicit JSON nulls and any
// additional fields, matching the Python SDK's opaque Mapping body); otherwise
// the typed params value is marshalled normally.
func agencyBody(typed any, raw RawBody) any {
	if raw != nil {
		return raw
	}
	return typed
}

// creatorUUIDsParam converts a slice of creator UUIDs into the []string form
// encodeQuery renders as repeated query keys, returning nil for an empty input
// so the parameter is omitted entirely.
func creatorUUIDsParam(uuids []string) []string {
	if len(uuids) == 0 {
		return nil
	}
	return uuids
}
