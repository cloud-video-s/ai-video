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
	"ai-video/internal/repository"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestFulfillAppleRenewalResetsOldPointsAndDeductsActiveFrozenVIP(t *testing.T) {
	db, service := newSubscriptionPointsTestService(t)
	now := time.Now().Truncate(time.Millisecond)
	oldExpires := now.Add(-time.Minute)
	newExpires := now.Add(30 * 24 * time.Hour)

	mustExecSubscriptionPointsTest(t, db, `INSERT INTO video_user (
		id, user_type, subscription_status, vip_started_at, vip_expires_at,
		vip_points, points_balance, frozen_points, payment_count,
		actual_amount_money, order_amount_money, subscription_payment_count,
		first_payment_met, payment_met, vip_level, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, 80, 20, 30, 1, 9.99, 9.99, 1, 1, 1, 1, ?, ?)`,
		1, domain.AppUserTypePaid, domain.AppUserSubscriptionSubscribed,
		now.Add(-30*24*time.Hour), oldExpires, now, now)
	mustExecSubscriptionPointsTest(t, db, `INSERT INTO video_order (
		id, order_no, user_id, product_type, product_id, bonus_points, status,
		order_type, pay_time, actual_amount_money, paid_amount, vip_level,
		vip_duration_days, created_at, updated_at
	) VALUES (1, 'renewal-order', 1, ?, 9, 100, ?, ?, ?, 9.99, 9.99, 2, 30, ?, ?)`,
		domain.OrderProductVIPSubscription, domain.OrderStatusPaid, domain.OrderTypeRenewal, now, now, now)
	mustExecSubscriptionPointsTest(t, db, `INSERT INTO video_user_generation_task
		(id, user_id, status, vip_score, deleted_at) VALUES
		(1, 1, ?, 30, NULL),
		(2, 1, ?, 20, NULL),
		(3, 1, ?, 40, ?)`,
		domain.GenerationTaskStatusRunning, domain.GenerationTaskStatusSuccess,
		domain.GenerationTaskStatusPending, now)

	if err := service.fulfillAppleOrder(context.Background(), &model.VideoOrder{OrderNo: "renewal-order"}, &newExpires); err != nil {
		t.Fatal(err)
	}

	var user model.VideoUser
	if err := db.First(&user, 1).Error; err != nil {
		t.Fatal(err)
	}
	if user.VipPoints != 70 || user.PointsBalance != 20 || user.FrozenPoints != 30 {
		t.Fatalf("balances = vip %d, points %d, frozen %d; want 70, 20, 30", user.VipPoints, user.PointsBalance, user.FrozenPoints)
	}
	if user.VipExpiresAt == nil || !user.VipExpiresAt.Equal(newExpires) ||
		user.SubscriptionStatus != domain.AppUserSubscriptionSubscribed || user.UserType != uint8(domain.AppUserTypePaid) {
		t.Fatalf("unexpected entitlement: expires=%v status=%d type=%d", user.VipExpiresAt, user.SubscriptionStatus, user.UserType)
	}

	var ledgers []model.VideoUserPointsLedger
	if err := db.Order("id ASC").Find(&ledgers).Error; err != nil {
		t.Fatal(err)
	}
	if len(ledgers) != 2 {
		t.Fatalf("ledger count = %d, want 2", len(ledgers))
	}
	clearLedger, giftLedger := ledgers[0], ledgers[1]
	if clearLedger.PointsChange != -80 || clearLedger.BalanceBefore != 100 ||
		clearLedger.BalanceAfter != 20 || clearLedger.SourceType != uint32(domain.PointsSourceExpireDeduct) {
		t.Fatalf("unexpected old-points ledger: %#v", clearLedger)
	}
	if giftLedger.PointsChange != 100 || giftLedger.BalanceBefore != 20 ||
		giftLedger.BalanceAfter != 90 || giftLedger.SourceType != uint32(domain.PointsSourceSubscriptionGift) {
		t.Fatalf("unexpected renewal gift ledger: %#v", giftLedger)
	}

	if err := service.fulfillAppleOrder(context.Background(), &model.VideoOrder{OrderNo: "renewal-order"}, &newExpires); err != nil {
		t.Fatal(err)
	}
	var ledgerCount int64
	if err := db.Model(&model.VideoUserPointsLedger{}).Count(&ledgerCount).Error; err != nil {
		t.Fatal(err)
	}
	if ledgerCount != 2 {
		t.Fatalf("ledger count after retry = %d, want 2", ledgerCount)
	}
}

func TestExpireAppleSubscriptionClearsEntitlementAndIgnoresStaleNotifications(t *testing.T) {
	t.Run("clear expired subscription", func(t *testing.T) {
		db, service := newSubscriptionPointsTestService(t)
		now := time.Now().Truncate(time.Millisecond)
		expiresAt := now.Add(-time.Minute)
		seedSubscriptionExpirationTest(t, db, expiresAt, 40)

		order := &model.VideoOrder{OrderNo: "expired-order", UserID: 1, ProductID: 9}
		if err := service.expireVIPFromAppleNotificationV2(context.Background(), order, expiresAt.UnixMilli()); err != nil {
			t.Fatal(err)
		}

		var user model.VideoUser
		if err := db.First(&user, 1).Error; err != nil {
			t.Fatal(err)
		}
		if user.VipPoints != 0 || user.VipExpiresAt != nil ||
			user.SubscriptionStatus != domain.AppUserSubscriptionExpired || user.UserType != uint8(domain.AppUserTypeFree) {
			t.Fatalf("unexpected expired entitlement: points=%d expires=%v status=%d type=%d", user.VipPoints, user.VipExpiresAt, user.SubscriptionStatus, user.UserType)
		}
		var ledger model.VideoUserPointsLedger
		if err := db.First(&ledger).Error; err != nil {
			t.Fatal(err)
		}
		if ledger.PointsChange != -40 || ledger.BalanceBefore != 50 || ledger.BalanceAfter != 10 ||
			ledger.SourceType != uint32(domain.PointsSourceExpireDeduct) {
			t.Fatalf("unexpected expiration ledger: %#v", ledger)
		}
	})

	t.Run("newer expiration wins", func(t *testing.T) {
		db, service := newSubscriptionPointsTestService(t)
		now := time.Now().Truncate(time.Millisecond)
		oldExpires := now.Add(-time.Minute)
		newExpires := now.Add(30 * 24 * time.Hour)
		seedSubscriptionExpirationTest(t, db, newExpires, 70)

		order := &model.VideoOrder{OrderNo: "expired-order", UserID: 1, ProductID: 9}
		if err := service.expireVIPFromAppleNotificationV2(context.Background(), order, oldExpires.UnixMilli()); err != nil {
			t.Fatal(err)
		}
		assertSubscriptionUnchanged(t, db, 70, newExpires)
	})

	t.Run("newer gift ledger wins", func(t *testing.T) {
		db, service := newSubscriptionPointsTestService(t)
		now := time.Now().Truncate(time.Millisecond)
		expiresAt := now.Add(-time.Minute)
		seedSubscriptionExpirationTest(t, db, expiresAt, 70)
		mustExecSubscriptionPointsTest(t, db, `INSERT INTO video_user_points_ledger (
			user_id, direction, points_change, balance_before, balance_after,
			description, source_type, order_code, occurred_at, created_at, updated_at
		) VALUES (1, 1, 100, 10, 110, 'new subscription', ?, 'new-order', ?, ?, ?)`,
			domain.PointsSourceSubscriptionGift, expiresAt.Add(time.Second), now, now)

		order := &model.VideoOrder{OrderNo: "expired-order", UserID: 1, ProductID: 9}
		if err := service.expireVIPFromAppleNotificationV2(context.Background(), order, expiresAt.UnixMilli()); err != nil {
			t.Fatal(err)
		}
		assertSubscriptionUnchanged(t, db, 70, expiresAt)
	})
}

func newSubscriptionPointsTestService(t *testing.T) (*gorm.DB, *Service) {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "-"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	statements := []string{
		`CREATE TABLE video_user (
			id INTEGER PRIMARY KEY, user_type INTEGER NOT NULL DEFAULT 1,
			subscription_status INTEGER NOT NULL DEFAULT 1,
			vip_started_at DATETIME NULL, vip_expires_at DATETIME NULL,
			vip_points INTEGER NOT NULL DEFAULT 0, points_balance INTEGER NOT NULL DEFAULT 0,
			frozen_points INTEGER NOT NULL DEFAULT 0, payment_count INTEGER NOT NULL DEFAULT 0,
			actual_amount_money REAL NOT NULL DEFAULT 0, last_paid_at DATETIME NULL,
			order_amount_money REAL NOT NULL DEFAULT 0, first_paid_at DATETIME NULL,
			first_payment_met INTEGER NOT NULL DEFAULT 0, payment_met INTEGER NOT NULL DEFAULT 0,
			subscription_payment_count INTEGER NOT NULL DEFAULT 0,
			one_time_payment_count INTEGER NOT NULL DEFAULT 0, vip_level INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL, updated_at DATETIME NOT NULL, deleted_at DATETIME NULL
		)`,
		`CREATE TABLE video_order (
			id INTEGER PRIMARY KEY, order_no TEXT NOT NULL UNIQUE, user_id INTEGER NOT NULL,
			product_type INTEGER NOT NULL, product_id INTEGER NOT NULL, bonus_points INTEGER NOT NULL DEFAULT 0,
			status INTEGER NOT NULL, order_type INTEGER NOT NULL DEFAULT 1, pay_time DATETIME NULL,
			actual_amount_money REAL NOT NULL DEFAULT 0, paid_amount REAL NOT NULL DEFAULT 0,
			vip_level INTEGER NOT NULL DEFAULT 0, vip_duration_days INTEGER NOT NULL DEFAULT 0,
			completed_at DATETIME NULL, created_at DATETIME NOT NULL, updated_at DATETIME NOT NULL,
			deleted_at DATETIME NULL
		)`,
		`CREATE TABLE video_user_points_ledger (
			id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL, direction INTEGER NOT NULL,
			points_change INTEGER NOT NULL, balance_before INTEGER NOT NULL, balance_after INTEGER NOT NULL,
			description TEXT, source_type INTEGER NOT NULL, order_code TEXT, points_id INTEGER NOT NULL DEFAULT 0,
			vip_id INTEGER NOT NULL DEFAULT 0, occurred_at DATETIME NOT NULL, admin_id INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL, updated_at DATETIME NOT NULL, deleted_at DATETIME NULL
		)`,
		`CREATE TABLE video_user_generation_task (
			id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL, status INTEGER NOT NULL,
			vip_score INTEGER NOT NULL DEFAULT 0, deleted_at DATETIME NULL
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
	return db, &Service{
		orders: repository.NewOrderRepo(), users: repository.NewAppUserRepo(),
		ledgers: repository.NewCommercePointsLedgerRepo(), tasks: repository.NewUserGenerationTaskRepo(),
	}
}

func seedSubscriptionExpirationTest(t *testing.T, db *gorm.DB, expiresAt time.Time, vipPoints int64) {
	t.Helper()
	now := time.Now().Truncate(time.Millisecond)
	mustExecSubscriptionPointsTest(t, db, `INSERT INTO video_user (
		id, user_type, subscription_status, vip_started_at, vip_expires_at,
		vip_points, points_balance, frozen_points, vip_level, created_at, updated_at
	) VALUES (1, ?, ?, ?, ?, ?, 10, 0, 1, ?, ?)`,
		domain.AppUserTypePaid, domain.AppUserSubscriptionSubscribed,
		now.Add(-30*24*time.Hour), expiresAt, vipPoints, now, now)
}

func assertSubscriptionUnchanged(t *testing.T, db *gorm.DB, wantPoints int64, wantExpires time.Time) {
	t.Helper()
	var user model.VideoUser
	if err := db.First(&user, 1).Error; err != nil {
		t.Fatal(err)
	}
	if user.VipPoints != wantPoints || user.VipExpiresAt == nil || !user.VipExpiresAt.Equal(wantExpires) ||
		user.SubscriptionStatus != domain.AppUserSubscriptionSubscribed || user.UserType != uint8(domain.AppUserTypePaid) {
		t.Fatalf("subscription changed: points=%d expires=%v status=%d type=%d", user.VipPoints, user.VipExpiresAt, user.SubscriptionStatus, user.UserType)
	}
}

func mustExecSubscriptionPointsTest(t *testing.T, db *gorm.DB, sql string, args ...any) {
	t.Helper()
	if err := db.Exec(sql, args...).Error; err != nil {
		t.Fatal(err)
	}
}
