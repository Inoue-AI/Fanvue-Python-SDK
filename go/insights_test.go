package fanvue

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

const earningsJSON = `{
  "data": [
    {
      "currency": "USD",
      "date": "2026-01-03T00:00:00.000Z",
      "gross": 1000,
      "messageUuid": "msg-1",
      "net": 800,
      "postUuid": null,
      "source": "message",
      "transactionOrderId": "ord-1",
      "transactionOrderStatus": "availableForPayout",
      "user": {
        "uuid": "u-9",
        "handle": "fan9",
        "displayName": "Fan Nine",
        "nickname": null,
        "isTopSpender": true
      }
    },
    {
      "currency": null,
      "date": "2026-01-04T00:00:00.000Z",
      "gross": 500,
      "net": 400,
      "source": "tip",
      "transactionOrderId": "ord-2",
      "transactionOrderStatus": "pendingBalance",
      "user": null
    }
  ],
  "nextCursor": "cur-2"
}`

func TestGetEarnings(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/insights/earnings" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("startDate") != "2026-01-01" || q.Get("endDate") != "2026-01-31" {
			t.Errorf("unexpected date query: %q", r.URL.RawQuery)
		}
		if q.Get("cursor") != "cur-1" || q.Get("size") != "50" {
			t.Errorf("unexpected pagination query: %q", r.URL.RawQuery)
		}
		sources := q["source"]
		if len(sources) != 2 || sources[0] != "message" || sources[1] != "tip" {
			t.Errorf("source must be repeated keys, got %v", sources)
		}
		_, _ = io.WriteString(w, earningsJSON)
	})

	page, err := c.GetEarnings(context.Background(), GetEarningsParams{
		StartDate: ptrString("2026-01-01"),
		EndDate:   ptrString("2026-01-31"),
		Source:    []EarningsSource{EarningsSourceMessage, EarningsSourceTip},
		Cursor:    ptrString("cur-1"),
		Size:      ptrInt(50),
	})
	if err != nil {
		t.Fatalf("GetEarnings: %v", err)
	}
	if len(page.Data) != 2 {
		t.Fatalf("expected 2 items, got %d", len(page.Data))
	}
	first := page.Data[0]
	if first.Currency == nil || *first.Currency != "USD" {
		t.Errorf("currency: got %v", first.Currency)
	}
	if first.Source != EarningsSourceMessage {
		t.Errorf("source: got %q", first.Source)
	}
	if first.User == nil || !first.User.IsTopSpender {
		t.Errorf("first user: got %+v", first.User)
	}
	second := page.Data[1]
	if second.Currency != nil {
		t.Errorf("second currency should be nil, got %v", *second.Currency)
	}
	if second.User != nil {
		t.Errorf("second user should be nil, got %+v", second.User)
	}
	if page.NextCursor == nil || *page.NextCursor != "cur-2" {
		t.Errorf("nextCursor: got %v", page.NextCursor)
	}
}

func TestGetEarnings_OmitsEmptySource(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if _, present := r.URL.Query()["source"]; present {
			t.Errorf("empty source slice must be omitted, query=%q", r.URL.RawQuery)
		}
		_, _ = io.WriteString(w, `{"data":[],"nextCursor":null}`)
	})
	if _, err := c.GetEarnings(context.Background(), GetEarningsParams{}); err != nil {
		t.Fatalf("GetEarnings: %v", err)
	}
}

func TestGetEarningsPercentile(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/insights/earnings/percentile" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, `{"percentile": 0.01}`)
	})
	out, err := c.GetEarningsPercentile(context.Background())
	if err != nil {
		t.Fatalf("GetEarningsPercentile: %v", err)
	}
	if out.Percentile != 0.01 {
		t.Errorf("percentile: got %v", out.Percentile)
	}
}

const earningsSummaryJSON = `{
  "averageByDayOfWeek": {"1": 10, "2": 20, "3": 30, "4": 40, "5": 50, "6": 60, "7": 70},
  "averageByHourOfDay": {"0": 1.5, "13": 9.9},
  "breakdownBySource": {
    "messages": {"gross": 100, "net": 80},
    "other": {"gross": 0, "net": 0},
    "posts": {"gross": 200, "net": 160},
    "referrals": {"gross": 0, "net": 0},
    "renewals": {"gross": 300, "net": 240},
    "subs": {"gross": 400, "net": 320},
    "tips": {"gross": 50, "net": 40}
  },
  "earningsByType": {
    "messages": {"gross": 100, "net": 80},
    "renewals": {"gross": 300, "net": 240},
    "subs": {"gross": 400, "net": 320},
    "tips": {"gross": 50, "net": 40}
  },
  "overTime": [{"gross": 100, "net": 80, "periodStart": "2026-01-01T00:00:00.000Z"}],
  "period": {"endDate": null, "granularity": "day", "startDate": "2026-01-01T00:00:00.000Z", "timezone": "UTC"},
  "totals": {
    "allTime": {"gross": 1050, "net": 840},
    "thisMonth": {
      "gross": 1050, "grossChangePercentage": null, "net": 840,
      "netChangePercentage": 12.5, "previousMonthGross": 900, "previousMonthNet": 700
    }
  }
}`

func TestGetEarningsSummary(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/insights/earnings/summary" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("granularity") != "day" || q.Get("timezone") != "UTC" {
			t.Errorf("unexpected query: %q", r.URL.RawQuery)
		}
		_, _ = io.WriteString(w, earningsSummaryJSON)
	})
	out, err := c.GetEarningsSummary(context.Background(), GetEarningsSummaryParams{
		Granularity: ptrGranularity(GranularityDay),
		Timezone:    ptrString("UTC"),
	})
	if err != nil {
		t.Fatalf("GetEarningsSummary: %v", err)
	}
	if out.AverageByDayOfWeek.Day1 != 10 || out.AverageByDayOfWeek.Day7 != 70 {
		t.Errorf("averageByDayOfWeek: got %+v", out.AverageByDayOfWeek)
	}
	if out.AverageByHourOfDay["13"] != 9.9 {
		t.Errorf("averageByHourOfDay: got %v", out.AverageByHourOfDay)
	}
	if out.BreakdownBySource.Posts.Net != 160 {
		t.Errorf("breakdownBySource.posts.net: got %v", out.BreakdownBySource.Posts.Net)
	}
	if out.Period.EndDate != nil {
		t.Errorf("period.endDate should be nil, got %v", *out.Period.EndDate)
	}
	if out.Period.Granularity != GranularityDay {
		t.Errorf("period.granularity: got %q", out.Period.Granularity)
	}
	if out.Totals.ThisMonth.GrossChangePercentage != nil {
		t.Errorf("grossChangePercentage should be nil, got %v", *out.Totals.ThisMonth.GrossChangePercentage)
	}
	if out.Totals.ThisMonth.NetChangePercentage == nil || *out.Totals.ThisMonth.NetChangePercentage != 12.5 {
		t.Errorf("netChangePercentage: got %v", out.Totals.ThisMonth.NetChangePercentage)
	}
}

const spendingJSON = `{
  "data": [
    {
      "currency": "USD",
      "date": "2026-01-03T00:00:00.000Z",
      "gross": 1000,
      "net": 800,
      "source": "refund",
      "user": {
        "uuid": "u-9", "handle": "fan9", "displayName": "Fan Nine",
        "nickname": null, "isTopSpender": false
      }
    }
  ],
  "nextCursor": null
}`

func TestGetSpending(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/insights/spending" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		sources := r.URL.Query()["source"]
		if len(sources) != 1 || sources[0] != "refund" {
			t.Errorf("source: got %v", sources)
		}
		_, _ = io.WriteString(w, spendingJSON)
	})
	page, err := c.GetSpending(context.Background(), GetSpendingParams{
		Source: []SpendingSource{SpendingSourceRefund},
	})
	if err != nil {
		t.Fatalf("GetSpending: %v", err)
	}
	if len(page.Data) != 1 {
		t.Fatalf("expected 1 item, got %d", len(page.Data))
	}
	if page.Data[0].Source != SpendingSourceRefund {
		t.Errorf("source: got %q", page.Data[0].Source)
	}
	if page.NextCursor != nil {
		t.Errorf("nextCursor should be nil, got %v", *page.NextCursor)
	}
}

const subscribersJSON = `{
  "data": [
    {"cancelledSubscribersCount": 2, "date": "2026-01-01T00:00:00.000Z", "newSubscribersCount": 5, "total": 3}
  ],
  "nextCursor": "cur-x"
}`

func TestGetSubscribers(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/insights/subscribers" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		if r.URL.Query().Get("size") != "10" {
			t.Errorf("unexpected query: %q", r.URL.RawQuery)
		}
		_, _ = io.WriteString(w, subscribersJSON)
	})
	page, err := c.GetSubscribers(context.Background(), GetSubscribersParams{Size: ptrInt(10)})
	if err != nil {
		t.Fatalf("GetSubscribers: %v", err)
	}
	if len(page.Data) != 1 || page.Data[0].NewSubscribersCount != 5 {
		t.Fatalf("unexpected data: %+v", page.Data)
	}
	if page.Data[0].Total != 3 {
		t.Errorf("total: got %v", page.Data[0].Total)
	}
	if page.NextCursor == nil || *page.NextCursor != "cur-x" {
		t.Errorf("nextCursor: got %v", page.NextCursor)
	}
}

const topSpendersJSON = `{
  "data": [
    {
      "gross": 5000, "messages": 12, "net": 4000,
      "user": {
        "avatarUrl": "https://cdn.example.com/a.png", "displayName": "Fan Nine",
        "handle": "fan9", "isTopSpender": true, "nickname": "Niner",
        "registeredAt": "2025-12-01T00:00:00.000Z", "uuid": "u-9"
      }
    }
  ],
  "pagination": {"hasMore": true, "page": 1, "size": 20}
}`

func TestGetTopSpenders(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/insights/top-spenders" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		if r.URL.Query().Get("page") != "1" || r.URL.Query().Get("size") != "20" {
			t.Errorf("unexpected query: %q", r.URL.RawQuery)
		}
		_, _ = io.WriteString(w, topSpendersJSON)
	})
	page, err := c.GetTopSpenders(context.Background(), GetTopSpendersParams{
		Page: ptrInt(1), Size: ptrInt(20),
	})
	if err != nil {
		t.Fatalf("GetTopSpenders: %v", err)
	}
	if len(page.Data) != 1 {
		t.Fatalf("expected 1 item, got %d", len(page.Data))
	}
	item := page.Data[0]
	if item.Gross != 5000 || item.Messages != 12 {
		t.Errorf("amounts: gross=%v messages=%v", item.Gross, item.Messages)
	}
	if item.User.UUID != "u-9" || !item.User.IsTopSpender {
		t.Errorf("user: got %+v", item.User)
	}
	if item.User.AvatarURL == nil || *item.User.AvatarURL != "https://cdn.example.com/a.png" {
		t.Errorf("avatarUrl: got %v", item.User.AvatarURL)
	}
	if !page.Pagination.HasMore || page.Pagination.Page != 1 {
		t.Errorf("pagination: got %+v", page.Pagination)
	}
}

const fanInsightsJSON = `{
  "spending": {
    "lastPurchaseAt": "2026-01-03T00:00:00.000Z",
    "maxSinglePayment": {"gross": 1000, "total": 800},
    "sources": {"message": {"gross": 1500, "total": 1200}},
    "total": {"gross": 2500, "total": 2000}
  },
  "status": "subscriber",
  "subscription": {"autoRenewalEnabled": true, "createdAt": "2025-12-01T00:00:00.000Z", "renewsAt": "2026-02-01T00:00:00.000Z"}
}`

func TestGetFanInsights(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/insights/fans/u-9" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, fanInsightsJSON)
	})
	out, err := c.GetFanInsights(context.Background(), "u-9")
	if err != nil {
		t.Fatalf("GetFanInsights: %v", err)
	}
	if out.Status != FanStatusSubscriber {
		t.Errorf("status: got %q", out.Status)
	}
	if out.Spending.Sources["message"].Total != 1200 {
		t.Errorf("spending source total: got %v", out.Spending.Sources["message"])
	}
	if out.Spending.LastPurchaseAt == nil {
		t.Error("lastPurchaseAt should be set")
	}
	if !out.Subscription.AutoRenewalEnabled {
		t.Error("autoRenewalEnabled should be true")
	}
}

func TestGetFanInsights_EmptyUUID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when user UUID is empty")
	})
	if _, err := c.GetFanInsights(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty user UUID")
	}
}

const bulkFanInsightsJSON = `{
  "errors": [{"code": "NOT_FOUND", "fanUuid": "u-bad", "message": "fan not found"}],
  "results": {
    "u-9": {
      "spending": {
        "lastPurchaseAt": null,
        "maxSinglePayment": {"gross": 1000, "total": 800},
        "sources": {},
        "total": {"gross": 1000, "total": 800}
      },
      "status": "follower",
      "subscription": {"autoRenewalEnabled": false, "createdAt": null, "renewsAt": null}
    },
    "u-bad": null
  }
}`

func TestGetBulkFanInsights(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/insights/fans" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		if r.URL.Query().Get("fanUuids") != "u-9,u-bad" {
			t.Errorf("unexpected fanUuids: %q", r.URL.RawQuery)
		}
		_, _ = io.WriteString(w, bulkFanInsightsJSON)
	})
	out, err := c.GetBulkFanInsights(context.Background(), "u-9,u-bad")
	if err != nil {
		t.Fatalf("GetBulkFanInsights: %v", err)
	}
	if len(out.Errors) != 1 || out.Errors[0].Code != "NOT_FOUND" {
		t.Errorf("errors: got %+v", out.Errors)
	}
	good := out.Results["u-9"]
	if good == nil || good.Status != FanStatusFollower {
		t.Errorf("u-9 result: got %+v", good)
	}
	if bad, present := out.Results["u-bad"]; !present || bad != nil {
		t.Errorf("u-bad should be present and nil, got present=%v val=%+v", present, bad)
	}
}

func TestGetBulkFanInsights_Empty(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when fanUuids is empty")
	})
	if _, err := c.GetBulkFanInsights(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty fanUuids")
	}
}

const batchFanInsightsJSON = `{
  "u-9": {
    "spending": {
      "lastPurchaseAt": null,
      "maxSinglePayment": {"gross": 0, "total": 0},
      "sources": {},
      "total": {"gross": 0, "total": 0}
    },
    "status": "expired",
    "subscription": {"autoRenewalEnabled": false, "createdAt": null, "renewsAt": null}
  },
  "u-bad": {"error": "forbidden"}
}`

func TestBatchFanInsights(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/insights/fans/batch" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var got map[string]json.RawMessage
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("body not JSON: %v (%s)", err, body)
		}
		if _, present := got["fanUuids"]; !present {
			t.Errorf("expected fanUuids key in body, got %s", body)
		}
		_, _ = io.WriteString(w, batchFanInsightsJSON)
	})
	out, err := c.BatchFanInsights(context.Background(), RawBody(`{"fanUuids":["u-9","u-bad"]}`))
	if err != nil {
		t.Fatalf("BatchFanInsights: %v", err)
	}
	good, present := out["u-9"]
	if !present || good.Status != FanStatusExpired || good.Error != "" {
		t.Errorf("u-9 result: got %+v", good)
	}
	if good.Spending == nil {
		t.Error("u-9 spending should be populated")
	}
	bad, present := out["u-bad"]
	if !present || bad.Error != "forbidden" {
		t.Errorf("u-bad result: got %+v", bad)
	}
	if bad.Spending != nil {
		t.Errorf("u-bad spending should be nil, got %+v", bad.Spending)
	}
}

func TestBatchFanInsights_EmptyBody(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when body is empty")
	})
	if _, err := c.BatchFanInsights(context.Background(), nil); err == nil {
		t.Fatal("expected error for empty body")
	}
}

const trackingLinksJSON = `{
  "data": [
    {
      "clicks": 42,
      "createdAt": "2026-01-01T00:00:00.000Z",
      "earnings": {"totalGross": 1000, "totalNet": 800},
      "engagement": {"acquiredFollowers": 10, "acquiredSubscribers": 3, "totalFollowers": 50, "totalSubscribers": 20},
      "externalSocialPlatform": "instagram",
      "linkUrl": "https://fanvue.com/track/abc",
      "name": "IG bio",
      "uuid": "tl-1"
    },
    {
      "clicks": 0,
      "createdAt": "2026-01-02T00:00:00.000Z",
      "earnings": null,
      "engagement": {"acquiredFollowers": 0, "acquiredSubscribers": 0, "totalFollowers": 0, "totalSubscribers": 0},
      "externalSocialPlatform": "other",
      "linkUrl": "https://fanvue.com/track/def",
      "name": "Misc",
      "uuid": "tl-2"
    }
  ],
  "nextCursor": "cur-tl"
}`

func TestListTrackingLinks(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/tracking-links" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("limit") != "25" || q.Get("cursor") != "cur-0" {
			t.Errorf("unexpected query: %q", r.URL.RawQuery)
		}
		if q.Get("createdAfter") != "2026-01-01" || q.Get("createdBefore") != "2026-02-01" {
			t.Errorf("unexpected date query: %q", r.URL.RawQuery)
		}
		_, _ = io.WriteString(w, trackingLinksJSON)
	})
	page, err := c.ListTrackingLinks(context.Background(), ListTrackingLinksParams{
		Limit:         ptrInt(25),
		Cursor:        ptrString("cur-0"),
		CreatedAfter:  ptrString("2026-01-01"),
		CreatedBefore: ptrString("2026-02-01"),
	})
	if err != nil {
		t.Fatalf("ListTrackingLinks: %v", err)
	}
	if len(page.Data) != 2 {
		t.Fatalf("expected 2 links, got %d", len(page.Data))
	}
	first := page.Data[0]
	if first.ExternalSocialPlatform != PlatformInstagram {
		t.Errorf("platform: got %q", first.ExternalSocialPlatform)
	}
	if first.Earnings == nil || first.Earnings.TotalNet != 800 {
		t.Errorf("earnings: got %+v", first.Earnings)
	}
	if first.Engagement.AcquiredFollowers != 10 {
		t.Errorf("engagement: got %+v", first.Engagement)
	}
	if page.Data[1].Earnings != nil {
		t.Errorf("second earnings should be nil, got %+v", page.Data[1].Earnings)
	}
	if page.NextCursor == nil || *page.NextCursor != "cur-tl" {
		t.Errorf("nextCursor: got %v", page.NextCursor)
	}
}

const createTrackingLinkJSON = `{
  "clicks": 0,
  "createdAt": "2026-01-05T00:00:00.000Z",
  "earnings": null,
  "engagement": {"acquiredFollowers": 0, "acquiredSubscribers": 0, "totalFollowers": 0, "totalSubscribers": 0},
  "externalSocialPlatform": "tiktok",
  "linkUrl": "https://fanvue.com/track/ghi",
  "name": "TikTok bio",
  "uuid": "tl-3"
}`

func TestCreateTrackingLink(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/tracking-links" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var got map[string]any
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("body not JSON: %v (%s)", err, body)
		}
		if got["name"] != "TikTok bio" {
			t.Errorf("body name: got %v", got["name"])
		}
		if got["externalSocialPlatform"] != "tiktok" {
			t.Errorf("body platform: got %v", got["externalSocialPlatform"])
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, createTrackingLinkJSON)
	})
	link, err := c.CreateTrackingLink(context.Background(),
		RawBody(`{"name":"TikTok bio","externalSocialPlatform":"tiktok"}`))
	if err != nil {
		t.Fatalf("CreateTrackingLink: %v", err)
	}
	if link.UUID != "tl-3" {
		t.Errorf("uuid: got %q", link.UUID)
	}
	if link.ExternalSocialPlatform != PlatformTikTok {
		t.Errorf("platform: got %q", link.ExternalSocialPlatform)
	}
}

func TestCreateTrackingLink_NilBodySendsNoPayload(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if len(body) != 0 {
			t.Errorf("nil body must send no payload, got %s", body)
		}
		if ct := r.Header.Get("Content-Type"); ct != "" {
			t.Errorf("nil body must not set Content-Type, got %q", ct)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, createTrackingLinkJSON)
	})
	if _, err := c.CreateTrackingLink(context.Background(), nil); err != nil {
		t.Fatalf("CreateTrackingLink: %v", err)
	}
}

func TestDeleteTrackingLink(t *testing.T) {
	called := false
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/tracking-links/tl-1" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	if err := c.DeleteTrackingLink(context.Background(), "tl-1"); err != nil {
		t.Fatalf("DeleteTrackingLink: %v", err)
	}
	if !called {
		t.Error("server was not called")
	}
}

func TestDeleteTrackingLink_EmptyUUID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when UUID is empty")
	})
	if err := c.DeleteTrackingLink(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty UUID")
	}
}

const trackingLinkUsersJSON = `{
  "data": [
    {
      "avatarUrl": "https://cdn.example.com/a.png",
      "displayName": "Fan Nine",
      "handle": "fan9",
      "isTopSpender": true,
      "nickname": "Niner",
      "registeredAt": "2025-12-01T00:00:00.000Z",
      "status": "subscriber",
      "uuid": "u-9"
    },
    {
      "avatarUrl": null,
      "displayName": "Fan Ten",
      "handle": "fan10",
      "isTopSpender": false,
      "nickname": null,
      "registeredAt": "2025-12-02T00:00:00.000Z",
      "status": null,
      "uuid": "u-10"
    }
  ],
  "nextCursor": null
}`

func TestListTrackingLinkUsers(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/tracking-links/tl-1/users" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		if r.URL.Query().Get("limit") != "10" || r.URL.Query().Get("cursor") != "cur-u" {
			t.Errorf("unexpected query: %q", r.URL.RawQuery)
		}
		_, _ = io.WriteString(w, trackingLinkUsersJSON)
	})
	page, err := c.ListTrackingLinkUsers(context.Background(), "tl-1", ListTrackingLinkUsersParams{
		Limit:  ptrInt(10),
		Cursor: ptrString("cur-u"),
	})
	if err != nil {
		t.Fatalf("ListTrackingLinkUsers: %v", err)
	}
	if len(page.Data) != 2 {
		t.Fatalf("expected 2 users, got %d", len(page.Data))
	}
	first := page.Data[0]
	if first.Status == nil || *first.Status != "subscriber" {
		t.Errorf("status: got %v", first.Status)
	}
	if first.AvatarURL == nil || *first.AvatarURL != "https://cdn.example.com/a.png" {
		t.Errorf("avatarUrl: got %v", first.AvatarURL)
	}
	second := page.Data[1]
	if second.Status != nil {
		t.Errorf("second status should be nil, got %v", *second.Status)
	}
	if second.AvatarURL != nil {
		t.Errorf("second avatarUrl should be nil, got %v", *second.AvatarURL)
	}
	if page.NextCursor != nil {
		t.Errorf("nextCursor should be nil, got %v", *page.NextCursor)
	}
}

func TestListTrackingLinkUsers_EmptyUUID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when UUID is empty")
	})
	if _, err := c.ListTrackingLinkUsers(context.Background(), "", ListTrackingLinkUsersParams{}); err == nil {
		t.Fatal("expected error for empty UUID")
	}
}

func TestGetUserTrackingMetadata(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/tracking-links/tl-1/users/u-9/metadata" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, `{"metadata": {"utm_source": "instagram", "campaign": "spring"}}`)
	})
	out, err := c.GetUserTrackingMetadata(context.Background(), "tl-1", "u-9")
	if err != nil {
		t.Fatalf("GetUserTrackingMetadata: %v", err)
	}
	if out.Metadata == nil || out.Metadata["utm_source"] != "instagram" {
		t.Errorf("metadata: got %+v", out.Metadata)
	}
}

func TestGetUserTrackingMetadata_NullMetadata(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"metadata": null}`)
	})
	out, err := c.GetUserTrackingMetadata(context.Background(), "tl-1", "u-9")
	if err != nil {
		t.Fatalf("GetUserTrackingMetadata: %v", err)
	}
	if out.Metadata != nil {
		t.Errorf("metadata should be nil, got %+v", out.Metadata)
	}
}

func TestGetUserTrackingMetadata_EmptyArgs(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when args are empty")
	})
	if _, err := c.GetUserTrackingMetadata(context.Background(), "", "u-9"); err == nil {
		t.Fatal("expected error for empty tracking-link UUID")
	}
	if _, err := c.GetUserTrackingMetadata(context.Background(), "tl-1", ""); err == nil {
		t.Fatal("expected error for empty user UUID")
	}
}

func ptrGranularity(g EarningsGranularity) *EarningsGranularity { return &g }
