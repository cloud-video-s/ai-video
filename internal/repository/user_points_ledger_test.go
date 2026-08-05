package repository

import (
	"context"
	"testing"

	"ai-video/internal/config"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestUserPointsLedgerPageListUsesLatestReferenceFields(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:user-points-ledger-latest-fields?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	previousDB := config.DB
	config.DB = db
	t.Cleanup(func() { config.DB = previousDB })
	statements := []string{
		`CREATE TABLE video_user (
			id INTEGER PRIMARY KEY, username TEXT, imei TEXT, login_account TEXT, email TEXT, deleted_at DATETIME
		)`,
		`CREATE TABLE video_points_package (
			id INTEGER PRIMARY KEY, product_code TEXT, name TEXT, deleted_at DATETIME
		)`,
		`CREATE TABLE video_user_points_ledger (
			id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL, direction INTEGER NOT NULL,
			points_change INTEGER NOT NULL, balance_before INTEGER NOT NULL, balance_after INTEGER NOT NULL,
			description TEXT, source_type INTEGER NOT NULL, order_code TEXT,
			points_id INTEGER NOT NULL DEFAULT 0, vip_id INTEGER NOT NULL DEFAULT 0,
			occurred_at DATETIME NOT NULL, admin_id INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL, updated_at DATETIME NOT NULL, deleted_at DATETIME
		)`,
		`INSERT INTO video_user (id, username) VALUES (1, 'tester')`,
		`INSERT INTO video_points_package (id, product_code, name) VALUES (7, 'points-100', '100 Points')`,
		`INSERT INTO video_user_points_ledger (
			id, user_id, direction, points_change, balance_before, balance_after, description,
			source_type, order_code, points_id, occurred_at, admin_id, created_at, updated_at
		) VALUES (10, 1, 1, 100, 20, 120, 'purchase', 2, 'order-1', 7, CURRENT_TIMESTAMP, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}

	records, total, summary, err := NewUserPointsLedgerRepo().PageList(context.Background(), 1, 20, &UserPointsLedgerFilter{
		PointsID: 7, OrderCode: "order-1", SourceType: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(records) != 1 || summary.IncomeTotal != 100 {
		t.Fatalf("records=%v total=%d summary=%#v", records, total, summary)
	}
	record := records[0]
	if record.PointsChange != 100 || record.BalanceBefore != 20 || record.BalanceAfter != 120 ||
		record.OrderCode != "order-1" || record.PointsID != 7 || record.PointsPackage == nil || record.PointsPackage.ID != 7 {
		t.Fatalf("latest references were not loaded: %#v", record)
	}
}
