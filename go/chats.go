package fanvue

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
)

// Message is a single chat message between the authenticated user and a
// counterpart. Free-form sub-objects (sender, recipient, pricing) are kept as
// raw JSON so the focused parity client does not have to model the full chat
// schema; callers that need them can unmarshal on demand.
type Message struct {
	UUID         string          `json:"uuid"`
	Type         string          `json:"type"`
	Text         *string         `json:"text"`
	HasMedia     *bool           `json:"hasMedia"`
	MediaType    *string         `json:"mediaType"`
	MediaUUIDs   []string        `json:"mediaUuids,omitempty"`
	IsRead       bool            `json:"isRead"`
	SentAt       *string         `json:"sentAt"`
	SentByUserID *string         `json:"sentByUserId"`
	PurchasedAt  *string         `json:"purchasedAt"`
	Sender       json.RawMessage `json:"sender,omitempty"`
	Recipient    json.RawMessage `json:"recipient,omitempty"`
	Pricing      json.RawMessage `json:"pricing,omitempty"`
}

// MessagesPage is one page of messages from a chat conversation.
type MessagesPage struct {
	Data       []Message  `json:"data"`
	Pagination Pagination `json:"pagination"`
}

// ListMessagesParams configures a page request to
// GET /chats/{userUuid}/messages.
type ListMessagesParams struct {
	Page       *int
	Size       *int
	MarkAsRead *bool
}

// ListMessages returns one page of messages between the authenticated user and
// the counterpart identified by userUUID. Messages are newest-first.
//
// GET /chats/{userUuid}/messages — scope: read:chat.
func (c *Client) ListMessages(
	ctx context.Context, userUUID string, p ListMessagesParams,
) (*MessagesPage, error) {
	if userUUID == "" {
		return nil, errors.New("fanvue: ListMessages requires a non-empty user UUID")
	}
	path := "/chats/" + url.PathEscape(userUUID) + "/messages"
	query := encodeQuery(map[string]any{
		"page":       p.Page,
		"size":       p.Size,
		"markAsRead": p.MarkAsRead,
	})
	out := &MessagesPage{}
	if err := c.doJSON(ctx, http.MethodGet, path, query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// SendMessageParams is the request body for sending a chat message. At least
// one of Text or MediaUUIDs should be supplied; Price marks the content
// pay-to-view.
type SendMessageParams struct {
	Text         *string  `json:"text,omitempty"`
	MediaUUIDs   []string `json:"mediaUuids,omitempty"`
	Price        *float64 `json:"price,omitempty"`
	TemplateUUID *string  `json:"templateUuid,omitempty"`
}

// SendMessageResult is the response from sending a message.
type SendMessageResult struct {
	MessageUUID string `json:"messageUuid"`
}

// SendMessage publishes a message into an existing chat conversation with the
// counterpart identified by userUUID. This is the chat publish operation the
// Inoue AI platform drives.
//
// POST /chats/{userUuid}/message — scope: write:chat.
func (c *Client) SendMessage(
	ctx context.Context, userUUID string, p SendMessageParams,
) (*SendMessageResult, error) {
	if userUUID == "" {
		return nil, errors.New("fanvue: SendMessage requires a non-empty user UUID")
	}
	if p.Text == nil && len(p.MediaUUIDs) == 0 {
		return nil, errors.New("fanvue: SendMessage requires Text or MediaUUIDs")
	}
	path := "/chats/" + url.PathEscape(userUUID) + "/message"
	out := &SendMessageResult{}
	if err := c.doJSON(ctx, http.MethodPost, path, nil, p, out); err != nil {
		return nil, err
	}
	return out, nil
}

// SmartListID enumerates the system-managed (smart) chat lists Fanvue exposes.
// These same identifiers are used both as the {uuid} path segment for
// GetSmartListMembers and as repeated smartListIds query values on ListChats.
type SmartListID string

const (
	// SmartListSubscribers is every active subscriber.
	SmartListSubscribers SmartListID = "subscribers"
	// SmartListAutoRenewing is subscribers whose subscription auto-renews.
	SmartListAutoRenewing SmartListID = "auto_renewing"
	// SmartListNonRenewing is subscribers who have cancelled auto-renew.
	SmartListNonRenewing SmartListID = "non_renewing"
	// SmartListFollowers is every follower (free, non-subscribing).
	SmartListFollowers SmartListID = "followers"
	// SmartListFreeTrialSubscribers is subscribers currently in a free trial.
	SmartListFreeTrialSubscribers SmartListID = "free_trial_subscribers"
	// SmartListExpiredSubscribers is users whose subscription has lapsed.
	SmartListExpiredSubscribers SmartListID = "expired_subscribers"
	// SmartListSpentMoreThan50 is users who have spent more than $50.
	SmartListSpentMoreThan50 SmartListID = "spent_more_than_50"
	// SmartListMuted is chats the creator has muted.
	SmartListMuted SmartListID = "muted"
)

// ChatFilter enumerates the filters accepted by ListChats. Multiple filters may
// be combined; they are sent as repeated filter query values.
type ChatFilter string

const (
	// ChatFilterUnread restricts to chats with unread messages.
	ChatFilterUnread ChatFilter = "unread"
	// ChatFilterSubscribers restricts to chats with subscribers.
	ChatFilterSubscribers ChatFilter = "subscribers"
	// ChatFilterFollowers restricts to chats with followers.
	ChatFilterFollowers ChatFilter = "followers"
	// ChatFilterOnline restricts to counterparts currently online.
	ChatFilterOnline ChatFilter = "online"
	// ChatFilterRecentSubscribers restricts to recently subscribed users.
	ChatFilterRecentSubscribers ChatFilter = "recent_subscribers"
	// ChatFilterNotAnswered restricts to chats awaiting a reply.
	ChatFilterNotAnswered ChatFilter = "not_answered"
	// ChatFilterSpentMoreThan50 restricts to users who spent more than $50.
	ChatFilterSpentMoreThan50 ChatFilter = "spent_more_than_50"
	// ChatFilterOnFreeTrial restricts to users on a free trial.
	ChatFilterOnFreeTrial ChatFilter = "on_free_trial"
	// ChatFilterHasTipped restricts to users who have tipped.
	ChatFilterHasTipped ChatFilter = "has_tipped"
	// ChatFilterSpenders restricts to users who have spent money.
	ChatFilterSpenders ChatFilter = "spenders"
	// ChatFilterExcludeCreators excludes other creators from the results.
	ChatFilterExcludeCreators ChatFilter = "exclude_creators"
	// ChatFilterSubscribedTo restricts to creators the user is subscribed to.
	ChatFilterSubscribedTo ChatFilter = "subscribed_to"
	// ChatFilterNotMuted excludes muted chats.
	ChatFilterNotMuted ChatFilter = "not_muted"
	// ChatFilterArchived restricts to archived chats.
	ChatFilterArchived ChatFilter = "archived"
)

// ChatSortBy enumerates the ordering options accepted by ListChats.
type ChatSortBy string

const (
	// SortMostRecentMessages orders by most recent message activity.
	SortMostRecentMessages ChatSortBy = "most_recent_messages"
	// SortOnlineNow orders by counterparts currently online.
	SortOnlineNow ChatSortBy = "online_now"
	// SortMostUnansweredChats orders by chats with the most unanswered messages.
	SortMostUnansweredChats ChatSortBy = "most_unanswered_chats"
)

// ChatLastMessage is a compact summary of the most recent message in a chat, as
// returned in a ListChats entry. Nullable fields use pointers so a JSON null is
// distinguishable from a zero value.
type ChatLastMessage struct {
	UUID         string  `json:"uuid"`
	Type         string  `json:"type"`
	Text         *string `json:"text"`
	HasMedia     *bool   `json:"hasMedia"`
	MediaType    *string `json:"mediaType"`
	SenderUUID   string  `json:"senderUuid"`
	SentAt       *string `json:"sentAt"`
	SentByUserID *string `json:"sentByUserId"`
}

// ChatUser is the chat counterpart summary returned in a ListChats entry.
type ChatUser struct {
	UUID         string  `json:"uuid"`
	Handle       string  `json:"handle"`
	DisplayName  string  `json:"displayName"`
	Nickname     *string `json:"nickname"`
	AvatarURL    *string `json:"avatarUrl"`
	IsTopSpender bool    `json:"isTopSpender"`
	RegisteredAt string  `json:"registeredAt"`
}

// Chat is one entry in a page of the authenticated creator's chats.
type Chat struct {
	User                ChatUser         `json:"user"`
	LastMessage         *ChatLastMessage `json:"lastMessage"`
	LastMessageAt       *string          `json:"lastMessageAt"`
	CreatedAt           *string          `json:"createdAt"`
	IsMuted             bool             `json:"isMuted"`
	IsRead              bool             `json:"isRead"`
	UnreadMessagesCount float64          `json:"unreadMessagesCount"`
}

// ChatsPage is one page of the authenticated creator's chats.
type ChatsPage struct {
	Data       []Chat     `json:"data"`
	Pagination Pagination `json:"pagination"`
}

// ListChatsParams configures a page request to GET /chats. All fields are
// optional; unset pointers and empty slices are omitted from the query string.
type ListChatsParams struct {
	Page         *int
	Size         *int
	CustomListID *string
	SmartListIDs []SmartListID
	Filter       []ChatFilter
	Search       *string
	SortBy       *ChatSortBy
}

// ListChats returns one page of the authenticated creator's chats.
//
// GET /chats — scope: read:chat.
func (c *Client) ListChats(ctx context.Context, p ListChatsParams) (*ChatsPage, error) {
	query := encodeQuery(map[string]any{
		"page":         p.Page,
		"size":         p.Size,
		"customListId": p.CustomListID,
		"smartListIds": smartListIDsToStrings(p.SmartListIDs),
		"filter":       chatFiltersToStrings(p.Filter),
		"search":       p.Search,
		"sortBy":       chatSortByToString(p.SortBy),
	})
	out := &ChatsPage{}
	if err := c.doJSON(ctx, http.MethodGet, "/chats", query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateChatParams is the POST /chats request body. UserUUID is required and
// identifies the counterpart the new chat conversation is opened with.
//
// RawBody, when non-nil, is sent verbatim and takes precedence over the typed
// field, mirroring the Python SDK's opaque-body signature.
type CreateChatParams struct {
	UserUUID string  `json:"userUuid"`
	RawBody  RawBody `json:"-"`
}

// CreateChatResult is the response from POST /chats.
type CreateChatResult struct {
	Message string `json:"message"`
}

// CreateChat opens a new chat conversation with the user identified by
// UserUUID.
//
// POST /chats — scope: write:chat.
func (c *Client) CreateChat(ctx context.Context, p CreateChatParams) (*CreateChatResult, error) {
	if p.RawBody == nil && p.UserUUID == "" {
		return nil, errors.New("fanvue: CreateChat requires a non-empty UserUUID")
	}
	out := &CreateChatResult{}
	if err := c.doJSON(ctx, http.MethodPost, "/chats", nil, chatBody(p, p.RawBody), out); err != nil {
		return nil, err
	}
	return out, nil
}

// UpdateChatParams is the PATCH /chats/{userUuid} request body. Every field is
// optional: a nil pointer is omitted so the property is left unchanged. Nickname
// is capped at 30 characters server-side.
//
// RawBody, when non-nil, is sent verbatim and takes precedence over the typed
// fields, mirroring the Python SDK's opaque-body signature.
type UpdateChatParams struct {
	IsRead   *bool   `json:"isRead,omitempty"`
	IsMuted  *bool   `json:"isMuted,omitempty"`
	Nickname *string `json:"nickname,omitempty"`
	RawBody  RawBody `json:"-"`
}

// UpdateChat updates properties of the chat conversation with the counterpart
// identified by userUUID (read status, mute status, or nickname). The endpoint
// returns no body.
//
// PATCH /chats/{userUuid} — scope: write:chat.
func (c *Client) UpdateChat(ctx context.Context, userUUID string, p UpdateChatParams) error {
	if userUUID == "" {
		return errors.New("fanvue: UpdateChat requires a non-empty user UUID")
	}
	path := "/chats/" + url.PathEscape(userUUID)
	return c.doJSON(ctx, http.MethodPatch, path, nil, chatBody(p, p.RawBody), nil)
}

// UnreadNotificationCounts breaks down the unread notification totals by type.
type UnreadNotificationCounts struct {
	NewFollower    float64 `json:"newFollower"`
	NewPostComment float64 `json:"newPostComment"`
	NewPostLike    float64 `json:"newPostLike"`
	NewPromotion   float64 `json:"newPromotion"`
	NewPurchase    float64 `json:"newPurchase"`
	NewSubscriber  float64 `json:"newSubscriber"`
	NewTip         float64 `json:"newTip"`
}

// UnreadChatsCount is the aggregate unread state for the authenticated user:
// unread chats, unread messages, and a per-type notification breakdown.
type UnreadChatsCount struct {
	UnreadChatsCount    float64                  `json:"unreadChatsCount"`
	UnreadMessagesCount float64                  `json:"unreadMessagesCount"`
	UnreadNotifications UnreadNotificationCounts `json:"unreadNotifications"`
}

// GetUnreadChatsCount returns the unread chats, messages, and notifications
// count for the authenticated user.
//
// GET /chats/unread — scope: read:chat.
func (c *Client) GetUnreadChatsCount(ctx context.Context) (*UnreadChatsCount, error) {
	out := &UnreadChatsCount{}
	if err := c.doJSON(ctx, http.MethodGet, "/chats/unread", nil, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// BatchStatusParams is the POST /chats/statuses request body. UserUUIDs is
// required (1–100 UUIDs) and identifies the counterparts whose online status to
// resolve.
//
// RawBody, when non-nil, is sent verbatim and takes precedence over the typed
// field, mirroring the Python SDK's opaque-body signature.
type BatchStatusParams struct {
	UserUUIDs []string `json:"userUuids"`
	RawBody   RawBody  `json:"-"`
}

// UserStatus is a single user's online status, keyed by user UUID in the
// GetBatchStatuses response. LastSeenAt is nil when the user has online
// visibility disabled or has never been seen.
type UserStatus struct {
	IsOnline   bool    `json:"isOnline"`
	LastSeenAt *string `json:"lastSeenAt"`
}

// GetBatchStatuses returns the online status and last-seen timestamp for
// multiple users in a single request, keyed by user UUID. At most 100 UUIDs may
// be supplied per call.
//
// POST /chats/statuses — scope: read:chat.
func (c *Client) GetBatchStatuses(ctx context.Context, p BatchStatusParams) (map[string]UserStatus, error) {
	if p.RawBody == nil && len(p.UserUUIDs) == 0 {
		return nil, errors.New("fanvue: GetBatchStatuses requires at least one user UUID")
	}
	out := map[string]UserStatus{}
	if err := c.doJSON(ctx, http.MethodPost, "/chats/statuses", nil, chatBody(p, p.RawBody), &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CustomList is one entry in a page of the authenticated creator's custom chat
// lists.
type CustomList struct {
	UUID         string  `json:"uuid"`
	Name         string  `json:"name"`
	MembersCount float64 `json:"membersCount"`
	CreatedAt    *string `json:"createdAt"`
}

// CustomListsPage is one page of custom lists.
type CustomListsPage struct {
	Data       []CustomList `json:"data"`
	Pagination Pagination   `json:"pagination"`
}

// GetCustomListsParams configures a page request to GET /chats/lists/custom.
type GetCustomListsParams struct {
	Page   *int
	Size   *int
	Search *string
}

// GetCustomLists returns one page of the authenticated creator's custom chat
// lists.
//
// GET /chats/lists/custom — scope: read:chat.
func (c *Client) GetCustomLists(ctx context.Context, p GetCustomListsParams) (*CustomListsPage, error) {
	query := encodeQuery(map[string]any{
		"page":   p.Page,
		"size":   p.Size,
		"search": p.Search,
	})
	out := &CustomListsPage{}
	if err := c.doJSON(ctx, http.MethodGet, "/chats/lists/custom", query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateCustomListParams is the POST /chats/lists/custom request body. Name is
// required (1–100 characters).
//
// RawBody, when non-nil, is sent verbatim and takes precedence over the typed
// field, mirroring the Python SDK's opaque-body signature.
type CreateCustomListParams struct {
	Name    string  `json:"name"`
	RawBody RawBody `json:"-"`
}

// CreateCustomListResult is the response from POST /chats/lists/custom.
type CreateCustomListResult struct {
	UUID      string  `json:"uuid"`
	Name      string  `json:"name"`
	CreatedAt *string `json:"createdAt"`
}

// CreateCustomList creates a new custom list for organizing contacts.
//
// POST /chats/lists/custom — scope: write:chat.
func (c *Client) CreateCustomList(ctx context.Context, p CreateCustomListParams) (*CreateCustomListResult, error) {
	if p.RawBody == nil && p.Name == "" {
		return nil, errors.New("fanvue: CreateCustomList requires a non-empty Name")
	}
	out := &CreateCustomListResult{}
	if err := c.doJSON(ctx, http.MethodPost, "/chats/lists/custom", nil, chatBody(p, p.RawBody), out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListMember is a single member of a custom or smart list.
type ListMember struct {
	UUID        string `json:"uuid"`
	Handle      string `json:"handle"`
	DisplayName string `json:"displayName"`
	IsCreator   bool   `json:"isCreator"`
}

// ListMembersPage is one page of members in a custom or smart list.
type ListMembersPage struct {
	Data       []ListMember `json:"data"`
	Pagination Pagination   `json:"pagination"`
}

// GetCustomListMembersParams configures a page request to
// GET /chats/lists/custom/{uuid}.
type GetCustomListMembersParams struct {
	Page *int
	Size *int
}

// GetCustomListMembers returns one page of members in the custom list
// identified by uuid.
//
// GET /chats/lists/custom/{uuid} — scope: read:chat read:fan.
func (c *Client) GetCustomListMembers(ctx context.Context, uuid string, p GetCustomListMembersParams) (*ListMembersPage, error) {
	if uuid == "" {
		return nil, errors.New("fanvue: GetCustomListMembers requires a non-empty list UUID")
	}
	path := "/chats/lists/custom/" + url.PathEscape(uuid)
	query := encodeQuery(map[string]any{"page": p.Page, "size": p.Size})
	out := &ListMembersPage{}
	if err := c.doJSON(ctx, http.MethodGet, path, query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// UpdateCustomListParams is the PATCH /chats/lists/custom/{uuid} request body.
// Name is required (1–100 characters) and renames the list.
//
// RawBody, when non-nil, is sent verbatim and takes precedence over the typed
// field, mirroring the Python SDK's opaque-body signature.
type UpdateCustomListParams struct {
	Name    string  `json:"name"`
	RawBody RawBody `json:"-"`
}

// UpdateCustomList renames an existing custom list. The endpoint returns no
// body.
//
// PATCH /chats/lists/custom/{uuid} — scope: write:chat.
func (c *Client) UpdateCustomList(ctx context.Context, uuid string, p UpdateCustomListParams) error {
	if uuid == "" {
		return errors.New("fanvue: UpdateCustomList requires a non-empty list UUID")
	}
	if p.RawBody == nil && p.Name == "" {
		return errors.New("fanvue: UpdateCustomList requires a non-empty Name")
	}
	path := "/chats/lists/custom/" + url.PathEscape(uuid)
	return c.doJSON(ctx, http.MethodPatch, path, nil, chatBody(p, p.RawBody), nil)
}

// DeleteCustomList deletes a custom list. All members are removed from the list.
// The endpoint returns no body.
//
// DELETE /chats/lists/custom/{uuid} — scope: write:chat.
func (c *Client) DeleteCustomList(ctx context.Context, uuid string) error {
	if uuid == "" {
		return errors.New("fanvue: DeleteCustomList requires a non-empty list UUID")
	}
	path := "/chats/lists/custom/" + url.PathEscape(uuid)
	return c.doJSON(ctx, http.MethodDelete, path, nil, nil, nil)
}

// AddMembersToCustomListParams is the POST /chats/lists/custom/{uuid}/members
// request body. UserUUIDs is required (1–100 UUIDs); users already on the list
// are skipped.
//
// RawBody, when non-nil, is sent verbatim and takes precedence over the typed
// field, mirroring the Python SDK's opaque-body signature.
type AddMembersToCustomListParams struct {
	UserUUIDs []string `json:"userUuids"`
	RawBody   RawBody  `json:"-"`
}

// AddMembersToCustomListResult is the response from adding members to a custom
// list: how many users were newly added versus skipped (already present).
type AddMembersToCustomListResult struct {
	Added   float64 `json:"added"`
	Skipped float64 `json:"skipped"`
}

// AddMembersToCustomList adds one or more users to the custom list identified by
// uuid. Existing members are skipped. At most 100 UUIDs may be supplied per
// call.
//
// POST /chats/lists/custom/{uuid}/members — scope: write:chat.
func (c *Client) AddMembersToCustomList(ctx context.Context, uuid string, p AddMembersToCustomListParams) (*AddMembersToCustomListResult, error) {
	if uuid == "" {
		return nil, errors.New("fanvue: AddMembersToCustomList requires a non-empty list UUID")
	}
	if p.RawBody == nil && len(p.UserUUIDs) == 0 {
		return nil, errors.New("fanvue: AddMembersToCustomList requires at least one user UUID")
	}
	path := "/chats/lists/custom/" + url.PathEscape(uuid) + "/members"
	out := &AddMembersToCustomListResult{}
	if err := c.doJSON(ctx, http.MethodPost, path, nil, chatBody(p, p.RawBody), out); err != nil {
		return nil, err
	}
	return out, nil
}

// RemoveMemberFromCustomList removes a single user from the custom list
// identified by uuid. The endpoint returns no body.
//
// DELETE /chats/lists/custom/{uuid}/members/{userUuid} — scope: write:chat.
func (c *Client) RemoveMemberFromCustomList(ctx context.Context, uuid, userUUID string) error {
	if uuid == "" {
		return errors.New("fanvue: RemoveMemberFromCustomList requires a non-empty list UUID")
	}
	if userUUID == "" {
		return errors.New("fanvue: RemoveMemberFromCustomList requires a non-empty user UUID")
	}
	path := "/chats/lists/custom/" + url.PathEscape(uuid) +
		"/members/" + url.PathEscape(userUUID)
	return c.doJSON(ctx, http.MethodDelete, path, nil, nil, nil)
}

// SmartList is a single system-managed chat list with its current member count.
type SmartList struct {
	UUID  SmartListID `json:"uuid"`
	Name  string      `json:"name"`
	Count float64     `json:"count"`
}

// GetSmartLists returns the system-managed (smart) chat lists available to the
// authenticated creator. The endpoint returns a bare array, not a paginated
// envelope.
//
// GET /chats/lists/smart — scope: read:chat.
func (c *Client) GetSmartLists(ctx context.Context) ([]SmartList, error) {
	out := []SmartList{}
	if err := c.doJSON(ctx, http.MethodGet, "/chats/lists/smart", nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetSmartListMembersParams configures a page request to
// GET /chats/lists/smart/{uuid}.
type GetSmartListMembersParams struct {
	Page *int
	Size *int
}

// GetSmartListMembers returns one page of members in the smart list identified
// by id (one of the SmartList* constants).
//
// GET /chats/lists/smart/{uuid} — scope: read:chat.
func (c *Client) GetSmartListMembers(ctx context.Context, id SmartListID, p GetSmartListMembersParams) (*ListMembersPage, error) {
	if id == "" {
		return nil, errors.New("fanvue: GetSmartListMembers requires a non-empty smart list id")
	}
	path := "/chats/lists/smart/" + url.PathEscape(string(id))
	query := encodeQuery(map[string]any{"page": p.Page, "size": p.Size})
	out := &ListMembersPage{}
	if err := c.doJSON(ctx, http.MethodGet, path, query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// TemplateMessage is a saved chat message template. Nullable fields use pointers
// so a JSON null is distinguishable from a zero value; Price is in cents.
type TemplateMessage struct {
	UUID       string   `json:"uuid"`
	Text       *string  `json:"text"`
	FolderName *string  `json:"folderName"`
	MediaUUIDs []string `json:"mediaUuids"`
	Price      *float64 `json:"price"`
}

// TemplateMessagesPage is one page of saved chat message templates.
type TemplateMessagesPage struct {
	Data       []TemplateMessage `json:"data"`
	Pagination Pagination        `json:"pagination"`
}

// ListTemplateMessagesParams configures a page request to GET /chats/templates.
type ListTemplateMessagesParams struct {
	Page       *int
	Size       *int
	FolderName *string
}

// ListTemplateMessages returns one page of the authenticated creator's saved
// chat message templates.
//
// GET /chats/templates — scope: read:chat.
func (c *Client) ListTemplateMessages(ctx context.Context, p ListTemplateMessagesParams) (*TemplateMessagesPage, error) {
	query := encodeQuery(map[string]any{
		"page":       p.Page,
		"size":       p.Size,
		"folderName": p.FolderName,
	})
	out := &TemplateMessagesPage{}
	if err := c.doJSON(ctx, http.MethodGet, "/chats/templates", query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetTemplateMessage fetches a single saved chat message template by UUID.
//
// GET /chats/templates/{templateUuid} — scope: read:chat.
func (c *Client) GetTemplateMessage(ctx context.Context, templateUUID string) (*TemplateMessage, error) {
	if templateUUID == "" {
		return nil, errors.New("fanvue: GetTemplateMessage requires a non-empty template UUID")
	}
	path := "/chats/templates/" + url.PathEscape(templateUUID)
	out := &TemplateMessage{}
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// chatBody resolves the body sent for a chats write request. When raw is
// supplied it is sent verbatim (preserving explicit JSON nulls and any
// additional fields, matching the Python SDK's opaque Mapping body); otherwise
// the typed params value is marshalled normally.
func chatBody(typed any, raw RawBody) any {
	if raw != nil {
		return raw
	}
	return typed
}

// smartListIDsToStrings flattens a slice of SmartListID into the []string form
// encodeQuery renders as repeated query keys. It returns nil for an empty input
// so the parameter is omitted entirely.
func smartListIDsToStrings(ids []SmartListID) []string {
	if len(ids) == 0 {
		return nil
	}
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = string(id)
	}
	return out
}

// chatFiltersToStrings flattens a slice of ChatFilter into the []string form
// encodeQuery renders as repeated query keys. It returns nil for an empty input
// so the parameter is omitted entirely.
func chatFiltersToStrings(filters []ChatFilter) []string {
	if len(filters) == 0 {
		return nil
	}
	out := make([]string, len(filters))
	for i, f := range filters {
		out[i] = string(f)
	}
	return out
}

// chatSortByToString converts an optional ChatSortBy into the *string form
// encodeQuery expects, returning nil when unset so the parameter is omitted.
func chatSortByToString(sortBy *ChatSortBy) *string {
	if sortBy == nil {
		return nil
	}
	s := string(*sortBy)
	return &s
}
