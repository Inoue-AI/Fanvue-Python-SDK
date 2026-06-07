package fanvue

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

const creatorEarningsPageJSON = `{
  "data": [
    {
      "currency": "usd",
      "date": "2026-01-01T00:00:00.000Z",
      "gross": 1000,
      "messageUuid": null,
      "net": 900,
      "postUuid": null,
      "source": "tip",
      "transactionOrderId": "ord-1",
      "transactionOrderStatus": "availableForPayout",
      "user": {
        "displayName": "Fan",
        "handle": "fan",
        "isTopSpender": true,
        "nickname": null,
        "uuid": "u-1"
      }
    }
  ],
  "nextCursor": "cur-next"
}`

const creatorEarningsSummaryJSON = `{
  "averageByDayOfWeek": {"1": 1, "2": 2, "3": 3, "4": 4, "5": 5, "6": 6, "7": 7},
  "averageByHourOfDay": {"0": 0.5, "23": 9.5},
  "breakdownBySource": {
    "messages": {"gross": 10, "net": 9},
    "other": {"gross": 1, "net": 1},
    "posts": {"gross": 20, "net": 18},
    "referrals": {"gross": 0, "net": 0},
    "renewals": {"gross": 30, "net": 27},
    "subs": {"gross": 40, "net": 36},
    "tips": {"gross": 50, "net": 45}
  },
  "earningsByType": {
    "messages": {"gross": 10, "net": 9},
    "renewals": {"gross": 30, "net": 27},
    "subs": {"gross": 40, "net": 36},
    "tips": {"gross": 50, "net": 45}
  },
  "overTime": [{"gross": 100, "net": 90, "periodStart": "2026-01-01T00:00:00.000Z"}],
  "period": {"endDate": null, "granularity": "day", "startDate": null, "timezone": "UTC"},
  "totals": {
    "allTime": {"gross": 1000, "net": 900},
    "thisMonth": {
      "gross": 200, "grossChangePercentage": null, "net": 180,
      "netChangePercentage": 12.5, "previousMonthGross": 100, "previousMonthNet": 90
    }
  }
}`

const creatorInsightsSubscribersPageJSON = `{
  "data": [
    {"cancelledSubscribersCount": 2, "date": "2026-01-01T00:00:00.000Z", "newSubscribersCount": 5, "total": 3}
  ],
  "nextCursor": null
}`

const creatorTopSpendersPageJSON = `{
  "data": [
    {
      "gross": 500, "messages": 12, "net": 450,
      "user": {
        "avatarUrl": null, "displayName": "Whale", "handle": "whale",
        "isTopSpender": true, "nickname": null, "registeredAt": "2025-01-01T00:00:00.000Z", "uuid": "u-2"
      }
    }
  ],
  "pagination": {"hasMore": false, "page": 1, "size": 20}
}`

const creatorFansPageJSON = `{
  "data": [
    {
      "uuid": "u-3", "handle": "sub", "displayName": "Sub", "nickname": null,
      "isTopSpender": false, "avatarUrl": null, "registeredAt": "2025-06-01T00:00:00.000Z"
    }
  ],
  "pagination": {"page": 1, "size": 20, "hasMore": false}
}`

const onlineSubscribersJSON = `{
  "count": 2,
  "data": [
    {"lastSeenAt": "2026-01-01T00:00:00.000Z", "uuid": "u-4"},
    {"lastSeenAt": null, "uuid": "u-5"}
  ]
}`

const creatorNotificationsPageJSON = `{
  "data": [
    {
      "createdAt": "2026-01-01T00:00:00.000Z",
      "data": {"foo": "bar"},
      "eventType": 7,
      "isRead": false,
      "originator": null,
      "receiverUuid": "cr-1",
      "uuid": "n-1"
    }
  ],
  "pagination": {"page": 1, "size": 20, "hasMore": false}
}`

const creatorTrackingLinksPageJSON = `{
  "data": [
    {
      "clicks": 100,
      "createdAt": "2026-01-01T00:00:00.000Z",
      "earnings": {"totalGross": 500, "totalNet": 450},
      "engagement": {"acquiredFollowers": 10, "acquiredSubscribers": 5, "totalFollowers": 20, "totalSubscribers": 8},
      "externalSocialPlatform": "instagram",
      "linkUrl": "https://fanvue.com/x?ref=abc",
      "name": "IG Bio",
      "uuid": "tl-1"
    }
  ],
  "nextCursor": "cur-tl"
}`

const creatorTrackingLinkJSON = `{
  "clicks": 0,
  "createdAt": "2026-01-01T00:00:00.000Z",
  "earnings": null,
  "engagement": {"acquiredFollowers": 0, "acquiredSubscribers": 0, "totalFollowers": 0, "totalSubscribers": 0},
  "externalSocialPlatform": "tiktok",
  "linkUrl": "https://fanvue.com/x?ref=new",
  "name": "TikTok Bio",
  "uuid": "tl-2"
}`

const creatorTrackingLinkUsersPageJSON = `{
  "data": [
    {
      "avatarUrl": null, "displayName": "Tracked", "handle": "tracked",
      "isTopSpender": false, "nickname": null, "registeredAt": "2025-06-01T00:00:00.000Z",
      "status": "subscriber", "uuid": "u-6"
    }
  ],
  "nextCursor": null
}`

const creatorUserTrackingMetadataJSON = `{"metadata": {"utm_source": "ig", "utm_campaign": "spring"}}`

func TestGetCreatorEarnings(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/creators/cr-1/insights/earnings" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("startDate") != "2026-01-01" || q.Get("size") != "10" || q.Get("cursor") != "cur-0" {
			t.Errorf("unexpected query: %q", r.URL.RawQuery)
		}
		sources := q["source"]
		if len(sources) != 2 || sources[0] != "tip" || sources[1] != "post" {
			t.Errorf("unexpected source query: %v", sources)
		}
		_, _ = io.WriteString(w, creatorEarningsPageJSON)
	})

	page, err := c.GetCreatorEarnings(context.Background(), "cr-1", GetCreatorEarningsParams{
		StartDate: ptrString("2026-01-01"),
		Source:    []EarningsSource{EarningsSourceTip, EarningsSourcePost},
		Cursor:    ptrString("cur-0"),
		Size:      ptrInt(10),
	})
	if err != nil {
		t.Fatalf("GetCreatorEarnings: %v", err)
	}
	if len(page.Data) != 1 || page.Data[0].Source != EarningsSourceTip {
		t.Errorf("unexpected data: %+v", page.Data)
	}
	if page.NextCursor == nil || *page.NextCursor != "cur-next" {
		t.Errorf("nextCursor: %v", page.NextCursor)
	}
	if page.Data[0].User == nil || page.Data[0].User.UUID != "u-1" {
		t.Errorf("user: %+v", page.Data[0].User)
	}
}

func TestGetCreatorEarnings_EmptyUUID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be reached")
	})
	if _, err := c.GetCreatorEarnings(context.Background(), "", GetCreatorEarningsParams{}); err == nil {
		t.Fatal("expected error for empty creator UUID")
	}
}

func TestGetCreatorEarnings_OmitsEmptySource(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if _, present := r.URL.Query()["source"]; present {
			t.Errorf("empty source slice must be omitted, query=%q", r.URL.RawQuery)
		}
		_, _ = io.WriteString(w, creatorEarningsPageJSON)
	})
	if _, err := c.GetCreatorEarnings(context.Background(), "cr-1", GetCreatorEarningsParams{}); err != nil {
		t.Fatalf("GetCreatorEarnings: %v", err)
	}
}

func TestGetCreatorEarningsSummary(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/creators/cr-1/insights/earnings/summary" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("granularity") != "week" || q.Get("timezone") != "Europe/London" {
			t.Errorf("unexpected query: %q", r.URL.RawQuery)
		}
		_, _ = io.WriteString(w, creatorEarningsSummaryJSON)
	})

	gran := GranularityWeek
	sum, err := c.GetCreatorEarningsSummary(context.Background(), "cr-1", GetCreatorEarningsSummaryParams{
		Granularity: &gran,
		Timezone:    ptrString("Europe/London"),
	})
	if err != nil {
		t.Fatalf("GetCreatorEarningsSummary: %v", err)
	}
	if sum.AverageByDayOfWeek.Day7 != 7 {
		t.Errorf("day7: got %v", sum.AverageByDayOfWeek.Day7)
	}
	if sum.AverageByHourOfDay["23"] != 9.5 {
		t.Errorf("hour23: got %v", sum.AverageByHourOfDay["23"])
	}
	if sum.Totals.ThisMonth.GrossChangePercentage != nil {
		t.Errorf("grossChangePercentage should be nil")
	}
	if sum.Totals.ThisMonth.NetChangePercentage == nil || *sum.Totals.ThisMonth.NetChangePercentage != 12.5 {
		t.Errorf("netChangePercentage: %v", sum.Totals.ThisMonth.NetChangePercentage)
	}
	if sum.Period.Granularity != GranularityDay {
		t.Errorf("period granularity: %q", sum.Period.Granularity)
	}
}

func TestGetCreatorEarningsSummary_EmptyUUID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be reached")
	})
	if _, err := c.GetCreatorEarningsSummary(context.Background(), "", GetCreatorEarningsSummaryParams{}); err == nil {
		t.Fatal("expected error for empty creator UUID")
	}
}

func TestGetCreatorSubscribers(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/creators/cr-1/insights/subscribers" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		if r.URL.Query().Get("size") != "5" {
			t.Errorf("unexpected query: %q", r.URL.RawQuery)
		}
		_, _ = io.WriteString(w, creatorInsightsSubscribersPageJSON)
	})

	page, err := c.GetCreatorSubscribers(context.Background(), "cr-1", GetCreatorSubscribersParams{
		Size: ptrInt(5),
	})
	if err != nil {
		t.Fatalf("GetCreatorSubscribers: %v", err)
	}
	if len(page.Data) != 1 || page.Data[0].NewSubscribersCount != 5 || page.Data[0].Total != 3 {
		t.Errorf("unexpected data: %+v", page.Data)
	}
	if page.NextCursor != nil {
		t.Errorf("nextCursor should be nil, got %v", *page.NextCursor)
	}
}

func TestGetCreatorSubscribers_EmptyUUID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be reached")
	})
	if _, err := c.GetCreatorSubscribers(context.Background(), "", GetCreatorSubscribersParams{}); err == nil {
		t.Fatal("expected error for empty creator UUID")
	}
}

func TestGetCreatorTopSpenders(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/creators/cr-1/insights/top-spenders" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("page") != "1" || q.Get("size") != "20" {
			t.Errorf("unexpected query: %q", r.URL.RawQuery)
		}
		_, _ = io.WriteString(w, creatorTopSpendersPageJSON)
	})

	page, err := c.GetCreatorTopSpenders(context.Background(), "cr-1", GetCreatorTopSpendersParams{
		Page: ptrInt(1),
		Size: ptrInt(20),
	})
	if err != nil {
		t.Fatalf("GetCreatorTopSpenders: %v", err)
	}
	if len(page.Data) != 1 || page.Data[0].User.Handle != "whale" || page.Data[0].Gross != 500 {
		t.Errorf("unexpected data: %+v", page.Data)
	}
	if page.Pagination.HasMore {
		t.Error("hasMore should be false")
	}
}

func TestGetCreatorTopSpenders_EmptyUUID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be reached")
	})
	if _, err := c.GetCreatorTopSpenders(context.Background(), "", GetCreatorTopSpendersParams{}); err == nil {
		t.Fatal("expected error for empty creator UUID")
	}
}

func TestListCreatorSubscribers(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/creators/cr-1/subscribers" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("page") != "2" || q.Get("size") != "50" {
			t.Errorf("unexpected query: %q", r.URL.RawQuery)
		}
		_, _ = io.WriteString(w, creatorFansPageJSON)
	})

	page, err := c.ListCreatorSubscribers(context.Background(), "cr-1", ListFansParams{
		Page: ptrInt(2),
		Size: ptrInt(50),
	})
	if err != nil {
		t.Fatalf("ListCreatorSubscribers: %v", err)
	}
	if len(page.Data) != 1 || page.Data[0].Handle != "sub" {
		t.Errorf("unexpected data: %+v", page.Data)
	}
}

func TestListCreatorSubscribers_EmptyUUID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be reached")
	})
	if _, err := c.ListCreatorSubscribers(context.Background(), "", ListFansParams{}); err == nil {
		t.Fatal("expected error for empty creator UUID")
	}
}

func TestGetOnlineSubscribers(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/creators/cr-1/subscribers/online" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("limit") != "25" || q.Get("subscriberUuids") != "u-4,u-5" {
			t.Errorf("unexpected query: %q", r.URL.RawQuery)
		}
		_, _ = io.WriteString(w, onlineSubscribersJSON)
	})

	res, err := c.GetOnlineSubscribers(context.Background(), "cr-1", GetOnlineSubscribersParams{
		Limit:           ptrInt(25),
		SubscriberUUIDs: ptrString("u-4,u-5"),
	})
	if err != nil {
		t.Fatalf("GetOnlineSubscribers: %v", err)
	}
	if res.Count != 2 || len(res.Data) != 2 {
		t.Errorf("unexpected result: %+v", res)
	}
	if res.Data[0].LastSeenAt == nil || *res.Data[0].LastSeenAt == "" {
		t.Errorf("first lastSeenAt should be set: %+v", res.Data[0])
	}
	if res.Data[1].LastSeenAt != nil {
		t.Errorf("second lastSeenAt should be nil, got %v", *res.Data[1].LastSeenAt)
	}
}

func TestGetOnlineSubscribers_EmptyUUID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be reached")
	})
	if _, err := c.GetOnlineSubscribers(context.Background(), "", GetOnlineSubscribersParams{}); err == nil {
		t.Fatal("expected error for empty creator UUID")
	}
}

func TestListCreatorFollowers(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/creators/cr-1/followers" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, creatorFansPageJSON)
	})

	page, err := c.ListCreatorFollowers(context.Background(), "cr-1", ListFansParams{})
	if err != nil {
		t.Fatalf("ListCreatorFollowers: %v", err)
	}
	if len(page.Data) != 1 || page.Data[0].UUID != "u-3" {
		t.Errorf("unexpected data: %+v", page.Data)
	}
}

func TestListCreatorFollowers_EmptyUUID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be reached")
	})
	if _, err := c.ListCreatorFollowers(context.Background(), "", ListFansParams{}); err == nil {
		t.Fatal("expected error for empty creator UUID")
	}
}

func TestListCreatorNotifications(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/creators/cr-1/notifications" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		if r.URL.Query().Get("eventType") != "7" {
			t.Errorf("unexpected query: %q", r.URL.RawQuery)
		}
		_, _ = io.WriteString(w, creatorNotificationsPageJSON)
	})

	page, err := c.ListCreatorNotifications(context.Background(), "cr-1", ListNotificationsParams{
		EventType: ptrInt(7),
	})
	if err != nil {
		t.Fatalf("ListCreatorNotifications: %v", err)
	}
	if len(page.Data) != 1 || page.Data[0].EventType != 7 || page.Data[0].UUID != "n-1" {
		t.Errorf("unexpected data: %+v", page.Data)
	}
	if page.Data[0].Originator != nil {
		t.Errorf("originator should be nil")
	}
	var decoded map[string]string
	if err := json.Unmarshal(page.Data[0].Data, &decoded); err != nil {
		t.Fatalf("decode notification data: %v", err)
	}
	if decoded["foo"] != "bar" {
		t.Errorf("notification data: %+v", decoded)
	}
}

func TestListCreatorNotifications_EmptyUUID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be reached")
	})
	if _, err := c.ListCreatorNotifications(context.Background(), "", ListNotificationsParams{}); err == nil {
		t.Fatal("expected error for empty creator UUID")
	}
}

func TestListCreatorTrackingLinks(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/creators/cr-1/tracking-links" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("limit") != "15" || q.Get("createdAfter") != "2026-01-01" {
			t.Errorf("unexpected query: %q", r.URL.RawQuery)
		}
		_, _ = io.WriteString(w, creatorTrackingLinksPageJSON)
	})

	page, err := c.ListCreatorTrackingLinks(context.Background(), "cr-1", ListCreatorTrackingLinksParams{
		Limit:        ptrInt(15),
		CreatedAfter: ptrString("2026-01-01"),
	})
	if err != nil {
		t.Fatalf("ListCreatorTrackingLinks: %v", err)
	}
	if len(page.Data) != 1 || page.Data[0].UUID != "tl-1" {
		t.Errorf("unexpected data: %+v", page.Data)
	}
	if page.Data[0].Earnings == nil || page.Data[0].Earnings.TotalGross != 500 {
		t.Errorf("earnings: %+v", page.Data[0].Earnings)
	}
	if page.Data[0].ExternalSocialPlatform != PlatformInstagram {
		t.Errorf("platform: %q", page.Data[0].ExternalSocialPlatform)
	}
	if page.NextCursor == nil || *page.NextCursor != "cur-tl" {
		t.Errorf("nextCursor: %v", page.NextCursor)
	}
}

func TestListCreatorTrackingLinks_EmptyUUID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be reached")
	})
	if _, err := c.ListCreatorTrackingLinks(context.Background(), "", ListCreatorTrackingLinksParams{}); err == nil {
		t.Fatal("expected error for empty creator UUID")
	}
}

func TestCreateCreatorTrackingLink(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/creators/cr-1/tracking-links" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var got map[string]string
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if got["name"] != "TikTok Bio" || got["externalSocialPlatform"] != "tiktok" {
			t.Errorf("unexpected body: %s", body)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, creatorTrackingLinkJSON)
	})

	link, err := c.CreateCreatorTrackingLink(context.Background(), "cr-1",
		RawBody(`{"name":"TikTok Bio","externalSocialPlatform":"tiktok"}`))
	if err != nil {
		t.Fatalf("CreateCreatorTrackingLink: %v", err)
	}
	if link.UUID != "tl-2" || link.ExternalSocialPlatform != PlatformTikTok {
		t.Errorf("unexpected link: %+v", link)
	}
	if link.Earnings != nil {
		t.Errorf("earnings should be nil, got %+v", link.Earnings)
	}
}

func TestCreateCreatorTrackingLink_NilBody(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if len(body) != 0 {
			t.Errorf("nil body must send no payload, got %q", body)
		}
		_, _ = io.WriteString(w, creatorTrackingLinkJSON)
	})
	if _, err := c.CreateCreatorTrackingLink(context.Background(), "cr-1", nil); err != nil {
		t.Fatalf("CreateCreatorTrackingLink: %v", err)
	}
}

func TestCreateCreatorTrackingLink_EmptyUUID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be reached")
	})
	if _, err := c.CreateCreatorTrackingLink(context.Background(), "", nil); err == nil {
		t.Fatal("expected error for empty creator UUID")
	}
}

func TestDeleteCreatorTrackingLink(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/creators/cr-1/tracking-links/tl-1" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	if err := c.DeleteCreatorTrackingLink(context.Background(), "cr-1", "tl-1"); err != nil {
		t.Fatalf("DeleteCreatorTrackingLink: %v", err)
	}
}

func TestDeleteCreatorTrackingLink_EmptyArgs(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be reached")
	})
	if err := c.DeleteCreatorTrackingLink(context.Background(), "", "tl-1"); err == nil {
		t.Fatal("expected error for empty creator UUID")
	}
	if err := c.DeleteCreatorTrackingLink(context.Background(), "cr-1", ""); err == nil {
		t.Fatal("expected error for empty tracking-link UUID")
	}
}

func TestListCreatorTrackingLinkUsers(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/creators/cr-1/tracking-links/tl-1/users" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("limit") != "10" || q.Get("cursor") != "cur-u" {
			t.Errorf("unexpected query: %q", r.URL.RawQuery)
		}
		_, _ = io.WriteString(w, creatorTrackingLinkUsersPageJSON)
	})

	page, err := c.ListCreatorTrackingLinkUsers(context.Background(), "cr-1", "tl-1",
		ListCreatorTrackingLinkUsersParams{Limit: ptrInt(10), Cursor: ptrString("cur-u")})
	if err != nil {
		t.Fatalf("ListCreatorTrackingLinkUsers: %v", err)
	}
	if len(page.Data) != 1 || page.Data[0].UUID != "u-6" {
		t.Errorf("unexpected data: %+v", page.Data)
	}
	if page.Data[0].Status == nil || *page.Data[0].Status != "subscriber" {
		t.Errorf("status: %v", page.Data[0].Status)
	}
}

func TestListCreatorTrackingLinkUsers_EmptyArgs(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be reached")
	})
	if _, err := c.ListCreatorTrackingLinkUsers(context.Background(), "", "tl-1", ListCreatorTrackingLinkUsersParams{}); err == nil {
		t.Fatal("expected error for empty creator UUID")
	}
	if _, err := c.ListCreatorTrackingLinkUsers(context.Background(), "cr-1", "", ListCreatorTrackingLinkUsersParams{}); err == nil {
		t.Fatal("expected error for empty tracking-link UUID")
	}
}

func TestGetCreatorUserTrackingMetadata(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/creators/cr-1/tracking-links/tl-1/users/u-6/metadata" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, creatorUserTrackingMetadataJSON)
	})

	meta, err := c.GetCreatorUserTrackingMetadata(context.Background(), "cr-1", "tl-1", "u-6")
	if err != nil {
		t.Fatalf("GetCreatorUserTrackingMetadata: %v", err)
	}
	if meta.Metadata["utm_source"] != "ig" || meta.Metadata["utm_campaign"] != "spring" {
		t.Errorf("unexpected metadata: %+v", meta.Metadata)
	}
}

func TestGetCreatorUserTrackingMetadata_EmptyArgs(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be reached")
	})
	if _, err := c.GetCreatorUserTrackingMetadata(context.Background(), "", "tl-1", "u-6"); err == nil {
		t.Fatal("expected error for empty creator UUID")
	}
	if _, err := c.GetCreatorUserTrackingMetadata(context.Background(), "cr-1", "", "u-6"); err == nil {
		t.Fatal("expected error for empty tracking-link UUID")
	}
	if _, err := c.GetCreatorUserTrackingMetadata(context.Background(), "cr-1", "tl-1", ""); err == nil {
		t.Fatal("expected error for empty user UUID")
	}
}
