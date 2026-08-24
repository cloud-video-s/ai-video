package commerce

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"ai-video/internal/adjustevent"
	"ai-video/internal/config"
	"ai-video/internal/domain"
	"ai-video/internal/gen/model"
	"ai-video/internal/pkg/adjust"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type capturedAdjustBusinessEvent struct {
	userID  uint64
	action  adjust.EventToken
	options adjustevent.EnqueueOptions
}

func TestPointsPaymentCompletionCreditsOnlyPurchasedPointsAndReportsPayment(t *testing.T) {
	db := newPointsPaymentTestDB(t)
	user, order := seedPointsPaymentOrder(t, db, domain.OrderStatusPaid)
	events := captureAdjustBusinessEvents(t)

	service := NewService()
	if err := service.fulfillAppleOrder(context.Background(), order, nil); err != nil {
		t.Fatal(err)
	}

	var updatedUser model.VideoUser
	if err := db.First(&updatedUser, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if updatedUser.PointsBalance != 140 {
		t.Fatalf("points_balance = %d, want 140", updatedUser.PointsBalance)
	}
	if updatedUser.VipPoints != 25 {
		t.Fatalf("vip_points = %d, want unchanged value 25", updatedUser.VipPoints)
	}

	var ledgers []model.VideoUserPointsLedger
	if err := db.Find(&ledgers).Error; err != nil {
		t.Fatal(err)
	}
	if len(ledgers) != 1 {
		t.Fatalf("points ledgers = %d, want 1", len(ledgers))
	}
	ledger := ledgers[0]
	if ledger.PointsChange != 40 || ledger.BalanceBefore != 100 || ledger.BalanceAfter != 140 {
		t.Fatalf("points ledger balances = change %d, before %d, after %d", ledger.PointsChange, ledger.BalanceBefore, ledger.BalanceAfter)
	}
	if ledger.SourceType != uint32(domain.PointsSourcePurchase) || ledger.PointsID != order.ProductID ||
		ledger.VipID != 0 || ledger.OrderCode != order.OrderNo {
		t.Fatalf("points ledger association = %#v", ledger)
	}

	var completed model.VideoOrder
	if err := db.First(&completed, order.ID).Error; err != nil {
		t.Fatal(err)
	}
	if completed.Status != domain.OrderStatusEnd || completed.CompletedAt.IsZero() {
		t.Fatalf("completed order status = %d, completed_at = %v", completed.Status, completed.CompletedAt)
	}
	if len(*events) != 1 || (*events)[0].userID != user.ID ||
		(*events)[0].action != adjust.EventTokenPayment || (*events)[0].options.OrderNo != order.OrderNo ||
		!(*events)[0].options.OccurredAt.Equal(order.PayTime) {
		t.Fatalf("Adjust events = %#v, want one payment event", *events)
	}

	// A repeated callback sees the completed order under the row lock and must
	// not duplicate either the balance change, ledger, or Adjust event.
	if err := service.fulfillAppleOrder(context.Background(), order, nil); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&updatedUser, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if updatedUser.PointsBalance != 140 {
		t.Fatalf("points_balance after repeated callback = %d, want 140", updatedUser.PointsBalance)
	}
	var ledgerCount int64
	if err := db.Model(&model.VideoUserPointsLedger{}).Count(&ledgerCount).Error; err != nil {
		t.Fatal(err)
	}
	if ledgerCount != 1 || len(*events) != 1 {
		t.Fatalf("repeated callback produced %d ledgers and %d events", ledgerCount, len(*events))
	}
}

func TestAppleOneTimeChargeCompletesPointsOrder(t *testing.T) {
	db := newPointsPaymentTestDB(t)
	user, order := seedPointsPaymentOrder(t, db, domain.OrderStatusPending)
	events := captureAdjustBusinessEvents(t)
	transactionID := "apple-one-time-charge-1"
	requestID := appleClientRequestID(transactionID)
	if err := db.Model(&model.VideoOrder{}).Where("id = ?", order.ID).
		Update("client_request_id", requestID).Error; err != nil {
		t.Fatal(err)
	}

	decoded := &DecodedAppleNotificationV2{
		NotificationType:    AppleNotificationOneTimeCharge,
		TransactionID:       transactionID,
		OriginalTransaction: transactionID,
		ProductID:           order.ProductCode,
		PurchaseDate:        time.Now().Truncate(time.Second).UnixMilli(),
		Price:               1990,
		Currency:            "usd",
		SignedTransaction:   "verified-signed-transaction",
	}
	service := NewService()
	found, err := service.findAppleNotificationV2Order(context.Background(), decoded)
	if err != nil {
		t.Fatal(err)
	}
	if found == nil || found.ID != order.ID {
		t.Fatalf("findAppleNotificationV2Order() = %#v, want order %d", found, order.ID)
	}
	completed, err := service.processAppleOneTimeChargeTransaction(context.Background(), found, decoded)
	if err != nil {
		t.Fatal(err)
	}
	if completed == nil || completed.Status != domain.OrderStatusEnd {
		t.Fatalf("completed order = %#v, want status %d", completed, domain.OrderStatusEnd)
	}
	if completed.ThirdOrderNo != transactionID || completed.OriginalTransactionID != transactionID ||
		completed.PaymentEvidence != decoded.SignedTransaction {
		t.Fatalf("completed Apple transaction fields = %#v", completed)
	}

	var updatedUser model.VideoUser
	if err := db.First(&updatedUser, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if updatedUser.PointsBalance != 140 || updatedUser.VipPoints != 25 ||
		updatedUser.OneTimePaymentCount != 1 {
		t.Fatalf("one-time charge balances/counter = points %d, vip %d, count %d",
			updatedUser.PointsBalance, updatedUser.VipPoints, updatedUser.OneTimePaymentCount)
	}
	var ledgerCount int64
	if err := db.Model(&model.VideoUserPointsLedger{}).Count(&ledgerCount).Error; err != nil {
		t.Fatal(err)
	}
	if ledgerCount != 1 || len(*events) != 1 {
		t.Fatalf("one-time charge produced %d ledgers and %d events, want one each", ledgerCount, len(*events))
	}

	// Apple retries must resolve the stored transaction and observe the
	// completed order without crediting the points a second time.
	found, err = service.findAppleNotificationV2Order(context.Background(), decoded)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.processAppleOneTimeChargeTransaction(context.Background(), found, decoded); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&updatedUser, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.VideoUserPointsLedger{}).Count(&ledgerCount).Error; err != nil {
		t.Fatal(err)
	}
	if updatedUser.PointsBalance != 140 || ledgerCount != 1 || len(*events) != 1 {
		t.Fatalf("repeated one-time charge changed state: points=%d ledgers=%d events=%d",
			updatedUser.PointsBalance, ledgerCount, len(*events))
	}
}

func TestPointsPaymentCompletionRollsBackWhenLedgerWriteFails(t *testing.T) {
	db := newPointsPaymentTestDB(t)
	user, order := seedPointsPaymentOrder(t, db, domain.OrderStatusPaid)
	events := captureAdjustBusinessEvents(t)
	if err := db.Exec(`CREATE TRIGGER reject_points_ledger
		BEFORE INSERT ON video_user_points_ledger
		BEGIN SELECT RAISE(ABORT, 'forced ledger failure'); END`).Error; err != nil {
		t.Fatal(err)
	}

	err := NewService().fulfillAppleOrder(context.Background(), order, nil)
	if err == nil {
		t.Fatal("fulfillAppleOrder() succeeded, want ledger failure")
	}

	var updatedUser model.VideoUser
	if err := db.First(&updatedUser, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if updatedUser.PointsBalance != 100 || updatedUser.VipPoints != 25 || updatedUser.PaymentCount != 0 {
		t.Fatalf("user changed after rollback: points=%d vip=%d payments=%d", updatedUser.PointsBalance, updatedUser.VipPoints, updatedUser.PaymentCount)
	}
	var ledgerCount int64
	if err := db.Model(&model.VideoUserPointsLedger{}).Count(&ledgerCount).Error; err != nil {
		t.Fatal(err)
	}
	var storedOrder model.VideoOrder
	if err := db.First(&storedOrder, order.ID).Error; err != nil {
		t.Fatal(err)
	}
	if ledgerCount != 0 || storedOrder.Status != domain.OrderStatusPaid || len(*events) != 0 {
		t.Fatalf("rollback result: ledgers=%d order_status=%d events=%d", ledgerCount, storedOrder.Status, len(*events))
	}
}

func TestFailedPointsPaymentCallbackDoesNotCreditOrReport(t *testing.T) {
	db := newPointsPaymentTestDB(t)
	user, order := seedPointsPaymentOrder(t, db, domain.OrderStatusPending)
	events := captureAdjustBusinessEvents(t)

	_, err := NewService().ConfirmApplePayment(context.Background(), order.OrderNo, ApplePaymentResult{
		TransactionID: "apple-transaction-failed", ProductCode: "another-product",
		Currency: "USD", PaidAmount: order.PayableAmount, PurchaseDate: time.Now(),
	})
	if !errors.Is(err, ErrPaymentMismatch) {
		t.Fatalf("ConfirmApplePayment() error = %v, want ErrPaymentMismatch", err)
	}

	var updatedUser model.VideoUser
	if err := db.First(&updatedUser, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	var storedOrder model.VideoOrder
	if err := db.First(&storedOrder, order.ID).Error; err != nil {
		t.Fatal(err)
	}
	var ledgerCount int64
	if err := db.Model(&model.VideoUserPointsLedger{}).Count(&ledgerCount).Error; err != nil {
		t.Fatal(err)
	}
	if updatedUser.PointsBalance != 100 || updatedUser.VipPoints != 25 || ledgerCount != 0 || len(*events) != 0 {
		t.Fatalf("failed callback changed benefits: points=%d vip=%d ledgers=%d events=%d", updatedUser.PointsBalance, updatedUser.VipPoints, ledgerCount, len(*events))
	}
	if storedOrder.Status != domain.OrderStatusCancelled {
		t.Fatalf("failed callback order status = %d, want cancelled", storedOrder.Status)
	}
}

func newPointsPaymentTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:points-payment-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	// This deliberately minimal schema exists only in the isolated in-memory
	// test database. The generated models use MySQL-specific unsigned types, so
	// SQLite AutoMigrate is not suitable here.
	statements := []string{
		`CREATE TABLE video_user (
			id INTEGER PRIMARY KEY, username TEXT NOT NULL DEFAULT '',
			vip_points INTEGER NOT NULL DEFAULT 0, points_balance INTEGER NOT NULL DEFAULT 0,
			payment_count INTEGER NOT NULL DEFAULT 0,
			subscription_payment_count INTEGER NOT NULL DEFAULT 0,
			one_time_payment_count INTEGER NOT NULL DEFAULT 0,
			order_amount_money REAL NOT NULL DEFAULT 0,
			actual_amount_money REAL NOT NULL DEFAULT 0,
			first_paid_at DATETIME, last_paid_at DATETIME,
			payment_met INTEGER NOT NULL DEFAULT 0, first_payment_met INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE video_order (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			order_no TEXT NOT NULL, client_request_id TEXT NOT NULL,
			user_id INTEGER NOT NULL, product_type INTEGER NOT NULL, product_id INTEGER NOT NULL,
			product_code TEXT NOT NULL, product_name TEXT NOT NULL, currency TEXT NOT NULL,
			product_amount REAL NOT NULL DEFAULT 0, payable_amount REAL NOT NULL DEFAULT 0,
			paid_amount REAL NOT NULL DEFAULT 0, actual_amount_money REAL NOT NULL DEFAULT 0,
			bonus_points INTEGER NOT NULL DEFAULT 0, vip_level INTEGER NOT NULL DEFAULT 0,
			vip_duration_days INTEGER NOT NULL DEFAULT 0, status INTEGER NOT NULL,
			pay_type INTEGER NOT NULL, pay_time DATETIME, pay_channel INTEGER NOT NULL DEFAULT 1,
			third_order_no TEXT, original_transaction_id TEXT, payment_evidence TEXT,
			cancel_reason TEXT, cancelled_at DATETIME, completed_at DATETIME, expires_at DATETIME,
			order_type INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE video_user_points_ledger (
			id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL,
			direction INTEGER NOT NULL, points_change INTEGER NOT NULL,
			balance_before INTEGER NOT NULL, balance_after INTEGER NOT NULL,
			description TEXT, source_type INTEGER NOT NULL,
			order_code TEXT, points_id INTEGER NOT NULL DEFAULT 0, vip_id INTEGER NOT NULL DEFAULT 0,
			occurred_at DATETIME NOT NULL, admin_id INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL, updated_at DATETIME, deleted_at DATETIME
		)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	previousDB := config.DB
	config.DB = db
	t.Cleanup(func() { config.DB = previousDB })
	return db
}

func seedPointsPaymentOrder(t *testing.T, db *gorm.DB, status uint32) (*model.VideoUser, *model.VideoOrder) {
	t.Helper()
	if err := db.Exec(`INSERT INTO video_user (
		id, username, points_balance, vip_points, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?)`, 77, "points-buyer", 100, 25, time.Now(), time.Now()).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now().Truncate(time.Second)
	if err := db.Exec(`INSERT INTO video_order (
		order_no, client_request_id, user_id, product_type, product_id,
		product_code, product_name, currency, product_amount, payable_amount,
		paid_amount, actual_amount_money, bonus_points, status, pay_type,
		pay_time, expires_at, order_type, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"points-order-1", "points-request-1", 77, domain.OrderProductPointsPackage, 9,
		"points-40", "40 points", "USD", 1.99, 1.99, 1.99, 1.5, 40, status,
		domain.PaymentMethodAppleIAP, now, now.Add(time.Hour), domain.OrderTypeNewPurchase, now, now,
	).Error; err != nil {
		t.Fatal(err)
	}
	user := &model.VideoUser{}
	if err := db.First(user, 77).Error; err != nil {
		t.Fatal(err)
	}
	order := &model.VideoOrder{}
	if err := db.Where("order_no = ?", "points-order-1").First(order).Error; err != nil {
		t.Fatal(err)
	}
	return user, order
}

func captureAdjustBusinessEvents(t *testing.T) *[]capturedAdjustBusinessEvent {
	t.Helper()
	previous := enqueueAdjustBusinessEvent
	events := make([]capturedAdjustBusinessEvent, 0, 1)
	enqueueAdjustBusinessEvent = func(_ context.Context, userID uint64, action adjust.EventToken, options adjustevent.EnqueueOptions) error {
		events = append(events, capturedAdjustBusinessEvent{userID: userID, action: action, options: options})
		return nil
	}
	t.Cleanup(func() { enqueueAdjustBusinessEvent = previous })
	return &events
}
