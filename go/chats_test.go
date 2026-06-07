package fanvue

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

// --- list_chats -------------------------------------------------------------

func TestListChats_Success(t *testing.T) {
	const body = `{
	  "data": [
	    {
	      "user": {
	        "uuid": "u-1",
	        "handle": "fan",
	        "displayName": "Fan One",
	        "nickname": null,
	        "avatarUrl": null,
	        "isTopSpender": true,
	        "registeredAt": "2026-01-01T00:00:00.000Z"
	      },
	      "lastMessage": {
	        "uuid": "m-1",
	        "type": "SINGLE_RECIPIENT",
	        "text": "hi",
	        "hasMedia": false,
	        "mediaType": null,
	        "senderUuid": "u-1",
	        "sentAt": "2026-01-02T00:00:00.000Z",
	        "sentByUserId": "u-1"
	      },
	      "lastMessageAt": "2026-01-02T00:00:00.000Z",
	      "createdAt": "2026-01-01T00:00:00.000Z",
	      "isMuted": false,
	      "isRead": true,
	      "unreadMessagesCount": 0
	    }
	  ],
	  "pagination": {"page": 1, "size": 10, "hasMore": false}
	}`

	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/chats" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("page") != "1" {
			t.Errorf("page: got %q", q.Get("page"))
		}
		if q.Get("customListId") != "list-1" {
			t.Errorf("customListId: got %q", q.Get("customListId"))
		}
		if q.Get("sortBy") != "most_recent_messages" {
			t.Errorf("sortBy: got %q", q.Get("sortBy"))
		}
		// Slice params must be repeated keys, matching httpx serialization.
		smart := q["smartListIds"]
		if len(smart) != 2 || smart[0] != "subscribers" || smart[1] != "followers" {
			t.Errorf("smartListIds: got %v", smart)
		}
		filters := q["filter"]
		if len(filters) != 2 || filters[0] != "unread" || filters[1] != "online" {
			t.Errorf("filter: got %v", filters)
		}
		_, _ = io.WriteString(w, body)
	})

	page, err := c.ListChats(context.Background(), ListChatsParams{
		Page:         ptrInt(1),
		Size:         ptrInt(10),
		CustomListID: ptrString("list-1"),
		SmartListIDs: []SmartListID{SmartListSubscribers, SmartListFollowers},
		Filter:       []ChatFilter{ChatFilterUnread, ChatFilterOnline},
		SortBy:       ptrChatSortBy(SortMostRecentMessages),
	})
	if err != nil {
		t.Fatalf("ListChats: %v", err)
	}
	if len(page.Data) != 1 {
		t.Fatalf("expected 1 chat, got %d", len(page.Data))
	}
	chat := page.Data[0]
	if chat.User.UUID != "u-1" || !chat.User.IsTopSpender {
		t.Errorf("user: got %+v", chat.User)
	}
	if chat.LastMessage == nil || chat.LastMessage.UUID != "m-1" {
		t.Errorf("lastMessage: got %+v", chat.LastMessage)
	}
	if page.Pagination.HasMore {
		t.Error("hasMore should be false")
	}
}

func TestListChats_OmitsUnsetSliceParams(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if _, ok := q["smartListIds"]; ok {
			t.Error("smartListIds must be omitted when empty")
		}
		if _, ok := q["filter"]; ok {
			t.Error("filter must be omitted when empty")
		}
		if _, ok := q["sortBy"]; ok {
			t.Error("sortBy must be omitted when nil")
		}
		_, _ = io.WriteString(w, `{"data":[],"pagination":{"page":1,"size":10,"hasMore":false}}`)
	})
	if _, err := c.ListChats(context.Background(), ListChatsParams{}); err != nil {
		t.Fatalf("ListChats: %v", err)
	}
}

// --- create_chat ------------------------------------------------------------

func TestCreateChat_Success(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/chats" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		got := decodeBody(t, r)
		if got["userUuid"] != "u-9" {
			t.Errorf("userUuid: got %v", got["userUuid"])
		}
		_, _ = io.WriteString(w, `{"message":"created"}`)
	})

	res, err := c.CreateChat(context.Background(), CreateChatParams{UserUUID: "u-9"})
	if err != nil {
		t.Fatalf("CreateChat: %v", err)
	}
	if res.Message != "created" {
		t.Errorf("message: got %q", res.Message)
	}
}

func TestCreateChat_RequiresUserUUID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called without a UserUUID")
	})
	if _, err := c.CreateChat(context.Background(), CreateChatParams{}); err == nil {
		t.Fatal("expected error when UserUUID is empty")
	}
}

func TestCreateChat_RawBodyVerbatim(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		got := decodeBody(t, r)
		if got["userUuid"] != "raw-uuid" {
			t.Errorf("expected raw body verbatim, got %v", got)
		}
		_, _ = io.WriteString(w, `{"message":"created"}`)
	})
	_, err := c.CreateChat(context.Background(), CreateChatParams{
		UserUUID: "ignored",
		RawBody:  RawBody(`{"userUuid":"raw-uuid"}`),
	})
	if err != nil {
		t.Fatalf("CreateChat: %v", err)
	}
}

// --- update_chat ------------------------------------------------------------

func TestUpdateChat_Success(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/chats/u-1" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		got := decodeBody(t, r)
		if got["isRead"] != true {
			t.Errorf("isRead: got %v", got["isRead"])
		}
		if _, present := got["isMuted"]; present {
			t.Error("isMuted should be omitted when unset")
		}
		w.WriteHeader(http.StatusNoContent)
	})

	read := true
	if err := c.UpdateChat(context.Background(), "u-1", UpdateChatParams{IsRead: &read}); err != nil {
		t.Fatalf("UpdateChat: %v", err)
	}
}

func TestUpdateChat_RequiresUserUUID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called without a user UUID")
	})
	if err := c.UpdateChat(context.Background(), "", UpdateChatParams{}); err == nil {
		t.Fatal("expected error when user UUID is empty")
	}
}

// --- get_unread_chats_count -------------------------------------------------

func TestGetUnreadChatsCount_Success(t *testing.T) {
	const body = `{
	  "unreadChatsCount": 3,
	  "unreadMessagesCount": 7,
	  "unreadNotifications": {
	    "newFollower": 1,
	    "newPostComment": 0,
	    "newPostLike": 2,
	    "newPromotion": 0,
	    "newPurchase": 4,
	    "newSubscriber": 1,
	    "newTip": 5
	  }
	}`
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chats/unread" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, body)
	})
	res, err := c.GetUnreadChatsCount(context.Background())
	if err != nil {
		t.Fatalf("GetUnreadChatsCount: %v", err)
	}
	if res.UnreadChatsCount != 3 || res.UnreadMessagesCount != 7 {
		t.Errorf("counts: got %+v", res)
	}
	if res.UnreadNotifications.NewTip != 5 {
		t.Errorf("newTip: got %v", res.UnreadNotifications.NewTip)
	}
}

// --- get_batch_statuses -----------------------------------------------------

func TestGetBatchStatuses_Success(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/chats/statuses" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		var got struct {
			UserUUIDs []string `json:"userUuids"`
		}
		raw, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatalf("body not JSON: %v (%s)", err, raw)
		}
		if len(got.UserUUIDs) != 2 {
			t.Errorf("userUuids: got %v", got.UserUUIDs)
		}
		_, _ = io.WriteString(w, `{
		  "u-1": {"isOnline": true, "lastSeenAt": "2026-01-02T00:00:00.000Z"},
		  "u-2": {"isOnline": false, "lastSeenAt": null}
		}`)
	})

	res, err := c.GetBatchStatuses(context.Background(), BatchStatusParams{
		UserUUIDs: []string{"u-1", "u-2"},
	})
	if err != nil {
		t.Fatalf("GetBatchStatuses: %v", err)
	}
	if !res["u-1"].IsOnline {
		t.Errorf("u-1 should be online: %+v", res["u-1"])
	}
	if res["u-2"].IsOnline || res["u-2"].LastSeenAt != nil {
		t.Errorf("u-2 should be offline with nil lastSeenAt: %+v", res["u-2"])
	}
}

func TestGetBatchStatuses_RequiresUsers(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called without user UUIDs")
	})
	if _, err := c.GetBatchStatuses(context.Background(), BatchStatusParams{}); err == nil {
		t.Fatal("expected error when UserUUIDs is empty")
	}
}

// --- get_custom_lists -------------------------------------------------------

func TestGetCustomLists_Success(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chats/lists/custom" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		if r.URL.Query().Get("search") != "vip" {
			t.Errorf("search: got %q", r.URL.Query().Get("search"))
		}
		_, _ = io.WriteString(w, `{
		  "data": [{"uuid": "cl-1", "name": "VIPs", "membersCount": 12, "createdAt": null}],
		  "pagination": {"page": 1, "size": 10, "hasMore": false}
		}`)
	})
	page, err := c.GetCustomLists(context.Background(), GetCustomListsParams{Search: ptrString("vip")})
	if err != nil {
		t.Fatalf("GetCustomLists: %v", err)
	}
	if len(page.Data) != 1 || page.Data[0].Name != "VIPs" || page.Data[0].MembersCount != 12 {
		t.Errorf("data: got %+v", page.Data)
	}
}

// --- create_custom_list -----------------------------------------------------

func TestCreateCustomList_Success(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		got := decodeBody(t, r)
		if got["name"] != "Big Spenders" {
			t.Errorf("name: got %v", got["name"])
		}
		_, _ = io.WriteString(w, `{"uuid": "cl-9", "name": "Big Spenders", "createdAt": "2026-01-01T00:00:00.000Z"}`)
	})
	res, err := c.CreateCustomList(context.Background(), CreateCustomListParams{Name: "Big Spenders"})
	if err != nil {
		t.Fatalf("CreateCustomList: %v", err)
	}
	if res.UUID != "cl-9" || res.Name != "Big Spenders" {
		t.Errorf("result: got %+v", res)
	}
}

func TestCreateCustomList_RequiresName(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called without a name")
	})
	if _, err := c.CreateCustomList(context.Background(), CreateCustomListParams{}); err == nil {
		t.Fatal("expected error when Name is empty")
	}
}

// --- get_custom_list_members ------------------------------------------------

func TestGetCustomListMembers_Success(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chats/lists/custom/cl-1" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		if r.URL.Query().Get("size") != "5" {
			t.Errorf("size: got %q", r.URL.Query().Get("size"))
		}
		_, _ = io.WriteString(w, `{
		  "data": [{"uuid": "u-1", "handle": "fan", "displayName": "Fan", "isCreator": false}],
		  "pagination": {"page": 1, "size": 5, "hasMore": true}
		}`)
	})
	page, err := c.GetCustomListMembers(context.Background(), "cl-1", GetCustomListMembersParams{Size: ptrInt(5)})
	if err != nil {
		t.Fatalf("GetCustomListMembers: %v", err)
	}
	if len(page.Data) != 1 || page.Data[0].Handle != "fan" {
		t.Errorf("data: got %+v", page.Data)
	}
	if !page.Pagination.HasMore {
		t.Error("hasMore should be true")
	}
}

func TestGetCustomListMembers_RequiresUUID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called without a list UUID")
	})
	if _, err := c.GetCustomListMembers(context.Background(), "", GetCustomListMembersParams{}); err == nil {
		t.Fatal("expected error when list UUID is empty")
	}
}

// --- update_custom_list -----------------------------------------------------

func TestUpdateCustomList_Success(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/chats/lists/custom/cl-1" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		got := decodeBody(t, r)
		if got["name"] != "Renamed" {
			t.Errorf("name: got %v", got["name"])
		}
		w.WriteHeader(http.StatusNoContent)
	})
	if err := c.UpdateCustomList(context.Background(), "cl-1", UpdateCustomListParams{Name: "Renamed"}); err != nil {
		t.Fatalf("UpdateCustomList: %v", err)
	}
}

func TestUpdateCustomList_RequiresName(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called without a name")
	})
	if err := c.UpdateCustomList(context.Background(), "cl-1", UpdateCustomListParams{}); err == nil {
		t.Fatal("expected error when Name is empty")
	}
}

// --- delete_custom_list -----------------------------------------------------

func TestDeleteCustomList_Success(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/chats/lists/custom/cl-1" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	if err := c.DeleteCustomList(context.Background(), "cl-1"); err != nil {
		t.Fatalf("DeleteCustomList: %v", err)
	}
}

func TestDeleteCustomList_RequiresUUID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called without a list UUID")
	})
	if err := c.DeleteCustomList(context.Background(), ""); err == nil {
		t.Fatal("expected error when list UUID is empty")
	}
}

// --- add_members_to_custom_list ---------------------------------------------

func TestAddMembersToCustomList_Success(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/chats/lists/custom/cl-1/members" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		var got struct {
			UserUUIDs []string `json:"userUuids"`
		}
		raw, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatalf("body not JSON: %v (%s)", err, raw)
		}
		if len(got.UserUUIDs) != 2 {
			t.Errorf("userUuids: got %v", got.UserUUIDs)
		}
		_, _ = io.WriteString(w, `{"added": 1, "skipped": 1}`)
	})
	res, err := c.AddMembersToCustomList(context.Background(), "cl-1", AddMembersToCustomListParams{
		UserUUIDs: []string{"u-1", "u-2"},
	})
	if err != nil {
		t.Fatalf("AddMembersToCustomList: %v", err)
	}
	if res.Added != 1 || res.Skipped != 1 {
		t.Errorf("result: got %+v", res)
	}
}

func TestAddMembersToCustomList_RequiresMembers(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called without members")
	})
	if _, err := c.AddMembersToCustomList(context.Background(), "cl-1", AddMembersToCustomListParams{}); err == nil {
		t.Fatal("expected error when UserUUIDs is empty")
	}
}

// --- remove_member_from_custom_list -----------------------------------------

func TestRemoveMemberFromCustomList_Success(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/chats/lists/custom/cl-1/members/u-9" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	if err := c.RemoveMemberFromCustomList(context.Background(), "cl-1", "u-9"); err != nil {
		t.Fatalf("RemoveMemberFromCustomList: %v", err)
	}
}

func TestRemoveMemberFromCustomList_RequiresIDs(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called without ids")
	})
	if err := c.RemoveMemberFromCustomList(context.Background(), "", "u-9"); err == nil {
		t.Fatal("expected error when list UUID is empty")
	}
	if err := c.RemoveMemberFromCustomList(context.Background(), "cl-1", ""); err == nil {
		t.Fatal("expected error when user UUID is empty")
	}
}

// --- get_smart_lists --------------------------------------------------------

func TestGetSmartLists_Success(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chats/lists/smart" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		// The endpoint returns a bare array, not a paginated envelope.
		_, _ = io.WriteString(w, `[
		  {"uuid": "subscribers", "name": "Subscribers", "count": 42},
		  {"uuid": "followers", "name": "Followers", "count": 100}
		]`)
	})
	lists, err := c.GetSmartLists(context.Background())
	if err != nil {
		t.Fatalf("GetSmartLists: %v", err)
	}
	if len(lists) != 2 {
		t.Fatalf("expected 2 smart lists, got %d", len(lists))
	}
	if lists[0].UUID != SmartListSubscribers || lists[0].Count != 42 {
		t.Errorf("list[0]: got %+v", lists[0])
	}
}

// --- get_smart_list_members -------------------------------------------------

func TestGetSmartListMembers_Success(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chats/lists/smart/subscribers" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		if r.URL.Query().Get("page") != "2" {
			t.Errorf("page: got %q", r.URL.Query().Get("page"))
		}
		_, _ = io.WriteString(w, `{
		  "data": [{"uuid": "u-1", "handle": "fan", "displayName": "Fan", "isCreator": false}],
		  "pagination": {"page": 2, "size": 10, "hasMore": false}
		}`)
	})
	page, err := c.GetSmartListMembers(context.Background(), SmartListSubscribers, GetSmartListMembersParams{Page: ptrInt(2)})
	if err != nil {
		t.Fatalf("GetSmartListMembers: %v", err)
	}
	if len(page.Data) != 1 || page.Data[0].UUID != "u-1" {
		t.Errorf("data: got %+v", page.Data)
	}
}

func TestGetSmartListMembers_RequiresID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called without a smart list id")
	})
	if _, err := c.GetSmartListMembers(context.Background(), "", GetSmartListMembersParams{}); err == nil {
		t.Fatal("expected error when smart list id is empty")
	}
}

// --- list_template_messages -------------------------------------------------

func TestListTemplateMessages_Success(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chats/templates" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		if r.URL.Query().Get("folderName") != "Welcome" {
			t.Errorf("folderName: got %q", r.URL.Query().Get("folderName"))
		}
		_, _ = io.WriteString(w, `{
		  "data": [{"uuid": "t-1", "text": "Welcome!", "folderName": "Welcome", "mediaUuids": [], "price": null}],
		  "pagination": {"page": 1, "size": 10, "hasMore": false}
		}`)
	})
	page, err := c.ListTemplateMessages(context.Background(), ListTemplateMessagesParams{FolderName: ptrString("Welcome")})
	if err != nil {
		t.Fatalf("ListTemplateMessages: %v", err)
	}
	if len(page.Data) != 1 || page.Data[0].UUID != "t-1" {
		t.Errorf("data: got %+v", page.Data)
	}
	if page.Data[0].Price != nil {
		t.Errorf("price should be nil, got %v", *page.Data[0].Price)
	}
}

// --- get_template_message ---------------------------------------------------

func TestGetTemplateMessage_Success(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chats/templates/t-1" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, `{"uuid": "t-1", "text": "Hi", "folderName": null, "mediaUuids": ["m-1"], "price": 500}`)
	})
	tpl, err := c.GetTemplateMessage(context.Background(), "t-1")
	if err != nil {
		t.Fatalf("GetTemplateMessage: %v", err)
	}
	if tpl.UUID != "t-1" || tpl.Text == nil || *tpl.Text != "Hi" {
		t.Errorf("template: got %+v", tpl)
	}
	if tpl.Price == nil || *tpl.Price != 500 {
		t.Errorf("price: got %v", tpl.Price)
	}
	if len(tpl.MediaUUIDs) != 1 || tpl.MediaUUIDs[0] != "m-1" {
		t.Errorf("mediaUuids: got %v", tpl.MediaUUIDs)
	}
}

func TestGetTemplateMessage_RequiresUUID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called without a template UUID")
	})
	if _, err := c.GetTemplateMessage(context.Background(), ""); err == nil {
		t.Fatal("expected error when template UUID is empty")
	}
}

// ptrChatSortBy returns a pointer to a ChatSortBy, for test ergonomics.
func ptrChatSortBy(s ChatSortBy) *ChatSortBy { return &s }
