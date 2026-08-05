package commerce

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"ai-video/internal/config"
	"ai-video/internal/domain"
	"ai-video/internal/gen/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestConfirmApplePurchaseCompletesInitialSubscription(t *testing.T) {
	db := newAppleCommerceTestDB(t)
	signer := newAppleJWSTestSigner(t)
	now := signer.now
	expires := now.Add(30 * 24 * time.Hour)
	if err := db.Exec(`INSERT INTO video_user (
		id, device_code, user_type, subscription_status, vip_points, points_balance,
		payment_count, subscription_payment_count, one_time_payment_count, order_count,
		actual_amount_money, order_amount_money, first_payment_met, payment_met,
		vip_level, created_at, updated_at
	) VALUES (1, 'device-1', 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, ?, ?)`, now, now).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO video_vip_subscription (
		id, suk_code, name, level_id, currency, first_subscription_price,
		first_subscription_revenue, first_bonus_points, subscription_price,
		subscription_revenue, subscription_points, v_ip_duration_days, status
	) VALUES (1, 'vip.monthly', 'VIP Monthly', 2, 'USD', 4.99, 3.50, 20, 9.99, 7.00, 100, 30, 1)`).Error; err != nil {
		t.Fatal(err)
	}
	signed := appleSignedTransaction{
		TransactionID: "tx-initial", OriginalTransactionID: "tx-initial",
		BundleID: "com.example.app", ProductID: "vip.monthly",
		PurchaseDate: now.UnixMilli(), ExpiresDate: expires.UnixMilli(),
		Quantity: 1, Type: "Auto-Renewable Subscription", Environment: "Sandbox",
		Price: 4990, Currency: "USD", TransactionReason: "PURCHASE", SignedDate: now.UnixMilli(),
	}
	evidence := signer.sign(t, signed)
	service := NewService()
	service.appleRootCAs = signer.roots

	result, err := service.ConfirmApplePurchase(context.Background(), 1, "com.example.app", ApplePurchaseRequest{
		ShopType: 1, BundleID: "com.example.app", ExpirationDate: &expires, IsActive: true,
		OriginalTransactionID: "tx-initial", ProductID: "vip.monthly", PurchaseDate: now,
		SignedTransactionInfo: evidence, TransactionID: "tx-initial",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != domain.OrderStatusEnd || result.TransactionID != "tx-initial" || result.PaidAmount != 4.99 {
		t.Fatalf("unexpected initial purchase response: %#v", result)
	}
	initialOrder, err := service.orders.GetByOrderNo(context.Background(), result.OrderNo, false)
	if err != nil {
		t.Fatal(err)
	}
	if initialOrder.OrderType != domain.OrderTypeNewPurchase {
		t.Fatalf("initial order_type=%d, want %d", initialOrder.OrderType, domain.OrderTypeNewPurchase)
	}
	var user struct {
		SubscriptionStatus       uint8
		PaymentCount             int64
		SubscriptionPaymentCount int64
		VipPoints                uint64
		VipExpiresAt             time.Time
		ActualAmountMoney        float64
	}
	if err := db.Table("video_user").Where("id = 1").Scan(&user).Error; err != nil {
		t.Fatal(err)
	}
	if user.SubscriptionStatus != domain.AppUserSubscriptionSubscribed || user.PaymentCount != 1 ||
		user.SubscriptionPaymentCount != 1 || user.VipPoints != 20 ||
		!user.VipExpiresAt.Equal(expires) || user.ActualAmountMoney != 3.5 {
		t.Fatalf("initial subscription was not fulfilled: %#v", user)
	}
}

func TestAppleRenewalCreatesOneCompletedOrder(t *testing.T) {
	db := newAppleCommerceTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Millisecond)
	oldExpires := now.Add(24 * time.Hour)
	newExpires := now.Add(31 * 24 * time.Hour)

	if err := db.Exec(`INSERT INTO video_user (
		id, device_code, user_type, subscription_status, vip_started_at, vip_expires_at,
		vip_points, points_balance, payment_count, subscription_payment_count,
		one_time_payment_count, order_count, actual_amount_money, order_amount_money,
		first_payment_met, payment_met, vip_level, created_at, updated_at
	) VALUES (?, 'device-1', 2, 2, ?, ?, 10, 0, 1, 1, 0, 1, 5, 9.99, 1, 1, 2, ?, ?)`,
		1, now.Add(-30*24*time.Hour), oldExpires, now, now).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO video_vip_subscription (
		id, suk_code, name, level_id, currency, first_subscription_price,
		first_subscription_revenue, first_bonus_points, subscription_price,
		subscription_revenue, subscription_points, v_ip_duration_days, status
	) VALUES (1, 'vip.monthly', 'VIP Monthly', 2, 'USD', 4.99, 3.50, 20, 9.99, 7.00, 100, 30, 1)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO video_order (
		order_no, client_request_id, user_id, product_type, product_id, product_code,
		product_name, currency, product_amount, discount_amount, payable_amount,
		paid_amount, actual_amount_money, refunded_amount, bonus_points, vip_level,
		vip_duration_days, status, pay_type, third_order_no,
		original_transaction_id, payment_evidence, pay_time, completed_at, expires_at,
		created_at, updated_at
	) VALUES (
		'initial-order', ?, 1, 1, 1, 'vip.monthly', 'VIP Monthly', 'USD',
		4.99, 0, 4.99, 4.99, 3.50, 0, 20, 2, 30, 4, ?,
		'tx-original', 'tx-original', 'initial-evidence', ?, ?, ?, ?, ?
	)`, appleClientRequestID("tx-original"), domain.PaymentMethodAppleIAP, now.Add(-30*24*time.Hour), now.Add(-30*24*time.Hour),
		now.Add(-30*24*time.Hour), now.Add(-30*24*time.Hour), now).Error; err != nil {
		t.Fatal(err)
	}

	service := NewService()
	base, err := service.orders.GetByOrderNo(ctx, "initial-order", false)
	if err != nil {
		t.Fatal(err)
	}
	decoded := &DecodedAppleNotificationV2{
		NotificationType: AppleNotificationDidRenew,
		BundleID:         "com.example.app", Environment: "Sandbox",
		TransactionID: "tx-renewal", OriginalTransaction: "tx-original",
		ProductID: "vip.monthly", PurchaseDate: now.UnixMilli(),
		ExpiresDate: newExpires.UnixMilli(), Price: 9990, Currency: "USD",
		SignedTransaction: "verified-renewal-evidence",
	}
	completed, err := service.processAppleSubscriptionTransaction(ctx, base, decoded, true)
	if err != nil {
		t.Fatal(err)
	}
	if completed.Status != domain.OrderStatusEnd || completed.ThirdOrderNo != "tx-renewal" ||
		completed.OriginalTransactionID != "tx-original" || completed.PaidAmount != 9.99 ||
		completed.ActualAmountMoney != 7 || completed.BonusPoints != 100 ||
		completed.OrderType != domain.OrderTypeRenewal {
		t.Fatalf("unexpected renewal order: %#v", completed)
	}
	assertAppleRenewalState(t, db, newExpires, 2, 2, 110, 1)

	// Apple's delivery is at-least-once. Processing the same transaction again
	// must not create another order, grant points, or increment counters.
	repeated, err := service.processAppleSubscriptionTransaction(ctx, completed, decoded, true)
	if err != nil {
		t.Fatal(err)
	}
	if repeated.ID != completed.ID {
		t.Fatalf("duplicate notification returned order %d, want %d", repeated.ID, completed.ID)
	}
	assertAppleRenewalState(t, db, newExpires, 2, 2, 110, 1)

	if err := service.reflectAppleRenewalStatusChange(ctx, completed, AppleSubtypeAutoRenewDisabled, newExpires.UnixMilli()); err != nil {
		t.Fatal(err)
	}
	var cancelled struct {
		SubscriptionStatus uint8
		VipExpiresAt       time.Time
	}
	if err := db.Table("video_user").Select("subscription_status, vip_expires_at").Where("id = 1").Scan(&cancelled).Error; err != nil {
		t.Fatal(err)
	}
	if cancelled.SubscriptionStatus != domain.AppUserSubscriptionCancelled ||
		!cancelled.VipExpiresAt.Equal(newExpires) {
		t.Fatalf("cancellation changed current entitlement: %#v", cancelled)
	}
	if err := db.Table("video_user").Where("id = 1").Update("subscription_status", domain.AppUserSubscriptionSubscribed).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.reflectAppleRenewalStatusChange(ctx, base, AppleSubtypeAutoRenewDisabled, oldExpires.UnixMilli()); err != nil {
		t.Fatal(err)
	}
	var currentStatus uint8
	if err := db.Table("video_user").Where("id = 1").Pluck("subscription_status", &currentStatus).Error; err != nil {
		t.Fatal(err)
	}
	if currentStatus != domain.AppUserSubscriptionSubscribed {
		t.Fatalf("stale cancellation overwrote newer renewal: status=%d", currentStatus)
	}
}

func TestApplyVIPEntitlementDoesNotUndoLaterCancellation(t *testing.T) {
	now := time.Now()
	expires := now.Add(30 * 24 * time.Hour)
	user := &model.VideoUser{
		SubscriptionStatus: domain.AppUserSubscriptionCancelled,
		VipExpiresAt:       &expires,
	}
	updates := map[string]any{}
	applyVIPEntitlement(user, &model.VideoOrder{VipLevel: 2}, now, &expires, updates)
	if _, ok := updates["subscription_status"]; ok {
		t.Fatalf("duplicate renewal overwrote cancellation: %#v", updates)
	}
}

func assertAppleRenewalState(
	t *testing.T,
	db *gorm.DB,
	wantExpires time.Time,
	wantOrders, wantPayments int64,
	wantVIPPoints uint64,
	wantLedgers int64,
) {
	t.Helper()
	var orderCount, ledgerCount int64
	if err := db.Table("video_order").Count(&orderCount).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Table("video_user_points_ledger").Count(&ledgerCount).Error; err != nil {
		t.Fatal(err)
	}
	if ledgerCount > 0 {
		var sourceType uint32
		if err := db.Table("video_user_points_ledger").Order("id DESC").Pluck("source_type", &sourceType).Error; err != nil {
			t.Fatal(err)
		}
		if sourceType != domain.PointsSourceSubscriptionGift {
			t.Fatalf("renewal ledger source_type=%d, want %d", sourceType, domain.PointsSourceSubscriptionGift)
		}
	}
	var user struct {
		SubscriptionStatus       uint8
		PaymentCount             int64
		SubscriptionPaymentCount int64
		VipPoints                uint64
		VipExpiresAt             time.Time
	}
	if err := db.Table("video_user").Where("id = 1").Scan(&user).Error; err != nil {
		t.Fatal(err)
	}
	if orderCount != wantOrders || ledgerCount != wantLedgers || user.PaymentCount != wantPayments ||
		user.SubscriptionPaymentCount != wantPayments || user.VipPoints != wantVIPPoints ||
		user.SubscriptionStatus != domain.AppUserSubscriptionSubscribed ||
		!user.VipExpiresAt.Equal(wantExpires) {
		t.Fatalf("unexpected renewal state: orders=%d ledgers=%d user=%#v", orderCount, ledgerCount, user)
	}
}

func newAppleCommerceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + strings.NewReplacer("/", "-", "\\", "-", " ", "-").Replace(t.Name()) + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	statements := []string{
		`CREATE TABLE video_user (
			id INTEGER PRIMARY KEY, device_code TEXT NOT NULL, user_type INTEGER NOT NULL,
			subscription_status INTEGER NOT NULL, vip_started_at DATETIME NULL,
			vip_expires_at DATETIME NULL, vip_points INTEGER NOT NULL DEFAULT 0,
			points_balance INTEGER NOT NULL DEFAULT 0, payment_count INTEGER NOT NULL DEFAULT 0,
			subscription_payment_count INTEGER NOT NULL DEFAULT 0,
			one_time_payment_count INTEGER NOT NULL DEFAULT 0, order_count INTEGER NOT NULL DEFAULT 0,
			actual_amount_money REAL NOT NULL DEFAULT 0, order_amount_money REAL NOT NULL DEFAULT 0,
			refund_amount_money REAL NOT NULL DEFAULT 0, first_order_created_at DATETIME NULL,
			first_paid_at DATETIME NULL, last_paid_at DATETIME NULL,
			first_payment_met INTEGER NOT NULL DEFAULT 0, payment_met INTEGER NOT NULL DEFAULT 0,
			vip_level INTEGER NOT NULL DEFAULT 0, created_at DATETIME NOT NULL, updated_at DATETIME NOT NULL,
			deleted_at DATETIME NULL
		)`,
		`CREATE TABLE video_vip_subscription (
			id INTEGER PRIMARY KEY, suk_code TEXT NOT NULL, name TEXT NOT NULL, level_id INTEGER NOT NULL,
			currency TEXT NOT NULL, first_subscription_price REAL NOT NULL DEFAULT 0,
			first_subscription_revenue REAL NOT NULL DEFAULT 0, first_bonus_points INTEGER NOT NULL DEFAULT 0,
			subscription_price REAL NOT NULL DEFAULT 0, subscription_revenue REAL NOT NULL DEFAULT 0,
			subscription_points INTEGER NOT NULL DEFAULT 0, v_ip_duration_days INTEGER NOT NULL DEFAULT 0,
			status INTEGER NOT NULL DEFAULT 1, deleted_at DATETIME NULL
		)`,
		`CREATE TABLE video_order (
			id INTEGER PRIMARY KEY AUTOINCREMENT, order_no TEXT NOT NULL UNIQUE,
			client_request_id TEXT NOT NULL UNIQUE, user_id INTEGER NOT NULL,
			product_type INTEGER NOT NULL, product_id INTEGER NOT NULL, product_code TEXT NOT NULL,
			product_name TEXT NOT NULL, currency TEXT NOT NULL, product_amount REAL NOT NULL DEFAULT 0,
			discount_amount REAL NOT NULL DEFAULT 0, payable_amount REAL NOT NULL DEFAULT 0,
			paid_amount REAL NOT NULL DEFAULT 0, actual_amount_money REAL NOT NULL DEFAULT 0,
			refunded_amount REAL NOT NULL DEFAULT 0, bonus_points INTEGER NOT NULL DEFAULT 0,
			vip_level INTEGER NOT NULL DEFAULT 0, vip_duration_days INTEGER NOT NULL DEFAULT 0,
			status INTEGER NOT NULL, pay_type INTEGER NOT NULL, third_order_no TEXT NULL,
			original_transaction_id TEXT NULL, payment_evidence TEXT NULL, failure_code TEXT NULL,
			failure_message TEXT NULL, cancel_reason TEXT NULL, pay_time DATETIME NULL,
			completed_at DATETIME NULL, cancelled_at DATETIME NULL, expires_at DATETIME NULL,
			order_type INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL, updated_at DATETIME NOT NULL, deleted_at DATETIME NULL,
			UNIQUE(pay_type, third_order_no)
		)`,
		`CREATE TABLE video_user_points_ledger (
			id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL, direction INTEGER NOT NULL,
			points_change INTEGER NOT NULL, balance_before INTEGER NOT NULL, balance_after INTEGER NOT NULL,
			description TEXT NULL, source_type INTEGER NOT NULL, order_code TEXT NULL,
			points_id INTEGER NOT NULL DEFAULT 0, vip_id INTEGER NOT NULL DEFAULT 0,
			occurred_at DATETIME NOT NULL, admin_id INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL, updated_at DATETIME NULL, deleted_at DATETIME NULL
		)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(fmt.Errorf("create isolated test schema: %w", err))
		}
	}
	previousDB := config.DB
	config.DB = db
	t.Cleanup(func() { config.DB = previousDB })
	return db
}
