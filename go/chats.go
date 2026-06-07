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
