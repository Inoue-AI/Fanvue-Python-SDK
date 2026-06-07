package fanvue

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestListVaultFolders(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/vault/folders" {
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
	page, err := c.ListVaultFolders(context.Background())
	if err != nil {
		t.Fatalf("ListVaultFolders: %v", err)
	}
	if len(page.Data) != 2 {
		t.Fatalf("data: got %d", len(page.Data))
	}
	if page.Data[0].Name != "Best Of" || page.Data[0].MediaCount != 4 {
		t.Errorf("folder[0]: got %+v", page.Data[0])
	}
	if page.Data[0].CreatedAt == nil || *page.Data[0].CreatedAt == "" {
		t.Errorf("folder[0] createdAt: got %v", page.Data[0].CreatedAt)
	}
	if page.Data[1].CreatedAt != nil {
		t.Errorf("folder[1] createdAt should be nil, got %v", *page.Data[1].CreatedAt)
	}
	if page.Pagination.HasMore {
		t.Error("hasMore should be false")
	}
}

func TestCreateVaultFolder(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/vault/folders" {
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
	folder, err := c.CreateVaultFolder(context.Background(), RawBody(`{"name":"New Folder"}`))
	if err != nil {
		t.Fatalf("CreateVaultFolder: %v", err)
	}
	if folder.Name != "New Folder" || folder.MediaCount != 0 {
		t.Errorf("result: got %+v", folder)
	}
}

func TestCreateVaultFolder_RequiresBody(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when body is empty")
	})
	if _, err := c.CreateVaultFolder(context.Background(), nil); err == nil {
		t.Fatal("expected error for empty body")
	}
}

func TestGetVaultFolder(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.EscapedPath() != "/vault/folders/Best%20Of" {
			t.Errorf("unexpected path: %q", r.URL.EscapedPath())
		}
		_, _ = io.WriteString(w, `{"createdAt": "2026-01-01T00:00:00.000Z", "mediaCount": 4, "name": "Best Of"}`)
	})
	folder, err := c.GetVaultFolder(context.Background(), "Best Of")
	if err != nil {
		t.Fatalf("GetVaultFolder: %v", err)
	}
	if folder.Name != "Best Of" || folder.MediaCount != 4 {
		t.Errorf("result: got %+v", folder)
	}
}

func TestGetVaultFolder_RequiresName(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when folder name is empty")
	})
	if _, err := c.GetVaultFolder(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty folder name")
	}
}

func TestRenameVaultFolder(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/vault/folders/Old" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var got map[string]any
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("body not JSON: %v (%s)", err, body)
		}
		if got["name"] != "New" {
			t.Errorf("body name: got %v", got["name"])
		}
		w.WriteHeader(http.StatusNoContent)
	})
	if err := c.RenameVaultFolder(context.Background(), "Old", RawBody(`{"name":"New"}`)); err != nil {
		t.Fatalf("RenameVaultFolder: %v", err)
	}
}

func TestRenameVaultFolder_RequiresBody(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when body is empty")
	})
	if err := c.RenameVaultFolder(context.Background(), "Old", nil); err == nil {
		t.Fatal("expected error for empty body")
	}
}

func TestDeleteVaultFolder(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/vault/folders/Trash" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	if err := c.DeleteVaultFolder(context.Background(), "Trash"); err != nil {
		t.Fatalf("DeleteVaultFolder: %v", err)
	}
}

func TestDeleteVaultFolder_RequiresName(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when folder name is empty")
	})
	if err := c.DeleteVaultFolder(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty folder name")
	}
}

func TestListVaultFolderMedia(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.EscapedPath() != "/vault/folders/Best%20Of/media" {
			t.Errorf("unexpected path: %q", r.URL.EscapedPath())
		}
		q := r.URL.Query()
		if q.Get("page") != "2" || q.Get("size") != "5" {
			t.Errorf("pagination: page=%q size=%q", q.Get("page"), q.Get("size"))
		}
		_, _ = io.WriteString(w, `{
		  "data": [
		    {"createdAt": "2026-01-01T00:00:00.000Z", "mediaType": "image", "name": "a.jpg", "uuid": "m-1"},
		    {"createdAt": null, "mediaType": "video", "name": null, "uuid": "m-2"}
		  ],
		  "pagination": {"page": 2, "size": 5, "hasMore": true}
		}`)
	})
	page, err := c.ListVaultFolderMedia(context.Background(), "Best Of", ListVaultFolderMediaParams{
		Page: ptrInt(2),
		Size: ptrInt(5),
	})
	if err != nil {
		t.Fatalf("ListVaultFolderMedia: %v", err)
	}
	if len(page.Data) != 2 {
		t.Fatalf("data: got %d", len(page.Data))
	}
	if page.Data[0].UUID != "m-1" || page.Data[0].MediaType != "image" {
		t.Errorf("media[0]: got %+v", page.Data[0])
	}
	if page.Data[0].Name == nil || *page.Data[0].Name != "a.jpg" {
		t.Errorf("media[0] name: got %v", page.Data[0].Name)
	}
	if page.Data[1].Name != nil {
		t.Errorf("media[1] name should be nil, got %v", *page.Data[1].Name)
	}
	if !page.Pagination.HasMore {
		t.Error("hasMore should be true")
	}
}

func TestListVaultFolderMedia_RequiresName(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when folder name is empty")
	})
	if _, err := c.ListVaultFolderMedia(context.Background(), "", ListVaultFolderMediaParams{}); err == nil {
		t.Fatal("expected error for empty folder name")
	}
}

func TestAttachMediaToVaultFolder(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.EscapedPath() != "/vault/folders/Best%20Of/media" {
			t.Errorf("unexpected path: %q", r.URL.EscapedPath())
		}
		body, _ := io.ReadAll(r.Body)
		var got map[string]any
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("body not JSON: %v (%s)", err, body)
		}
		if _, ok := got["mediaUuids"]; !ok {
			t.Errorf("expected mediaUuids in body, got %s", body)
		}
		_, _ = io.WriteString(w, `{"addedCount": 3}`)
	})
	res, err := c.AttachMediaToVaultFolder(
		context.Background(), "Best Of", RawBody(`{"mediaUuids":["m-1","m-2","m-3"]}`),
	)
	if err != nil {
		t.Fatalf("AttachMediaToVaultFolder: %v", err)
	}
	if res.AddedCount != 3 {
		t.Errorf("addedCount: got %v", res.AddedCount)
	}
}

func TestAttachMediaToVaultFolder_RequiresBody(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when body is empty")
	})
	if _, err := c.AttachMediaToVaultFolder(context.Background(), "Best Of", nil); err == nil {
		t.Fatal("expected error for empty body")
	}
}

func TestDetachMediaFromVaultFolder(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.EscapedPath() != "/vault/folders/Best%20Of/media/m-1" {
			t.Errorf("unexpected path: %q", r.URL.EscapedPath())
		}
		w.WriteHeader(http.StatusNoContent)
	})
	if err := c.DetachMediaFromVaultFolder(context.Background(), "Best Of", "m-1"); err != nil {
		t.Fatalf("DetachMediaFromVaultFolder: %v", err)
	}
}

func TestDetachMediaFromVaultFolder_RequiresArgs(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when args are empty")
	})
	if err := c.DetachMediaFromVaultFolder(context.Background(), "", "m-1"); err == nil {
		t.Fatal("expected error for empty folder name")
	}
	if err := c.DetachMediaFromVaultFolder(context.Background(), "Best Of", ""); err == nil {
		t.Fatal("expected error for empty media UUID")
	}
}

func TestListCollections(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/collections" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, `{
		  "data": [
		    {"createdAt": "2026-01-01T00:00:00.000Z", "label": "Top", "uuid": "col-1"},
		    {"createdAt": null, "label": "Misc", "uuid": "col-2"}
		  ],
		  "pagination": {"page": 1, "size": 20, "hasMore": false}
		}`)
	})
	page, err := c.ListCollections(context.Background())
	if err != nil {
		t.Fatalf("ListCollections: %v", err)
	}
	if len(page.Data) != 2 {
		t.Fatalf("data: got %d", len(page.Data))
	}
	if page.Data[0].UUID != "col-1" || page.Data[0].Label != "Top" {
		t.Errorf("collection[0]: got %+v", page.Data[0])
	}
	if page.Data[1].CreatedAt != nil {
		t.Errorf("collection[1] createdAt should be nil, got %v", *page.Data[1].CreatedAt)
	}
}

func TestCreateCollection(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/collections" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var got map[string]any
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("body not JSON: %v (%s)", err, body)
		}
		if got["label"] != "New Collection" {
			t.Errorf("body label: got %v", got["label"])
		}
		_, _ = io.WriteString(w, `{"createdAt": "2026-01-01T00:00:00.000Z", "label": "New Collection", "uuid": "col-9"}`)
	})
	col, err := c.CreateCollection(context.Background(), RawBody(`{"label":"New Collection"}`))
	if err != nil {
		t.Fatalf("CreateCollection: %v", err)
	}
	if col.UUID != "col-9" || col.Label != "New Collection" {
		t.Errorf("result: got %+v", col)
	}
}

func TestCreateCollection_RequiresBody(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when body is empty")
	})
	if _, err := c.CreateCollection(context.Background(), nil); err == nil {
		t.Fatal("expected error for empty body")
	}
}

func TestRenameCollection(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/collections/col-1" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var got map[string]any
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("body not JSON: %v (%s)", err, body)
		}
		if got["label"] != "Renamed" {
			t.Errorf("body label: got %v", got["label"])
		}
		w.WriteHeader(http.StatusNoContent)
	})
	if err := c.RenameCollection(context.Background(), "col-1", RawBody(`{"label":"Renamed"}`)); err != nil {
		t.Fatalf("RenameCollection: %v", err)
	}
}

func TestRenameCollection_RequiresArgs(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when args are empty")
	})
	if err := c.RenameCollection(context.Background(), "", RawBody(`{"label":"x"}`)); err == nil {
		t.Fatal("expected error for empty collection UUID")
	}
	if err := c.RenameCollection(context.Background(), "col-1", nil); err == nil {
		t.Fatal("expected error for empty body")
	}
}

func TestDeleteCollection(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/collections/col-1" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	if err := c.DeleteCollection(context.Background(), "col-1"); err != nil {
		t.Fatalf("DeleteCollection: %v", err)
	}
}

func TestDeleteCollection_RequiresUUID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when UUID is empty")
	})
	if err := c.DeleteCollection(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty collection UUID")
	}
}

func TestListNotifications(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/notifications" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("page") != "1" || q.Get("size") != "25" {
			t.Errorf("pagination: page=%q size=%q", q.Get("page"), q.Get("size"))
		}
		if q.Get("eventType") != "7" {
			t.Errorf("eventType: got %q", q.Get("eventType"))
		}
		_, _ = io.WriteString(w, `{
		  "data": [
		    {
		      "createdAt": "2026-01-01T00:00:00.000Z",
		      "data": {"postUuid": "p-1", "amount": null},
		      "eventType": 7,
		      "isRead": false,
		      "originator": {
		        "avatarUrl": "https://cdn.example/a.jpg",
		        "displayName": "Fan One",
		        "handle": "fan1",
		        "isCreator": false,
		        "uuid": "u-1"
		      },
		      "receiverUuid": "u-self",
		      "uuid": "n-1"
		    },
		    {
		      "createdAt": "2026-01-02T00:00:00.000Z",
		      "data": {},
		      "eventType": 3,
		      "isRead": true,
		      "originator": null,
		      "receiverUuid": "u-self",
		      "uuid": "n-2"
		    }
		  ],
		  "pagination": {"page": 1, "size": 25, "hasMore": true}
		}`)
	})
	page, err := c.ListNotifications(context.Background(), ListNotificationsParams{
		Page:      ptrInt(1),
		Size:      ptrInt(25),
		EventType: ptrInt(7),
	})
	if err != nil {
		t.Fatalf("ListNotifications: %v", err)
	}
	if len(page.Data) != 2 {
		t.Fatalf("data: got %d", len(page.Data))
	}
	n0 := page.Data[0]
	if n0.UUID != "n-1" || n0.EventType != 7 || n0.IsRead {
		t.Errorf("notification[0]: got %+v", n0)
	}
	if n0.Originator == nil || n0.Originator.Handle != "fan1" || n0.Originator.IsCreator {
		t.Errorf("notification[0] originator: got %+v", n0.Originator)
	}
	var data0 map[string]any
	if err := json.Unmarshal(n0.Data, &data0); err != nil {
		t.Fatalf("notification[0] data not JSON: %v", err)
	}
	if data0["postUuid"] != "p-1" {
		t.Errorf("notification[0] data postUuid: got %v", data0["postUuid"])
	}
	if page.Data[1].Originator != nil {
		t.Errorf("notification[1] originator should be nil, got %+v", page.Data[1].Originator)
	}
	if !page.Pagination.HasMore {
		t.Error("hasMore should be true")
	}
}

func TestListNotifications_OmitsUnsetParams(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if _, ok := q["page"]; ok {
			t.Errorf("page should be omitted, got %q", q.Get("page"))
		}
		if _, ok := q["eventType"]; ok {
			t.Errorf("eventType should be omitted, got %q", q.Get("eventType"))
		}
		_, _ = io.WriteString(w, `{"data": [], "pagination": {"page": 1, "size": 20, "hasMore": false}}`)
	})
	if _, err := c.ListNotifications(context.Background(), ListNotificationsParams{}); err != nil {
		t.Fatalf("ListNotifications: %v", err)
	}
}

func TestListNotifications_Unauthorized(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"message": "token expired"}`)
	})
	_, err := c.ListNotifications(context.Background(), ListNotificationsParams{})
	apiErr, ok := AsError(err)
	if !ok {
		t.Fatalf("expected *Error, got %v", err)
	}
	if !apiErr.IsUnauthorized() {
		t.Errorf("expected 401, got %d", apiErr.StatusCode)
	}
}
