package fanvue

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

const fullPostJSON = `{
  "uuid": "p-1",
  "text": "hello",
  "audience": "subscribers",
  "collections": [{"uuid": "col-1", "label": "Best Of"}],
  "commentsCount": 3,
  "likesCount": 12,
  "isPinned": true,
  "mediaUuids": ["m-1", "m-2"],
  "mediaPreviewUuid": null,
  "price": 500,
  "expiresAt": null,
  "publishAt": null,
  "publishedAt": "2026-01-02T00:00:00.000Z",
  "createdAt": "2026-01-01T00:00:00.000Z",
  "tips": {"count": 2, "totalGross": 1000, "totalNet": 800}
}`

func assertFullPost(t *testing.T, p *Post) {
	t.Helper()
	if p.UUID != "p-1" {
		t.Errorf("uuid: got %q", p.UUID)
	}
	if p.Text == nil || *p.Text != "hello" {
		t.Errorf("text: got %v", p.Text)
	}
	if p.Audience != AudienceSubscribers {
		t.Errorf("audience: got %q", p.Audience)
	}
	if !p.IsPinned {
		t.Error("isPinned should be true")
	}
	if p.Price == nil || *p.Price != 500 {
		t.Errorf("price: got %v", p.Price)
	}
	if p.MediaPreviewUUID != nil {
		t.Errorf("mediaPreviewUuid should be nil, got %v", *p.MediaPreviewUUID)
	}
	if len(p.MediaUUIDs) != 2 {
		t.Errorf("mediaUuids: got %v", p.MediaUUIDs)
	}
	if p.Tips == nil || p.Tips.TotalNet != 800 {
		t.Errorf("tips: got %+v", p.Tips)
	}
}

func TestUpdatePost(t *testing.T) {
	newText := "edited"
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/posts/p-1" {
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

	post, err := c.UpdatePost(context.Background(), "p-1", UpdatePostParams{Text: &newText})
	if err != nil {
		t.Fatalf("UpdatePost: %v", err)
	}
	assertFullPost(t, post)
}

func TestUpdatePost_RawBodyVerbatim(t *testing.T) {
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
	_, err := c.UpdatePost(context.Background(), "p-1", UpdatePostParams{
		RawBody: RawBody(`{"text":null}`),
		Text:    ptrString("ignored"),
	})
	if err != nil {
		t.Fatalf("UpdatePost: %v", err)
	}
}

func TestUpdatePost_InvalidAudience(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called for an invalid audience")
	})
	bad := PostAudience("everyone")
	if _, err := c.UpdatePost(context.Background(), "p-1", UpdatePostParams{Audience: &bad}); err == nil {
		t.Fatal("expected error for invalid audience")
	}
}

func TestUpdatePost_EmptyUUID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when post UUID is empty")
	})
	if _, err := c.UpdatePost(context.Background(), "", UpdatePostParams{}); err == nil {
		t.Fatal("expected error for empty post UUID")
	}
}

func TestDeletePost(t *testing.T) {
	called := false
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/posts/p-1" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	if err := c.DeletePost(context.Background(), "p-1"); err != nil {
		t.Fatalf("DeletePost: %v", err)
	}
	if !called {
		t.Error("server was not called")
	}
}

func TestDeletePost_EmptyUUID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when post UUID is empty")
	})
	if err := c.DeletePost(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty post UUID")
	}
}

func TestPinPost(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/posts/p-1/pin" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, fullPostJSON)
	})

	post, err := c.PinPost(context.Background(), "p-1")
	if err != nil {
		t.Fatalf("PinPost: %v", err)
	}
	assertFullPost(t, post)
}

func TestPinPost_EmptyUUID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when post UUID is empty")
	})
	if _, err := c.PinPost(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty post UUID")
	}
}

func TestUnpinPost(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/posts/p-1/pin" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, fullPostJSON)
	})

	post, err := c.UnpinPost(context.Background(), "p-1")
	if err != nil {
		t.Fatalf("UnpinPost: %v", err)
	}
	assertFullPost(t, post)
}

func TestUnpinPost_EmptyUUID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when post UUID is empty")
	})
	if _, err := c.UnpinPost(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty post UUID")
	}
}

func TestRepostPost(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/posts/p-1/repost" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, fullPostJSON)
	})

	post, err := c.RepostPost(context.Background(), "p-1")
	if err != nil {
		t.Fatalf("RepostPost: %v", err)
	}
	assertFullPost(t, post)
}

func TestRepostPost_EmptyUUID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when post UUID is empty")
	})
	if _, err := c.RepostPost(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty post UUID")
	}
}

const createCommentJSON = `{
  "uuid": "cmt-1",
  "text": "nice post",
  "createdAt": "2026-01-03T00:00:00.000Z",
  "updatedAt": null
}`

func TestCreatePostComment(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/posts/p-1/comments" {
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
		_, _ = io.WriteString(w, createCommentJSON)
	})

	cmt, err := c.CreatePostComment(context.Background(), "p-1", CreatePostCommentParams{Text: "nice post"})
	if err != nil {
		t.Fatalf("CreatePostComment: %v", err)
	}
	if cmt.UUID != "cmt-1" {
		t.Errorf("uuid: got %q", cmt.UUID)
	}
	if cmt.Text != "nice post" {
		t.Errorf("text: got %q", cmt.Text)
	}
	if cmt.UpdatedAt != nil {
		t.Errorf("updatedAt should be nil, got %v", *cmt.UpdatedAt)
	}
}

func TestCreatePostComment_EmptyText(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when text is empty")
	})
	if _, err := c.CreatePostComment(context.Background(), "p-1", CreatePostCommentParams{}); err == nil {
		t.Fatal("expected error for empty text")
	}
}

func TestCreatePostComment_EmptyUUID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when post UUID is empty")
	})
	if _, err := c.CreatePostComment(context.Background(), "", CreatePostCommentParams{Text: "x"}); err == nil {
		t.Fatal("expected error for empty post UUID")
	}
}

func TestDeletePostComment(t *testing.T) {
	called := false
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/posts/p-1/comments/cmt-1" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	if err := c.DeletePostComment(context.Background(), "p-1", "cmt-1"); err != nil {
		t.Fatalf("DeletePostComment: %v", err)
	}
	if !called {
		t.Error("server was not called")
	}
}

func TestDeletePostComment_EmptyArgs(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when args are empty")
	})
	if err := c.DeletePostComment(context.Background(), "", "cmt-1"); err == nil {
		t.Fatal("expected error for empty post UUID")
	}
	if err := c.DeletePostComment(context.Background(), "p-1", ""); err == nil {
		t.Fatal("expected error for empty comment UUID")
	}
}

const postCommentsJSON = `{
  "data": [
    {
      "uuid": "cmt-1",
      "text": "great",
      "createdAt": "2026-01-03T00:00:00.000Z",
      "updatedAt": null,
      "user": {
        "uuid": "u-9",
        "handle": "fan9",
        "displayName": "Fan Nine",
        "nickname": null,
        "isTopSpender": true
      }
    },
    {
      "uuid": "cmt-2",
      "text": "deleted user",
      "createdAt": "2026-01-04T00:00:00.000Z",
      "updatedAt": "2026-01-05T00:00:00.000Z",
      "user": null
    }
  ],
  "pagination": {"page": 1, "size": 20, "hasMore": false}
}`

func TestGetPostComments(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/posts/p-1/comments" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		if r.URL.Query().Get("page") != "1" || r.URL.Query().Get("size") != "20" {
			t.Errorf("unexpected query: %q", r.URL.RawQuery)
		}
		_, _ = io.WriteString(w, postCommentsJSON)
	})

	page, err := c.GetPostComments(context.Background(), "p-1", GetPostCommentsParams{
		Page: ptrInt(1), Size: ptrInt(20),
	})
	if err != nil {
		t.Fatalf("GetPostComments: %v", err)
	}
	if len(page.Data) != 2 {
		t.Fatalf("expected 2 comments, got %d", len(page.Data))
	}
	first := page.Data[0]
	if first.User == nil || first.User.Handle != "fan9" {
		t.Errorf("first comment user: got %+v", first.User)
	}
	if !first.User.IsTopSpender {
		t.Error("first comment user should be top spender")
	}
	second := page.Data[1]
	if second.User != nil {
		t.Errorf("second comment user should be nil, got %+v", second.User)
	}
	if second.UpdatedAt == nil {
		t.Error("second comment updatedAt should be set")
	}
	if page.Pagination.HasMore {
		t.Error("hasMore should be false")
	}
}

func TestGetPostComments_EmptyUUID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when post UUID is empty")
	})
	if _, err := c.GetPostComments(context.Background(), "", GetPostCommentsParams{}); err == nil {
		t.Fatal("expected error for empty post UUID")
	}
}

const postLikesJSON = `{
  "data": [
    {
      "createdAt": "2026-01-03T00:00:00.000Z",
      "user": {
        "uuid": "u-9",
        "handle": "fan9",
        "displayName": "Fan Nine",
        "nickname": "Niner",
        "avatarUrl": "https://cdn.example.com/a.png",
        "isTopSpender": false,
        "registeredAt": "2025-12-01T00:00:00.000Z"
      }
    },
    {
      "createdAt": "2026-01-04T00:00:00.000Z",
      "user": null
    }
  ],
  "pagination": {"page": 2, "size": 5, "hasMore": true}
}`

func TestGetPostLikes(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/posts/p-1/likes" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, postLikesJSON)
	})

	page, err := c.GetPostLikes(context.Background(), "p-1", GetPostLikesParams{})
	if err != nil {
		t.Fatalf("GetPostLikes: %v", err)
	}
	if len(page.Data) != 2 {
		t.Fatalf("expected 2 likes, got %d", len(page.Data))
	}
	first := page.Data[0]
	if first.User == nil || first.User.UUID != "u-9" {
		t.Errorf("first like user: got %+v", first.User)
	}
	if first.User.AvatarURL == nil || *first.User.AvatarURL != "https://cdn.example.com/a.png" {
		t.Errorf("avatarUrl: got %v", first.User.AvatarURL)
	}
	if first.User.Nickname == nil || *first.User.Nickname != "Niner" {
		t.Errorf("nickname: got %v", first.User.Nickname)
	}
	if page.Data[1].User != nil {
		t.Errorf("second like user should be nil, got %+v", page.Data[1].User)
	}
	if !page.Pagination.HasMore {
		t.Error("hasMore should be true")
	}
}

func TestGetPostLikes_EmptyUUID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when post UUID is empty")
	})
	if _, err := c.GetPostLikes(context.Background(), "", GetPostLikesParams{}); err == nil {
		t.Fatal("expected error for empty post UUID")
	}
}

const postTipsJSON = `{
  "data": [
    {
      "createdAt": "2026-01-03T00:00:00.000Z",
      "gross": 1000,
      "net": 800,
      "user": {
        "uuid": "u-9",
        "handle": "fan9",
        "displayName": "Fan Nine",
        "nickname": null,
        "avatarUrl": null,
        "isTopSpender": true,
        "registeredAt": "2025-12-01T00:00:00.000Z"
      }
    },
    {
      "createdAt": null,
      "gross": 500,
      "net": 400,
      "user": null
    }
  ],
  "pagination": {"page": 1, "size": 10, "hasMore": false}
}`

func TestGetPostTips(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/posts/p-1/tips" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, postTipsJSON)
	})

	page, err := c.GetPostTips(context.Background(), "p-1", GetPostTipsParams{})
	if err != nil {
		t.Fatalf("GetPostTips: %v", err)
	}
	if len(page.Data) != 2 {
		t.Fatalf("expected 2 tips, got %d", len(page.Data))
	}
	first := page.Data[0]
	if first.Gross != 1000 || first.Net != 800 {
		t.Errorf("first tip amounts: gross=%v net=%v", first.Gross, first.Net)
	}
	if first.CreatedAt == nil {
		t.Error("first tip createdAt should be set")
	}
	if first.User == nil || !first.User.IsTopSpender {
		t.Errorf("first tip user: got %+v", first.User)
	}
	second := page.Data[1]
	if second.CreatedAt != nil {
		t.Errorf("second tip createdAt should be nil, got %v", *second.CreatedAt)
	}
	if second.User != nil {
		t.Errorf("second tip user should be nil, got %+v", second.User)
	}
}

func TestGetPostTips_EmptyUUID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when post UUID is empty")
	})
	if _, err := c.GetPostTips(context.Background(), "", GetPostTipsParams{}); err == nil {
		t.Fatal("expected error for empty post UUID")
	}
}
