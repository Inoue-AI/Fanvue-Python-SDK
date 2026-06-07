package fanvue

import (
	"context"
	"io"
	"net/http"
	"testing"
)

func TestGetCurrentUser_TypedAndNullable(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/users/me" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_, _ = io.WriteString(w, currentUserJSON)
	})
	user, err := c.GetCurrentUser(context.Background())
	if err != nil {
		t.Fatalf("GetCurrentUser: %v", err)
	}
	if user.UUID != "u-1" || user.Email != "creator@example.com" {
		t.Fatalf("unexpected user: %+v", user)
	}
	if user.AvatarURL != nil || user.UpdatedAt != nil {
		t.Fatalf("expected nullable fields to decode as nil, got %+v", user)
	}
}

func TestListPosts_PassesPaginationAndDecodes(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/posts" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("page"); got != "2" {
			t.Errorf("expected page=2, got %q", got)
		}
		if _, ok := r.URL.Query()["size"]; ok {
			t.Error("size should be omitted when nil")
		}
		_, _ = io.WriteString(w, `{
		  "data": [
		    {"uuid":"post-1","text":"hi","audience":"subscribers","collections":[],
		     "commentsCount":0,"likesCount":3,"isPinned":false,"mediaUuids":[],
		     "mediaPreviewUuid":null,"price":null,"expiresAt":null,"publishAt":null,
		     "publishedAt":"2026-04-01T00:00:00.000Z","createdAt":"2026-04-01T00:00:00.000Z",
		     "tips":{"count":0,"totalGross":0,"totalNet":0}}
		  ],
		  "pagination": {"page":2,"size":20,"hasMore":false}
		}`)
	})
	page, err := c.ListPosts(context.Background(), ListPostsParams{Page: ptrInt(2)})
	if err != nil {
		t.Fatalf("ListPosts: %v", err)
	}
	if len(page.Data) != 1 || page.Data[0].UUID != "post-1" {
		t.Fatalf("unexpected posts: %+v", page.Data)
	}
	if page.Data[0].Audience != AudienceSubscribers {
		t.Fatalf("unexpected audience: %q", page.Data[0].Audience)
	}
	if page.Data[0].LikesCount != 3 || page.Pagination.HasMore {
		t.Fatalf("unexpected page data: %+v", page)
	}
}

func TestGetPost_RequiresUUID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("server must not be called when post UUID is empty")
	})
	if _, err := c.GetPost(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty post UUID")
	}
}

func TestGetPost_Success(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/posts/post-42" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_, _ = io.WriteString(w, `{"uuid":"post-42","text":"pinned","audience":"followers-and-subscribers",
		  "isPinned":true,"likesCount":7,"commentsCount":2,"mediaUuids":[],"collections":[],
		  "mediaPreviewUuid":null,"price":null,"expiresAt":null,"publishAt":null,
		  "publishedAt":null,"createdAt":"2026-03-01T00:00:00.000Z",
		  "tips":{"count":0,"totalGross":0,"totalNet":0}}`)
	})
	post, err := c.GetPost(context.Background(), "post-42")
	if err != nil {
		t.Fatalf("GetPost: %v", err)
	}
	if post.UUID != "post-42" || !post.IsPinned {
		t.Fatalf("unexpected post: %+v", post)
	}
	if post.Audience != AudienceFollowersAndSubscribers {
		t.Fatalf("unexpected audience: %q", post.Audience)
	}
}

func TestCreatePost_SendsBodyAndReturnsTyped(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/posts" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		body := decodeBody(t, r)
		if body["audience"] != "subscribers" {
			t.Errorf("unexpected audience: %v", body["audience"])
		}
		if body["text"] != "Launch day!" {
			t.Errorf("unexpected text: %v", body["text"])
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"uuid":"post-new","text":"Launch day!","audience":"subscribers",
		  "createdAt":"2026-04-01T00:00:00.000Z","publishedAt":"2026-04-01T00:00:00.000Z",
		  "mediaPreviewUuid":null,"price":null,"expiresAt":null,"publishAt":null}`)
	})
	post, err := c.CreatePost(context.Background(), CreatePostParams{
		Audience:   AudienceSubscribers,
		Text:       ptrString("Launch day!"),
		MediaUUIDs: []string{"m-1"},
	})
	if err != nil {
		t.Fatalf("CreatePost: %v", err)
	}
	if post.UUID != "post-new" {
		t.Fatalf("unexpected created post: %+v", post)
	}
}

func TestCreatePost_RequiresAudience(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("server must not be called when audience is empty")
	})
	if _, err := c.CreatePost(context.Background(), CreatePostParams{Text: ptrString("x")}); err == nil {
		t.Fatal("expected error for missing audience")
	}
}

func TestCreatePost_RejectsInvalidAudience(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("server must not be called for an invalid audience")
	})
	_, err := c.CreatePost(context.Background(), CreatePostParams{Audience: PostAudience("everyone")})
	if err == nil {
		t.Fatal("expected error for invalid audience")
	}
}

func TestListSubscribers_Success(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/subscribers" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_, _ = io.WriteString(w, `{
		  "data": [{"uuid":"fan-1","handle":"fan","displayName":"Fan One",
		    "nickname":null,"isTopSpender":false,"avatarUrl":null,
		    "registeredAt":"2026-02-01T00:00:00.000Z"}],
		  "pagination": {"page":1,"size":20,"hasMore":true}
		}`)
	})
	subs, err := c.ListSubscribers(context.Background(), ListFansParams{Page: ptrInt(1), Size: ptrInt(20)})
	if err != nil {
		t.Fatalf("ListSubscribers: %v", err)
	}
	if len(subs.Data) != 1 || subs.Data[0].UUID != "fan-1" {
		t.Fatalf("unexpected subscribers: %+v", subs.Data)
	}
	if !subs.Pagination.HasMore || subs.Data[0].Nickname != nil {
		t.Fatalf("unexpected page: %+v", subs)
	}
}

func TestSendMessage_PublishesToChat(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/chats/fan-uuid/message" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		body := decodeBody(t, r)
		if body["text"] != "Thanks for subscribing!" {
			t.Errorf("unexpected body text: %v", body["text"])
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"messageUuid":"msg-1"}`)
	})
	res, err := c.SendMessage(context.Background(), "fan-uuid", SendMessageParams{
		Text: ptrString("Thanks for subscribing!"),
	})
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if res.MessageUUID != "msg-1" {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestSendMessage_RequiresContent(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("server must not be called when message has no text or media")
	})
	if _, err := c.SendMessage(context.Background(), "fan-uuid", SendMessageParams{}); err == nil {
		t.Fatal("expected error when no content is supplied")
	}
}

func TestListMessages_Success(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chats/fan-uuid/messages" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("markAsRead"); got != "true" {
			t.Errorf("expected markAsRead=true, got %q", got)
		}
		_, _ = io.WriteString(w, `{
		  "data": [{"uuid":"m-1","type":"SINGLE_RECIPIENT","text":"hi","isRead":true,
		    "hasMedia":false,"mediaType":null,"mediaUuids":[],"sentAt":"2026-05-01T00:00:00.000Z",
		    "sentByUserId":null,"purchasedAt":null}],
		  "pagination": {"page":1,"size":20,"hasMore":false}
		}`)
	})
	markRead := true
	page, err := c.ListMessages(context.Background(), "fan-uuid", ListMessagesParams{MarkAsRead: &markRead})
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	if len(page.Data) != 1 || page.Data[0].UUID != "m-1" || !page.Data[0].IsRead {
		t.Fatalf("unexpected messages: %+v", page.Data)
	}
}

func TestContextCancellationPropagates(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, currentUserJSON)
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before the call
	if _, err := c.GetCurrentUser(ctx); err == nil {
		t.Fatal("expected error from cancelled context")
	}
}
