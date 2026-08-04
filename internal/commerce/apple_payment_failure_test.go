package commerce

import (
	"context"
	"errors"
	"testing"
	"time"

	"ai-video/internal/config"
	"ai-video/internal/domain"
	"ai-video/internal/gen/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestApplePaymentFailureCancelsPendingOrder(t *testing.T) {
	db := appleFailureTestDB(t)
	service := NewService()
	ctx := context.Background()
	order, err := service.CreateOrder(ctx, CreateOrderRequest{
		UserID: 1, ProductType: domain.OrderProductPointsPackage, ProductID: 10,
		PaymentMethod: domain.PaymentMethodAppleIAP, ClientRequestID: "request-payment-failure",
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := service.ConfirmApplePayment(ctx, order.OrderNo, ApplePaymentResult{}); err == nil {
		t.Fatal("expected invalid Apple payment result")
	}
	var cancelled model.VideoOrder
	if err := db.First(&cancelled, order.ID).Error; err != nil {
		t.Fatal(err)
	}
	if cancelled.Status != domain.OrderStatusCancelled {
		t.Fatalf("status=%s, want %s", cancelled.Status, domain.OrderStatusCancelled)
	}
	if cancelled.CancelReason != "Apple payment confirmation failed" || cancelled.CancelledAt.IsZero() {
		t.Fatalf("unexpected cancellation metadata: %#v", cancelled)
	}
}

func TestAppleFailureNotificationCancelsPendingOrder(t *testing.T) {
	db := appleFailureTestDB(t)
	service := NewService()
	signer := newAppleJWSTestSigner(t)
	service.appleRootCAs = signer.roots
	ctx := context.Background()
	transactionID := "apple-failed-renewal-1"
	order, err := service.CreateOrder(ctx, CreateOrderRequest{
		UserID: 1, ProductType: domain.OrderProductPointsPackage, ProductID: 10,
		PaymentMethod:   domain.PaymentMethodAppleIAP,
		ClientRequestID: appleClientRequestID(transactionID),
	})
	if err != nil {
		t.Fatal(err)
	}

	signedDate := signer.now.UnixMilli()
	signedTransaction := signer.sign(t, map[string]any{
		"transactionId": transactionID, "originalTransactionId": "apple-original-1",
		"bundleId": "com.example.video", "productId": "points.small",
		"environment": "Sandbox", "signedDate": signedDate,
	})
	signedPayload := signer.sign(t, map[string]any{
		"notificationType": AppleNotificationDidFailToRenew,
		"notificationUUID": "notification-failure-1",
		"version":          "2.0", "signedDate": signedDate,
		"data": map[string]any{
			"appAppleId": int64(123456789), "bundleId": "com.example.video", "environment": "Sandbox",
			"signedTransactionInfo": signedTransaction,
		},
	})
	summary, err := service.HandleAppleServerNotification(ctx, signedPayload)
	if err != nil {
		t.Fatal(err)
	}
	if !summary.Processed || summary.Action != "cancel_failed_payment" || summary.AffectedOrderNo != order.OrderNo {
		t.Fatalf("unexpected summary: %#v", summary)
	}

	var cancelled model.VideoOrder
	if err := db.First(&cancelled, order.ID).Error; err != nil {
		t.Fatal(err)
	}
	if cancelled.Status != domain.OrderStatusCancelled || cancelled.CancelReason != "Apple DID_FAIL_TO_RENEW notification" {
		t.Fatalf("unexpected cancelled order: %#v", cancelled)
	}

	second, err := service.HandleAppleServerNotification(ctx, signedPayload)
	if err != nil || !second.Processed || second.Message != "pending Apple order was already cancelled" {
		t.Fatalf("idempotent callback summary=%#v, err=%v", second, err)
	}
}

func TestAppleFailureNotificationDoesNotCancelPaidOrder(t *testing.T) {
	db := appleFailureTestDB(t)
	service := NewService()
	signer := newAppleJWSTestSigner(t)
	service.appleRootCAs = signer.roots
	ctx := context.Background()
	transactionID := "apple-already-paid-1"
	order, err := service.CreateOrder(ctx, CreateOrderRequest{
		UserID: 1, ProductType: domain.OrderProductPointsPackage, ProductID: 10,
		PaymentMethod: domain.PaymentMethodAppleIAP, ClientRequestID: "request-already-paid",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.VideoOrder{}).Where("id = ?", order.ID).Updates(map[string]any{
		"status": domain.OrderStatusPaid, "third_order_no": transactionID,
	}).Error; err != nil {
		t.Fatal(err)
	}

	signedDate := signer.now.UnixMilli()
	signedTransaction := signer.sign(t, map[string]any{
		"transactionId": transactionID, "originalTransactionId": transactionID,
		"bundleId": "com.example.video", "productId": "points.small",
		"environment": "Sandbox", "signedDate": signedDate,
	})
	signedPayload := signer.sign(t, map[string]any{
		"notificationType": AppleNotificationDidFailToRenew,
		"notificationUUID": "notification-paid-1", "version": "2.0", "signedDate": signedDate,
		"data": map[string]any{
			"appAppleId": int64(123456789), "bundleId": "com.example.video", "environment": "Sandbox",
			"signedTransactionInfo": signedTransaction,
		},
	})
	summary, err := service.HandleAppleServerNotification(ctx, signedPayload)
	if err != nil {
		t.Fatal(err)
	}
	if !summary.Processed || summary.Message != "matching Apple order is not pending; left unchanged" {
		t.Fatalf("unexpected summary: %#v", summary)
	}
	var unchanged model.VideoOrder
	if err := db.First(&unchanged, order.ID).Error; err != nil {
		t.Fatal(err)
	}
	if unchanged.Status != domain.OrderStatusPaid || !unchanged.CancelledAt.IsZero() {
		t.Fatalf("paid order was changed: %#v", unchanged)
	}
}

func TestAppleRefundReversedDoesNotRevokePaidOrder(t *testing.T) {
	db := appleFailureTestDB(t)
	service := NewService()
	signer := newAppleJWSTestSigner(t)
	service.appleRootCAs = signer.roots
	ctx := context.Background()
	transactionID := "apple-refund-reversed-1"
	order, err := service.CreateOrder(ctx, CreateOrderRequest{
		UserID: 1, ProductType: domain.OrderProductPointsPackage, ProductID: 10,
		PaymentMethod: domain.PaymentMethodAppleIAP, ClientRequestID: "request-refund-reversed",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.VideoOrder{}).Where("id = ?", order.ID).Updates(map[string]any{
		"status": domain.OrderStatusPaid, "third_order_no": transactionID,
		"original_transaction_id": transactionID,
	}).Error; err != nil {
		t.Fatal(err)
	}

	signedDate := signer.now.UnixMilli()
	signedTransaction := signer.sign(t, map[string]any{
		"transactionId": transactionID, "originalTransactionId": transactionID,
		"bundleId": "com.example.video", "productId": "points.small",
		"environment": "Sandbox", "signedDate": signedDate,
	})
	signedPayload := signer.sign(t, map[string]any{
		"notificationType": AppleNotificationRefundReversed,
		"notificationUUID": "notification-refund-reversed-1", "version": "2.0", "signedDate": signedDate,
		"data": map[string]any{
			"appAppleId": int64(123456789), "bundleId": "com.example.video", "environment": "Sandbox",
			"signedTransactionInfo": signedTransaction,
		},
	})
	summary, err := service.HandleAppleServerNotification(ctx, signedPayload)
	if err != nil {
		t.Fatal(err)
	}
	if !summary.Processed || summary.Action != "refund_reversed" {
		t.Fatalf("unexpected summary: %#v", summary)
	}
	var unchanged model.VideoOrder
	if err := db.First(&unchanged, order.ID).Error; err != nil {
		t.Fatal(err)
	}
	if unchanged.Status != domain.OrderStatusPaid {
		t.Fatalf("refund reversal revoked a paid order: %#v", unchanged)
	}
}

func TestAppleRenewalNotificationIsIdempotent(t *testing.T) {
	db := appleFailureTestDB(t)
	service := NewService()
	signer := newAppleJWSTestSigner(t)
	service.appleRootCAs = signer.roots
	ctx := context.Background()
	transactionID := "apple-renewal-1"
	order, err := service.CreateOrder(ctx, CreateOrderRequest{
		UserID: 1, ProductType: domain.OrderProductPointsPackage, ProductID: 10,
		PaymentMethod: domain.PaymentMethodAppleIAP, ClientRequestID: "request-renewal",
	})
	if err != nil {
		t.Fatal(err)
	}
	oldExpiry := signer.now.Add(30 * time.Minute)
	newExpiry := signer.now.Add(time.Hour)
	if err := db.Model(&model.VideoOrder{}).Where("id = ?", order.ID).Updates(map[string]any{
		"status": domain.OrderStatusPaid, "product_type": domain.OrderProductVIPSubscription,
		"third_order_no": transactionID, "original_transaction_id": transactionID,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.VideoUser{}).Where("id = ?", 1).Updates(map[string]any{
		"vip_expires_at": oldExpiry, "subscription_payment_count": 0,
	}).Error; err != nil {
		t.Fatal(err)
	}

	signedDate := signer.now.UnixMilli()
	signedTransaction := signer.sign(t, map[string]any{
		"transactionId": transactionID, "originalTransactionId": transactionID,
		"bundleId": "com.example.video", "productId": "vip.monthly",
		"expiresDate": newExpiry.UnixMilli(), "environment": "Sandbox", "signedDate": signedDate,
	})
	signedPayload := signer.sign(t, map[string]any{
		"notificationType": AppleNotificationDidRenew,
		"notificationUUID": "notification-renewal-1", "version": "2.0", "signedDate": signedDate,
		"data": map[string]any{
			"appAppleId": int64(123456789), "bundleId": "com.example.video", "environment": "Sandbox",
			"signedTransactionInfo": signedTransaction,
		},
	})
	for i := 0; i < 2; i++ {
		if _, err := service.HandleAppleServerNotification(ctx, signedPayload); err != nil {
			t.Fatal(err)
		}
	}
	var user model.VideoUser
	if err := db.First(&user, 1).Error; err != nil {
		t.Fatal(err)
	}
	if user.SubscriptionPaymentCount != 1 || user.VipExpiresAt == nil || !user.VipExpiresAt.Equal(newExpiry) {
		t.Fatalf("renewal was not idempotent: %#v", user)
	}
}

func TestAppleNotificationRejectsUnknownBundleAndMissingProductionAppID(t *testing.T) {
	appleFailureTestDB(t)
	service := NewService()
	signer := newAppleJWSTestSigner(t)
	service.appleRootCAs = signer.roots
	signedDate := signer.now.UnixMilli()

	tests := []struct {
		name   string
		data   map[string]any
		wanted error
	}{
		{
			name: "unknown bundle",
			data: map[string]any{
				"appAppleId": int64(123456789), "bundleId": "com.unknown.video", "environment": "Sandbox",
			},
			wanted: ErrAppleBundleMismatch,
		},
		{
			name: "production app ID required",
			data: map[string]any{
				"bundleId": "com.example.video", "environment": "Production",
			},
			wanted: ErrAppleEvidenceInvalid,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			payload := signer.sign(t, map[string]any{
				"notificationType": AppleNotificationTest,
				"notificationUUID": "notification-validation", "version": "2.0", "signedDate": signedDate,
				"data": test.data,
			})
			if _, err := service.HandleAppleServerNotification(context.Background(), payload); !errors.Is(err, test.wanted) {
				t.Fatalf("error=%v, want %v", err, test.wanted)
			}
		})
	}
}

func appleFailureTestDB(t *testing.T) *gorm.DB {
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
		`CREATE TABLE video_package (
			id INTEGER PRIMARY KEY, package_code TEXT NOT NULL UNIQUE,
			status INTEGER NOT NULL, system_type INTEGER NOT NULL, deleted_at DATETIME
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
			payment_method TEXT NOT NULL, third_order_no TEXT,
			original_transaction_id TEXT, payment_evidence TEXT,
			failure_code TEXT, failure_message TEXT, cancel_reason TEXT,
			paid_at DATETIME, cancelled_at DATETIME, expires_at DATETIME,
			created_at DATETIME NOT NULL, updated_at DATETIME NOT NULL, deleted_at DATETIME,
			UNIQUE (payment_method, third_order_no)
		)`,
		`INSERT INTO video_user (id) VALUES (1)`,
		`INSERT INTO video_points_package (id, product_code, name, points, currency, sale_price, status)
			VALUES (10, 'points.small', 'Small points pack', 100, 'USD', 9.99, 1)`,
		`INSERT INTO video_package (id, package_code, status, system_type)
			VALUES (1, 'com.example.video', 1, 1)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	return db
}
