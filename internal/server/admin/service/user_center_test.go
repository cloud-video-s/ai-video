package service

import (
	"context"
	"testing"
	"time"

	"ai-video/internal/config"
	"ai-video/internal/domain"
	"ai-video/internal/gen/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestNewAppUserUsesLatestSchemaFields(t *testing.T) {
	status := domain.AppUserStatusBlacklisted
	startedAt := time.Now().Add(-time.Hour)
	expiresAt := time.Now().Add(24 * time.Hour)
	user := newAppUser(&CreateAppUserRequest{
		DeviceCode: " device ", Username: " user ", ClientCountry: " CN ", ServerCountry: " US ",
		PackageCode: " package ", IMEI: " imei ", Phone: " 123 ", ActiveLong: 7200,
		VIPStartedAt: &startedAt, VIPExpiresAt: &expiresAt, VIPPoints: -5, PointsBalance: -20,
		PaymentMet: true, FirstPaymentMet: true, Registered: true, VIPLevel: 3, Status: &status,
	})

	if user.DeviceCode != "device" || user.ClientCountry != "CN" || user.ServerCountry != "US" ||
		user.PackageCode != "package" || user.IMEI != "imei" || user.Phone != "123" {
		t.Fatalf("string fields were not normalized: %#v", user)
	}
	if user.ActiveLong != 7200 || user.VipPoints != -5 || user.PointsBalance != -20 ||
		user.VIPStartedAt != &startedAt || user.VipExpiresAt != &expiresAt {
		t.Fatalf("latest schema fields were not mapped: %#v", user)
	}
	if user.LoginType != uint8(domain.AppUserLoginGuest) || user.UserType != uint8(domain.AppUserTypeFree) ||
		user.SubscriptionStatus != domain.AppUserSubscriptionNotSubscribed {
		t.Fatalf("defaults were not applied: %#v", user)
	}
	if user.PaymentMet != 1 || user.FirstPaymentMet != 1 || user.Registered != 1 ||
		user.IsBlacklisted != 1 || user.Status != domain.AppUserStatusBlacklisted || user.VIPLevel != 3 {
		t.Fatalf("state fields were not mapped: %#v", user)
	}
}

func TestUserCenterVIPAndAccessOperations(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:user-center-operations?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	config.DB = db
	if err := db.Exec(`CREATE TABLE video_user (
		id INTEGER PRIMARY KEY, device_code TEXT, imei TEXT, username TEXT, app_name TEXT,
		client_country TEXT, server_country TEXT, email TEXT, third_code TEXT, phone TEXT,
		login_account TEXT, package_code TEXT,
		active_long INTEGER NOT NULL DEFAULT 0, points_balance INTEGER NOT NULL DEFAULT 0,
		vip_points INTEGER NOT NULL DEFAULT 0,
		status INTEGER NOT NULL DEFAULT 1, token_version INTEGER NOT NULL DEFAULT 0,
		vip_level INTEGER NOT NULL DEFAULT 0, vip_started_at DATETIME, vip_expires_at DATETIME,
		user_type INTEGER NOT NULL DEFAULT 1, subscription_status INTEGER NOT NULL DEFAULT 1,
		is_frozen BOOLEAN NOT NULL DEFAULT 0, is_blacklisted BOOLEAN NOT NULL DEFAULT 0,
		created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
	)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE video_user_identity (
		id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL, email TEXT
	)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE video_user_points_ledger (
		id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL, direction INTEGER NOT NULL,
		points_change INTEGER NOT NULL, balance_before INTEGER NOT NULL, balance_after INTEGER NOT NULL,
		description TEXT, source_type INTEGER NOT NULL DEFAULT 1, order_code TEXT,
		points_id INTEGER NOT NULL DEFAULT 0, vip_id INTEGER NOT NULL DEFAULT 0,
		occurred_at DATETIME NOT NULL, admin_id INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL, updated_at DATETIME NOT NULL, deleted_at DATETIME
	)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO video_user (id, device_code, imei, username, app_name, vip_points) VALUES (1, 'device-1', 'imei-1', 'legacy', 'app', 9)`).Error; err != nil {
		t.Fatal(err)
	}
	user := model.VideoUser{ID: 1}

	service := NewAppUserService()
	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	if err := service.GrantVIP(context.Background(), user.ID, 42, &GrantUserVIPRequest{Level: 3, ExpiresAt: expiresAt}); err != nil {
		t.Fatal(err)
	}
	updated, err := service.GetByID(context.Background(), user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.VipPoints != 9 {
		t.Fatalf("VIP points=%d after default zero gift, want existing balance 9", updated.VipPoints)
	}
	var ledgerCount int64
	if err := db.Table("video_user_points_ledger").Count(&ledgerCount).Error; err != nil {
		t.Fatal(err)
	}
	if ledgerCount != 0 {
		t.Fatalf("default zero gift created %d ledger rows, want 0", ledgerCount)
	}
	if err := service.GrantVIP(context.Background(), user.ID, 42, &GrantUserVIPRequest{Level: 3, VIPPoints: 120, ExpiresAt: expiresAt}); err != nil {
		t.Fatal(err)
	}
	if err := service.SetFrozen(context.Background(), user.ID, true); err != nil {
		t.Fatal(err)
	}

	updated, err = service.GetByID(context.Background(), user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.VIPLevel != 3 || updated.VipPoints != 129 || updated.VipExpiresAt == nil || updated.IsFrozen != 1 || updated.Status != domain.AppUserStatusFrozen {
		t.Fatalf("unexpected updated user: level=%d expires=%v frozen=%v status=%d", updated.VIPLevel, updated.VipExpiresAt, updated.IsFrozen, updated.Status)
	}
	var ledger model.VideoUserPointsLedger
	if err := db.Table("video_user_points_ledger").First(&ledger).Error; err != nil {
		t.Fatal(err)
	}
	if ledger.PointsChange != 120 || ledger.BalanceBefore != 9 || ledger.BalanceAfter != 129 ||
		ledger.AdminID != 42 || ledger.SourceType != uint32(domain.PointsSourceAdminOp) {
		t.Fatalf("unexpected VIP gift ledger: %#v", ledger)
	}
	if updated.TokenVersion != 1 {
		t.Fatalf("token version=%d, want 1", updated.TokenVersion)
	}
	if _, _, err := service.repo.GetAuthState(context.Background(), user.ID); err == nil {
		t.Fatal("frozen user must not have a valid auth state")
	}

	if err := service.SetBlacklisted(context.Background(), user.ID, true); err != nil {
		t.Fatal(err)
	}
	updated, err = service.GetByID(context.Background(), user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.IsBlacklisted != 1 || updated.Status != domain.AppUserStatusBlacklisted {
		t.Fatalf("blacklisted=%d status=%d, want 1/%d", updated.IsBlacklisted, updated.Status, domain.AppUserStatusBlacklisted)
	}
	if err := service.SetBlacklisted(context.Background(), user.ID, false); err != nil {
		t.Fatal(err)
	}
	updated, err = service.GetByID(context.Background(), user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != domain.AppUserStatusFrozen {
		t.Fatalf("status=%d, want frozen status after removing blacklist", updated.Status)
	}
	if err := service.SetFrozen(context.Background(), user.ID, false); err != nil {
		t.Fatal(err)
	}
	if _, _, err := service.repo.GetAuthState(context.Background(), user.ID); err != nil {
		t.Fatalf("normal user should have valid auth state: %v", err)
	}

	clientCountry := " CN "
	serverCountry := " US "
	activeLong := uint64(3600)
	pointsBalance := int64(-25)
	vipPoints := int64(-5)
	updated, err = service.Update(context.Background(), user.ID, &UpdateAppUserRequest{
		ClientCountry: &clientCountry, ServerCountry: &serverCountry, ActiveLong: &activeLong,
		PointsBalance: &pointsBalance, VIPPoints: &vipPoints,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.ClientCountry != "CN" || updated.ServerCountry != "US" || updated.ActiveLong != activeLong ||
		updated.PointsBalance != pointsBalance || updated.VipPoints != vipPoints {
		t.Fatalf("latest user fields were not updated: %#v", updated)
	}

	notFrozen := false
	users, total, err := service.List(context.Background(), 1, 20, &ListAppUserRequest{
		ClientCountry: "CN", IsFrozen: &notFrozen,
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(users) != 1 || users[0].ID != user.ID {
		t.Fatalf("filtered users=%v total=%d, want user %d", users, total, user.ID)
	}
	users, total, err = service.List(context.Background(), 1, 20, &ListAppUserRequest{Keyword: "device-1"})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(users) != 1 || users[0].ID != user.ID {
		t.Fatalf("device-code lookup users=%v total=%d, want user %d", users, total, user.ID)
	}
}
