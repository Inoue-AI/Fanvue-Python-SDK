package fanvue

import (
	"context"
	"errors"
	"net/http"
	"net/url"
)

// ChatMediaType enumerates the media kinds a chat media item can be. It is the
// shared type for the mediaType field returned by ListMedia and
// GetMessageMediaByUUIDs, and for the mediaType query filter on ListMedia.
//
// Note that responses may additionally carry the synthetic "unknown" value
// (ChatMediaTypeUnknown) for media whose type could not be resolved, whereas
// the ListMedia query filter only accepts the four concrete kinds — matching
// the Python SDK's Literal split.
type ChatMediaType string

const (
	// ChatMediaTypeImage is a still image.
	ChatMediaTypeImage ChatMediaType = "image"
	// ChatMediaTypeVideo is a video clip.
	ChatMediaTypeVideo ChatMediaType = "video"
	// ChatMediaTypeAudio is an audio clip.
	ChatMediaTypeAudio ChatMediaType = "audio"
	// ChatMediaTypeDocument is a document attachment.
	ChatMediaTypeDocument ChatMediaType = "document"
	// ChatMediaTypeUnknown is returned in responses when the media type could
	// not be determined. It is never a valid query filter value.
	ChatMediaTypeUnknown ChatMediaType = "unknown"
)

// MediaVariantType enumerates the rendered variants Fanvue produces for a media
// item (full-resolution, thumbnails, and the blurred locked-content preview).
type MediaVariantType string

const (
	// MediaVariantMain is the full-resolution rendition.
	MediaVariantMain MediaVariantType = "main"
	// MediaVariantThumbnail is the standard thumbnail rendition.
	MediaVariantThumbnail MediaVariantType = "thumbnail"
	// MediaVariantThumbnailGallery is the gallery-sized thumbnail rendition.
	MediaVariantThumbnailGallery MediaVariantType = "thumbnail_gallery"
	// MediaVariantBlurred is the blurred preview shown for locked content.
	MediaVariantBlurred MediaVariantType = "blurred"
)

// MassMessageStatus enumerates the lifecycle states of a mass message.
type MassMessageStatus string

const (
	// MassMessageStatusSent indicates the mass message has been delivered.
	MassMessageStatusSent MassMessageStatus = "SENT"
	// MassMessageStatusUnsent indicates the mass message has not been sent.
	MassMessageStatusUnsent MassMessageStatus = "UNSENT"
	// MassMessageStatusSending indicates delivery is in progress.
	MassMessageStatusSending MassMessageStatus = "SENDING"
	// MassMessageStatusFailed indicates delivery failed.
	MassMessageStatusFailed MassMessageStatus = "FAILED"
	// MassMessageStatusModerated indicates the mass message was blocked by
	// moderation.
	MassMessageStatusModerated MassMessageStatus = "MODERATED"
	// MassMessageStatusScheduled indicates the mass message is scheduled for a
	// future send.
	MassMessageStatusScheduled MassMessageStatus = "SCHEDULED"
)

// DeleteMessage deletes a single message from the chat conversation with the
// counterpart identified by userUUID. The endpoint returns no body.
//
// DELETE /chats/{userUuid}/messages/{messageUuid} — scope: write:chat.
func (c *Client) DeleteMessage(ctx context.Context, userUUID, messageUUID string) error {
	if userUUID == "" {
		return errors.New("fanvue: DeleteMessage requires a non-empty user UUID")
	}
	if messageUUID == "" {
		return errors.New("fanvue: DeleteMessage requires a non-empty message UUID")
	}
	path := "/chats/" + url.PathEscape(userUUID) +
		"/messages/" + url.PathEscape(messageUUID)
	return c.doJSON(ctx, http.MethodDelete, path, nil, nil, nil)
}

// MediaVariant is one rendered variant of a chat media item. Nullable
// dimensions use pointers so a JSON null is distinguishable from a zero value;
// URL is omitted (nil) when the caller is not entitled to that variant.
type MediaVariant struct {
	DisplayPosition float64          `json:"displayPosition"`
	Height          *float64         `json:"height"`
	LengthMs        *float64         `json:"lengthMs"`
	URL             *string          `json:"url,omitempty"`
	VariantType     MediaVariantType `json:"variantType"`
	Width           *float64         `json:"width"`
}

// MessageMediaError is a per-UUID failure entry returned by
// GetMessageMediaByUUIDs when a requested media UUID could not be resolved.
type MessageMediaError struct {
	// Code is "NOT_IN_MESSAGE" or "INTERNAL".
	Code      string `json:"code"`
	MediaUUID string `json:"mediaUuid"`
	Message   string `json:"message"`
}

// MessageMedia is a single resolved chat media item. Nullable fields use
// pointers so a JSON null is distinguishable from a zero value; Variants is nil
// when the caller is not entitled to any rendition.
type MessageMedia struct {
	CreatedAt   *string        `json:"created_at"`
	MediaType   ChatMediaType  `json:"mediaType"`
	MessageUUID string         `json:"messageUuid"`
	Name        *string        `json:"name"`
	OwnerUUID   string         `json:"ownerUuid"`
	SentAt      *string        `json:"sentAt"`
	UUID        string         `json:"uuid"`
	Variants    []MediaVariant `json:"variants,omitempty"`
}

// MessageMediaByUUIDs is the response from GetMessageMediaByUUIDs. Results maps
// each requested media UUID to its resolved item, or to a JSON null (nil
// pointer) when that UUID failed; the parallel Errors slice carries the failure
// detail.
type MessageMediaByUUIDs struct {
	Errors  []MessageMediaError      `json:"errors"`
	Results map[string]*MessageMedia `json:"results"`
}

// GetMessageMediaByUUIDsParams configures a request to
// GET /chats/{userUuid}/messages/{messageUuid}/media.
//
// MediaUUIDs is required and is a comma-separated list of media UUIDs to
// resolve, sent verbatim as the single mediaUuids query value — mirroring the
// Python SDK's str parameter exactly (it is NOT serialized as repeated keys).
// Variants is an optional comma-separated list of variant types to include.
type GetMessageMediaByUUIDsParams struct {
	MediaUUIDs string
	Variants   *string
}

// GetMessageMediaByUUIDs resolves one or more media UUIDs attached to a chat
// message into their media items (with variant renditions), for the message
// identified by messageUUID within the conversation with userUUID. UUIDs that
// cannot be resolved appear in the response's Errors slice and as a null entry
// in Results.
//
// GET /chats/{userUuid}/messages/{messageUuid}/media — scope: read:chat.
func (c *Client) GetMessageMediaByUUIDs(
	ctx context.Context, userUUID, messageUUID string, p GetMessageMediaByUUIDsParams,
) (*MessageMediaByUUIDs, error) {
	if userUUID == "" {
		return nil, errors.New("fanvue: GetMessageMediaByUUIDs requires a non-empty user UUID")
	}
	if messageUUID == "" {
		return nil, errors.New("fanvue: GetMessageMediaByUUIDs requires a non-empty message UUID")
	}
	if p.MediaUUIDs == "" {
		return nil, errors.New("fanvue: GetMessageMediaByUUIDs requires non-empty MediaUUIDs")
	}
	path := "/chats/" + url.PathEscape(userUUID) +
		"/messages/" + url.PathEscape(messageUUID) + "/media"
	query := encodeQuery(map[string]any{
		"mediaUuids": p.MediaUUIDs,
		"variants":   p.Variants,
	})
	out := &MessageMediaByUUIDs{}
	if err := c.doJSON(ctx, http.MethodGet, path, query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// ChatMedia is one entry in a page of media exchanged within a chat
// conversation. Nullable fields use pointers so a JSON null is distinguishable
// from a zero value; Variants is nil when the caller is not entitled to any
// rendition.
type ChatMedia struct {
	CreatedAt   *string        `json:"created_at"`
	MediaType   ChatMediaType  `json:"mediaType"`
	MessageUUID string         `json:"messageUuid"`
	Name        *string        `json:"name"`
	OwnerUUID   string         `json:"ownerUuid"`
	SentAt      *string        `json:"sentAt"`
	UUID        string         `json:"uuid"`
	Variants    []MediaVariant `json:"variants,omitempty"`
}

// ChatMediaPage is one cursor-paginated page of media from a chat conversation.
// NextCursor is nil on the final page; pass it back as ListMediaParams.Cursor to
// fetch the next page.
type ChatMediaPage struct {
	Data       []ChatMedia `json:"data"`
	NextCursor *string     `json:"nextCursor"`
}

// ListMediaParams configures a page request to GET /chats/{userUuid}/media. All
// fields are optional. Cursor drives the opaque cursor-based pagination (unlike
// the page/size envelope used elsewhere); MediaType filters by kind.
type ListMediaParams struct {
	Cursor    *string
	MediaType *ChatMediaType
	Limit     *int
}

// ListMedia returns one cursor-paginated page of media exchanged within the chat
// conversation with the counterpart identified by userUUID.
//
// GET /chats/{userUuid}/media — scope: read:chat.
func (c *Client) ListMedia(
	ctx context.Context, userUUID string, p ListMediaParams,
) (*ChatMediaPage, error) {
	if userUUID == "" {
		return nil, errors.New("fanvue: ListMedia requires a non-empty user UUID")
	}
	path := "/chats/" + url.PathEscape(userUUID) + "/media"
	query := encodeQuery(map[string]any{
		"cursor":    p.Cursor,
		"mediaType": chatMediaTypeToString(p.MediaType),
		"limit":     p.Limit,
	})
	out := &ChatMediaPage{}
	if err := c.doJSON(ctx, http.MethodGet, path, query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// BatchMessagePricingUSD holds the USD price (in cents) of a paid message in a
// batch result.
type BatchMessagePricingUSD struct {
	Price float64 `json:"price"`
}

// BatchMessagePricing wraps the per-currency pricing of a paid message.
type BatchMessagePricing struct {
	USD BatchMessagePricingUSD `json:"USD"`
}

// BatchMessageParticipant is the sender or recipient summary on a batch message.
// Recipient UUID/handle may be nil when the account is no longer resolvable.
type BatchMessageParticipant struct {
	Handle *string `json:"handle"`
	UUID   *string `json:"uuid"`
}

// BatchMessage is a single message returned within a successful per-chat batch
// result. Nullable fields use pointers so a JSON null is distinguishable from a
// zero value.
type BatchMessage struct {
	HasMedia     *bool                   `json:"hasMedia"`
	IsRead       bool                    `json:"isRead"`
	MediaType    *ChatMediaType          `json:"mediaType"`
	MediaUUIDs   []string                `json:"mediaUuids"`
	Pricing      *BatchMessagePricing    `json:"pricing"`
	PurchasedAt  *string                 `json:"purchasedAt"`
	Recipient    BatchMessageParticipant `json:"recipient"`
	Sender       BatchMessageParticipant `json:"sender"`
	SentAt       *string                 `json:"sentAt"`
	SentByUserID *string                 `json:"sentByUserId"`
	Text         *string                 `json:"text"`
	Type         string                  `json:"type"`
	UUID         string                  `json:"uuid"`
}

// BatchChatResult is the per-chat entry in a MessagesBatch response. On success
// Error is empty and Messages/HasMore/OldestMessageUUID are populated; on
// failure Error is set ("forbidden", "not_found", or "internal") and the other
// fields are zero. This mirrors the Python SDK's discriminated union
// (option-1 success vs option-2 error) keyed on the presence of Error.
type BatchChatResult struct {
	HasMore           bool           `json:"hasMore"`
	Messages          []BatchMessage `json:"messages"`
	OldestMessageUUID *string        `json:"oldestMessageUuid"`
	Error             string         `json:"error,omitempty"`
}

// MessagesBatchResult is the response from MessagesBatch: a per-chat map keyed
// by chat (user) UUID. Each value is either a success result or an error result
// (see BatchChatResult).
type MessagesBatchResult struct {
	ByChat map[string]BatchChatResult `json:"byChat"`
}

// MessagesBatchParams is the POST /chats/messages/batch request body. The typed
// fields express the common shape (per-chat selectors); RawBody, when non-nil,
// is sent verbatim and takes precedence over the typed fields, mirroring the
// Python SDK's opaque Mapping[str, Any] body exactly.
type MessagesBatchParams struct {
	// UserUUIDs selects which chats to fetch messages from in bulk.
	UserUUIDs []string `json:"userUuids,omitempty"`
	// Limit caps the number of messages returned per chat.
	Limit *int `json:"limit,omitempty"`
	// Cursor optionally resumes from a prior position per chat.
	Cursor *string `json:"cursor,omitempty"`

	// RawBody, when non-nil, is sent as the request body verbatim and takes
	// precedence over every typed field above, matching the Python SDK's opaque
	// body signature.
	RawBody RawBody `json:"-"`
}

// MessagesBatch fetches recent messages from multiple chats in a single
// request, keyed by chat (user) UUID. Chats the caller cannot access are
// returned as per-chat error entries rather than failing the whole request.
//
// POST /chats/messages/batch — scope: read:chat.
func (c *Client) MessagesBatch(
	ctx context.Context, p MessagesBatchParams,
) (*MessagesBatchResult, error) {
	if p.RawBody == nil && len(p.UserUUIDs) == 0 {
		return nil, errors.New("fanvue: MessagesBatch requires at least one user UUID")
	}
	out := &MessagesBatchResult{}
	if err := c.doJSON(ctx, http.MethodPost, "/chats/messages/batch", nil, chatMessageBody(p, p.RawBody), out); err != nil {
		return nil, err
	}
	return out, nil
}

// MassMessage is one entry in a page of the authenticated creator's mass
// messages. Amounts are in cents; nullable fields use pointers so a JSON null is
// distinguishable from a zero value.
type MassMessage struct {
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

// MassMessagesPage is one page of the authenticated creator's mass messages.
type MassMessagesPage struct {
	Data       []MassMessage `json:"data"`
	Pagination Pagination    `json:"pagination"`
}

// ListMassMessagesParams configures a page request to GET /chats/mass-messages.
// All fields are optional. IncludeDeleted, when non-nil, includes soft-deleted
// mass messages in the result.
type ListMassMessagesParams struct {
	Page           *int
	Size           *int
	IncludeDeleted *bool
}

// ListMassMessages returns one page of the authenticated creator's mass
// messages.
//
// GET /chats/mass-messages — scope: read:chat.
func (c *Client) ListMassMessages(
	ctx context.Context, p ListMassMessagesParams,
) (*MassMessagesPage, error) {
	query := encodeQuery(map[string]any{
		"page":           p.Page,
		"size":           p.Size,
		"includeDeleted": boolToLiteralString(p.IncludeDeleted),
	})
	out := &MassMessagesPage{}
	if err := c.doJSON(ctx, http.MethodGet, "/chats/mass-messages", query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// SendMassMessageResult is the response from SendMassMessage. CreatedAt is nil
// when the mass message is scheduled rather than sent immediately.
type SendMassMessageResult struct {
	CreatedAt      *string `json:"createdAt"`
	ID             string  `json:"id"`
	RecipientCount float64 `json:"recipientCount"`
}

// SendMassMessageParams is the POST /chats/mass-messages request body. The typed
// fields express the common shape; RawBody, when non-nil, is sent verbatim and
// takes precedence over the typed fields, mirroring the Python SDK's opaque
// Mapping[str, Any] body exactly. Price is in cents; ScheduledAt schedules the
// send for a future ISO-8601 timestamp.
type SendMassMessageParams struct {
	Text         *string  `json:"text,omitempty"`
	MediaUUIDs   []string `json:"mediaUuids,omitempty"`
	Price        *float64 `json:"price,omitempty"`
	ScheduledAt  *string  `json:"scheduledAt,omitempty"`
	SmartListIDs []string `json:"smartListIds,omitempty"`
	CustomListID *string  `json:"customListId,omitempty"`

	// RawBody, when non-nil, is sent as the request body verbatim and takes
	// precedence over every typed field above, matching the Python SDK's opaque
	// body signature.
	RawBody RawBody `json:"-"`
}

// SendMassMessage sends (or schedules) a mass message to a list of the
// authenticated creator's contacts.
//
// POST /chats/mass-messages — scope: write:chat.
func (c *Client) SendMassMessage(
	ctx context.Context, p SendMassMessageParams,
) (*SendMassMessageResult, error) {
	out := &SendMassMessageResult{}
	if err := c.doJSON(ctx, http.MethodPost, "/chats/mass-messages", nil, chatMessageBody(p, p.RawBody), out); err != nil {
		return nil, err
	}
	return out, nil
}

// UpdateMassMessageParams is the PATCH /chats/mass-messages/{messageUuid}
// request body. Every typed field is optional: a nil pointer (or nil slice) is
// omitted so the field is left unchanged. RawBody, when non-nil, is sent
// verbatim and takes precedence over the typed fields, matching the Python SDK's
// opaque Mapping[str, Any] body exactly.
type UpdateMassMessageParams struct {
	Text        *string  `json:"text,omitempty"`
	MediaUUIDs  []string `json:"mediaUuids,omitempty"`
	Price       *float64 `json:"price,omitempty"`
	ScheduledAt *string  `json:"scheduledAt,omitempty"`

	// RawBody, when non-nil, is sent as the request body verbatim and takes
	// precedence over every typed field above, matching the Python SDK's opaque
	// body signature.
	RawBody RawBody `json:"-"`
}

// UpdateMassMessage updates a scheduled mass message identified by messageUUID
// (e.g. to revise its text, media, price, or scheduled send time). Only
// scheduled (not yet sent) mass messages can be updated. The endpoint returns no
// body.
//
// PATCH /chats/mass-messages/{messageUuid} — scope: write:chat.
func (c *Client) UpdateMassMessage(
	ctx context.Context, messageUUID string, p UpdateMassMessageParams,
) error {
	if messageUUID == "" {
		return errors.New("fanvue: UpdateMassMessage requires a non-empty message UUID")
	}
	path := "/chats/mass-messages/" + url.PathEscape(messageUUID)
	return c.doJSON(ctx, http.MethodPatch, path, nil, chatMessageBody(p, p.RawBody), nil)
}

// DeleteMassMessage deletes a mass message identified by messageUUID. The
// endpoint returns no body.
//
// DELETE /chats/mass-messages/{messageUuid} — scope: write:chat.
func (c *Client) DeleteMassMessage(ctx context.Context, messageUUID string) error {
	if messageUUID == "" {
		return errors.New("fanvue: DeleteMassMessage requires a non-empty message UUID")
	}
	path := "/chats/mass-messages/" + url.PathEscape(messageUUID)
	return c.doJSON(ctx, http.MethodDelete, path, nil, nil, nil)
}

// chatMessageBody resolves the body sent for a chat-messages write request. When
// raw is supplied it is sent verbatim (preserving explicit JSON nulls and any
// additional fields, matching the Python SDK's opaque Mapping body); otherwise
// the typed params value is marshalled normally.
func chatMessageBody(typed any, raw RawBody) any {
	if raw != nil {
		return raw
	}
	return typed
}

// chatMediaTypeToString converts an optional ChatMediaType into the *string form
// encodeQuery expects, returning nil when unset so the parameter is omitted.
func chatMediaTypeToString(mt *ChatMediaType) *string {
	if mt == nil {
		return nil
	}
	s := string(*mt)
	return &s
}

// boolToLiteralString renders an optional bool as the "true"/"false" string the
// Fanvue API expects for its Literal['true','false'] query parameters (e.g.
// includeDeleted), returning nil when unset so the parameter is omitted. This
// mirrors the Python SDK, which types these parameters as the string literals
// rather than a native bool.
func boolToLiteralString(b *bool) *string {
	if b == nil {
		return nil
	}
	s := "false"
	if *b {
		s = "true"
	}
	return &s
}
