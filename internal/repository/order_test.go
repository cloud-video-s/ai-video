package repository

import (
	"context"
	"testing"
	"time"

	"ai-video/internal/config"
	"ai-video/internal/domain"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestOrderRepoPageAdminAssociatesPurchaser(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:order-admin?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	statements := []string{
		`CREATE TABLE video_user (
			id INTEGER PRIMARY KEY, username TEXT, login_account TEXT, email TEXT,
			phone TEXT, imei TEXT, device_code TEXT, user_type INTEGER, status INTEGER,
			deleted_at DATETIME NULL
		)`,
		`CREATE TABLE video_user_identity (
			id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL, email TEXT, display_name TEXT
		)`,
		`CREATE TABLE video_order (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			order_no TEXT NOT NULL,
			client_request_id TEXT NOT NULL,
			user_id INTEGER NOT NULL,
			product_type INTEGER NOT NULL,
			product_id INTEGER NOT NULL,
			product_code TEXT NOT NULL,
			product_name TEXT NOT NULL,
			currency TEXT NOT NULL,
			product_amount REAL NOT NULL DEFAULT 0,
			discount_amount REAL NOT NULL DEFAULT 0,
			payable_amount REAL NOT NULL DEFAULT 0,
			paid_amount REAL NOT NULL DEFAULT 0,
			refunded_amount REAL NOT NULL DEFAULT 0,
			bonus_points INTEGER NOT NULL DEFAULT 0,
			vip_level INTEGER NOT NULL DEFAULT 0,
			vip_duration_days INTEGER NOT NULL DEFAULT 0,
			status INTEGER NOT NULL,
			payment_method TEXT NOT NULL,
			third_order_no TEXT NULL,
			original_transaction_id TEXT NULL,
			payment_evidence TEXT NULL,
			failure_code TEXT NULL,
			failure_message TEXT NULL,
			cancel_reason TEXT NULL,
			pay_at DATETIME NULL,
			cancelled_at DATETIME NULL,
			expires_at DATETIME NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			deleted_at DATETIME NULL
		)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Exec(`INSERT INTO video_user (
		id, username, login_account, email, phone, imei, device_code, user_type, status
	) VALUES (7, 'Alice', 'alice-login', 'alice@example.com', '13800000000', 'imei-7', 'device-7', 2, 1)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO video_user_identity (id, user_id, email, display_name)
		VALUES (1, 7, 'apple-relay@example.com', 'Alice Apple')`).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now().Truncate(time.Second)
	if err := db.Exec(`INSERT INTO video_order (
		order_no, client_request_id, user_id, product_type, product_id, product_code,
		product_name, currency, product_amount, discount_amount, payable_amount,
		paid_amount, refunded_amount, bonus_points, vip_level, vip_duration_days,
		status, payment_method, third_order_no, original_transaction_id,
		payment_evidence, pay_at, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"order-1", "request-1", 7, 1, 3, "vip-monthly", "VIP Monthly", "USD",
		19.99, 0, 19.99, 19.99, 2.5, 0, 1, 30, domain.OrderStatusPaid, "apple_iap",
		"transaction-1", "original-1", `{"transactionId":"transaction-1"}`, now, now, now,
	).Error; err != nil {
		t.Fatal(err)
	}

	previousDB := config.DB
	config.DB = db
	t.Cleanup(func() { config.DB = previousDB })

	repo := NewOrderRepo()
	records, total, summary, err := repo.PageAdmin(context.Background(), 1, 20, &OrderAdminFilter{
		ProductType: 1, Status: domain.OrderStatusPaid, Keyword: "apple-relay@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(records) != 1 {
		t.Fatalf("PageAdmin() returned %d records, total %d", len(records), total)
	}
	if records[0].User == nil || records[0].User.Username != "Alice" {
		t.Fatalf("PageAdmin() purchaser = %#v", records[0].User)
	}
	if summary.PaidOrderCount != 1 || len(summary.Amounts) != 1 ||
		summary.Amounts[0].Currency != "USD" || summary.Amounts[0].PaidTotal != 19.99 ||
		summary.Amounts[0].RefundedTotal != 2.5 {
		t.Fatalf("PageAdmin() summary = %#v", summary)
	}

	detail, err := repo.GetAdminDetail(context.Background(), records[0].Order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Order.OrderNo != "order-1" || detail.User == nil || detail.User.ID != 7 {
		t.Fatalf("GetAdminDetail() = %#v", detail)
	}
}
