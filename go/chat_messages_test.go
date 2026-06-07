package fanvue

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

// ptrBool, ptrFloat64, and ptrChatMediaType are local pointer helpers for the
// chat-messages tests, complementing ptrString/ptrInt from client_test.go.
func ptrBool(b bool) *bool                            { return &b }
func ptrFloat64(f float64) *float64                   { return &f }
func ptrChatMediaType(t ChatMediaType) *ChatMediaType { return &t }

// --- delete_message ---------------------------------------------------------

func TestDeleteMessage_Success(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/chats/u-1/messages/m-1" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	if err := c.DeleteMessage(context.Background(), "u-1", "m-1"); err != nil {
		t.Fatalf("DeleteMessage: %v", err)
	}
}

func TestDeleteMessage_RequiresUUIDs(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler must not be called when validation fails")
	})
	if err := c.DeleteMessage(context.Background(), "", "m-1"); err == nil {
		t.Error("expected error for empty user UUID")
	}
	if err := c.DeleteMessage(context.Background(), "u-1", ""); err == nil {
		t.Error("expected error for empty message UUID")
	}
}

func TestDeleteMessage_PropagatesAPIError(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"message":"message not found"}`)
	})
	err := c.DeleteMessage(context.Background(), "u-1", "m-1")
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := AsError(err)
	if !ok || !apiErr.IsNotFound() {
		t.Errorf("expected 404 Error, got %v", err)
	}
}

// --- get_message_media_by_uuids ---------------------------------------------

func TestGetMessageMediaByUUIDs_Success(t *testing.T) {
	const body = `{
	  "errors": [
	    {"code": "NOT_IN_MESSAGE", "mediaUuid": "media-2", "message": "not in message"}
	  ],
	  "results": {
	    "media-1": {
	      "created_at": "2026-01-01T00:00:00.000Z",
	      "mediaType": "image",
	      "messageUuid": "m-1",
	      "name": "pic.jpg",
	      "ownerUuid": "u-owner",
	      "sentAt": "2026-01-01T00:00:01.000Z",
	      "uuid": "media-1",
	      "variants": [
	        {"displayPosition": 0, "height": 1080, "lengthMs": null, "url": "https://cdn/x", "variantType": "main", "width": 1920}
	      ]
	    },
	    "media-2": null
	  }
	}`
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/chats/u-1/messages/m-1/media" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		q := r.URL.Query()
		// mediaUuids must be a single verbatim value, NOT repeated keys.
		if got := q["mediaUuids"]; len(got) != 1 || got[0] != "media-1,media-2" {
			t.Errorf("mediaUuids: got %v", got)
		}
		if q.Get("variants") != "main,thumbnail" {
			t.Errorf("variants: got %q", q.Get("variants"))
		}
		_, _ = io.WriteString(w, body)
	})

	res, err := c.GetMessageMediaByUUIDs(context.Background(), "u-1", "m-1",
		GetMessageMediaByUUIDsParams{
			MediaUUIDs: "media-1,media-2",
			Variants:   ptrString("main,thumbnail"),
		})
	if err != nil {
		t.Fatalf("GetMessageMediaByUUIDs: %v", err)
	}
	if len(res.Errors) != 1 || res.Errors[0].Code != "NOT_IN_MESSAGE" || res.Errors[0].MediaUUID != "media-2" {
		t.Errorf("errors: got %+v", res.Errors)
	}
	got1, ok := res.Results["media-1"]
	if !ok || got1 == nil {
		t.Fatalf("results[media-1]: got %v", got1)
	}
	if got1.MediaType != ChatMediaTypeImage || len(got1.Variants) != 1 || got1.Variants[0].VariantType != MediaVariantMain {
		t.Errorf("media-1: got %+v", got1)
	}
	got2, ok := res.Results["media-2"]
	if !ok || got2 != nil {
		t.Errorf("results[media-2]: expected explicit null, got %v", got2)
	}
}

func TestGetMessageMediaByUUIDs_OmitsVariantsWhenNil(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.URL.Query()["variants"]; ok {
			t.Error("variants must be omitted when nil")
		}
		_, _ = io.WriteString(w, `{"errors":[],"results":{}}`)
	})
	if _, err := c.GetMessageMediaByUUIDs(context.Background(), "u-1", "m-1",
		GetMessageMediaByUUIDsParams{MediaUUIDs: "media-1"}); err != nil {
		t.Fatalf("GetMessageMediaByUUIDs: %v", err)
	}
}

func TestGetMessageMediaByUUIDs_Validation(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler must not be called when validation fails")
	})
	p := GetMessageMediaByUUIDsParams{MediaUUIDs: "media-1"}
	if _, err := c.GetMessageMediaByUUIDs(context.Background(), "", "m-1", p); err == nil {
		t.Error("expected error for empty user UUID")
	}
	if _, err := c.GetMessageMediaByUUIDs(context.Background(), "u-1", "", p); err == nil {
		t.Error("expected error for empty message UUID")
	}
	if _, err := c.GetMessageMediaByUUIDs(context.Background(), "u-1", "m-1",
		GetMessageMediaByUUIDsParams{}); err == nil {
		t.Error("expected error for empty MediaUUIDs")
	}
}

// --- list_media -------------------------------------------------------------

func TestListMedia_Success(t *testing.T) {
	const body = `{
	  "data": [
	    {
	      "created_at": "2026-01-01T00:00:00.000Z",
	      "mediaType": "video",
	      "messageUuid": "m-1",
	      "name": null,
	      "ownerUuid": "u-owner",
	      "sentAt": "2026-01-01T00:00:01.000Z",
	      "uuid": "media-1",
	      "variants": [
	        {"displayPosition": 0, "height": null, "lengthMs": 5000, "url": null, "variantType": "thumbnail", "width": null}
	      ]
	    }
	  ],
	  "nextCursor": "cursor-2"
	}`
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/chats/u-1/media" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("cursor") != "cursor-1" {
			t.Errorf("cursor: got %q", q.Get("cursor"))
		}
		if q.Get("mediaType") != "video" {
			t.Errorf("mediaType: got %q", q.Get("mediaType"))
		}
		if q.Get("limit") != "25" {
			t.Errorf("limit: got %q", q.Get("limit"))
		}
		_, _ = io.WriteString(w, body)
	})

	page, err := c.ListMedia(context.Background(), "u-1", ListMediaParams{
		Cursor:    ptrString("cursor-1"),
		MediaType: ptrChatMediaType(ChatMediaTypeVideo),
		Limit:     ptrInt(25),
	})
	if err != nil {
		t.Fatalf("ListMedia: %v", err)
	}
	if len(page.Data) != 1 {
		t.Fatalf("expected 1 item, got %d", len(page.Data))
	}
	item := page.Data[0]
	if item.MediaType != ChatMediaTypeVideo || item.UUID != "media-1" {
		t.Errorf("item: got %+v", item)
	}
	if len(item.Variants) != 1 || item.Variants[0].VariantType != MediaVariantThumbnail {
		t.Errorf("variants: got %+v", item.Variants)
	}
	if item.Variants[0].URL != nil {
		t.Errorf("url should be nil, got %v", *item.Variants[0].URL)
	}
	if page.NextCursor == nil || *page.NextCursor != "cursor-2" {
		t.Errorf("nextCursor: got %v", page.NextCursor)
	}
}

func TestListMedia_OmitsUnsetParams(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		for _, key := range []string{"cursor", "mediaType", "limit"} {
			if _, ok := q[key]; ok {
				t.Errorf("%s must be omitted when unset", key)
			}
		}
		_, _ = io.WriteString(w, `{"data":[],"nextCursor":null}`)
	})
	page, err := c.ListMedia(context.Background(), "u-1", ListMediaParams{})
	if err != nil {
		t.Fatalf("ListMedia: %v", err)
	}
	if page.NextCursor != nil {
		t.Errorf("nextCursor should be nil, got %v", *page.NextCursor)
	}
}

func TestListMedia_RequiresUserUUID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler must not be called when validation fails")
	})
	if _, err := c.ListMedia(context.Background(), "", ListMediaParams{}); err == nil {
		t.Error("expected error for empty user UUID")
	}
}

// --- messages_batch ---------------------------------------------------------

func TestMessagesBatch_Success(t *testing.T) {
	const body = `{
	  "byChat": {
	    "u-ok": {
	      "hasMore": true,
	      "oldestMessageUuid": "m-old",
	      "messages": [
	        {
	          "hasMedia": false,
	          "isRead": true,
	          "mediaType": null,
	          "mediaUuids": [],
	          "pricing": null,
	          "purchasedAt": null,
	          "recipient": {"handle": "fan", "uuid": "u-ok"},
	          "sender": {"handle": "creator", "uuid": "u-me"},
	          "sentAt": "2026-01-01T00:00:00.000Z",
	          "sentByUserId": "u-me",
	          "text": "hi",
	          "type": "SINGLE_RECIPIENT",
	          "uuid": "m-1"
	        }
	      ]
	    },
	    "u-bad": {"error": "forbidden"}
	  }
	}`
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/chats/messages/batch" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		sent := decodeBody(t, r)
		uuids, ok := sent["userUuids"].([]any)
		if !ok || len(uuids) != 2 {
			t.Errorf("userUuids: got %v", sent["userUuids"])
		}
		_, _ = io.WriteString(w, body)
	})

	res, err := c.MessagesBatch(context.Background(), MessagesBatchParams{
		UserUUIDs: []string{"u-ok", "u-bad"},
		Limit:     ptrInt(10),
	})
	if err != nil {
		t.Fatalf("MessagesBatch: %v", err)
	}
	ok := res.ByChat["u-ok"]
	if ok.Error != "" || !ok.HasMore || len(ok.Messages) != 1 {
		t.Errorf("u-ok: got %+v", ok)
	}
	if ok.Messages[0].UUID != "m-1" || ok.Messages[0].Type != "SINGLE_RECIPIENT" {
		t.Errorf("u-ok message: got %+v", ok.Messages[0])
	}
	if ok.Messages[0].Recipient.UUID == nil || *ok.Messages[0].Recipient.UUID != "u-ok" {
		t.Errorf("recipient: got %+v", ok.Messages[0].Recipient)
	}
	bad := res.ByChat["u-bad"]
	if bad.Error != "forbidden" {
		t.Errorf("u-bad: expected error 'forbidden', got %+v", bad)
	}
}

func TestMessagesBatch_RawBodyTakesPrecedence(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var sent map[string]any
		if err := json.Unmarshal(raw, &sent); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if _, ok := sent["userUuids"]; ok {
			t.Error("typed userUuids must not be sent when RawBody is set")
		}
		if sent["custom"] != "value" {
			t.Errorf("RawBody not sent verbatim: got %v", sent)
		}
		_, _ = io.WriteString(w, `{"byChat":{}}`)
	})
	_, err := c.MessagesBatch(context.Background(), MessagesBatchParams{
		UserUUIDs: []string{"ignored"},
		RawBody:   RawBody(`{"custom":"value"}`),
	})
	if err != nil {
		t.Fatalf("MessagesBatch: %v", err)
	}
}

func TestMessagesBatch_RequiresUserUUIDs(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler must not be called when validation fails")
	})
	if _, err := c.MessagesBatch(context.Background(), MessagesBatchParams{}); err == nil {
		t.Error("expected error for empty UserUUIDs")
	}
}

// --- list_mass_messages -----------------------------------------------------

func TestListMassMessages_Success(t *testing.T) {
	const body = `{
	  "data": [
	    {
	      "createdAt": "2026-01-01T00:00:00.000Z",
	      "mediaUuids": ["media-1"],
	      "price": 500,
	      "publishedAt": "2026-01-01T00:00:05.000Z",
	      "purchaseCount": 3,
	      "recipientCount": 100,
	      "scheduledAt": null,
	      "status": "SENT",
	      "text": "hello",
	      "totalRevenue": 1500,
	      "uuid": "mm-1",
	      "viewCount": 42
	    }
	  ],
	  "pagination": {"page": 1, "size": 10, "hasMore": false}
	}`
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/chats/mass-messages" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("page") != "1" || q.Get("size") != "10" {
			t.Errorf("page/size: got %q/%q", q.Get("page"), q.Get("size"))
		}
		// includeDeleted must serialize as the literal "true" string.
		if q.Get("includeDeleted") != "true" {
			t.Errorf("includeDeleted: got %q", q.Get("includeDeleted"))
		}
		_, _ = io.WriteString(w, body)
	})

	page, err := c.ListMassMessages(context.Background(), ListMassMessagesParams{
		Page:           ptrInt(1),
		Size:           ptrInt(10),
		IncludeDeleted: ptrBool(true),
	})
	if err != nil {
		t.Fatalf("ListMassMessages: %v", err)
	}
	if len(page.Data) != 1 {
		t.Fatalf("expected 1 item, got %d", len(page.Data))
	}
	mm := page.Data[0]
	if mm.UUID != "mm-1" || mm.Status != MassMessageStatusSent {
		t.Errorf("mass message: got %+v", mm)
	}
	if mm.Price == nil || *mm.Price != 500 {
		t.Errorf("price: got %v", mm.Price)
	}
	if mm.ScheduledAt != nil {
		t.Errorf("scheduledAt should be nil, got %v", *mm.ScheduledAt)
	}
	if page.Pagination.HasMore {
		t.Error("hasMore should be false")
	}
}

func TestListMassMessages_IncludeDeletedFalse(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("includeDeleted"); got != "false" {
			t.Errorf("includeDeleted: expected 'false', got %q", got)
		}
		_, _ = io.WriteString(w, `{"data":[],"pagination":{"page":1,"size":10,"hasMore":false}}`)
	})
	if _, err := c.ListMassMessages(context.Background(), ListMassMessagesParams{
		IncludeDeleted: ptrBool(false),
	}); err != nil {
		t.Fatalf("ListMassMessages: %v", err)
	}
}

func TestListMassMessages_OmitsUnsetParams(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.URL.Query()["includeDeleted"]; ok {
			t.Error("includeDeleted must be omitted when nil")
		}
		_, _ = io.WriteString(w, `{"data":[],"pagination":{"page":1,"size":10,"hasMore":false}}`)
	})
	if _, err := c.ListMassMessages(context.Background(), ListMassMessagesParams{}); err != nil {
		t.Fatalf("ListMassMessages: %v", err)
	}
}

// --- send_mass_message ------------------------------------------------------

func TestSendMassMessage_Success(t *testing.T) {
	const body = `{"createdAt":"2026-01-01T00:00:00.000Z","id":"mm-1","recipientCount":100}`
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/chats/mass-messages" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		sent := decodeBody(t, r)
		if sent["text"] != "broadcast" {
			t.Errorf("text: got %v", sent["text"])
		}
		if sent["price"] != float64(500) {
			t.Errorf("price: got %v", sent["price"])
		}
		smart, ok := sent["smartListIds"].([]any)
		if !ok || len(smart) != 1 || smart[0] != "subscribers" {
			t.Errorf("smartListIds: got %v", sent["smartListIds"])
		}
		_, _ = io.WriteString(w, body)
	})

	res, err := c.SendMassMessage(context.Background(), SendMassMessageParams{
		Text:         ptrString("broadcast"),
		Price:        ptrFloat64(500),
		SmartListIDs: []string{"subscribers"},
	})
	if err != nil {
		t.Fatalf("SendMassMessage: %v", err)
	}
	if res.ID != "mm-1" || res.RecipientCount != 100 {
		t.Errorf("result: got %+v", res)
	}
	if res.CreatedAt == nil || *res.CreatedAt != "2026-01-01T00:00:00.000Z" {
		t.Errorf("createdAt: got %v", res.CreatedAt)
	}
}

func TestSendMassMessage_Scheduled(t *testing.T) {
	const body = `{"createdAt":null,"id":"mm-2","recipientCount":50}`
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		sent := decodeBody(t, r)
		if sent["scheduledAt"] != "2026-12-01T00:00:00.000Z" {
			t.Errorf("scheduledAt: got %v", sent["scheduledAt"])
		}
		_, _ = io.WriteString(w, body)
	})
	res, err := c.SendMassMessage(context.Background(), SendMassMessageParams{
		Text:        ptrString("later"),
		ScheduledAt: ptrString("2026-12-01T00:00:00.000Z"),
	})
	if err != nil {
		t.Fatalf("SendMassMessage: %v", err)
	}
	if res.CreatedAt != nil {
		t.Errorf("createdAt should be nil for scheduled, got %v", *res.CreatedAt)
	}
}

func TestSendMassMessage_RawBodyTakesPrecedence(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var sent map[string]any
		if err := json.Unmarshal(raw, &sent); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if _, ok := sent["text"]; ok {
			t.Error("typed text must not be sent when RawBody is set")
		}
		if sent["raw"] != "only" {
			t.Errorf("RawBody not sent verbatim: got %v", sent)
		}
		_, _ = io.WriteString(w, `{"createdAt":null,"id":"mm-3","recipientCount":0}`)
	})
	if _, err := c.SendMassMessage(context.Background(), SendMassMessageParams{
		Text:    ptrString("ignored"),
		RawBody: RawBody(`{"raw":"only"}`),
	}); err != nil {
		t.Fatalf("SendMassMessage: %v", err)
	}
}

// --- update_mass_message ----------------------------------------------------

func TestUpdateMassMessage_Success(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/chats/mass-messages/mm-1" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		sent := decodeBody(t, r)
		if sent["text"] != "edited" {
			t.Errorf("text: got %v", sent["text"])
		}
		if _, ok := sent["price"]; ok {
			t.Error("unset price must be omitted")
		}
		w.WriteHeader(http.StatusNoContent)
	})
	if err := c.UpdateMassMessage(context.Background(), "mm-1", UpdateMassMessageParams{
		Text: ptrString("edited"),
	}); err != nil {
		t.Fatalf("UpdateMassMessage: %v", err)
	}
}

func TestUpdateMassMessage_RawBodyTakesPrecedence(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var sent map[string]any
		if err := json.Unmarshal(raw, &sent); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if _, ok := sent["text"]; ok {
			t.Error("typed text must not be sent when RawBody is set")
		}
		// A RawBody can express an explicit JSON null to clear a field.
		val, present := sent["price"]
		if !present || val != nil {
			t.Errorf("expected explicit price:null, got present=%v val=%v", present, val)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	if err := c.UpdateMassMessage(context.Background(), "mm-1", UpdateMassMessageParams{
		Text:    ptrString("ignored"),
		RawBody: RawBody(`{"price":null}`),
	}); err != nil {
		t.Fatalf("UpdateMassMessage: %v", err)
	}
}

func TestUpdateMassMessage_RequiresMessageUUID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler must not be called when validation fails")
	})
	if err := c.UpdateMassMessage(context.Background(), "", UpdateMassMessageParams{}); err == nil {
		t.Error("expected error for empty message UUID")
	}
}

// --- delete_mass_message ----------------------------------------------------

func TestDeleteMassMessage_Success(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/chats/mass-messages/mm-1" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	if err := c.DeleteMassMessage(context.Background(), "mm-1"); err != nil {
		t.Fatalf("DeleteMassMessage: %v", err)
	}
}

func TestDeleteMassMessage_RequiresMessageUUID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler must not be called when validation fails")
	})
	if err := c.DeleteMassMessage(context.Background(), ""); err == nil {
		t.Error("expected error for empty message UUID")
	}
}

func TestDeleteMassMessage_PropagatesAPIError(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"message":"forbidden"}`)
	})
	err := c.DeleteMassMessage(context.Background(), "mm-1")
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := AsError(err)
	if !ok || !apiErr.IsForbidden() {
		t.Errorf("expected 403 Error, got %v", err)
	}
}
