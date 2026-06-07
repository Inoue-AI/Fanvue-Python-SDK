package fanvue

import (
	"context"
	"errors"
	"net/http"
	"net/url"
)

// CreatorChatLastMessage is a compact summary of the most recent message in a
// managed creator's chat, as returned in a ListCreatorChats entry. Nullable
// fields use pointers so a JSON null is distinguishable from a zero value. The
// shape is byte-identical to the self-scoped ChatLastMessage.
type CreatorChatLastMessage struct {
	UUID         string  `json:"uuid"`
	Type         string  `json:"type"`
	Text         *string `json:"text"`
	HasMedia     *bool   `json:"hasMedia"`
	MediaType    *string `json:"mediaType"`
	SenderUUID   string  `json:"senderUuid"`
	SentAt       *string `json:"sentAt"`
	SentByUserID *string `json:"sentByUserId"`
}

// CreatorChatUser is the chat counterpart summary returned in a ListCreatorChats
// entry. The shape is byte-identical to the self-scoped ChatUser.
type CreatorChatUser struct {
	UUID         string  `json:"uuid"`
	Handle       string  `json:"handle"`
	DisplayName  string  `json:"displayName"`
	Nickname     *string `json:"nickname"`
	AvatarURL    *string `json:"avatarUrl"`
	IsTopSpender bool    `json:"isTopSpender"`
	RegisteredAt string  `json:"registeredAt"`
}

// CreatorChat is one entry in a page of a managed creator's chats.
type CreatorChat struct {
	User                CreatorChatUser         `json:"user"`
	LastMessage         *CreatorChatLastMessage `json:"lastMessage"`
	LastMessageAt       *string                 `json:"lastMessageAt"`
	CreatedAt           *string                 `json:"createdAt"`
	IsMuted             bool                    `json:"isMuted"`
	IsRead              bool                    `json:"isRead"`
	UnreadMessagesCount float64                 `json:"unreadMessagesCount"`
}

// CreatorChatsPage is one page of a managed creator's chats, returned by
// GET /creators/{creatorUserUuid}/chats.
type CreatorChatsPage struct {
	Data       []CreatorChat `json:"data"`
	Pagination Pagination    `json:"pagination"`
}

// ListCreatorChatsParams configures a page request to
// GET /creators/{creatorUserUuid}/chats. All fields are optional; unset pointers
// are omitted from the query string.
type ListCreatorChatsParams struct {
	Page *int
	Size *int
}

// ListCreatorChats returns one page of a managed creator's chats.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// GET /creators/{creatorUserUuid}/chats — scopes: read:chat, read:creator.
func (c *Client) ListCreatorChats(
	ctx context.Context, creatorUserUUID string, p ListCreatorChatsParams,
) (*CreatorChatsPage, error) {
	if creatorUserUUID == "" {
		return nil, errors.New("fanvue: ListCreatorChats requires a non-empty creator user UUID")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) + "/chats"
	query := encodeQuery(map[string]any{"page": p.Page, "size": p.Size})
	out := &CreatorChatsPage{}
	if err := c.doJSON(ctx, http.MethodGet, path, query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateCreatorChatParams is the POST /creators/{creatorUserUuid}/chats request
// body. UserUUID is required and identifies the counterpart the new chat
// conversation is opened with.
//
// RawBody, when non-nil, is sent verbatim and takes precedence over the typed
// field, mirroring the Python SDK's opaque Mapping[str, Any] body.
type CreateCreatorChatParams struct {
	UserUUID string  `json:"userUuid"`
	RawBody  RawBody `json:"-"`
}

// CreateCreatorChatResult is the response from
// POST /creators/{creatorUserUuid}/chats.
type CreateCreatorChatResult struct {
	Message string `json:"message"`
}

// CreateCreatorChat opens a new chat conversation, on behalf of a managed
// creator, with the user identified by UserUUID.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// POST /creators/{creatorUserUuid}/chats — scopes: write:chat, write:creator.
func (c *Client) CreateCreatorChat(
	ctx context.Context, creatorUserUUID string, p CreateCreatorChatParams,
) (*CreateCreatorChatResult, error) {
	if creatorUserUUID == "" {
		return nil, errors.New("fanvue: CreateCreatorChat requires a non-empty creator user UUID")
	}
	if p.RawBody == nil && p.UserUUID == "" {
		return nil, errors.New("fanvue: CreateCreatorChat requires a non-empty UserUUID")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) + "/chats"
	out := &CreateCreatorChatResult{}
	if err := c.doJSON(ctx, http.MethodPost, path, nil, chatBody(p, p.RawBody), out); err != nil {
		return nil, err
	}
	return out, nil
}

// UpdateCreatorChatParams is the PATCH /creators/{creatorUserUuid}/chats/{userUuid}
// request body. Every typed field is optional: a nil pointer is omitted so the
// property is left unchanged. Nickname is capped at 30 characters server-side.
//
// RawBody, when non-nil, is sent verbatim and takes precedence over the typed
// fields, mirroring the Python SDK's opaque Mapping[str, Any] body.
type UpdateCreatorChatParams struct {
	IsRead   *bool   `json:"isRead,omitempty"`
	IsMuted  *bool   `json:"isMuted,omitempty"`
	Nickname *string `json:"nickname,omitempty"`
	RawBody  RawBody `json:"-"`
}

// UpdateCreatorChat updates properties of a managed creator's chat with the
// counterpart identified by userUUID (read status, mute status, or nickname).
// The endpoint returns no body.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// PATCH /creators/{creatorUserUuid}/chats/{userUuid} — scopes: write:chat, write:creator.
func (c *Client) UpdateCreatorChat(
	ctx context.Context, creatorUserUUID, userUUID string, p UpdateCreatorChatParams,
) error {
	if creatorUserUUID == "" {
		return errors.New("fanvue: UpdateCreatorChat requires a non-empty creator user UUID")
	}
	if userUUID == "" {
		return errors.New("fanvue: UpdateCreatorChat requires a non-empty user UUID")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) +
		"/chats/" + url.PathEscape(userUUID)
	return c.doJSON(ctx, http.MethodPatch, path, nil, chatBody(p, p.RawBody), nil)
}

// CreatorMessagePricingUSD holds the USD price (in cents) of a paid creator chat
// message.
type CreatorMessagePricingUSD struct {
	Price float64 `json:"price"`
}

// CreatorMessagePricing wraps the per-currency pricing of a paid creator chat
// message.
type CreatorMessagePricing struct {
	USD CreatorMessagePricingUSD `json:"USD"`
}

// CreatorMessageRecipient is the recipient summary on a creator chat message.
// Both fields are nil when the account is no longer resolvable.
type CreatorMessageRecipient struct {
	Handle *string `json:"handle"`
	UUID   *string `json:"uuid"`
}

// CreatorMessageSender is the sender summary on a creator chat message.
type CreatorMessageSender struct {
	Handle string `json:"handle"`
	UUID   string `json:"uuid"`
}

// CreatorMessage is a single message in a managed creator's chat conversation.
// Nullable fields use pointers so a JSON null is distinguishable from a zero
// value; MediaType uses the four-value ChatMediaType (the responses here never
// carry the synthetic "unknown" variant).
type CreatorMessage struct {
	UUID         string                  `json:"uuid"`
	Type         string                  `json:"type"`
	Text         *string                 `json:"text"`
	HasMedia     *bool                   `json:"hasMedia"`
	MediaType    *ChatMediaType          `json:"mediaType"`
	MediaUUIDs   []string                `json:"mediaUuids"`
	IsRead       bool                    `json:"isRead"`
	Pricing      *CreatorMessagePricing  `json:"pricing"`
	PurchasedAt  *string                 `json:"purchasedAt"`
	Recipient    CreatorMessageRecipient `json:"recipient"`
	Sender       CreatorMessageSender    `json:"sender"`
	SentAt       *string                 `json:"sentAt"`
	SentByUserID *string                 `json:"sentByUserId"`
}

// CreatorMessagesPage is one page of messages from a managed creator's chat
// conversation, returned by
// GET /creators/{creatorUserUuid}/chats/{userUuid}/messages.
type CreatorMessagesPage struct {
	Data       []CreatorMessage `json:"data"`
	Pagination Pagination       `json:"pagination"`
}

// ListCreatorMessagesParams configures a page request to
// GET /creators/{creatorUserUuid}/chats/{userUuid}/messages. All fields are
// optional. MarkAsRead, when non-nil, is sent as the Literal "true"/"false"
// string the API expects and marks the fetched messages as read.
type ListCreatorMessagesParams struct {
	Page       *int
	Size       *int
	MarkAsRead *bool
}

// ListCreatorMessages returns one page of messages between a managed creator and
// the user identified by userUUID. Messages are newest-first.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// GET /creators/{creatorUserUuid}/chats/{userUuid}/messages — scopes: read:chat, read:creator.
func (c *Client) ListCreatorMessages(
	ctx context.Context, creatorUserUUID, userUUID string, p ListCreatorMessagesParams,
) (*CreatorMessagesPage, error) {
	if creatorUserUUID == "" {
		return nil, errors.New("fanvue: ListCreatorMessages requires a non-empty creator user UUID")
	}
	if userUUID == "" {
		return nil, errors.New("fanvue: ListCreatorMessages requires a non-empty user UUID")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) +
		"/chats/" + url.PathEscape(userUUID) + "/messages"
	query := encodeQuery(map[string]any{
		"page":       p.Page,
		"size":       p.Size,
		"markAsRead": boolToLiteralString(p.MarkAsRead),
	})
	out := &CreatorMessagesPage{}
	if err := c.doJSON(ctx, http.MethodGet, path, query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// SendCreatorMessageParams is the request body for sending a message as a managed
// creator. At least one of Text or MediaUUIDs should be supplied; Price marks the
// content pay-to-view.
//
// RawBody, when non-nil, is sent verbatim and takes precedence over the typed
// fields, mirroring the Python SDK's opaque Mapping[str, Any] body.
type SendCreatorMessageParams struct {
	Text         *string  `json:"text,omitempty"`
	MediaUUIDs   []string `json:"mediaUuids,omitempty"`
	Price        *float64 `json:"price,omitempty"`
	TemplateUUID *string  `json:"templateUuid,omitempty"`
	RawBody      RawBody  `json:"-"`
}

// SendCreatorMessageResult is the response from sending a creator chat message.
type SendCreatorMessageResult struct {
	MessageUUID string `json:"messageUuid"`
}

// SendCreatorMessage publishes a message, on behalf of a managed creator, into an
// existing chat conversation with the user identified by userUUID.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// POST /creators/{creatorUserUuid}/chats/{userUuid}/message — scopes: write:chat, write:creator.
func (c *Client) SendCreatorMessage(
	ctx context.Context, creatorUserUUID, userUUID string, p SendCreatorMessageParams,
) (*SendCreatorMessageResult, error) {
	if creatorUserUUID == "" {
		return nil, errors.New("fanvue: SendCreatorMessage requires a non-empty creator user UUID")
	}
	if userUUID == "" {
		return nil, errors.New("fanvue: SendCreatorMessage requires a non-empty user UUID")
	}
	if p.RawBody == nil && p.Text == nil && len(p.MediaUUIDs) == 0 {
		return nil, errors.New("fanvue: SendCreatorMessage requires Text or MediaUUIDs")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) +
		"/chats/" + url.PathEscape(userUUID) + "/message"
	out := &SendCreatorMessageResult{}
	if err := c.doJSON(ctx, http.MethodPost, path, nil, chatBody(p, p.RawBody), out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreatorMessageMediaError is a per-UUID failure entry returned by
// GetCreatorMessageMediaByUUIDs when a requested media UUID could not be
// resolved.
type CreatorMessageMediaError struct {
	// Code is "NOT_IN_MESSAGE" or "INTERNAL".
	Code      string `json:"code"`
	MediaUUID string `json:"mediaUuid"`
	Message   string `json:"message"`
}

// CreatorMessageMedia is a single resolved media item attached to a managed
// creator's chat message. Nullable fields use pointers so a JSON null is
// distinguishable from a zero value; Variants is nil when the caller is not
// entitled to any rendition.
type CreatorMessageMedia struct {
	CreatedAt   *string        `json:"created_at"`
	MediaType   ChatMediaType  `json:"mediaType"`
	MessageUUID string         `json:"messageUuid"`
	Name        *string        `json:"name"`
	OwnerUUID   string         `json:"ownerUuid"`
	SentAt      *string        `json:"sentAt"`
	UUID        string         `json:"uuid"`
	Variants    []MediaVariant `json:"variants,omitempty"`
}

// CreatorMessageMediaByUUIDs is the response from
// GetCreatorMessageMediaByUUIDs. Results maps each requested media UUID to its
// resolved item, or to a JSON null (nil pointer) when that UUID failed; the
// parallel Errors slice carries the failure detail.
type CreatorMessageMediaByUUIDs struct {
	Errors  []CreatorMessageMediaError      `json:"errors"`
	Results map[string]*CreatorMessageMedia `json:"results"`
}

// GetCreatorMessageMediaByUUIDsParams configures a request to
// GET /creators/{creatorUserUuid}/chats/{userUuid}/messages/{messageUuid}/media.
//
// MediaUUIDs is required and is a comma-separated list of media UUIDs to resolve,
// sent verbatim as the single mediaUuids query value — mirroring the Python SDK's
// str parameter exactly (it is NOT serialized as repeated keys). Variants is an
// optional comma-separated list of variant types to include.
type GetCreatorMessageMediaByUUIDsParams struct {
	MediaUUIDs string
	Variants   *string
}

// GetCreatorMessageMediaByUUIDs resolves one or more media UUIDs attached to a
// managed creator's chat message into their media items (with variant
// renditions), for the message identified by messageUUID within the conversation
// with userUUID. UUIDs that cannot be resolved appear in the response's Errors
// slice and as a null entry in Results.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// GET /creators/{creatorUserUuid}/chats/{userUuid}/messages/{messageUuid}/media —
// scopes: read:chat, read:creator.
func (c *Client) GetCreatorMessageMediaByUUIDs(
	ctx context.Context, creatorUserUUID, userUUID, messageUUID string, p GetCreatorMessageMediaByUUIDsParams,
) (*CreatorMessageMediaByUUIDs, error) {
	if creatorUserUUID == "" {
		return nil, errors.New("fanvue: GetCreatorMessageMediaByUUIDs requires a non-empty creator user UUID")
	}
	if userUUID == "" {
		return nil, errors.New("fanvue: GetCreatorMessageMediaByUUIDs requires a non-empty user UUID")
	}
	if messageUUID == "" {
		return nil, errors.New("fanvue: GetCreatorMessageMediaByUUIDs requires a non-empty message UUID")
	}
	if p.MediaUUIDs == "" {
		return nil, errors.New("fanvue: GetCreatorMessageMediaByUUIDs requires non-empty MediaUUIDs")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) +
		"/chats/" + url.PathEscape(userUUID) +
		"/messages/" + url.PathEscape(messageUUID) + "/media"
	query := encodeQuery(map[string]any{
		"mediaUuids": p.MediaUUIDs,
		"variants":   p.Variants,
	})
	out := &CreatorMessageMediaByUUIDs{}
	if err := c.doJSON(ctx, http.MethodGet, path, query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreatorChatMedia is one entry in a page of media exchanged within a managed
// creator's chat conversation. Nullable fields use pointers so a JSON null is
// distinguishable from a zero value; Variants is nil when the caller is not
// entitled to any rendition.
type CreatorChatMedia struct {
	CreatedAt   *string        `json:"created_at"`
	MediaType   ChatMediaType  `json:"mediaType"`
	MessageUUID string         `json:"messageUuid"`
	Name        *string        `json:"name"`
	OwnerUUID   string         `json:"ownerUuid"`
	SentAt      *string        `json:"sentAt"`
	UUID        string         `json:"uuid"`
	Variants    []MediaVariant `json:"variants,omitempty"`
}

// CreatorChatMediaPage is one cursor-paginated page of media from a managed
// creator's chat conversation. NextCursor is nil on the final page; pass it back
// as ListCreatorChatMediaParams.Cursor to fetch the next page.
type CreatorChatMediaPage struct {
	Data       []CreatorChatMedia `json:"data"`
	NextCursor *string            `json:"nextCursor"`
}

// ListCreatorChatMediaParams configures a page request to
// GET /creators/{creatorUserUuid}/chats/{userUuid}/media. All fields are
// optional. Cursor drives the opaque cursor-based pagination (unlike the
// page/size envelope used elsewhere); MediaType filters by kind.
type ListCreatorChatMediaParams struct {
	Cursor    *string
	MediaType *ChatMediaType
	Limit     *int
}

// ListCreatorChatMedia returns one cursor-paginated page of media exchanged
// within a managed creator's chat conversation with the user identified by
// userUUID.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// GET /creators/{creatorUserUuid}/chats/{userUuid}/media — scopes: read:chat, read:creator.
func (c *Client) ListCreatorChatMedia(
	ctx context.Context, creatorUserUUID, userUUID string, p ListCreatorChatMediaParams,
) (*CreatorChatMediaPage, error) {
	if creatorUserUUID == "" {
		return nil, errors.New("fanvue: ListCreatorChatMedia requires a non-empty creator user UUID")
	}
	if userUUID == "" {
		return nil, errors.New("fanvue: ListCreatorChatMedia requires a non-empty user UUID")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) +
		"/chats/" + url.PathEscape(userUUID) + "/media"
	query := encodeQuery(map[string]any{
		"cursor":    p.Cursor,
		"mediaType": chatMediaTypeToString(p.MediaType),
		"limit":     p.Limit,
	})
	out := &CreatorChatMediaPage{}
	if err := c.doJSON(ctx, http.MethodGet, path, query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreatorMassMessage is one entry in a page of a managed creator's mass messages.
// Amounts are in cents; nullable fields use pointers so a JSON null is
// distinguishable from a zero value.
type CreatorMassMessage struct {
	CreatedAt      *string           `json:"createdAt"`
	MediaUUIDs     []string          `json:"mediaUuids"`
	Price          *float64          `json:"price"`
	PublishedAt    *string           `json:"publishedAt"`
	PurchaseCount  float64           `json:"purchaseCount"`
	RecipientCount float64           `json:"recipientCount"`
	ScheduledAt    *string           `json:"scheduledAt"`
	Status         MassMessageStatus `json:"status"`
	Text           *string           `json:"text"`
	TotalRevenue   float64           `json:"totalRevenue"`
	UUID           string            `json:"uuid"`
	ViewCount      float64           `json:"viewCount"`
}

// CreatorMassMessagesPage is one page of a managed creator's mass messages,
// returned by GET /creators/{creatorUserUuid}/chats/mass-messages.
type CreatorMassMessagesPage struct {
	Data       []CreatorMassMessage `json:"data"`
	Pagination Pagination           `json:"pagination"`
}

// ListCreatorMassMessagesParams configures a page request to
// GET /creators/{creatorUserUuid}/chats/mass-messages. All fields are optional.
// IncludeDeleted, when non-nil, is sent as the Literal "true"/"false" string the
// API expects and includes soft-deleted mass messages in the result.
type ListCreatorMassMessagesParams struct {
	Page           *int
	Size           *int
	IncludeDeleted *bool
}

// ListCreatorMassMessages returns one page of a managed creator's mass messages.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// GET /creators/{creatorUserUuid}/chats/mass-messages — scopes: read:chat, read:creator.
func (c *Client) ListCreatorMassMessages(
	ctx context.Context, creatorUserUUID string, p ListCreatorMassMessagesParams,
) (*CreatorMassMessagesPage, error) {
	if creatorUserUUID == "" {
		return nil, errors.New("fanvue: ListCreatorMassMessages requires a non-empty creator user UUID")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) + "/chats/mass-messages"
	query := encodeQuery(map[string]any{
		"page":           p.Page,
		"size":           p.Size,
		"includeDeleted": boolToLiteralString(p.IncludeDeleted),
	})
	out := &CreatorMassMessagesPage{}
	if err := c.doJSON(ctx, http.MethodGet, path, query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// SendCreatorMassMessageParams is the
// POST /creators/{creatorUserUuid}/chats/mass-messages request body. The typed
// fields express the common shape; Price is in cents; ScheduledAt schedules the
// send for a future ISO-8601 timestamp.
//
// RawBody, when non-nil, is sent verbatim and takes precedence over the typed
// fields, mirroring the Python SDK's opaque Mapping[str, Any] body.
type SendCreatorMassMessageParams struct {
	Text         *string  `json:"text,omitempty"`
	MediaUUIDs   []string `json:"mediaUuids,omitempty"`
	Price        *float64 `json:"price,omitempty"`
	ScheduledAt  *string  `json:"scheduledAt,omitempty"`
	SmartListIDs []string `json:"smartListIds,omitempty"`
	CustomListID *string  `json:"customListId,omitempty"`
	RawBody      RawBody  `json:"-"`
}

// SendCreatorMassMessageResult is the response from
// POST /creators/{creatorUserUuid}/chats/mass-messages. CreatedAt is nil when the
// mass message is scheduled rather than sent immediately.
type SendCreatorMassMessageResult struct {
	CreatedAt      *string `json:"createdAt"`
	ID             string  `json:"id"`
	RecipientCount float64 `json:"recipientCount"`
}

// SendCreatorMassMessage sends (or schedules) a mass message, on behalf of a
// managed creator, to a list of that creator's contacts.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// POST /creators/{creatorUserUuid}/chats/mass-messages — scopes: write:chat, write:creator.
func (c *Client) SendCreatorMassMessage(
	ctx context.Context, creatorUserUUID string, p SendCreatorMassMessageParams,
) (*SendCreatorMassMessageResult, error) {
	if creatorUserUUID == "" {
		return nil, errors.New("fanvue: SendCreatorMassMessage requires a non-empty creator user UUID")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) + "/chats/mass-messages"
	out := &SendCreatorMassMessageResult{}
	if err := c.doJSON(ctx, http.MethodPost, path, nil, chatBody(p, p.RawBody), out); err != nil {
		return nil, err
	}
	return out, nil
}

// UpdateCreatorMassMessageParams is the
// PATCH /creators/{creatorUserUuid}/chats/mass-messages/{messageUuid} request
// body. Every typed field is optional: a nil pointer (or nil slice) is omitted so
// the field is left unchanged.
//
// RawBody, when non-nil, is sent verbatim and takes precedence over the typed
// fields, mirroring the Python SDK's opaque Mapping[str, Any] body.
type UpdateCreatorMassMessageParams struct {
	Text        *string  `json:"text,omitempty"`
	MediaUUIDs  []string `json:"mediaUuids,omitempty"`
	Price       *float64 `json:"price,omitempty"`
	ScheduledAt *string  `json:"scheduledAt,omitempty"`
	RawBody     RawBody  `json:"-"`
}

// UpdateCreatorMassMessage updates a managed creator's scheduled mass message
// identified by messageUUID (e.g. to revise its text, media, price, or scheduled
// send time). Only scheduled (not yet sent) mass messages can be updated. The
// endpoint returns no body.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// PATCH /creators/{creatorUserUuid}/chats/mass-messages/{messageUuid} —
// scopes: write:chat, write:creator.
func (c *Client) UpdateCreatorMassMessage(
	ctx context.Context, creatorUserUUID, messageUUID string, p UpdateCreatorMassMessageParams,
) error {
	if creatorUserUUID == "" {
		return errors.New("fanvue: UpdateCreatorMassMessage requires a non-empty creator user UUID")
	}
	if messageUUID == "" {
		return errors.New("fanvue: UpdateCreatorMassMessage requires a non-empty message UUID")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) +
		"/chats/mass-messages/" + url.PathEscape(messageUUID)
	return c.doJSON(ctx, http.MethodPatch, path, nil, chatBody(p, p.RawBody), nil)
}

// DeleteCreatorMassMessage deletes a managed creator's mass message identified by
// messageUUID. The endpoint returns no body.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// DELETE /creators/{creatorUserUuid}/chats/mass-messages/{messageUuid} —
// scopes: write:chat, write:creator.
func (c *Client) DeleteCreatorMassMessage(
	ctx context.Context, creatorUserUUID, messageUUID string,
) error {
	if creatorUserUUID == "" {
		return errors.New("fanvue: DeleteCreatorMassMessage requires a non-empty creator user UUID")
	}
	if messageUUID == "" {
		return errors.New("fanvue: DeleteCreatorMassMessage requires a non-empty message UUID")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) +
		"/chats/mass-messages/" + url.PathEscape(messageUUID)
	return c.doJSON(ctx, http.MethodDelete, path, nil, nil, nil)
}

// CreatorCustomList is one entry in a page of a managed creator's custom chat
// lists.
type CreatorCustomList struct {
	UUID         string  `json:"uuid"`
	Name         string  `json:"name"`
	MembersCount float64 `json:"membersCount"`
	CreatedAt    *string `json:"createdAt"`
}

// CreatorCustomListsPage is one page of a managed creator's custom chat lists,
// returned by GET /creators/{creatorUserUuid}/chats/lists/custom.
type CreatorCustomListsPage struct {
	Data       []CreatorCustomList `json:"data"`
	Pagination Pagination          `json:"pagination"`
}

// GetCreatorCustomListsParams configures a page request to
// GET /creators/{creatorUserUuid}/chats/lists/custom.
type GetCreatorCustomListsParams struct {
	Page *int
	Size *int
}

// GetCreatorCustomLists returns one page of a managed creator's custom chat
// lists.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// GET /creators/{creatorUserUuid}/chats/lists/custom — scopes: read:chat, read:creator.
func (c *Client) GetCreatorCustomLists(
	ctx context.Context, creatorUserUUID string, p GetCreatorCustomListsParams,
) (*CreatorCustomListsPage, error) {
	if creatorUserUUID == "" {
		return nil, errors.New("fanvue: GetCreatorCustomLists requires a non-empty creator user UUID")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) + "/chats/lists/custom"
	query := encodeQuery(map[string]any{"page": p.Page, "size": p.Size})
	out := &CreatorCustomListsPage{}
	if err := c.doJSON(ctx, http.MethodGet, path, query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreatorListMember is a single member of a managed creator's custom or smart
// list.
type CreatorListMember struct {
	UUID        string `json:"uuid"`
	Handle      string `json:"handle"`
	DisplayName string `json:"displayName"`
	IsCreator   bool   `json:"isCreator"`
}

// CreatorListMembersPage is one page of members in a managed creator's custom or
// smart list.
type CreatorListMembersPage struct {
	Data       []CreatorListMember `json:"data"`
	Pagination Pagination          `json:"pagination"`
}

// GetCreatorCustomListMembersParams configures a page request to
// GET /creators/{creatorUserUuid}/chats/lists/custom/{uuid}.
type GetCreatorCustomListMembersParams struct {
	Page *int
	Size *int
}

// GetCreatorCustomListMembers returns one page of members in the managed
// creator's custom list identified by listUUID.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// GET /creators/{creatorUserUuid}/chats/lists/custom/{uuid} — scopes: read:chat, read:creator.
func (c *Client) GetCreatorCustomListMembers(
	ctx context.Context, creatorUserUUID, listUUID string, p GetCreatorCustomListMembersParams,
) (*CreatorListMembersPage, error) {
	if creatorUserUUID == "" {
		return nil, errors.New("fanvue: GetCreatorCustomListMembers requires a non-empty creator user UUID")
	}
	if listUUID == "" {
		return nil, errors.New("fanvue: GetCreatorCustomListMembers requires a non-empty list UUID")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) +
		"/chats/lists/custom/" + url.PathEscape(listUUID)
	query := encodeQuery(map[string]any{"page": p.Page, "size": p.Size})
	out := &CreatorListMembersPage{}
	if err := c.doJSON(ctx, http.MethodGet, path, query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreatorSmartList is a single system-managed chat list for a managed creator
// with its current member count.
type CreatorSmartList struct {
	UUID  SmartListID `json:"uuid"`
	Name  string      `json:"name"`
	Count float64     `json:"count"`
}

// GetCreatorSmartLists returns the system-managed (smart) chat lists available to
// a managed creator. The endpoint returns a bare array, not a paginated envelope.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// GET /creators/{creatorUserUuid}/chats/lists/smart — scopes: read:chat, read:creator.
func (c *Client) GetCreatorSmartLists(
	ctx context.Context, creatorUserUUID string,
) ([]CreatorSmartList, error) {
	if creatorUserUUID == "" {
		return nil, errors.New("fanvue: GetCreatorSmartLists requires a non-empty creator user UUID")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) + "/chats/lists/smart"
	out := []CreatorSmartList{}
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetCreatorSmartListMembersParams configures a page request to
// GET /creators/{creatorUserUuid}/chats/lists/smart/{uuid}.
type GetCreatorSmartListMembersParams struct {
	Page *int
	Size *int
}

// GetCreatorSmartListMembers returns one page of members in the managed creator's
// smart list identified by id (one of the SmartList* constants).
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// GET /creators/{creatorUserUuid}/chats/lists/smart/{uuid} — scopes: read:chat, read:creator.
func (c *Client) GetCreatorSmartListMembers(
	ctx context.Context, creatorUserUUID string, id SmartListID, p GetCreatorSmartListMembersParams,
) (*CreatorListMembersPage, error) {
	if creatorUserUUID == "" {
		return nil, errors.New("fanvue: GetCreatorSmartListMembers requires a non-empty creator user UUID")
	}
	if id == "" {
		return nil, errors.New("fanvue: GetCreatorSmartListMembers requires a non-empty smart list id")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) +
		"/chats/lists/smart/" + url.PathEscape(string(id))
	query := encodeQuery(map[string]any{"page": p.Page, "size": p.Size})
	out := &CreatorListMembersPage{}
	if err := c.doJSON(ctx, http.MethodGet, path, query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}
