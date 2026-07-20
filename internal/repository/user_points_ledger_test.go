package repository

import (
	"context"
	"testing"
	"time"

	"ai-video/internal/app"
	"ai-video/internal/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestUserPointsLedgerPageListFiltersSummarizesAndPreloads(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:user-points-ledger?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	app.DB = db
	statements := []string{
		`CREATE TABLE video_user (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			imei TEXT NOT NULL,
			username TEXT NOT NULL,
			login_account TEXT,
			google_email TEXT,
			appid_email TEXT,
			user_type INTEGER NOT NULL DEFAULT 1,
			deleted_at DATETIME
		)`,
		`CREATE TABLE video_points_package (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			product_id TEXT NOT NULL,
			name TEXT NOT NULL,
			package_id INTEGER NOT NULL,
			resource_type TEXT NOT NULL,
			points INTEGER NOT NULL,
			currency TEXT NOT NULL,
			sale_price NUMERIC NOT NULL,
			actual_revenue NUMERIC NOT NULL,
			original_price NUMERIC NOT NULL DEFAULT 0,
			status INTEGER NOT NULL DEFAULT 1,
			deleted_at DATETIME
		)`,
		`CREATE TABLE video_user_points_ledger (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			direction INTEGER NOT NULL,
			points_change INTEGER NOT NULL,
			balance_before INTEGER NOT NULL,
			balance_after INTEGER NOT NULL,
			source_type TEXT NOT NULL,
			business_id TEXT,
			points_package_id INTEGER,
			operator_admin_id INTEGER,
			description TEXT,
			occurred_at DATETIME NOT NULL,
			created_at DATETIME NOT NULL
		)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}

	if err := db.Exec(`INSERT INTO video_user (imei, username, login_account, google_email, user_type) VALUES (?, ?, ?, ?, ?)`,
		"device-alice", "Alice", "alice@example.com", "alice@example.com", 1).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO video_user (imei, username, login_account, google_email, user_type) VALUES (?, ?, ?, ?, ?)`,
		"device-bob", "Bob", "bob@example.com", "bob@example.com", 2).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO video_points_package (product_id, name, package_id, resource_type, points, currency, sale_price, actual_revenue, original_price, status) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"credits_1000", "1000 Credits", 10, "credits", 1000, "USD", 9.99, 8.99, 0, 1).Error; err != nil {
		t.Fatal(err)
	}

	baseTime := time.Date(2026, 7, 17, 9, 0, 0, 0, time.Local)
	userID, otherUserID, packageID := uint64(1), uint64(2), uint64(1)
	entries := []model.VideoUserPointsLedger{
		{UserID: userID, Direction: model.PointsDirectionIncome, PointsChange: 1000, BalanceBefore: 0, BalanceAfter: 1000, SourceType: "purchase", BusinessID: "ORDER-100", PointsPackageID: &packageID, Description: "purchase credits", OccurredAt: baseTime},
		{UserID: userID, Direction: model.PointsDirectionExpense, PointsChange: -250, BalanceBefore: 1000, BalanceAfter: 750, SourceType: "consume", BusinessID: "TASK-200", Description: "render video", OccurredAt: baseTime.Add(time.Hour)},
		{UserID: otherUserID, Direction: model.PointsDirectionIncome, PointsChange: 50, BalanceBefore: 0, BalanceAfter: 50, SourceType: "reward", BusinessID: "EVENT-300", Description: "daily reward", OccurredAt: baseTime.Add(2 * time.Hour)},
	}
	if err := db.Create(&entries).Error; err != nil {
		t.Fatal(err)
	}

	repo := NewUserPointsLedgerRepo()
	list, total, summary, err := repo.PageList(context.Background(), 1, 20, &UserPointsLedgerFilter{UserID: userID})
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(list) != 2 {
		t.Fatalf("total=%d len(list)=%d, want 2 and 2", total, len(list))
	}
	if summary.IncomeTotal != 1000 || summary.ExpenseTotal != 250 {
		t.Fatalf("summary=%+v, want income=1000 expense=250", summary)
	}
	if list[0].BusinessID != "TASK-200" || list[1].BusinessID != "ORDER-100" {
		t.Fatalf("unexpected order: %q, %q", list[0].BusinessID, list[1].BusinessID)
	}
	if list[1].User.Username != "Alice" || list[1].PointsPackage == nil || list[1].PointsPackage.ProductID != "credits_1000" {
		t.Fatalf("associations were not preloaded: %#v", list[1])
	}

	keywordList, keywordTotal, _, err := repo.PageList(context.Background(), 1, 20, &UserPointsLedgerFilter{Keyword: "Alice"})
	if err != nil {
		t.Fatal(err)
	}
	if keywordTotal != 2 || len(keywordList) != 2 {
		t.Fatalf("user keyword total=%d len=%d, want 2", keywordTotal, len(keywordList))
	}
}
