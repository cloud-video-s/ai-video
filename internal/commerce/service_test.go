package commerce

import (
	"context"
	"errors"
	"testing"

	"ai-video/internal/config"
	"ai-video/internal/domain"
	"ai-video/internal/gen/model"
	"ai-video/internal/repository"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestPurchaseAndConsumePointsAreIdempotent(t *testing.T) {
	db := commerceTestDB(t)
	service := NewService()
	ctx := context.Background()

	order, err := service.CreateOrder(ctx, CreateOrderRequest{
		UserID: 1, ProductType: domain.OrderProductPointsPackage, ProductID: 10,
		PaymentMethod: domain.PaymentMethodAppleIAP, ClientRequestID: "request-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if order.ProductCode != "points.small" || order.BonusPoints != 100 || order.PayableAmount != 9.99 {
		t.Fatalf("unexpected product snapshot: %#v", order)
	}

	payment := ApplePaymentResult{
		TransactionID: "apple-transaction-1", ProductCode: "points.small",
		Currency: "USD", PaidAmount: 9.99, SignedTransaction: "verified-jws",
	}
	paid, err := service.ConfirmApplePayment(ctx, order.OrderNo, payment)
	if err != nil {
		t.Fatal(err)
	}
	if paid.Status != domain.OrderStatusPaid {
		t.Fatalf("status=%s", paid.Status)
	}
	second, err := service.ConfirmApplePayment(ctx, order.OrderNo, payment)
	if err != nil || second.ID != paid.ID {
		t.Fatalf("idempotent payment = %#v, %v", second, err)
	}
	otherOrder, err := service.CreateOrder(ctx, CreateOrderRequest{
		UserID: 1, ProductType: domain.OrderProductPointsPackage, ProductID: 10,
		PaymentMethod: domain.PaymentMethodAppleIAP, ClientRequestID: "request-2",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ConfirmApplePayment(ctx, otherOrder.OrderNo, payment); !errors.Is(err, ErrPaymentTransactionUsed) {
		t.Fatalf("reused Apple transaction error=%v", err)
	}

	firstConsume, err := service.ConsumePoints(ctx, ConsumePointsRequest{UserID: 1, WorkID: "work-1", ModeKey: "text-to-video", Points: 30})
	if err != nil {
		t.Fatal(err)
	}
	secondConsume, err := service.ConsumePoints(ctx, ConsumePointsRequest{UserID: 1, WorkID: "work-1", ModeKey: "text-to-video", Points: 30})
	if err != nil || secondConsume.ID != firstConsume.ID {
		t.Fatalf("idempotent consume = %#v, %v", secondConsume, err)
	}

	var user model.VideoUser
	if err := db.First(&user, 1).Error; err != nil {
		t.Fatal(err)
	}
	if user.PointsBalance != 70 {
		t.Fatalf("points balance=%d, want 70", user.PointsBalance)
	}
	var ledgerCount int64
	if err := db.Model(&model.VideoUserPointsLedger{}).Count(&ledgerCount).Error; err != nil {
		t.Fatal(err)
	}
	if ledgerCount != 3 { // one legacy row, one purchase credit, one consumption
		t.Fatalf("ledger count=%d, want 3", ledgerCount)
	}
}

func TestCancelledOrderCannotBePaid(t *testing.T) {
	commerceTestDB(t)
	service := NewService()
	ctx := context.Background()
	order, err := service.CreateOrder(ctx, CreateOrderRequest{
		UserID: 1, ProductType: domain.OrderProductPointsPackage, ProductID: 10,
		PaymentMethod: domain.PaymentMethodAppleIAP, ClientRequestID: "request-cancel",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.CancelOrder(ctx, 1, order.OrderNo, "user cancelled"); err != nil {
		t.Fatal(err)
	}
	_, err = service.ConfirmApplePayment(ctx, order.OrderNo, ApplePaymentResult{
		TransactionID: "apple-cancelled", ProductCode: "points.small", Currency: "USD", PaidAmount: 9.99,
	})
	if !errors.Is(err, repository.ErrOrderNotPending) {
		t.Fatalf("cancelled payment error=%v", err)
	}
}

func commerceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	config.DB = db
	statements := []string{
		`CREATE TABLE video_user (
			id INTEGER PRIMARY KEY, points_balance INTEGER NOT NULL DEFAULT 0,
			order_count INTEGER NOT NULL DEFAULT 0, payment_count INTEGER NOT NULL DEFAULT 0,
			subscription_payment_count INTEGER NOT NULL DEFAULT 0, one_time_payment_count INTEGER NOT NULL DEFAULT 0,
			order_amount_money DECIMAL(12,2) NOT NULL DEFAULT 0, actual_amount_money DECIMAL(12,2) NOT NULL DEFAULT 0,
			first_order_created_at DATETIME, first_paid_at DATETIME, last_paid_at DATETIME,
			payment_met BOOLEAN NOT NULL DEFAULT 0, first_payment_met BOOLEAN NOT NULL DEFAULT 0,
			vip_level INTEGER NOT NULL DEFAULT 0, vip_started_at DATETIME, vip_expires_at DATETIME,
			user_type INTEGER NOT NULL DEFAULT 1, subscription_status INTEGER NOT NULL DEFAULT 1,
			status INTEGER NOT NULL DEFAULT 1, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE video_points_package (
			id INTEGER PRIMARY KEY, product_code TEXT NOT NULL, name TEXT NOT NULL, points INTEGER NOT NULL,
			currency TEXT NOT NULL, sale_price DECIMAL(12,2) NOT NULL, status INTEGER NOT NULL, deleted_at DATETIME
		)`,
		`CREATE TABLE video_user_points_ledger (
			id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL, direction INTEGER NOT NULL,
			points_change INTEGER NOT NULL, balance_before INTEGER NOT NULL, balance_after INTEGER NOT NULL,
			source_type TEXT NOT NULL, business_id TEXT, points_package_id INTEGER, operator_admin_id INTEGER,
			description TEXT, occurred_at DATETIME NOT NULL, created_at DATETIME NOT NULL,
			order_id INTEGER, work_id TEXT, mode_key TEXT, idempotency_key TEXT UNIQUE
		)`,
		`CREATE TABLE video_order (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			order_no TEXT NOT NULL UNIQUE, client_request_id TEXT NOT NULL UNIQUE,
			user_id INTEGER NOT NULL, product_type TEXT NOT NULL, product_id INTEGER NOT NULL,
			product_code TEXT NOT NULL, product_name TEXT NOT NULL, currency TEXT NOT NULL,
			product_amount DECIMAL(12,2) NOT NULL DEFAULT 0,
			discount_amount DECIMAL(12,2) NOT NULL DEFAULT 0,
			payable_amount DECIMAL(12,2) NOT NULL DEFAULT 0,
			paid_amount DECIMAL(12,2) NOT NULL DEFAULT 0,
			refunded_amount DECIMAL(12,2) NOT NULL DEFAULT 0,
			bonus_points INTEGER NOT NULL DEFAULT 0, vip_level INTEGER NOT NULL DEFAULT 0,
			vip_duration_days INTEGER NOT NULL DEFAULT 0, status TEXT NOT NULL,
			payment_method TEXT NOT NULL, provider_transaction_id TEXT,
			original_transaction_id TEXT, payment_evidence TEXT,
			failure_code TEXT, failure_message TEXT, cancel_reason TEXT,
			paid_at DATETIME, cancelled_at DATETIME, expires_at DATETIME,
			created_at DATETIME NOT NULL, updated_at DATETIME NOT NULL,
			UNIQUE (payment_method, provider_transaction_id)
		)`,
		`INSERT INTO video_user (id) VALUES (1)`,
		`INSERT INTO video_points_package (id, product_code, name, points, currency, sale_price, status)
			VALUES (10, 'points.small', 'Small points pack', 100, 'USD', 9.99, 1)`,
		`INSERT INTO video_user_points_ledger
			(user_id, direction, points_change, balance_before, balance_after, source_type, occurred_at, created_at)
			VALUES (1, 1, 0, 0, 0, 'legacy', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	return db
}
