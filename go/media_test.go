package fanvue

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"testing"
)

const finalisedMediaJSON = `{
  "uuid": "m-1",
  "status": "ready",
  "caption": "a caption",
  "createdAt": "2026-01-01T00:00:00.000Z",
  "description": "desc",
  "mediaType": "image",
  "name": "file.jpg",
  "purchasedByFan": true,
  "recommendedPrice": 999,
  "tags": {
    "bodyParts": ["face"],
    "bodyType": ["slim"],
    "description": "tagdesc",
    "hairColor": ["brown"],
    "importantTags": ["a"],
    "isNsfw": false,
    "mediaType": "image",
    "nsfwCategory": [],
    "otherTags": ["b"],
    "people": ["one"],
    "position": ["standing"],
    "setting": ["indoor"],
    "sexActs": [],
    "sexObjects": [],
    "skinColor": ["fair"],
    "tags": ["c"]
  },
  "url": "https://cdn.example/m-1",
  "variants": [
    {
      "displayPosition": 0,
      "height": 1080,
      "lengthMs": null,
      "url": "https://cdn.example/m-1/main",
      "uuid": "v-1",
      "variantType": "main",
      "width": 1920
    }
  ]
}`

func assertFinalisedMedia(t *testing.T, m *MediaItem) {
	t.Helper()
	if m.UUID != "m-1" {
		t.Errorf("uuid: got %q", m.UUID)
	}
	if m.Status != MediaStatusReady {
		t.Errorf("status: got %q", m.Status)
	}
	if m.MediaType == nil || *m.MediaType != MediaTypeImage {
		t.Errorf("mediaType: got %v", m.MediaType)
	}
	if m.PurchasedByFan == nil || !*m.PurchasedByFan {
		t.Errorf("purchasedByFan: got %v", m.PurchasedByFan)
	}
	if m.RecommendedPrice == nil || *m.RecommendedPrice != 999 {
		t.Errorf("recommendedPrice: got %v", m.RecommendedPrice)
	}
	if m.Tags == nil || m.Tags.IsNsfw {
		t.Errorf("tags: got %+v", m.Tags)
	}
	if len(m.Variants) != 1 {
		t.Fatalf("variants: got %d", len(m.Variants))
	}
	v := m.Variants[0]
	if v.UUID != "v-1" || v.VariantType != MediaVariantMain {
		t.Errorf("variant: got %+v", v)
	}
	if v.URL == nil || *v.URL != "https://cdn.example/m-1/main" {
		t.Errorf("variant url: got %v", v.URL)
	}
	if v.LengthMs != nil {
		t.Errorf("variant lengthMs should be nil, got %v", *v.LengthMs)
	}
	if v.Width == nil || *v.Width != 1920 {
		t.Errorf("variant width: got %v", v.Width)
	}
}

func TestGetUserMedia(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/media" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("page") != "2" || q.Get("size") != "10" {
			t.Errorf("pagination: page=%q size=%q", q.Get("page"), q.Get("size"))
		}
		if q.Get("mediaType") != "image" {
			t.Errorf("mediaType: got %q", q.Get("mediaType"))
		}
		if q.Get("usage") != "ppv" {
			t.Errorf("usage: got %q", q.Get("usage"))
		}
		if q.Get("purchasedBy") != "fan-1" {
			t.Errorf("purchasedBy: got %q", q.Get("purchasedBy"))
		}
		statuses := q["status"]
		sort.Strings(statuses)
		if len(statuses) != 2 || statuses[0] != "processing" || statuses[1] != "ready" {
			t.Errorf("status repeated keys: got %v", q["status"])
		}
		variants := q["variants"]
		sort.Strings(variants)
		if len(variants) != 2 || variants[0] != "main" || variants[1] != "thumbnail" {
			t.Errorf("variants repeated keys: got %v", q["variants"])
		}
		_, _ = io.WriteString(w, `{
		  "data": [
		    {"uuid": "m-pending", "status": "processing"},
		    `+finalisedMediaJSON+`
		  ],
		  "pagination": {"page": 2, "size": 10, "hasMore": false}
		}`)
	})

	page, err := c.GetUserMedia(context.Background(), ListUserMediaParams{
		Page:        ptrInt(2),
		Size:        ptrInt(10),
		MediaType:   mediaTypePtr(MediaTypeImage),
		Usage:       mediaUsagePtr(MediaUsagePPV),
		PurchasedBy: ptrString("fan-1"),
		Status:      []MediaStatus{MediaStatusReady, MediaStatusProcessing},
		Variants:    []MediaVariantType{MediaVariantMain, MediaVariantThumbnail},
	})
	if err != nil {
		t.Fatalf("GetUserMedia: %v", err)
	}
	if len(page.Data) != 2 {
		t.Fatalf("data: got %d", len(page.Data))
	}
	// Non-finalised item carries only uuid + status.
	if page.Data[0].UUID != "m-pending" || page.Data[0].Status != MediaStatusProcessing {
		t.Errorf("pending item: got %+v", page.Data[0])
	}
	if page.Data[0].MediaType != nil {
		t.Errorf("pending mediaType should be nil, got %v", page.Data[0].MediaType)
	}
	assertFinalisedMedia(t, &page.Data[1])
	if page.Pagination.Page != 2 || page.Pagination.HasMore {
		t.Errorf("pagination: got %+v", page.Pagination)
	}
}

func TestGetUserMedia_OmitsUnsetParams(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("expected no query params, got %q", r.URL.RawQuery)
		}
		_, _ = io.WriteString(w, `{"data": [], "pagination": {"page": 1, "size": 20, "hasMore": false}}`)
	})
	if _, err := c.GetUserMedia(context.Background(), ListUserMediaParams{}); err != nil {
		t.Fatalf("GetUserMedia: %v", err)
	}
}

func TestGetUserMediaByUUID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/media/m-1" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		if r.URL.Query().Get("purchasedBy") != "fan-9" {
			t.Errorf("purchasedBy: got %q", r.URL.Query().Get("purchasedBy"))
		}
		if got := r.URL.Query()["variants"]; len(got) != 1 || got[0] != "main" {
			t.Errorf("variants: got %v", got)
		}
		_, _ = io.WriteString(w, finalisedMediaJSON)
	})
	m, err := c.GetUserMediaByUUID(context.Background(), "m-1", GetUserMediaByUUIDParams{
		PurchasedBy: ptrString("fan-9"),
		Variants:    []MediaVariantType{MediaVariantMain},
	})
	if err != nil {
		t.Fatalf("GetUserMediaByUUID: %v", err)
	}
	assertFinalisedMedia(t, m)
}

func TestGetUserMediaByUUID_RequiresUUID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when UUID is empty")
	})
	if _, err := c.GetUserMediaByUUID(context.Background(), "", GetUserMediaByUUIDParams{}); err == nil {
		t.Fatal("expected error for empty UUID")
	}
}

func TestGetBulkMedia(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/media/bulk" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("mediaUuids") != "m-1,m-2" {
			t.Errorf("mediaUuids: got %q", q.Get("mediaUuids"))
		}
		if q.Get("variants") != "main,thumbnail" {
			t.Errorf("variants: got %q", q.Get("variants"))
		}
		_, _ = io.WriteString(w, `{
		  "errors": [{"code": "NOT_FOUND", "mediaUuid": "m-2", "message": "missing"}],
		  "results": {
		    "m-1": `+finalisedMediaJSON+`,
		    "m-2": null
		  }
		}`)
	})
	res, err := c.GetBulkMedia(context.Background(), GetBulkMediaParams{
		MediaUUIDs: "m-1,m-2",
		Variants:   ptrString("main,thumbnail"),
	})
	if err != nil {
		t.Fatalf("GetBulkMedia: %v", err)
	}
	if len(res.Errors) != 1 || res.Errors[0].Code != "NOT_FOUND" || res.Errors[0].MediaUUID != "m-2" {
		t.Errorf("errors: got %+v", res.Errors)
	}
	item, ok := res.Results["m-1"]
	if !ok || item == nil {
		t.Fatalf("results[m-1]: got %v", res.Results["m-1"])
	}
	assertFinalisedMedia(t, item)
	if v, ok := res.Results["m-2"]; !ok || v != nil {
		t.Errorf("results[m-2] should be present and nil, got ok=%v v=%v", ok, v)
	}
}

func TestGetBulkMedia_RequiresMediaUUIDs(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when MediaUUIDs is empty")
	})
	if _, err := c.GetBulkMedia(context.Background(), GetBulkMediaParams{}); err == nil {
		t.Fatal("expected error for empty MediaUUIDs")
	}
}

func TestGetEntitledMedia(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/media/m-1/entitled" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		if r.URL.Query().Get("consumerId") != "c-1" {
			t.Errorf("consumerId: got %q", r.URL.Query().Get("consumerId"))
		}
		if got := r.URL.Query()["variants"]; len(got) != 1 || got[0] != "main" {
			t.Errorf("variants: got %v", got)
		}
		_, _ = io.WriteString(w, finalisedMediaJSON)
	})
	m, err := c.GetEntitledMedia(context.Background(), "m-1", GetEntitledMediaParams{
		ConsumerID: "c-1",
		Variants:   []MediaVariantType{MediaVariantMain},
	})
	if err != nil {
		t.Fatalf("GetEntitledMedia: %v", err)
	}
	assertFinalisedMedia(t, m)
}

func TestGetEntitledMedia_RequiresConsumerID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when ConsumerID is empty")
	})
	if _, err := c.GetEntitledMedia(context.Background(), "m-1", GetEntitledMediaParams{}); err == nil {
		t.Fatal("expected error for empty ConsumerID")
	}
}

func TestGetMediaLinkPurchaseStatus(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/media/links/link-1/purchased" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, `{"purchased": true}`)
	})
	status, err := c.GetMediaLinkPurchaseStatus(context.Background(), "link-1")
	if err != nil {
		t.Fatalf("GetMediaLinkPurchaseStatus: %v", err)
	}
	if !status.Purchased {
		t.Error("expected purchased=true")
	}
}

func TestGetMediaLinkPurchaseStatus_NotFound(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"message": "media link not found"}`)
	})
	_, err := c.GetMediaLinkPurchaseStatus(context.Background(), "missing")
	apiErr, ok := AsError(err)
	if !ok {
		t.Fatalf("expected *Error, got %v", err)
	}
	if !apiErr.IsNotFound() {
		t.Errorf("expected 404, got %d", apiErr.StatusCode)
	}
}

func TestGrantMedia(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/media/m-1/grant" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var got map[string]any
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("body not JSON: %v (%s)", err, body)
		}
		if got["consumerId"] != "c-1" {
			t.Errorf("body consumerId: got %v", got["consumerId"])
		}
		_, _ = io.WriteString(w, `{"entitlementId": "ent-1", "status": "granted"}`)
	})
	res, err := c.GrantMedia(context.Background(), "m-1", RawBody(`{"consumerId":"c-1"}`))
	if err != nil {
		t.Fatalf("GrantMedia: %v", err)
	}
	if res.EntitlementID != "ent-1" || res.Status != "granted" {
		t.Errorf("result: got %+v", res)
	}
}

func TestGrantMedia_RequiresBody(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when body is empty")
	})
	if _, err := c.GrantMedia(context.Background(), "m-1", nil); err == nil {
		t.Fatal("expected error for empty body")
	}
}

func TestCreateUploadSession(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/media/uploads" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var got map[string]any
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("body not JSON: %v (%s)", err, body)
		}
		if got["mediaType"] != "image" {
			t.Errorf("body mediaType: got %v", got["mediaType"])
		}
		_, _ = io.WriteString(w, `{"mediaUuid": "m-1", "uploadId": "up-1"}`)
	})
	res, err := c.CreateUploadSession(context.Background(), RawBody(`{"mediaType":"image","parts":3}`))
	if err != nil {
		t.Fatalf("CreateUploadSession: %v", err)
	}
	if res.MediaUUID != "m-1" || res.UploadID != "up-1" {
		t.Errorf("result: got %+v", res)
	}
}

func TestCreateUploadSession_RequiresBody(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when body is empty")
	})
	if _, err := c.CreateUploadSession(context.Background(), nil); err == nil {
		t.Fatal("expected error for empty body")
	}
}

func TestCompleteUploadSession(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/media/uploads/up-1" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var got map[string]any
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("body not JSON: %v (%s)", err, body)
		}
		if _, ok := got["parts"]; !ok {
			t.Errorf("expected parts in body, got %s", body)
		}
		_, _ = io.WriteString(w, `{"status": "processing"}`)
	})
	res, err := c.CompleteUploadSession(context.Background(), "up-1", RawBody(`{"parts":[{"partNumber":1,"etag":"e1"}]}`))
	if err != nil {
		t.Fatalf("CompleteUploadSession: %v", err)
	}
	if res.Status != MediaStatusProcessing {
		t.Errorf("status: got %q", res.Status)
	}
}

func TestCompleteUploadSession_RequiresUploadIDAndBody(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called on validation error")
	})
	if _, err := c.CompleteUploadSession(context.Background(), "", RawBody(`{}`)); err == nil {
		t.Fatal("expected error for empty upload ID")
	}
	if _, err := c.CompleteUploadSession(context.Background(), "up-1", nil); err == nil {
		t.Fatal("expected error for empty body")
	}
}

func TestGetUploadPartURL(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/media/uploads/up-1/parts/2/url" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, `{"url": "https://s3.example/signed?part=2"}`)
	})
	raw, err := c.GetUploadPartURL(context.Background(), "up-1", 2)
	if err != nil {
		t.Fatalf("GetUploadPartURL: %v", err)
	}
	var got struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("raw body not JSON: %v (%s)", err, raw)
	}
	if got.URL != "https://s3.example/signed?part=2" {
		t.Errorf("signed url: got %q", got.URL)
	}
}

func TestGetUploadPartURL_RequiresValidArgs(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called on validation error")
	})
	if _, err := c.GetUploadPartURL(context.Background(), "", 1); err == nil {
		t.Fatal("expected error for empty upload ID")
	}
	if _, err := c.GetUploadPartURL(context.Background(), "up-1", 0); err == nil {
		t.Fatal("expected error for non-positive part number")
	}
}

// mediaTypePtr and mediaUsagePtr are test helpers for building optional enum
// query parameters.
func mediaTypePtr(t MediaType) *MediaType    { return &t }
func mediaUsagePtr(u MediaUsage) *MediaUsage { return &u }
