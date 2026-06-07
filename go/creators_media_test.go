package fanvue

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"testing"
)

func TestGetCreatorMedia(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/creators/cr-1/media" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("page") != "2" || q.Get("size") != "10" {
			t.Errorf("pagination: page=%q size=%q", q.Get("page"), q.Get("size"))
		}
		if q.Get("mediaType") != "video" {
			t.Errorf("mediaType: got %q", q.Get("mediaType"))
		}
		if q.Get("folderName") != "Best Of" {
			t.Errorf("folderName: got %q", q.Get("folderName"))
		}
		if q.Get("usage") != "subscribers" {
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
		  "data": [`+finalisedMediaJSON+`, {"uuid": "m-2", "status": "created"}],
		  "pagination": {"page": 2, "size": 10, "hasMore": true}
		}`)
	})

	page, err := c.GetCreatorMedia(context.Background(), "cr-1", ListCreatorMediaParams{
		Page:        ptrInt(2),
		Size:        ptrInt(10),
		MediaType:   mediaTypePtr(MediaTypeVideo),
		FolderName:  ptrString("Best Of"),
		Usage:       mediaUsagePtr(MediaUsageSubscribers),
		PurchasedBy: ptrString("fan-1"),
		Status:      []MediaStatus{MediaStatusReady, MediaStatusProcessing},
		Variants:    []MediaVariantType{MediaVariantMain, MediaVariantThumbnail},
	})
	if err != nil {
		t.Fatalf("GetCreatorMedia: %v", err)
	}
	if len(page.Data) != 2 {
		t.Fatalf("data: got %d", len(page.Data))
	}
	assertFinalisedMedia(t, &page.Data[0])
	if page.Data[1].UUID != "m-2" || page.Data[1].Status != MediaStatusCreated {
		t.Errorf("non-finalised item: got %+v", page.Data[1])
	}
	if page.Data[1].MediaType != nil {
		t.Errorf("non-finalised mediaType should be nil, got %v", page.Data[1].MediaType)
	}
	if !page.Pagination.HasMore {
		t.Error("hasMore should be true")
	}
}

func TestGetCreatorMedia_OmitsUnsetParams(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("expected no query params, got %q", r.URL.RawQuery)
		}
		_, _ = io.WriteString(w, `{"data": [], "pagination": {"page": 1, "size": 20, "hasMore": false}}`)
	})
	if _, err := c.GetCreatorMedia(context.Background(), "cr-1", ListCreatorMediaParams{}); err != nil {
		t.Fatalf("GetCreatorMedia: %v", err)
	}
}

func TestGetCreatorMedia_RequiresCreatorUUID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called on validation error")
	})
	if _, err := c.GetCreatorMedia(context.Background(), "", ListCreatorMediaParams{}); err == nil {
		t.Fatal("expected error for empty creator UUID")
	}
}

func TestGetCreatorMediaByUUID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/creators/cr-1/media/m-1" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("purchasedBy") != "fan-9" {
			t.Errorf("purchasedBy: got %q", q.Get("purchasedBy"))
		}
		variants := q["variants"]
		if len(variants) != 1 || variants[0] != "thumbnail_gallery" {
			t.Errorf("variants: got %v", variants)
		}
		_, _ = io.WriteString(w, finalisedMediaJSON)
	})

	m, err := c.GetCreatorMediaByUUID(context.Background(), "cr-1", "m-1", GetCreatorMediaByUUIDParams{
		PurchasedBy: ptrString("fan-9"),
		Variants:    []MediaVariantType{MediaVariantThumbnailGallery},
	})
	if err != nil {
		t.Fatalf("GetCreatorMediaByUUID: %v", err)
	}
	assertFinalisedMedia(t, m)
}

func TestGetCreatorMediaByUUID_RequiresArgs(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called on validation error")
	})
	if _, err := c.GetCreatorMediaByUUID(context.Background(), "", "m-1", GetCreatorMediaByUUIDParams{}); err == nil {
		t.Fatal("expected error for empty creator UUID")
	}
	if _, err := c.GetCreatorMediaByUUID(context.Background(), "cr-1", "", GetCreatorMediaByUUIDParams{}); err == nil {
		t.Fatal("expected error for empty media UUID")
	}
}

func TestCreateCreatorUploadSession(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/creators/cr-1/media/uploads" {
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
	res, err := c.CreateCreatorUploadSession(context.Background(), "cr-1", RawBody(`{"mediaType":"image","parts":3}`))
	if err != nil {
		t.Fatalf("CreateCreatorUploadSession: %v", err)
	}
	if res.MediaUUID != "m-1" || res.UploadID != "up-1" {
		t.Errorf("result: got %+v", res)
	}
}

func TestCreateCreatorUploadSession_RequiresArgs(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called on validation error")
	})
	if _, err := c.CreateCreatorUploadSession(context.Background(), "", RawBody(`{}`)); err == nil {
		t.Fatal("expected error for empty creator UUID")
	}
	if _, err := c.CreateCreatorUploadSession(context.Background(), "cr-1", nil); err == nil {
		t.Fatal("expected error for empty body")
	}
}

func TestCompleteCreatorUploadSession(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/creators/cr-1/media/uploads/up-1" {
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
	res, err := c.CompleteCreatorUploadSession(context.Background(), "cr-1", "up-1", RawBody(`{"parts":[{"partNumber":1,"etag":"e1"}]}`))
	if err != nil {
		t.Fatalf("CompleteCreatorUploadSession: %v", err)
	}
	if res.Status != MediaStatusProcessing {
		t.Errorf("status: got %q", res.Status)
	}
}

func TestCompleteCreatorUploadSession_RequiresArgs(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called on validation error")
	})
	if _, err := c.CompleteCreatorUploadSession(context.Background(), "", "up-1", RawBody(`{}`)); err == nil {
		t.Fatal("expected error for empty creator UUID")
	}
	if _, err := c.CompleteCreatorUploadSession(context.Background(), "cr-1", "", RawBody(`{}`)); err == nil {
		t.Fatal("expected error for empty upload ID")
	}
	if _, err := c.CompleteCreatorUploadSession(context.Background(), "cr-1", "up-1", nil); err == nil {
		t.Fatal("expected error for empty body")
	}
}

func TestGetCreatorUploadPartURL(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/creators/cr-1/media/uploads/up-1/parts/2/url" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, `{"url": "https://s3.example/signed?part=2"}`)
	})
	raw, err := c.GetCreatorUploadPartURL(context.Background(), "cr-1", "up-1", 2)
	if err != nil {
		t.Fatalf("GetCreatorUploadPartURL: %v", err)
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

func TestGetCreatorUploadPartURL_RequiresValidArgs(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called on validation error")
	})
	if _, err := c.GetCreatorUploadPartURL(context.Background(), "", "up-1", 1); err == nil {
		t.Fatal("expected error for empty creator UUID")
	}
	if _, err := c.GetCreatorUploadPartURL(context.Background(), "cr-1", "", 1); err == nil {
		t.Fatal("expected error for empty upload ID")
	}
	if _, err := c.GetCreatorUploadPartURL(context.Background(), "cr-1", "up-1", 0); err == nil {
		t.Fatal("expected error for non-positive part number")
	}
}

func TestListCreatorVaultFolders(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/creators/cr-1/vault/folders" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, `{
		  "data": [
		    {"createdAt": "2026-01-01T00:00:00.000Z", "mediaCount": 4, "name": "Best Of"},
		    {"createdAt": null, "mediaCount": 0, "name": "Empty"}
		  ],
		  "pagination": {"page": 1, "size": 20, "hasMore": false}
		}`)
	})
	page, err := c.ListCreatorVaultFolders(context.Background(), "cr-1")
	if err != nil {
		t.Fatalf("ListCreatorVaultFolders: %v", err)
	}
	if len(page.Data) != 2 {
		t.Fatalf("data: got %d", len(page.Data))
	}
	if page.Data[0].Name != "Best Of" || page.Data[0].MediaCount != 4 {
		t.Errorf("folder[0]: got %+v", page.Data[0])
	}
	if page.Data[1].CreatedAt != nil {
		t.Errorf("folder[1] createdAt should be nil, got %v", *page.Data[1].CreatedAt)
	}
}

func TestListCreatorVaultFolders_RequiresCreatorUUID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called on validation error")
	})
	if _, err := c.ListCreatorVaultFolders(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty creator UUID")
	}
}

func TestCreateCreatorVaultFolder(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/creators/cr-1/vault/folders" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var got map[string]any
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("body not JSON: %v (%s)", err, body)
		}
		if got["name"] != "New Folder" {
			t.Errorf("body name: got %v", got["name"])
		}
		_, _ = io.WriteString(w, `{"createdAt": "2026-01-01T00:00:00.000Z", "mediaCount": 0, "name": "New Folder"}`)
	})
	folder, err := c.CreateCreatorVaultFolder(context.Background(), "cr-1", RawBody(`{"name":"New Folder"}`))
	if err != nil {
		t.Fatalf("CreateCreatorVaultFolder: %v", err)
	}
	if folder.Name != "New Folder" || folder.MediaCount != 0 {
		t.Errorf("folder: got %+v", folder)
	}
}

func TestCreateCreatorVaultFolder_RequiresArgs(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called on validation error")
	})
	if _, err := c.CreateCreatorVaultFolder(context.Background(), "", RawBody(`{}`)); err == nil {
		t.Fatal("expected error for empty creator UUID")
	}
	if _, err := c.CreateCreatorVaultFolder(context.Background(), "cr-1", nil); err == nil {
		t.Fatal("expected error for empty body")
	}
}

func TestGetCreatorVaultFolder(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/creators/cr-1/vault/folders/Best Of" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, `{"createdAt": null, "mediaCount": 7, "name": "Best Of"}`)
	})
	folder, err := c.GetCreatorVaultFolder(context.Background(), "cr-1", "Best Of")
	if err != nil {
		t.Fatalf("GetCreatorVaultFolder: %v", err)
	}
	if folder.Name != "Best Of" || folder.MediaCount != 7 {
		t.Errorf("folder: got %+v", folder)
	}
	if folder.CreatedAt != nil {
		t.Errorf("createdAt should be nil, got %v", *folder.CreatedAt)
	}
}

func TestGetCreatorVaultFolder_RequiresArgs(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called on validation error")
	})
	if _, err := c.GetCreatorVaultFolder(context.Background(), "", "Best Of"); err == nil {
		t.Fatal("expected error for empty creator UUID")
	}
	if _, err := c.GetCreatorVaultFolder(context.Background(), "cr-1", ""); err == nil {
		t.Fatal("expected error for empty folder name")
	}
}

func TestRenameCreatorVaultFolder(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/creators/cr-1/vault/folders/Old Name" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var got map[string]any
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("body not JSON: %v (%s)", err, body)
		}
		if got["name"] != "New Name" {
			t.Errorf("body name: got %v", got["name"])
		}
		w.WriteHeader(http.StatusNoContent)
	})
	if err := c.RenameCreatorVaultFolder(context.Background(), "cr-1", "Old Name", RawBody(`{"name":"New Name"}`)); err != nil {
		t.Fatalf("RenameCreatorVaultFolder: %v", err)
	}
}

func TestRenameCreatorVaultFolder_RequiresArgs(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called on validation error")
	})
	if err := c.RenameCreatorVaultFolder(context.Background(), "", "f", RawBody(`{}`)); err == nil {
		t.Fatal("expected error for empty creator UUID")
	}
	if err := c.RenameCreatorVaultFolder(context.Background(), "cr-1", "", RawBody(`{}`)); err == nil {
		t.Fatal("expected error for empty folder name")
	}
	if err := c.RenameCreatorVaultFolder(context.Background(), "cr-1", "f", nil); err == nil {
		t.Fatal("expected error for empty body")
	}
}

func TestDeleteCreatorVaultFolder(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/creators/cr-1/vault/folders/Best Of" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	if err := c.DeleteCreatorVaultFolder(context.Background(), "cr-1", "Best Of"); err != nil {
		t.Fatalf("DeleteCreatorVaultFolder: %v", err)
	}
}

func TestDeleteCreatorVaultFolder_RequiresArgs(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called on validation error")
	})
	if err := c.DeleteCreatorVaultFolder(context.Background(), "", "f"); err == nil {
		t.Fatal("expected error for empty creator UUID")
	}
	if err := c.DeleteCreatorVaultFolder(context.Background(), "cr-1", ""); err == nil {
		t.Fatal("expected error for empty folder name")
	}
}

func TestListCreatorVaultFolderMedia(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/creators/cr-1/vault/folders/Best Of/media" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("page") != "3" || q.Get("size") != "5" {
			t.Errorf("pagination: page=%q size=%q", q.Get("page"), q.Get("size"))
		}
		_, _ = io.WriteString(w, `{
		  "data": [
		    {"createdAt": "2026-01-01T00:00:00.000Z", "mediaType": "image", "name": "a.jpg", "uuid": "m-1"},
		    {"createdAt": null, "mediaType": "video", "name": null, "uuid": "m-2"}
		  ],
		  "pagination": {"page": 3, "size": 5, "hasMore": true}
		}`)
	})
	page, err := c.ListCreatorVaultFolderMedia(context.Background(), "cr-1", "Best Of", ListCreatorVaultFolderMediaParams{
		Page: ptrInt(3),
		Size: ptrInt(5),
	})
	if err != nil {
		t.Fatalf("ListCreatorVaultFolderMedia: %v", err)
	}
	if len(page.Data) != 2 {
		t.Fatalf("data: got %d", len(page.Data))
	}
	if page.Data[0].UUID != "m-1" || page.Data[0].MediaType != "image" {
		t.Errorf("item[0]: got %+v", page.Data[0])
	}
	if page.Data[0].Name == nil || *page.Data[0].Name != "a.jpg" {
		t.Errorf("item[0] name: got %v", page.Data[0].Name)
	}
	if page.Data[1].Name != nil {
		t.Errorf("item[1] name should be nil, got %v", *page.Data[1].Name)
	}
	if !page.Pagination.HasMore {
		t.Error("hasMore should be true")
	}
}

func TestListCreatorVaultFolderMedia_RequiresArgs(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called on validation error")
	})
	if _, err := c.ListCreatorVaultFolderMedia(context.Background(), "", "f", ListCreatorVaultFolderMediaParams{}); err == nil {
		t.Fatal("expected error for empty creator UUID")
	}
	if _, err := c.ListCreatorVaultFolderMedia(context.Background(), "cr-1", "", ListCreatorVaultFolderMediaParams{}); err == nil {
		t.Fatal("expected error for empty folder name")
	}
}

func TestAttachCreatorVaultMedia(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/creators/cr-1/vault/folders/Best Of/media" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var got map[string]any
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("body not JSON: %v (%s)", err, body)
		}
		if _, ok := got["mediaUuids"]; !ok {
			t.Errorf("expected mediaUuids in body, got %s", body)
		}
		_, _ = io.WriteString(w, `{"addedCount": 2}`)
	})
	res, err := c.AttachCreatorVaultMedia(context.Background(), "cr-1", "Best Of", RawBody(`{"mediaUuids":["m-1","m-2"]}`))
	if err != nil {
		t.Fatalf("AttachCreatorVaultMedia: %v", err)
	}
	if res.AddedCount != 2 {
		t.Errorf("addedCount: got %v", res.AddedCount)
	}
}

func TestAttachCreatorVaultMedia_RequiresArgs(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called on validation error")
	})
	if _, err := c.AttachCreatorVaultMedia(context.Background(), "", "f", RawBody(`{}`)); err == nil {
		t.Fatal("expected error for empty creator UUID")
	}
	if _, err := c.AttachCreatorVaultMedia(context.Background(), "cr-1", "", RawBody(`{}`)); err == nil {
		t.Fatal("expected error for empty folder name")
	}
	if _, err := c.AttachCreatorVaultMedia(context.Background(), "cr-1", "f", nil); err == nil {
		t.Fatal("expected error for empty body")
	}
}

func TestDetachCreatorVaultMedia(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/creators/cr-1/vault/folders/Best Of/media/m-1" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	if err := c.DetachCreatorVaultMedia(context.Background(), "cr-1", "Best Of", "m-1"); err != nil {
		t.Fatalf("DetachCreatorVaultMedia: %v", err)
	}
}

func TestDetachCreatorVaultMedia_RequiresArgs(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called on validation error")
	})
	if err := c.DetachCreatorVaultMedia(context.Background(), "", "f", "m-1"); err == nil {
		t.Fatal("expected error for empty creator UUID")
	}
	if err := c.DetachCreatorVaultMedia(context.Background(), "cr-1", "", "m-1"); err == nil {
		t.Fatal("expected error for empty folder name")
	}
	if err := c.DetachCreatorVaultMedia(context.Background(), "cr-1", "Best Of", ""); err == nil {
		t.Fatal("expected error for empty media UUID")
	}
}
