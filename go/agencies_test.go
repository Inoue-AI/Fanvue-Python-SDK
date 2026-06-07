package fanvue

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestCreateAgencyInvite(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/agencies/invites" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		raw, _ := io.ReadAll(r.Body)
		var sent map[string]any
		if err := json.Unmarshal(raw, &sent); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if sent["email"] != "team@example.com" {
			t.Errorf("email: got %v", sent["email"])
		}
		if sent["isAdmin"] != true {
			t.Errorf("isAdmin: got %v", sent["isAdmin"])
		}
		_, _ = io.WriteString(w, `{"inviteUuid":"inv-1","message":"Invite sent","success":true}`)
	})

	res, err := c.CreateAgencyInvite(context.Background(), CreateAgencyInviteParams{
		Email:   "team@example.com",
		IsAdmin: ptrBool(true),
	})
	if err != nil {
		t.Fatalf("CreateAgencyInvite: %v", err)
	}
	if res.InviteUUID != "inv-1" || !res.Success || res.Message != "Invite sent" {
		t.Errorf("unexpected result: %+v", res)
	}
}

func TestCreateAgencyInvite_RequiresEmail(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler must not be reached when Email and RawBody are empty")
	})
	if _, err := c.CreateAgencyInvite(context.Background(), CreateAgencyInviteParams{}); err == nil {
		t.Fatal("expected error when Email and RawBody are empty")
	}
}

func TestCreateAgencyInvite_RawBodyTakesPrecedence(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var sent map[string]any
		if err := json.Unmarshal(raw, &sent); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if _, ok := sent["email"]; ok {
			t.Error("typed email must not be sent when RawBody is set")
		}
		if sent["custom"] != "value" {
			t.Errorf("RawBody not sent verbatim: got %v", sent)
		}
		_, _ = io.WriteString(w, `{"inviteUuid":"inv-2","message":"ok","success":true}`)
	})
	_, err := c.CreateAgencyInvite(context.Background(), CreateAgencyInviteParams{
		Email:   "ignored@example.com",
		RawBody: RawBody(`{"custom":"value"}`),
	})
	if err != nil {
		t.Fatalf("CreateAgencyInvite: %v", err)
	}
}

func TestCreateCreatorInvite(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/agencies/creator-invites" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		raw, _ := io.ReadAll(r.Body)
		var sent map[string]any
		if err := json.Unmarshal(raw, &sent); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if sent["email"] != "creator@example.com" {
			t.Errorf("email: got %v", sent["email"])
		}
		_, _ = io.WriteString(w, `{"message":"Creator invited","success":true}`)
	})

	res, err := c.CreateCreatorInvite(context.Background(), CreateCreatorInviteParams{
		Email: "creator@example.com",
	})
	if err != nil {
		t.Fatalf("CreateCreatorInvite: %v", err)
	}
	if !res.Success || res.Message != "Creator invited" {
		t.Errorf("unexpected result: %+v", res)
	}
}

func TestCreateCreatorInvite_RequiresEmail(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler must not be reached when Email and RawBody are empty")
	})
	if _, err := c.CreateCreatorInvite(context.Background(), CreateCreatorInviteParams{}); err == nil {
		t.Fatal("expected error when Email and RawBody are empty")
	}
}

func TestGetChatterLeaderboard(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/agencies/insights/chatter-leaderboard" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("startDate") != "2026-01-01" || q.Get("endDate") != "2026-01-31" {
			t.Errorf("unexpected date query: %q", r.URL.RawQuery)
		}
		if q.Get("chatterUuids") != "ch-1,ch-2" {
			t.Errorf("chatterUuids must be a single value, got %q", q.Get("chatterUuids"))
		}
		_, _ = io.WriteString(w, `{"data":[
			{"activeHours":4.5,"avatarUrl":null,"avgResponseMs":null,"chatterName":"Alice",
			 "chatterUuid":"ch-1","eph":12.0,"goldenRatio":0.5,"messages":120,"ppvsSent":10,
			 "ppvsUnlocked":7,"revenue":5000,"unlockRatio":0.7},
			{"activeHours":2.0,"avatarUrl":"https://cdn/avatar.png","avgResponseMs":1500.0,
			 "chatterName":"Bob","chatterUuid":"ch-2","eph":8.0,"goldenRatio":0.3,"messages":60,
			 "ppvsSent":5,"ppvsUnlocked":2,"revenue":1600,"unlockRatio":0.4}
		]}`)
	})

	board, err := c.GetChatterLeaderboard(context.Background(), GetChatterLeaderboardParams{
		StartDate:    ptrString("2026-01-01"),
		EndDate:      ptrString("2026-01-31"),
		ChatterUUIDs: ptrString("ch-1,ch-2"),
	})
	if err != nil {
		t.Fatalf("GetChatterLeaderboard: %v", err)
	}
	if len(board.Data) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(board.Data))
	}
	first := board.Data[0]
	if first.ChatterUUID != "ch-1" || first.Messages != 120 || first.Revenue != 5000 {
		t.Errorf("first entry: %+v", first)
	}
	if first.AvatarURL != nil || first.AvgResponseMs != nil {
		t.Errorf("first entry nullable fields should be nil: %+v", first)
	}
	second := board.Data[1]
	if second.AvgResponseMs == nil || *second.AvgResponseMs != 1500.0 {
		t.Errorf("second avgResponseMs: %v", second.AvgResponseMs)
	}
	if second.AvatarURL == nil || *second.AvatarURL != "https://cdn/avatar.png" {
		t.Errorf("second avatarUrl: %v", second.AvatarURL)
	}
}

func TestListAgencyChats(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/agencies/chats" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("page") != "2" || q.Get("size") != "25" {
			t.Errorf("unexpected pagination query: %q", r.URL.RawQuery)
		}
		_, _ = io.WriteString(w, `{"data":[
			{"createdAt":"2026-01-02T00:00:00.000Z","creatorUuid":"cr-1","isMuted":false,
			 "isRead":true,"lastMessage":{"hasMedia":true,"mediaType":"image","senderUuid":"s-1",
			 "sentAt":"2026-01-02T01:00:00.000Z","sentByUserId":"u-1","text":"hi","type":"SINGLE_RECIPIENT",
			 "uuid":"m-1"},"lastMessageAt":"2026-01-02T01:00:00.000Z","unreadMessagesCount":0,
			 "user":{"avatarUrl":null,"displayName":"Fan One","handle":"fan1","isTopSpender":false,
			 "nickname":null,"registeredAt":"2025-12-01T00:00:00.000Z","uuid":"u-1"}},
			{"createdAt":null,"creatorUuid":"cr-2","isMuted":true,"isRead":false,"lastMessage":null,
			 "lastMessageAt":null,"unreadMessagesCount":3,
			 "user":{"avatarUrl":"https://cdn/a.png","displayName":"Fan Two","handle":"fan2",
			 "isTopSpender":true,"nickname":"VIP","registeredAt":"2025-11-01T00:00:00.000Z","uuid":"u-2"}}
		],"pagination":{"hasMore":true,"page":2,"size":25}}`)
	})

	page, err := c.ListAgencyChats(context.Background(), ListAgencyChatsParams{
		Page: ptrInt(2),
		Size: ptrInt(25),
	})
	if err != nil {
		t.Fatalf("ListAgencyChats: %v", err)
	}
	if len(page.Data) != 2 {
		t.Fatalf("expected 2 chats, got %d", len(page.Data))
	}
	if !page.Pagination.HasMore || page.Pagination.Page != 2 {
		t.Errorf("pagination: %+v", page.Pagination)
	}
	first := page.Data[0]
	if first.CreatorUUID != "cr-1" || first.LastMessage == nil || first.LastMessage.UUID != "m-1" {
		t.Errorf("first chat: %+v", first)
	}
	second := page.Data[1]
	if second.LastMessage != nil || !second.IsMuted || second.UnreadMessagesCount != 3 {
		t.Errorf("second chat: %+v", second)
	}
	if second.User.Nickname == nil || *second.User.Nickname != "VIP" {
		t.Errorf("second user nickname: %v", second.User.Nickname)
	}
}

func TestListAgencyEarningsByDay(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/agencies/earnings" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("startDate") != "2026-01-01" || q.Get("endDate") != "2026-01-31" {
			t.Errorf("unexpected date query: %q", r.URL.RawQuery)
		}
		if q.Get("page") != "1" || q.Get("size") != "100" {
			t.Errorf("unexpected pagination query: %q", r.URL.RawQuery)
		}
		creators := q["creatorUuids"]
		if len(creators) != 2 || creators[0] != "cr-1" || creators[1] != "cr-2" {
			t.Errorf("creatorUuids must be repeated keys, got %v", creators)
		}
		_, _ = io.WriteString(w, `{"data":[
			{"creatorUuid":"cr-1","currency":"USD","date":"2026-01-31T00:00:00.000Z","gross":12000,"net":9600},
			{"creatorUuid":"cr-2","currency":null,"date":"2026-01-30T00:00:00.000Z","gross":500,"net":400}
		],"pagination":{"hasMore":false,"page":1,"size":100}}`)
	})

	page, err := c.ListAgencyEarningsByDay(context.Background(), ListAgencyEarningsByDayParams{
		StartDate:    "2026-01-01",
		EndDate:      "2026-01-31",
		Page:         ptrInt(1),
		Size:         ptrInt(100),
		CreatorUUIDs: []string{"cr-1", "cr-2"},
	})
	if err != nil {
		t.Fatalf("ListAgencyEarningsByDay: %v", err)
	}
	if len(page.Data) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(page.Data))
	}
	first := page.Data[0]
	if first.CreatorUUID != "cr-1" || first.Gross != 12000 || first.Net != 9600 {
		t.Errorf("first row: %+v", first)
	}
	if first.Currency == nil || *first.Currency != "USD" {
		t.Errorf("first currency: %v", first.Currency)
	}
	if page.Data[1].Currency != nil {
		t.Errorf("second currency should be nil: %v", page.Data[1].Currency)
	}
}

func TestListAgencyEarningsByDay_RequiresDates(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler must not be reached when required dates are missing")
	})
	if _, err := c.ListAgencyEarningsByDay(context.Background(), ListAgencyEarningsByDayParams{
		EndDate: "2026-01-31",
	}); err == nil {
		t.Fatal("expected error when StartDate is empty")
	}
	if _, err := c.ListAgencyEarningsByDay(context.Background(), ListAgencyEarningsByDayParams{
		StartDate: "2026-01-01",
	}); err == nil {
		t.Fatal("expected error when EndDate is empty")
	}
}

func TestListAgencySubscribers(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/agencies/subscribers" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("page") != "1" || q.Get("size") != "50" {
			t.Errorf("unexpected pagination query: %q", r.URL.RawQuery)
		}
		creators := q["creatorUuids"]
		if len(creators) != 1 || creators[0] != "cr-1" {
			t.Errorf("creatorUuids: got %v", creators)
		}
		_, _ = io.WriteString(w, `{"data":[
			{"avatarUrl":null,"creatorUuid":"cr-1","displayName":"Sub One","expiresAt":"2026-02-01T00:00:00.000Z",
			 "handle":"sub1","isTopSpender":true,"nickname":null,"registeredAt":"2025-10-01T00:00:00.000Z",
			 "subscribedAt":"2026-01-01T00:00:00.000Z","uuid":"u-1"}
		],"pagination":{"hasMore":false,"page":1,"size":50}}`)
	})

	page, err := c.ListAgencySubscribers(context.Background(), ListAgencySubscribersParams{
		Page:         ptrInt(1),
		Size:         ptrInt(50),
		CreatorUUIDs: []string{"cr-1"},
	})
	if err != nil {
		t.Fatalf("ListAgencySubscribers: %v", err)
	}
	if len(page.Data) != 1 {
		t.Fatalf("expected 1 subscriber, got %d", len(page.Data))
	}
	sub := page.Data[0]
	if sub.CreatorUUID != "cr-1" || sub.UUID != "u-1" || !sub.IsTopSpender {
		t.Errorf("subscriber: %+v", sub)
	}
	if sub.ExpiresAt == nil || *sub.ExpiresAt != "2026-02-01T00:00:00.000Z" {
		t.Errorf("expiresAt: %v", sub.ExpiresAt)
	}
}

func TestListAgencySubscribers_OmitsCreatorUUIDsWhenEmpty(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.URL.Query()["creatorUuids"]; ok {
			t.Errorf("creatorUuids must be omitted when empty: %q", r.URL.RawQuery)
		}
		_, _ = io.WriteString(w, `{"data":[],"pagination":{"hasMore":false,"page":1,"size":20}}`)
	})
	if _, err := c.ListAgencySubscribers(context.Background(), ListAgencySubscribersParams{}); err != nil {
		t.Fatalf("ListAgencySubscribers: %v", err)
	}
}

func TestListAgencySubscribersHistory(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/agencies/subscribers-history" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("startDate") != "2026-01-01" || q.Get("endDate") != "2026-01-31" {
			t.Errorf("unexpected date query: %q", r.URL.RawQuery)
		}
		_, _ = io.WriteString(w, `{"data":[
			{"cancelledSubscribersCount":2,"creatorUuid":"cr-1","date":"2026-01-31T00:00:00.000Z",
			 "newSubscribersCount":5,"total":3}
		],"pagination":{"hasMore":false,"page":1,"size":31}}`)
	})

	page, err := c.ListAgencySubscribersHistory(context.Background(), ListAgencySubscribersHistoryParams{
		StartDate: "2026-01-01",
		EndDate:   "2026-01-31",
	})
	if err != nil {
		t.Fatalf("ListAgencySubscribersHistory: %v", err)
	}
	if len(page.Data) != 1 {
		t.Fatalf("expected 1 row, got %d", len(page.Data))
	}
	row := page.Data[0]
	if row.CreatorUUID != "cr-1" || row.NewSubscribersCount != 5 ||
		row.CancelledSubscribersCount != 2 || row.Total != 3 {
		t.Errorf("history row: %+v", row)
	}
}

func TestListAgencySubscribersHistory_RequiresDates(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler must not be reached when required dates are missing")
	})
	if _, err := c.ListAgencySubscribersHistory(context.Background(), ListAgencySubscribersHistoryParams{
		EndDate: "2026-01-31",
	}); err == nil {
		t.Fatal("expected error when StartDate is empty")
	}
	if _, err := c.ListAgencySubscribersHistory(context.Background(), ListAgencySubscribersHistoryParams{
		StartDate: "2026-01-01",
	}); err == nil {
		t.Fatal("expected error when EndDate is empty")
	}
}

func TestListTeamMembers(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/agencies/team-members" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		// Bare array response, not a paginated envelope.
		_, _ = io.WriteString(w, `[
			{"creatorAccess":[{"role":"ADMIN","uuid":"cr-1"},{"role":"CHATTER","uuid":"cr-2"}],
			 "displayName":"Alice","email":"alice@example.com","isAdmin":true,"nickname":null,"uuid":"tm-1"},
			{"creatorAccess":[],"displayName":"Bob","email":"bob@example.com","isAdmin":false,
			 "nickname":"Bobby","uuid":"tm-2"}
		]`)
	})

	members, err := c.ListTeamMembers(context.Background())
	if err != nil {
		t.Fatalf("ListTeamMembers: %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(members))
	}
	first := members[0]
	if first.UUID != "tm-1" || !first.IsAdmin || len(first.CreatorAccess) != 2 {
		t.Errorf("first member: %+v", first)
	}
	if first.CreatorAccess[0].Role != TeamMemberRoleAdmin || first.CreatorAccess[1].Role != TeamMemberRoleChatter {
		t.Errorf("creatorAccess roles: %+v", first.CreatorAccess)
	}
	second := members[1]
	if second.Nickname == nil || *second.Nickname != "Bobby" || second.IsAdmin {
		t.Errorf("second member: %+v", second)
	}
}

func TestUpdateTeamMember(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/agencies/team-members/tm-1" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		raw, _ := io.ReadAll(r.Body)
		var sent map[string]any
		if err := json.Unmarshal(raw, &sent); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if sent["isAdmin"] != false {
			t.Errorf("isAdmin: got %v", sent["isAdmin"])
		}
		if sent["nickname"] != "Lead" {
			t.Errorf("nickname: got %v", sent["nickname"])
		}
		_, _ = io.WriteString(w, `{"creatorAccess":[{"role":"CHATTER","uuid":"cr-2"}],
			"displayName":"Alice","email":"alice@example.com","isAdmin":false,"nickname":"Lead","uuid":"tm-1"}`)
	})

	res, err := c.UpdateTeamMember(context.Background(), "tm-1", UpdateTeamMemberParams{
		IsAdmin:  ptrBool(false),
		Nickname: ptrString("Lead"),
	})
	if err != nil {
		t.Fatalf("UpdateTeamMember: %v", err)
	}
	if res.UUID != "tm-1" || res.IsAdmin {
		t.Errorf("result: %+v", res)
	}
	if res.Nickname == nil || *res.Nickname != "Lead" {
		t.Errorf("nickname: %v", res.Nickname)
	}
	if len(res.CreatorAccess) != 1 || res.CreatorAccess[0].Role != TeamMemberRoleChatter {
		t.Errorf("creatorAccess: %+v", res.CreatorAccess)
	}
}

func TestUpdateTeamMember_RequiresUserID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler must not be reached when userID is empty")
	})
	if _, err := c.UpdateTeamMember(context.Background(), "", UpdateTeamMemberParams{}); err == nil {
		t.Fatal("expected error when userID is empty")
	}
}

func TestUpdateTeamMember_RawBodyTakesPrecedence(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var sent map[string]any
		if err := json.Unmarshal(raw, &sent); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if _, ok := sent["isAdmin"]; ok {
			t.Error("typed isAdmin must not be sent when RawBody is set")
		}
		if sent["custom"] != "value" {
			t.Errorf("RawBody not sent verbatim: got %v", sent)
		}
		_, _ = io.WriteString(w, `{"creatorAccess":[],"displayName":"A","email":"a@b.c","isAdmin":true,"nickname":null,"uuid":"tm-1"}`)
	})
	_, err := c.UpdateTeamMember(context.Background(), "tm-1", UpdateTeamMemberParams{
		IsAdmin: ptrBool(true),
		RawBody: RawBody(`{"custom":"value"}`),
	})
	if err != nil {
		t.Fatalf("UpdateTeamMember: %v", err)
	}
}

func TestListCreators(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/creators" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("page") != "1" || q.Get("size") != "10" {
			t.Errorf("unexpected pagination query: %q", r.URL.RawQuery)
		}
		_, _ = io.WriteString(w, `{"data":[
			{"avatarUrl":"https://cdn/a.png","displayName":"Creator One","handle":"creator1",
			 "isTopSpender":false,"nickname":null,"registeredAt":"2025-09-01T00:00:00.000Z","role":"owner","uuid":"cr-1"},
			{"avatarUrl":null,"displayName":"Creator Two","handle":"creator2","isTopSpender":true,
			 "nickname":"Two","registeredAt":"2025-08-01T00:00:00.000Z","uuid":"cr-2"}
		],"pagination":{"hasMore":true,"page":1,"size":10}}`)
	})

	page, err := c.ListCreators(context.Background(), ListCreatorsParams{
		Page: ptrInt(1),
		Size: ptrInt(10),
	})
	if err != nil {
		t.Fatalf("ListCreators: %v", err)
	}
	if len(page.Data) != 2 {
		t.Fatalf("expected 2 creators, got %d", len(page.Data))
	}
	first := page.Data[0]
	if first.UUID != "cr-1" || first.Role == nil || *first.Role != "owner" {
		t.Errorf("first creator: %+v", first)
	}
	second := page.Data[1]
	if second.Role != nil {
		t.Errorf("second role should be nil: %v", second.Role)
	}
	if !page.Pagination.HasMore {
		t.Errorf("pagination: %+v", page.Pagination)
	}
}

func TestListCreators_PropagatesAPIError(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"message":"insufficient scope"}`)
	})
	_, err := c.ListCreators(context.Background(), ListCreatorsParams{})
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
	apiErr, ok := AsError(err)
	if !ok || !apiErr.IsForbidden() {
		t.Errorf("expected forbidden *Error, got %v", err)
	}
}
