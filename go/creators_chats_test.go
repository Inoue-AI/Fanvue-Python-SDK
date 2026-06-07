package fanvue

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

const creatorChatsPageJSON = `{
  "data": [
    {
      "createdAt": "2026-01-01T00:00:00.000Z",
      "isMuted": false,
      "isRead": true,
      "lastMessage": {
        "uuid": "m-1",
        "type": "SINGLE_RECIPIENT",
        "text": "hey",
        "hasMedia": false,
        "mediaType": null,
        "senderUuid": "cr-1",
        "sentAt": "2026-01-02T00:00:00.000Z",
        "sentByUserId": "cr-1"
      },
      "lastMessageAt": "2026-01-02T00:00:00.000Z",
      "unreadMessagesCount": 0,
      "user": {
        "uuid": "u-9",
        "handle": "fan",
        "displayName": "Fan",
        "nickname": null,
        "avatarUrl": null,
        "isTopSpender": true,
        "registeredAt": "2025-12-01T00:00:00.000Z"
      }
    }
  ],
  "pagination": {"page": 1, "size": 20, "hasMore": false}
}`

const creatorChatCreatedJSON = `{"message": "Chat created"}`

const creatorMessagesPageJSON = `{
  "data": [
    {
      "uuid": "m-1",
      "type": "SINGLE_RECIPIENT",
      "text": "hello",
      "hasMedia": true,
      "mediaType": "image",
      "mediaUuids": ["md-1"],
      "isRead": false,
      "pricing": {"USD": {"price": 500}},
      "purchasedAt": null,
      "recipient": {"handle": "fan", "uuid": "u-9"},
      "sender": {"handle": "creator", "uuid": "cr-1"},
      "sentAt": "2026-01-02T00:00:00.000Z",
      "sentByUserId": "cr-1"
    }
  ],
  "pagination": {"page": 1, "size": 20, "hasMore": false}
}`

const sendCreatorMessageJSON = `{"messageUuid": "m-2"}`

const creatorMessageMediaByUUIDsJSON = `{
  "errors": [{"code": "NOT_IN_MESSAGE", "mediaUuid": "md-x", "message": "not found"}],
  "results": {
    "md-1": {
      "created_at": "2026-01-01T00:00:00.000Z",
      "mediaType": "image",
      "messageUuid": "m-1",
      "name": "pic.jpg",
      "ownerUuid": "cr-1",
      "sentAt": "2026-01-02T00:00:00.000Z",
      "uuid": "md-1",
      "variants": [
        {"displayPosition": 0, "height": 100, "lengthMs": null, "url": "https://cdn/x", "variantType": "main", "width": 200}
      ]
    },
    "md-x": null
  }
}`

const creatorChatMediaPageJSON = `{
  "data": [
    {
      "created_at": "2026-01-01T00:00:00.000Z",
      "mediaType": "video",
      "messageUuid": "m-1",
      "name": null,
      "ownerUuid": "cr-1",
      "sentAt": "2026-01-02T00:00:00.000Z",
      "uuid": "md-1",
      "variants": null
    }
  ],
  "nextCursor": "cur-2"
}`

const creatorMassMessagesPageJSON = `{
  "data": [
    {
      "createdAt": "2026-01-01T00:00:00.000Z",
      "mediaUuids": ["md-1"],
      "price": 1000,
      "publishedAt": null,
      "purchaseCount": 3,
      "recipientCount": 100,
      "scheduledAt": "2026-02-01T00:00:00.000Z",
      "status": "SCHEDULED",
      "text": "promo",
      "totalRevenue": 3000,
      "uuid": "mm-1",
      "viewCount": 42
    }
  ],
  "pagination": {"page": 1, "size": 20, "hasMore": false}
}`

const sendCreatorMassMessageJSON = `{"createdAt": null, "id": "mm-2", "recipientCount": 100}`

const creatorCustomListsPageJSON = `{
  "data": [{"uuid": "cl-1", "name": "VIPs", "membersCount": 5, "createdAt": "2026-01-01T00:00:00.000Z"}],
  "pagination": {"page": 1, "size": 20, "hasMore": false}
}`

const creatorListMembersPageJSON = `{
  "data": [{"uuid": "u-9", "handle": "fan", "displayName": "Fan", "isCreator": false}],
  "pagination": {"page": 1, "size": 20, "hasMore": false}
}`

const creatorSmartListsJSON = `[
  {"uuid": "subscribers", "name": "Subscribers", "count": 10},
  {"uuid": "followers", "name": "Followers", "count": 25}
]`

func TestListCreatorChats(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/creators/cr-1/chats" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("page") != "2" || q.Get("size") != "20" {
			t.Errorf("query: page=%q size=%q", q.Get("page"), q.Get("size"))
		}
		_, _ = io.WriteString(w, creatorChatsPageJSON)
	})

	page, err := c.ListCreatorChats(context.Background(), "cr-1", ListCreatorChatsParams{
		Page: ptrInt(2),
		Size: ptrInt(20),
	})
	if err != nil {
		t.Fatalf("ListCreatorChats: %v", err)
	}
	if len(page.Data) != 1 {
		t.Fatalf("data: got %d entries", len(page.Data))
	}
	entry := page.Data[0]
	if entry.User.Handle != "fan" || !entry.User.IsTopSpender {
		t.Errorf("user: got %+v", entry.User)
	}
	if entry.LastMessage == nil || entry.LastMessage.Type != "SINGLE_RECIPIENT" || entry.LastMessage.SenderUUID != "cr-1" {
		t.Errorf("lastMessage: got %+v", entry.LastMessage)
	}
	if page.Pagination.Page != 1 || page.Pagination.HasMore {
		t.Errorf("pagination: got %+v", page.Pagination)
	}
}

func TestListCreatorChats_OmitsUnsetParams(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if _, present := r.URL.Query()["page"]; present {
			t.Errorf("page should be omitted, query=%s", r.URL.RawQuery)
		}
		_, _ = io.WriteString(w, creatorChatsPageJSON)
	})
	if _, err := c.ListCreatorChats(context.Background(), "cr-1", ListCreatorChatsParams{}); err != nil {
		t.Fatalf("ListCreatorChats: %v", err)
	}
}

func TestListCreatorChats_EmptyUUID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when creator user UUID is empty")
	})
	if _, err := c.ListCreatorChats(context.Background(), "", ListCreatorChatsParams{}); err == nil {
		t.Fatal("expected error for empty creator user UUID")
	}
}

func TestCreateCreatorChat(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/creators/cr-1/chats" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		body := decodeBody(t, r)
		if body["userUuid"] != "u-9" {
			t.Errorf("body userUuid: got %v", body["userUuid"])
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, creatorChatCreatedJSON)
	})

	res, err := c.CreateCreatorChat(context.Background(), "cr-1", CreateCreatorChatParams{UserUUID: "u-9"})
	if err != nil {
		t.Fatalf("CreateCreatorChat: %v", err)
	}
	if res.Message != "Chat created" {
		t.Errorf("message: got %q", res.Message)
	}
}

func TestCreateCreatorChat_RawBodyVerbatim(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body := decodeBody(t, r)
		if body["userUuid"] != "raw-9" {
			t.Errorf("RawBody should win: got %v", body["userUuid"])
		}
		_, _ = io.WriteString(w, creatorChatCreatedJSON)
	})
	_, err := c.CreateCreatorChat(context.Background(), "cr-1", CreateCreatorChatParams{
		UserUUID: "ignored",
		RawBody:  RawBody(`{"userUuid":"raw-9"}`),
	})
	if err != nil {
		t.Fatalf("CreateCreatorChat: %v", err)
	}
}

func TestCreateCreatorChat_EmptyUUIDs(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when a UUID is empty")
	})
	if _, err := c.CreateCreatorChat(context.Background(), "", CreateCreatorChatParams{UserUUID: "u-9"}); err == nil {
		t.Fatal("expected error for empty creator user UUID")
	}
	if _, err := c.CreateCreatorChat(context.Background(), "cr-1", CreateCreatorChatParams{}); err == nil {
		t.Fatal("expected error for empty UserUUID")
	}
}

func TestUpdateCreatorChat(t *testing.T) {
	called := false
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/creators/cr-1/chats/u-9" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		body := decodeBody(t, r)
		if body["isMuted"] != true {
			t.Errorf("body isMuted: got %v", body["isMuted"])
		}
		if _, present := body["isRead"]; present {
			t.Errorf("isRead should be omitted when unset")
		}
		w.WriteHeader(http.StatusNoContent)
	})

	if err := c.UpdateCreatorChat(context.Background(), "cr-1", "u-9", UpdateCreatorChatParams{
		IsMuted: ptrBool(true),
	}); err != nil {
		t.Fatalf("UpdateCreatorChat: %v", err)
	}
	if !called {
		t.Error("server was not called")
	}
}

func TestUpdateCreatorChat_RawBodyVerbatim(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var got map[string]json.RawMessage
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("body not JSON: %v (%s)", err, body)
		}
		raw, present := got["nickname"]
		if !present || string(raw) != "null" {
			t.Errorf("expected explicit null nickname, got %s present=%v", raw, present)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	if err := c.UpdateCreatorChat(context.Background(), "cr-1", "u-9", UpdateCreatorChatParams{
		Nickname: ptrString("ignored"),
		RawBody:  RawBody(`{"nickname":null}`),
	}); err != nil {
		t.Fatalf("UpdateCreatorChat: %v", err)
	}
}

func TestUpdateCreatorChat_EmptyUUIDs(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when a UUID is empty")
	})
	if err := c.UpdateCreatorChat(context.Background(), "", "u-9", UpdateCreatorChatParams{}); err == nil {
		t.Fatal("expected error for empty creator user UUID")
	}
	if err := c.UpdateCreatorChat(context.Background(), "cr-1", "", UpdateCreatorChatParams{}); err == nil {
		t.Fatal("expected error for empty user UUID")
	}
}

func TestListCreatorMessages(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/creators/cr-1/chats/u-9/messages" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("markAsRead") != "true" {
			t.Errorf("markAsRead: got %q", q.Get("markAsRead"))
		}
		if q.Get("page") != "1" {
			t.Errorf("page: got %q", q.Get("page"))
		}
		_, _ = io.WriteString(w, creatorMessagesPageJSON)
	})

	page, err := c.ListCreatorMessages(context.Background(), "cr-1", "u-9", ListCreatorMessagesParams{
		Page:       ptrInt(1),
		MarkAsRead: ptrBool(true),
	})
	if err != nil {
		t.Fatalf("ListCreatorMessages: %v", err)
	}
	if len(page.Data) != 1 {
		t.Fatalf("data: got %d entries", len(page.Data))
	}
	m := page.Data[0]
	if m.UUID != "m-1" || m.Sender.UUID != "cr-1" {
		t.Errorf("message: got %+v", m)
	}
	if m.Pricing == nil || m.Pricing.USD.Price != 500 {
		t.Errorf("pricing: got %+v", m.Pricing)
	}
	if m.MediaType == nil || *m.MediaType != ChatMediaTypeImage {
		t.Errorf("mediaType: got %v", m.MediaType)
	}
	if len(m.MediaUUIDs) != 1 || m.MediaUUIDs[0] != "md-1" {
		t.Errorf("mediaUuids: got %v", m.MediaUUIDs)
	}
	if m.Recipient.Handle == nil || *m.Recipient.Handle != "fan" {
		t.Errorf("recipient handle: got %v", m.Recipient.Handle)
	}
}

func TestListCreatorMessages_MarkAsReadFalse(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("markAsRead"); got != "false" {
			t.Errorf("markAsRead: got %q want false", got)
		}
		_, _ = io.WriteString(w, creatorMessagesPageJSON)
	})
	if _, err := c.ListCreatorMessages(context.Background(), "cr-1", "u-9", ListCreatorMessagesParams{
		MarkAsRead: ptrBool(false),
	}); err != nil {
		t.Fatalf("ListCreatorMessages: %v", err)
	}
}

func TestListCreatorMessages_EmptyUUIDs(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when a UUID is empty")
	})
	if _, err := c.ListCreatorMessages(context.Background(), "", "u-9", ListCreatorMessagesParams{}); err == nil {
		t.Fatal("expected error for empty creator user UUID")
	}
	if _, err := c.ListCreatorMessages(context.Background(), "cr-1", "", ListCreatorMessagesParams{}); err == nil {
		t.Fatal("expected error for empty user UUID")
	}
}

func TestSendCreatorMessage(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/creators/cr-1/chats/u-9/message" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		body := decodeBody(t, r)
		if body["text"] != "hi" {
			t.Errorf("body text: got %v", body["text"])
		}
		if _, present := body["price"]; present {
			t.Errorf("price should be omitted when unset")
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, sendCreatorMessageJSON)
	})

	res, err := c.SendCreatorMessage(context.Background(), "cr-1", "u-9", SendCreatorMessageParams{
		Text: ptrString("hi"),
	})
	if err != nil {
		t.Fatalf("SendCreatorMessage: %v", err)
	}
	if res.MessageUUID != "m-2" {
		t.Errorf("messageUuid: got %q", res.MessageUUID)
	}
}

func TestSendCreatorMessage_RawBodyVerbatim(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body := decodeBody(t, r)
		if body["text"] != "raw" {
			t.Errorf("RawBody should win: got %v", body["text"])
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, sendCreatorMessageJSON)
	})
	_, err := c.SendCreatorMessage(context.Background(), "cr-1", "u-9", SendCreatorMessageParams{
		Text:    ptrString("ignored"),
		RawBody: RawBody(`{"text":"raw"}`),
	})
	if err != nil {
		t.Fatalf("SendCreatorMessage: %v", err)
	}
}

func TestSendCreatorMessage_RequiresContent(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called without content")
	})
	if _, err := c.SendCreatorMessage(context.Background(), "cr-1", "u-9", SendCreatorMessageParams{}); err == nil {
		t.Fatal("expected error when neither Text nor MediaUUIDs supplied")
	}
}

func TestSendCreatorMessage_EmptyUUIDs(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when a UUID is empty")
	})
	if _, err := c.SendCreatorMessage(context.Background(), "", "u-9", SendCreatorMessageParams{Text: ptrString("x")}); err == nil {
		t.Fatal("expected error for empty creator user UUID")
	}
	if _, err := c.SendCreatorMessage(context.Background(), "cr-1", "", SendCreatorMessageParams{Text: ptrString("x")}); err == nil {
		t.Fatal("expected error for empty user UUID")
	}
}

func TestGetCreatorMessageMediaByUUIDs(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/creators/cr-1/chats/u-9/messages/m-1/media" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("mediaUuids") != "md-1,md-x" {
			t.Errorf("mediaUuids: got %q", q.Get("mediaUuids"))
		}
		if q.Get("variants") != "main" {
			t.Errorf("variants: got %q", q.Get("variants"))
		}
		_, _ = io.WriteString(w, creatorMessageMediaByUUIDsJSON)
	})

	res, err := c.GetCreatorMessageMediaByUUIDs(context.Background(), "cr-1", "u-9", "m-1", GetCreatorMessageMediaByUUIDsParams{
		MediaUUIDs: "md-1,md-x",
		Variants:   ptrString("main"),
	})
	if err != nil {
		t.Fatalf("GetCreatorMessageMediaByUUIDs: %v", err)
	}
	if len(res.Errors) != 1 || res.Errors[0].Code != "NOT_IN_MESSAGE" {
		t.Errorf("errors: got %+v", res.Errors)
	}
	md, ok := res.Results["md-1"]
	if !ok || md == nil || md.UUID != "md-1" || md.MediaType != ChatMediaTypeImage {
		t.Errorf("results md-1: got %+v ok=%v", md, ok)
	}
	if len(md.Variants) != 1 || md.Variants[0].VariantType != MediaVariantMain {
		t.Errorf("variants: got %+v", md.Variants)
	}
	if v, ok := res.Results["md-x"]; !ok || v != nil {
		t.Errorf("results md-x should be present and nil, got %+v ok=%v", v, ok)
	}
}

func TestGetCreatorMessageMediaByUUIDs_RequiresMediaUUIDs(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called without mediaUuids")
	})
	if _, err := c.GetCreatorMessageMediaByUUIDs(context.Background(), "cr-1", "u-9", "m-1", GetCreatorMessageMediaByUUIDsParams{}); err == nil {
		t.Fatal("expected error for empty MediaUUIDs")
	}
}

func TestGetCreatorMessageMediaByUUIDs_EmptyUUIDs(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when a UUID is empty")
	})
	p := GetCreatorMessageMediaByUUIDsParams{MediaUUIDs: "md-1"}
	if _, err := c.GetCreatorMessageMediaByUUIDs(context.Background(), "", "u-9", "m-1", p); err == nil {
		t.Fatal("expected error for empty creator user UUID")
	}
	if _, err := c.GetCreatorMessageMediaByUUIDs(context.Background(), "cr-1", "", "m-1", p); err == nil {
		t.Fatal("expected error for empty user UUID")
	}
	if _, err := c.GetCreatorMessageMediaByUUIDs(context.Background(), "cr-1", "u-9", "", p); err == nil {
		t.Fatal("expected error for empty message UUID")
	}
}

func TestListCreatorChatMedia(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/creators/cr-1/chats/u-9/media" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("cursor") != "cur-1" {
			t.Errorf("cursor: got %q", q.Get("cursor"))
		}
		if q.Get("mediaType") != "video" {
			t.Errorf("mediaType: got %q", q.Get("mediaType"))
		}
		if q.Get("limit") != "10" {
			t.Errorf("limit: got %q", q.Get("limit"))
		}
		_, _ = io.WriteString(w, creatorChatMediaPageJSON)
	})

	mt := ChatMediaTypeVideo
	page, err := c.ListCreatorChatMedia(context.Background(), "cr-1", "u-9", ListCreatorChatMediaParams{
		Cursor:    ptrString("cur-1"),
		MediaType: &mt,
		Limit:     ptrInt(10),
	})
	if err != nil {
		t.Fatalf("ListCreatorChatMedia: %v", err)
	}
	if len(page.Data) != 1 || page.Data[0].MediaType != ChatMediaTypeVideo {
		t.Errorf("data: got %+v", page.Data)
	}
	if page.NextCursor == nil || *page.NextCursor != "cur-2" {
		t.Errorf("nextCursor: got %v", page.NextCursor)
	}
}

func TestListCreatorChatMedia_EmptyUUIDs(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when a UUID is empty")
	})
	if _, err := c.ListCreatorChatMedia(context.Background(), "", "u-9", ListCreatorChatMediaParams{}); err == nil {
		t.Fatal("expected error for empty creator user UUID")
	}
	if _, err := c.ListCreatorChatMedia(context.Background(), "cr-1", "", ListCreatorChatMediaParams{}); err == nil {
		t.Fatal("expected error for empty user UUID")
	}
}

func TestListCreatorMassMessages(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/creators/cr-1/chats/mass-messages" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("includeDeleted"); got != "true" {
			t.Errorf("includeDeleted: got %q", got)
		}
		_, _ = io.WriteString(w, creatorMassMessagesPageJSON)
	})

	page, err := c.ListCreatorMassMessages(context.Background(), "cr-1", ListCreatorMassMessagesParams{
		IncludeDeleted: ptrBool(true),
	})
	if err != nil {
		t.Fatalf("ListCreatorMassMessages: %v", err)
	}
	if len(page.Data) != 1 {
		t.Fatalf("data: got %d entries", len(page.Data))
	}
	mm := page.Data[0]
	if mm.UUID != "mm-1" || mm.Status != MassMessageStatusScheduled {
		t.Errorf("massMessage: got %+v", mm)
	}
	if mm.Price == nil || *mm.Price != 1000 {
		t.Errorf("price: got %v", mm.Price)
	}
}

func TestListCreatorMassMessages_EmptyUUID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when creator user UUID is empty")
	})
	if _, err := c.ListCreatorMassMessages(context.Background(), "", ListCreatorMassMessagesParams{}); err == nil {
		t.Fatal("expected error for empty creator user UUID")
	}
}

func TestSendCreatorMassMessage(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/creators/cr-1/chats/mass-messages" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		body := decodeBody(t, r)
		if body["text"] != "promo" {
			t.Errorf("body text: got %v", body["text"])
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, sendCreatorMassMessageJSON)
	})

	res, err := c.SendCreatorMassMessage(context.Background(), "cr-1", SendCreatorMassMessageParams{
		Text:         ptrString("promo"),
		SmartListIDs: []string{"subscribers"},
	})
	if err != nil {
		t.Fatalf("SendCreatorMassMessage: %v", err)
	}
	if res.ID != "mm-2" || res.RecipientCount != 100 {
		t.Errorf("result: got %+v", res)
	}
	if res.CreatedAt != nil {
		t.Errorf("createdAt should be nil, got %v", *res.CreatedAt)
	}
}

func TestSendCreatorMassMessage_RawBodyVerbatim(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body := decodeBody(t, r)
		if body["text"] != "raw" {
			t.Errorf("RawBody should win: got %v", body["text"])
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, sendCreatorMassMessageJSON)
	})
	_, err := c.SendCreatorMassMessage(context.Background(), "cr-1", SendCreatorMassMessageParams{
		Text:    ptrString("ignored"),
		RawBody: RawBody(`{"text":"raw"}`),
	})
	if err != nil {
		t.Fatalf("SendCreatorMassMessage: %v", err)
	}
}

func TestSendCreatorMassMessage_EmptyUUID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when creator user UUID is empty")
	})
	if _, err := c.SendCreatorMassMessage(context.Background(), "", SendCreatorMassMessageParams{}); err == nil {
		t.Fatal("expected error for empty creator user UUID")
	}
}

func TestUpdateCreatorMassMessage(t *testing.T) {
	called := false
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/creators/cr-1/chats/mass-messages/mm-1" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		body := decodeBody(t, r)
		if body["text"] != "edited" {
			t.Errorf("body text: got %v", body["text"])
		}
		w.WriteHeader(http.StatusNoContent)
	})

	if err := c.UpdateCreatorMassMessage(context.Background(), "cr-1", "mm-1", UpdateCreatorMassMessageParams{
		Text: ptrString("edited"),
	}); err != nil {
		t.Fatalf("UpdateCreatorMassMessage: %v", err)
	}
	if !called {
		t.Error("server was not called")
	}
}

func TestUpdateCreatorMassMessage_EmptyUUIDs(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when a UUID is empty")
	})
	if err := c.UpdateCreatorMassMessage(context.Background(), "", "mm-1", UpdateCreatorMassMessageParams{}); err == nil {
		t.Fatal("expected error for empty creator user UUID")
	}
	if err := c.UpdateCreatorMassMessage(context.Background(), "cr-1", "", UpdateCreatorMassMessageParams{}); err == nil {
		t.Fatal("expected error for empty message UUID")
	}
}

func TestDeleteCreatorMassMessage(t *testing.T) {
	called := false
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/creators/cr-1/chats/mass-messages/mm-1" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	if err := c.DeleteCreatorMassMessage(context.Background(), "cr-1", "mm-1"); err != nil {
		t.Fatalf("DeleteCreatorMassMessage: %v", err)
	}
	if !called {
		t.Error("server was not called")
	}
}

func TestDeleteCreatorMassMessage_EmptyUUIDs(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when a UUID is empty")
	})
	if err := c.DeleteCreatorMassMessage(context.Background(), "", "mm-1"); err == nil {
		t.Fatal("expected error for empty creator user UUID")
	}
	if err := c.DeleteCreatorMassMessage(context.Background(), "cr-1", ""); err == nil {
		t.Fatal("expected error for empty message UUID")
	}
}

func TestGetCreatorCustomLists(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/creators/cr-1/chats/lists/custom" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("page"); got != "1" {
			t.Errorf("page: got %q", got)
		}
		_, _ = io.WriteString(w, creatorCustomListsPageJSON)
	})

	page, err := c.GetCreatorCustomLists(context.Background(), "cr-1", GetCreatorCustomListsParams{Page: ptrInt(1)})
	if err != nil {
		t.Fatalf("GetCreatorCustomLists: %v", err)
	}
	if len(page.Data) != 1 || page.Data[0].UUID != "cl-1" || page.Data[0].Name != "VIPs" {
		t.Errorf("data: got %+v", page.Data)
	}
}

func TestGetCreatorCustomLists_EmptyUUID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when creator user UUID is empty")
	})
	if _, err := c.GetCreatorCustomLists(context.Background(), "", GetCreatorCustomListsParams{}); err == nil {
		t.Fatal("expected error for empty creator user UUID")
	}
}

func TestGetCreatorCustomListMembers(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/creators/cr-1/chats/lists/custom/cl-1" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, creatorListMembersPageJSON)
	})

	page, err := c.GetCreatorCustomListMembers(context.Background(), "cr-1", "cl-1", GetCreatorCustomListMembersParams{})
	if err != nil {
		t.Fatalf("GetCreatorCustomListMembers: %v", err)
	}
	if len(page.Data) != 1 || page.Data[0].Handle != "fan" {
		t.Errorf("data: got %+v", page.Data)
	}
}

func TestGetCreatorCustomListMembers_EmptyUUIDs(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when a UUID is empty")
	})
	if _, err := c.GetCreatorCustomListMembers(context.Background(), "", "cl-1", GetCreatorCustomListMembersParams{}); err == nil {
		t.Fatal("expected error for empty creator user UUID")
	}
	if _, err := c.GetCreatorCustomListMembers(context.Background(), "cr-1", "", GetCreatorCustomListMembersParams{}); err == nil {
		t.Fatal("expected error for empty list UUID")
	}
}

func TestGetCreatorSmartLists(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/creators/cr-1/chats/lists/smart" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, creatorSmartListsJSON)
	})

	lists, err := c.GetCreatorSmartLists(context.Background(), "cr-1")
	if err != nil {
		t.Fatalf("GetCreatorSmartLists: %v", err)
	}
	if len(lists) != 2 {
		t.Fatalf("lists: got %d entries", len(lists))
	}
	if lists[0].UUID != SmartListSubscribers || lists[0].Count != 10 {
		t.Errorf("list[0]: got %+v", lists[0])
	}
	if lists[1].UUID != SmartListFollowers {
		t.Errorf("list[1]: got %+v", lists[1])
	}
}

func TestGetCreatorSmartLists_EmptyUUID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when creator user UUID is empty")
	})
	if _, err := c.GetCreatorSmartLists(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty creator user UUID")
	}
}

func TestGetCreatorSmartListMembers(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/creators/cr-1/chats/lists/smart/subscribers" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("size"); got != "50" {
			t.Errorf("size: got %q", got)
		}
		_, _ = io.WriteString(w, creatorListMembersPageJSON)
	})

	page, err := c.GetCreatorSmartListMembers(context.Background(), "cr-1", SmartListSubscribers, GetCreatorSmartListMembersParams{
		Size: ptrInt(50),
	})
	if err != nil {
		t.Fatalf("GetCreatorSmartListMembers: %v", err)
	}
	if len(page.Data) != 1 || page.Data[0].UUID != "u-9" {
		t.Errorf("data: got %+v", page.Data)
	}
}

func TestGetCreatorSmartListMembers_EmptyArgs(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when an argument is empty")
	})
	if _, err := c.GetCreatorSmartListMembers(context.Background(), "", SmartListSubscribers, GetCreatorSmartListMembersParams{}); err == nil {
		t.Fatal("expected error for empty creator user UUID")
	}
	if _, err := c.GetCreatorSmartListMembers(context.Background(), "cr-1", "", GetCreatorSmartListMembersParams{}); err == nil {
		t.Fatal("expected error for empty smart list id")
	}
}
