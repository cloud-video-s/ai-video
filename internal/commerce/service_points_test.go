package commerce

import (
	"context"
	"testing"
	"time"

	"ai-video/internal/domain"
	"ai-video/internal/gen/model"

	"gorm.io/gorm"
)

func TestConsumePointsRecordsTotalBalanceBeforeDeduction(t *testing.T) {
	db := newAppleCommerceTestDB(t)
	if err := db.Exec(`INSERT INTO video_user (
		id, device_code, user_type, subscription_status, vip_points, points_balance, created_at, updated_at
	) VALUES (1, 'points-user', 2, 2, 30, 90, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`).Error; err != nil {
		t.Fatal(err)
	}

	ledger, err := NewService().ConsumePoints(context.Background(), ConsumePointsRequest{
		UserID: 1, Points: 50, Description: "test deduction",
	})
	if err != nil {
		t.Fatal(err)
	}
	if ledger.BalanceBefore != 120 || ledger.BalanceAfter != 70 || ledger.PointsChange != -50 ||
		ledger.Direction != int8(domain.PointsDirectionExpense) || ledger.SourceType != uint32(domain.PointsSourceModelConsume) {
		t.Fatalf("unexpected points ledger: %#v", ledger)
	}

	var user struct {
		VIPPoints     int64 `gorm:"column:vip_points"`
		PointsBalance int64 `gorm:"column:points_balance"`
	}
	if err := db.Table("video_user").Where("id = ?", 1).Take(&user).Error; err != nil {
		t.Fatal(err)
	}
	if user.VIPPoints != 0 || user.PointsBalance != 70 {
		t.Fatalf("unexpected balances after deduction: %#v", user)
	}
}

func TestRevokeVIPRecordsActuallyRemovedPoints(t *testing.T) {
	db := newAppleCommerceTestDB(t)
	if err := db.Exec(`INSERT INTO video_user (
		id, device_code, user_type, subscription_status, vip_points, points_balance, created_at, updated_at
	) VALUES (1, 'refund-user', 2, 2, 30, 90, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO video_order (
		order_no, client_request_id, user_id, product_type, product_id, product_code,
		product_name, currency, bonus_points, status, pay_type, created_at, updated_at
	) VALUES (?, ?, 1, ?, 7, 'vip.monthly', 'VIP Monthly', 'USD', 100, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		"vip-refund", "vip-refund-request", domain.OrderProductVIPSubscription,
		domain.OrderStatusPaid, domain.PaymentMethodAppleIAP).Error; err != nil {
		t.Fatal(err)
	}

	service := NewService()
	if err := service.revokePaidAppleOrder(context.Background(), &model.VideoOrder{
		OrderNo: "vip-refund", Status: domain.OrderStatusPaid,
	}, time.Now()); err != nil {
		t.Fatal(err)
	}

	ledger := latestPointsLedger(t, db)
	if ledger.PointsChange != -30 || ledger.BalanceBefore != 120 || ledger.BalanceAfter != 90 ||
		ledger.SourceType != uint32(domain.PointsSourceSubscriptionGift) || ledger.VipID != 7 || ledger.OrderCode != "vip-refund" {
		t.Fatalf("unexpected VIP revoke ledger: %#v", ledger)
	}
}

func TestExpireVIPRecordsRemainingPointsDeduction(t *testing.T) {
	db := newAppleCommerceTestDB(t)
	expiresAt := time.Now().Add(-time.Minute).Truncate(time.Millisecond)
	if err := db.Exec(`INSERT INTO video_user (
		id, device_code, user_type, subscription_status, vip_expires_at,
		vip_points, points_balance, created_at, updated_at
	) VALUES (1, 'expired-user', 2, 2, ?, 30, 90, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, expiresAt).Error; err != nil {
		t.Fatal(err)
	}

	service := NewService()
	if err := service.expireVIPFromAppleNotificationV2(context.Background(), &model.VideoOrder{
		UserID: 1, OrderNo: "vip-expired", ProductID: 8,
	}, expiresAt.UnixMilli()); err != nil {
		t.Fatal(err)
	}

	ledger := latestPointsLedger(t, db)
	if ledger.PointsChange != -30 || ledger.BalanceBefore != 120 || ledger.BalanceAfter != 90 ||
		ledger.SourceType != uint32(domain.PointsSourceExpireDeduct) || ledger.VipID != 8 || ledger.OrderCode != "vip-expired" {
		t.Fatalf("unexpected VIP expiration ledger: %#v", ledger)
	}
}

func latestPointsLedger(t *testing.T, db *gorm.DB) model.VideoUserPointsLedger {
	t.Helper()
	var ledger model.VideoUserPointsLedger
	if err := db.Table("video_user_points_ledger").Order("id DESC").Take(&ledger).Error; err != nil {
		t.Fatal(err)
	}
	return ledger
}
