package service

import (
	"testing"
	"time"

	"ai-video/internal/gen/model"
	"ai-video/internal/repository"
)

func TestParseOrderDateRangeIncludesWholeEndDate(t *testing.T) {
	from, to, err := parseOrderDateRange("2026-08-01", "2026-08-03")
	if err != nil {
		t.Fatal(err)
	}
	if from == nil || from.Format("2006-01-02 15:04:05") != "2026-08-01 00:00:00" {
		t.Fatalf("unexpected start: %v", from)
	}
	if to == nil || to.Format("2006-01-02 15:04:05") != "2026-08-04 00:00:00" {
		t.Fatalf("unexpected exclusive end: %v", to)
	}
	if _, _, err := parseOrderDateRange("2026-08-04", "2026-08-03"); err == nil {
		t.Fatal("expected reverse date range to be rejected")
	}
}

func TestOrderAdminViewIncludesPurchaserAndDetailEvidence(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	record := &repository.OrderAdminRecord{
		Order: model.VideoOrder{
			ID: 9, OrderNo: "order-9", UserID: 7, ProductType: 1,
			ProductID: 3, ProductCode: "vip-monthly", ProductName: "VIP Monthly",
			Currency: "USD", PayableAmount: 19.99, PaidAmount: 19.99,
			Status: "paid", PaymentMethod: "apple_iap", PaymentEvidence: `{"transactionId":"tx-9"}`,
			PaidAt: now, CreatedAt: now, UpdatedAt: now,
		},
		User: &model.VideoUser{ID: 7, Username: "Alice", Email: "alice@example.com", Status: 1},
	}

	listView := orderAdminView(record, false)
	if listView.User == nil || listView.User.Username != "Alice" {
		t.Fatalf("list purchaser = %#v", listView.User)
	}
	if listView.PaymentEvidence != "" {
		t.Fatal("list view must omit payment evidence")
	}
	if listView.PaidAt == nil || !listView.PaidAt.Equal(now) {
		t.Fatalf("paid_at = %v", listView.PaidAt)
	}

	detailView := orderAdminView(record, true)
	if detailView.PaymentEvidence != record.Order.PaymentEvidence {
		t.Fatalf("detail evidence = %q", detailView.PaymentEvidence)
	}
}
