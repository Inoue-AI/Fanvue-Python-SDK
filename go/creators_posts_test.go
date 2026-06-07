package fanvue

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

const creatorAccountJSON = `{
  "account": {
    "earnings": {"availableBalance": 100.5, "lastPayoutAt": null, "total": 9876.54},
    "fans": {"followers": 42, "subscribers": 17},
    "status": "active"
  },
  "avatarUrl": null,
  "bannerUrl": "https://cdn.example.com/banner.png",
  "bio": "managed creator",
  "createdAt": "2026-01-01T00:00:00.000Z",
  "displayName": "Managed Creator",
  "email": "creator@example.com",
  "handle": "managed",
  "isCreator": true,
  "updatedAt": null,
  "uuid": "cr-1"
}`

const creatorPostCreatedJSON = `{
  "uuid": "p-1",
  "audience": "subscribers",
  "text": "hello",
  "mediaPreviewUuid": null,
  "price": 500,
  "expiresAt": null,
  "publishAt": null,
  "publishedAt": "2026-01-02T00:00:00.000Z",
  "createdAt": "2026-01-01T00:00:00.000Z"
}`

const creatorPostsPageJSON = `{
  "data": [` + fullPostJSON + `],
  "pagination": {"page": 1, "size": 20, "hasMore": false}
}`

const creatorPostCommentsPageJSON = `{
  "data": [
    {
      "uuid": "c-1",
      "text": "nice post",
      "createdAt": "2026-01-03T00:00:00.000Z",
      "updatedAt": null,
      "user": {
        "uuid": "u-9",
        "handle": "fan",
        "displayName": "Fan",
        "nickname": null,
        "isTopSpender": true
      }
    }
  ],
  "pagination": {"page": 1, "size": 20, "hasMore": false}
}`

const creatorPostCommentJSON = `{
  "uuid": "c-1",
  "text": "nice post",
  "createdAt": "2026-01-03T00:00:00.000Z",
  "updatedAt": null
}`

func TestGetCreatorAccount(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/creators/cr-1/account" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, creatorAccountJSON)
	})

	acc, err := c.GetCreatorAccount(context.Background(), "cr-1")
	if err != nil {
		t.Fatalf("GetCreatorAccount: %v", err)
	}
	if acc.UUID != "cr-1" {
		t.Errorf("uuid: got %q", acc.UUID)
	}
	if acc.Account.Status != "active" {
		t.Errorf("status: got %q", acc.Account.Status)
	}
	if acc.Account.Earnings.Total != 9876.54 {
		t.Errorf("earnings total: got %v", acc.Account.Earnings.Total)
	}
	if acc.Account.Earnings.LastPayoutAt != nil {
		t.Errorf("lastPayoutAt should be nil, got %v", *acc.Account.Earnings.LastPayoutAt)
	}
	if acc.Account.Fans.Followers != 42 || acc.Account.Fans.Subscribers != 17 {
		t.Errorf("fans: got %+v", acc.Account.Fans)
	}
	if acc.AvatarURL != nil {
		t.Errorf("avatarUrl should be nil, got %v", *acc.AvatarURL)
	}
	if !acc.IsCreator {
		t.Error("isCreator should be true")
	}
}

func TestGetCreatorAccount_EmptyUUID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when creator user UUID is empty")
	})
	if _, err := c.GetCreatorAccount(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty creator user UUID")
	}
}

func TestGetCreatorPosts(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/creators/cr-1/posts" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("page") != "2" {
			t.Errorf("page: got %q", q.Get("page"))
		}
		if q.Get("size") != "20" {
			t.Errorf("size: got %q", q.Get("size"))
		}
		if q.Get("includeUnpublished") != "true" {
			t.Errorf("includeUnpublished: got %q", q.Get("includeUnpublished"))
		}
		_, _ = io.WriteString(w, creatorPostsPageJSON)
	})

	page, err := c.GetCreatorPosts(context.Background(), "cr-1", ListCreatorPostsParams{
		Page:               ptrInt(2),
		Size:               ptrInt(20),
		IncludeUnpublished: ptrBool(true),
	})
	if err != nil {
		t.Fatalf("GetCreatorPosts: %v", err)
	}
	if len(page.Data) != 1 {
		t.Fatalf("data: got %d entries", len(page.Data))
	}
	assertFullPost(t, &page.Data[0])
	if page.Pagination.Page != 1 || page.Pagination.HasMore {
		t.Errorf("pagination: got %+v", page.Pagination)
	}
}

func TestGetCreatorPosts_OmitsUnsetParams(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if _, present := q["page"]; present {
			t.Errorf("page should be omitted when unset, query=%s", r.URL.RawQuery)
		}
		if _, present := q["includeUnpublished"]; present {
			t.Errorf("includeUnpublished should be omitted when unset, query=%s", r.URL.RawQuery)
		}
		_, _ = io.WriteString(w, creatorPostsPageJSON)
	})

	if _, err := c.GetCreatorPosts(context.Background(), "cr-1", ListCreatorPostsParams{}); err != nil {
		t.Fatalf("GetCreatorPosts: %v", err)
	}
}

func TestGetCreatorPosts_IncludeUnpublishedFalse(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("includeUnpublished"); got != "false" {
			t.Errorf("includeUnpublished: got %q, want false", got)
		}
		_, _ = io.WriteString(w, creatorPostsPageJSON)
	})

	if _, err := c.GetCreatorPosts(context.Background(), "cr-1", ListCreatorPostsParams{
		IncludeUnpublished: ptrBool(false),
	}); err != nil {
		t.Fatalf("GetCreatorPosts: %v", err)
	}
}

func TestGetCreatorPosts_EmptyUUID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when creator user UUID is empty")
	})
	if _, err := c.GetCreatorPosts(context.Background(), "", ListCreatorPostsParams{}); err == nil {
		t.Fatal("expected error for empty creator user UUID")
	}
}

func TestCreateCreatorPost(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/creators/cr-1/posts" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var got map[string]any
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("body not JSON: %v (%s)", err, body)
		}
		if got["audience"] != "subscribers" {
			t.Errorf("body audience: got %v", got["audience"])
		}
		if got["text"] != "hello" {
			t.Errorf("body text: got %v", got["text"])
		}
		if _, present := got["price"]; present {
			t.Errorf("price should be omitted when unset, body=%s", body)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, creatorPostCreatedJSON)
	})

	post, err := c.CreateCreatorPost(context.Background(), "cr-1", CreatePostParams{
		Audience: AudienceSubscribers,
		Text:     ptrString("hello"),
	})
	if err != nil {
		t.Fatalf("CreateCreatorPost: %v", err)
	}
	if post.UUID != "p-1" {
		t.Errorf("uuid: got %q", post.UUID)
	}
	if post.Audience != AudienceSubscribers {
		t.Errorf("audience: got %q", post.Audience)
	}
	if post.Text == nil || *post.Text != "hello" {
		t.Errorf("text: got %v", post.Text)
	}
	if post.Price == nil || *post.Price != 500 {
		t.Errorf("price: got %v", post.Price)
	}
	if post.MediaPreviewUUID != nil {
		t.Errorf("mediaPreviewUuid should be nil, got %v", *post.MediaPreviewUUID)
	}
}

func TestCreateCreatorPost_InvalidAudience(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called for an invalid audience")
	})
	if _, err := c.CreateCreatorPost(context.Background(), "cr-1", CreatePostParams{
		Audience: PostAudience("everyone"),
	}); err == nil {
		t.Fatal("expected error for invalid audience")
	}
}

func TestCreateCreatorPost_MissingAudience(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when audience is missing")
	})
	if _, err := c.CreateCreatorPost(context.Background(), "cr-1", CreatePostParams{}); err == nil {
		t.Fatal("expected error for missing audience")
	}
}

func TestCreateCreatorPost_EmptyUUID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when creator user UUID is empty")
	})
	if _, err := c.CreateCreatorPost(context.Background(), "", CreatePostParams{
		Audience: AudienceSubscribers,
	}); err == nil {
		t.Fatal("expected error for empty creator user UUID")
	}
}

func TestUpdateCreatorPost(t *testing.T) {
	newText := "edited"
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/creators/cr-1/posts/p-1" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var got map[string]any
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("body not JSON: %v (%s)", err, body)
		}
		if got["text"] != "edited" {
			t.Errorf("body text: got %v", got["text"])
		}
		if _, present := got["price"]; present {
			t.Errorf("price should be omitted when unset, body=%s", body)
		}
		_, _ = io.WriteString(w, fullPostJSON)
	})

	post, err := c.UpdateCreatorPost(context.Background(), "cr-1", "p-1", UpdatePostParams{Text: &newText})
	if err != nil {
		t.Fatalf("UpdateCreatorPost: %v", err)
	}
	assertFullPost(t, post)
}

func TestUpdateCreatorPost_RawBodyVerbatim(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var got map[string]json.RawMessage
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("body not JSON: %v (%s)", err, body)
		}
		raw, present := got["text"]
		if !present {
			t.Fatalf("expected text key present, body=%s", body)
		}
		if string(raw) != "null" {
			t.Errorf("expected explicit null for text, got %s", raw)
		}
		_, _ = io.WriteString(w, fullPostJSON)
	})

	// RawBody must be sent verbatim, preserving the explicit null that a nil
	// pointer cannot express.
	_, err := c.UpdateCreatorPost(context.Background(), "cr-1", "p-1", UpdatePostParams{
		RawBody: RawBody(`{"text":null}`),
		Text:    ptrString("ignored"),
	})
	if err != nil {
		t.Fatalf("UpdateCreatorPost: %v", err)
	}
}

func TestUpdateCreatorPost_InvalidAudience(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called for an invalid audience")
	})
	bad := PostAudience("everyone")
	if _, err := c.UpdateCreatorPost(context.Background(), "cr-1", "p-1", UpdatePostParams{Audience: &bad}); err == nil {
		t.Fatal("expected error for invalid audience")
	}
}

func TestUpdateCreatorPost_EmptyUUIDs(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when a UUID is empty")
	})
	if _, err := c.UpdateCreatorPost(context.Background(), "", "p-1", UpdatePostParams{}); err == nil {
		t.Fatal("expected error for empty creator user UUID")
	}
	if _, err := c.UpdateCreatorPost(context.Background(), "cr-1", "", UpdatePostParams{}); err == nil {
		t.Fatal("expected error for empty post UUID")
	}
}

func TestDeleteCreatorPost(t *testing.T) {
	called := false
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/creators/cr-1/posts/p-1" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	if err := c.DeleteCreatorPost(context.Background(), "cr-1", "p-1"); err != nil {
		t.Fatalf("DeleteCreatorPost: %v", err)
	}
	if !called {
		t.Error("server was not called")
	}
}

func TestDeleteCreatorPost_EmptyUUIDs(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when a UUID is empty")
	})
	if err := c.DeleteCreatorPost(context.Background(), "", "p-1"); err == nil {
		t.Fatal("expected error for empty creator user UUID")
	}
	if err := c.DeleteCreatorPost(context.Background(), "cr-1", ""); err == nil {
		t.Fatal("expected error for empty post UUID")
	}
}

func TestPinCreatorPost(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/creators/cr-1/posts/p-1/pin" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, fullPostJSON)
	})

	post, err := c.PinCreatorPost(context.Background(), "cr-1", "p-1")
	if err != nil {
		t.Fatalf("PinCreatorPost: %v", err)
	}
	assertFullPost(t, post)
}

func TestPinCreatorPost_EmptyUUIDs(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when a UUID is empty")
	})
	if _, err := c.PinCreatorPost(context.Background(), "", "p-1"); err == nil {
		t.Fatal("expected error for empty creator user UUID")
	}
	if _, err := c.PinCreatorPost(context.Background(), "cr-1", ""); err == nil {
		t.Fatal("expected error for empty post UUID")
	}
}

func TestUnpinCreatorPost(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/creators/cr-1/posts/p-1/pin" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, fullPostJSON)
	})

	post, err := c.UnpinCreatorPost(context.Background(), "cr-1", "p-1")
	if err != nil {
		t.Fatalf("UnpinCreatorPost: %v", err)
	}
	assertFullPost(t, post)
}

func TestUnpinCreatorPost_EmptyUUIDs(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when a UUID is empty")
	})
	if _, err := c.UnpinCreatorPost(context.Background(), "", "p-1"); err == nil {
		t.Fatal("expected error for empty creator user UUID")
	}
	if _, err := c.UnpinCreatorPost(context.Background(), "cr-1", ""); err == nil {
		t.Fatal("expected error for empty post UUID")
	}
}

func TestRepostCreatorPost(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/creators/cr-1/posts/p-1/repost" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, fullPostJSON)
	})

	post, err := c.RepostCreatorPost(context.Background(), "cr-1", "p-1")
	if err != nil {
		t.Fatalf("RepostCreatorPost: %v", err)
	}
	assertFullPost(t, post)
}

func TestRepostCreatorPost_EmptyUUIDs(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when a UUID is empty")
	})
	if _, err := c.RepostCreatorPost(context.Background(), "", "p-1"); err == nil {
		t.Fatal("expected error for empty creator user UUID")
	}
	if _, err := c.RepostCreatorPost(context.Background(), "cr-1", ""); err == nil {
		t.Fatal("expected error for empty post UUID")
	}
}

func TestGetCreatorPostComments(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/creators/cr-1/posts/p-1/comments" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("page") != "1" {
			t.Errorf("page: got %q", q.Get("page"))
		}
		if q.Get("size") != "20" {
			t.Errorf("size: got %q", q.Get("size"))
		}
		_, _ = io.WriteString(w, creatorPostCommentsPageJSON)
	})

	page, err := c.GetCreatorPostComments(context.Background(), "cr-1", "p-1", GetCreatorPostCommentsParams{
		Page: ptrInt(1),
		Size: ptrInt(20),
	})
	if err != nil {
		t.Fatalf("GetCreatorPostComments: %v", err)
	}
	if len(page.Data) != 1 {
		t.Fatalf("data: got %d entries", len(page.Data))
	}
	entry := page.Data[0]
	if entry.UUID != "c-1" || entry.Text != "nice post" {
		t.Errorf("entry: got %+v", entry)
	}
	if entry.User == nil || entry.User.Handle != "fan" || !entry.User.IsTopSpender {
		t.Errorf("entry user: got %+v", entry.User)
	}
	if entry.UpdatedAt != nil {
		t.Errorf("updatedAt should be nil, got %v", *entry.UpdatedAt)
	}
}

func TestGetCreatorPostComments_EmptyUUIDs(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when a UUID is empty")
	})
	if _, err := c.GetCreatorPostComments(context.Background(), "", "p-1", GetCreatorPostCommentsParams{}); err == nil {
		t.Fatal("expected error for empty creator user UUID")
	}
	if _, err := c.GetCreatorPostComments(context.Background(), "cr-1", "", GetCreatorPostCommentsParams{}); err == nil {
		t.Fatal("expected error for empty post UUID")
	}
}

func TestCreateCreatorPostComment(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/creators/cr-1/posts/p-1/comments" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var got map[string]any
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("body not JSON: %v (%s)", err, body)
		}
		if got["text"] != "nice post" {
			t.Errorf("body text: got %v", got["text"])
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, creatorPostCommentJSON)
	})

	comment, err := c.CreateCreatorPostComment(context.Background(), "cr-1", "p-1", CreatePostCommentParams{
		Text: "nice post",
	})
	if err != nil {
		t.Fatalf("CreateCreatorPostComment: %v", err)
	}
	if comment.UUID != "c-1" || comment.Text != "nice post" {
		t.Errorf("comment: got %+v", comment)
	}
	if comment.UpdatedAt != nil {
		t.Errorf("updatedAt should be nil, got %v", *comment.UpdatedAt)
	}
}

func TestCreateCreatorPostComment_EmptyText(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when text is empty")
	})
	if _, err := c.CreateCreatorPostComment(context.Background(), "cr-1", "p-1", CreatePostCommentParams{}); err == nil {
		t.Fatal("expected error for empty text")
	}
}

func TestCreateCreatorPostComment_EmptyUUIDs(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when a UUID is empty")
	})
	if _, err := c.CreateCreatorPostComment(context.Background(), "", "p-1", CreatePostCommentParams{Text: "x"}); err == nil {
		t.Fatal("expected error for empty creator user UUID")
	}
	if _, err := c.CreateCreatorPostComment(context.Background(), "cr-1", "", CreatePostCommentParams{Text: "x"}); err == nil {
		t.Fatal("expected error for empty post UUID")
	}
}

func TestDeleteCreatorPostComment(t *testing.T) {
	called := false
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/creators/cr-1/posts/p-1/comments/c-1" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	if err := c.DeleteCreatorPostComment(context.Background(), "cr-1", "p-1", "c-1"); err != nil {
		t.Fatalf("DeleteCreatorPostComment: %v", err)
	}
	if !called {
		t.Error("server was not called")
	}
}

func TestDeleteCreatorPostComment_EmptyUUIDs(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when a UUID is empty")
	})
	if err := c.DeleteCreatorPostComment(context.Background(), "", "p-1", "c-1"); err == nil {
		t.Fatal("expected error for empty creator user UUID")
	}
	if err := c.DeleteCreatorPostComment(context.Background(), "cr-1", "", "c-1"); err == nil {
		t.Fatal("expected error for empty post UUID")
	}
	if err := c.DeleteCreatorPostComment(context.Background(), "cr-1", "p-1", ""); err == nil {
		t.Fatal("expected error for empty comment UUID")
	}
}
