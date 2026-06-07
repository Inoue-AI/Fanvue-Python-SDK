package fanvue

import (
	"context"
	"io"
	"net/http"
	"testing"
)

const accountJSON = `{
  "account": {
    "earnings": {
      "availableBalance": 1234.56,
      "lastPayoutAt": null,
      "total": 9876.54
    },
    "fans": {
      "followers": 42,
      "subscribers": 17
    },
    "status": "active"
  },
  "avatarUrl": null,
  "bannerUrl": "https://cdn.example.com/banner.png",
  "bio": "creator bio",
  "createdAt": "2026-01-01T00:00:00.000Z",
  "displayName": "Creator",
  "email": "creator@example.com",
  "handle": "creator",
  "isCreator": true,
  "updatedAt": null,
  "uuid": "u-1"
}`

func TestGetAccount(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/users/account" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, accountJSON)
	})

	acc, err := c.GetAccount(context.Background())
	if err != nil {
		t.Fatalf("GetAccount: %v", err)
	}
	if acc.UUID != "u-1" {
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
	if acc.BannerURL == nil || *acc.BannerURL != "https://cdn.example.com/banner.png" {
		t.Errorf("bannerUrl: got %v", acc.BannerURL)
	}
	if !acc.IsCreator {
		t.Error("isCreator should be true")
	}
}

const appCurrentUserSubscriptionJSON = `{
  "appUuid": "app-1",
  "cancelAtPeriodEnd": false,
  "currentPeriodEnd": "2026-07-01T00:00:00.000Z",
  "hasActiveSubscription": true,
  "managedCreators": [
    {
      "cancelAtPeriodEnd": true,
      "currentPeriodEnd": "2026-08-01T00:00:00.000Z",
      "hasActiveSubscription": true,
      "planName": "Pro",
      "planUuid": "plan-9",
      "status": "active",
      "userUuid": "creator-7"
    }
  ],
  "planName": "Basic",
  "planUuid": "plan-1",
  "status": "active",
  "userUuid": "u-1"
}`

func TestGetAppCurrentUserSubscription(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/apps/app-1/subscription/me" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, appCurrentUserSubscriptionJSON)
	})

	sub, err := c.GetAppCurrentUserSubscription(context.Background(), "app-1")
	if err != nil {
		t.Fatalf("GetAppCurrentUserSubscription: %v", err)
	}
	if sub.AppUUID != "app-1" {
		t.Errorf("appUuid: got %q", sub.AppUUID)
	}
	if !sub.HasActiveSubscription {
		t.Error("hasActiveSubscription should be true")
	}
	if sub.Status != "active" {
		t.Errorf("status: got %q", sub.Status)
	}
	if sub.PlanName == nil || *sub.PlanName != "Basic" {
		t.Errorf("planName: got %v", sub.PlanName)
	}
	if len(sub.ManagedCreators) != 1 {
		t.Fatalf("expected 1 managed creator, got %d", len(sub.ManagedCreators))
	}
	mc := sub.ManagedCreators[0]
	if mc.UserUUID != "creator-7" {
		t.Errorf("managed userUuid: got %q", mc.UserUUID)
	}
	if !mc.CancelAtPeriodEnd {
		t.Error("managed cancelAtPeriodEnd should be true")
	}
	if mc.Status != "active" {
		t.Errorf("managed status: got %q", mc.Status)
	}
}

func TestGetAppCurrentUserSubscription_EmptyUUID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when app UUID is empty")
	})
	if _, err := c.GetAppCurrentUserSubscription(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty app UUID")
	}
}

func TestGetAppCurrentUserSubscription_EmptyManagedCreators(t *testing.T) {
	const body = `{
  "appUuid": "app-2",
  "cancelAtPeriodEnd": false,
  "currentPeriodEnd": null,
  "hasActiveSubscription": false,
  "managedCreators": [],
  "planName": null,
  "planUuid": null,
  "status": "none",
  "userUuid": "u-2"
}`
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, body)
	})
	sub, err := c.GetAppCurrentUserSubscription(context.Background(), "app-2")
	if err != nil {
		t.Fatalf("GetAppCurrentUserSubscription: %v", err)
	}
	if sub.Status != "none" {
		t.Errorf("status: got %q", sub.Status)
	}
	if sub.CurrentPeriodEnd != nil {
		t.Errorf("currentPeriodEnd should be nil, got %v", *sub.CurrentPeriodEnd)
	}
	if sub.PlanName != nil || sub.PlanUUID != nil {
		t.Errorf("plan fields should be nil, got name=%v uuid=%v", sub.PlanName, sub.PlanUUID)
	}
	if len(sub.ManagedCreators) != 0 {
		t.Errorf("expected empty managedCreators, got %d", len(sub.ManagedCreators))
	}
}

const appSubscriptionStatusJSON = `{
  "appName": "My App",
  "appUuid": "app-1",
  "availability": "complete",
  "overallStatus": "active",
  "pricingPlans": [
    {
      "billingType": "recurring",
      "currencyCode": "USD",
      "interval": "monthly",
      "name": "Monthly Pro",
      "price": 9.99,
      "status": "active",
      "uuid": "plan-1"
    },
    {
      "billingType": "free",
      "currencyCode": "USD",
      "interval": null,
      "name": "Free",
      "price": 0,
      "status": "pending_setup",
      "uuid": "plan-2"
    }
  ]
}`

func TestGetAppSubscriptionStatus(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/apps/app-1/subscription-status" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, appSubscriptionStatusJSON)
	})

	status, err := c.GetAppSubscriptionStatus(context.Background(), "app-1")
	if err != nil {
		t.Fatalf("GetAppSubscriptionStatus: %v", err)
	}
	if status.AppName != "My App" {
		t.Errorf("appName: got %q", status.AppName)
	}
	if status.Availability != "complete" {
		t.Errorf("availability: got %q", status.Availability)
	}
	if status.OverallStatus != "active" {
		t.Errorf("overallStatus: got %q", status.OverallStatus)
	}
	if len(status.PricingPlans) != 2 {
		t.Fatalf("expected 2 pricing plans, got %d", len(status.PricingPlans))
	}
	recurring := status.PricingPlans[0]
	if recurring.BillingType != "recurring" {
		t.Errorf("plan[0] billingType: got %q", recurring.BillingType)
	}
	if recurring.Interval == nil || *recurring.Interval != "monthly" {
		t.Errorf("plan[0] interval: got %v", recurring.Interval)
	}
	if recurring.Price != 9.99 {
		t.Errorf("plan[0] price: got %v", recurring.Price)
	}
	free := status.PricingPlans[1]
	if free.Interval != nil {
		t.Errorf("plan[1] interval should be nil, got %v", *free.Interval)
	}
	if free.Status != "pending_setup" {
		t.Errorf("plan[1] status: got %q", free.Status)
	}
}

func TestGetAppSubscriptionStatus_EmptyUUID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called when app UUID is empty")
	})
	if _, err := c.GetAppSubscriptionStatus(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty app UUID")
	}
}
