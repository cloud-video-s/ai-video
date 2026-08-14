package generation

import (
	"context"
	"path/filepath"
	"testing"

	"ai-video/internal/config"
	"ai-video/internal/domain"
	"ai-video/internal/repository"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestConcurrentReservationsProduceContinuousLedgerBalances(t *testing.T) {
	db := newGenerationPointsTestDB(t)
	if err := db.Exec(`INSERT INTO video_user
		(id, subscription_status, vip_points, points_balance, frozen_points)
		VALUES (1, ?, 0, 280, 360)`, domain.AppUserSubscriptionSubscribed).Error; err != nil {
		t.Fatal(err)
	}

	manager := &Manager{AppUserRepo: repository.NewAppUserRepo()}
	for i, taskCode := range []string{"task-a", "task-b", "task-c", "task-d"} {
		state := &repository.GenerationTaskPointState{
			ID:          uint64(i + 1),
			UserID:      1,
			TaskCode:    taskCode,
			Score:       90,
			ScoreType:   TaskScoreTypeBalance,
			PointsScore: 90,
		}
		if err := repository.Transaction(context.Background(), func(ctx context.Context) error {
			return manager.consumeFrozenTaskPoints(ctx, state)
		}); err != nil {
			t.Fatalf("settle %s: %v", taskCode, err)
		}
	}

	var ledgers []struct {
		PointsChange  int64  `gorm:"column:points_change"`
		BalanceBefore uint64 `gorm:"column:balance_before"`
		BalanceAfter  uint64 `gorm:"column:balance_after"`
	}
	if err := db.Table("video_user_points_ledger").
		Select("points_change", "balance_before", "balance_after").
		Order("id ASC").Find(&ledgers).Error; err != nil {
		t.Fatal(err)
	}
	want := [][2]uint64{{640, 550}, {550, 460}, {460, 370}, {370, 280}}
	if len(ledgers) != len(want) {
		t.Fatalf("ledger count = %d, want %d", len(ledgers), len(want))
	}
	for i := range want {
		if ledgers[i].PointsChange != -90 ||
			ledgers[i].BalanceBefore != want[i][0] || ledgers[i].BalanceAfter != want[i][1] {
			t.Errorf("ledger %d = change %d, %d -> %d; want -90, %d -> %d",
				i, ledgers[i].PointsChange, ledgers[i].BalanceBefore, ledgers[i].BalanceAfter,
				want[i][0], want[i][1])
		}
	}

	assertGenerationPointBalances(t, db, 1, 0, 280, 0)
}

func TestReleaseFrozenTaskPointsUpdatesTheCorrectColumns(t *testing.T) {
	db := newGenerationPointsTestDB(t)
	tests := []struct {
		name       string
		userID     uint64
		status     uint8
		wantVIP    int64
		wantPoints int64
	}{
		{name: "active subscription restores both allocations", userID: 1, status: domain.AppUserSubscriptionSubscribed, wantVIP: 30, wantPoints: 340},
		{name: "expired subscription restores only purchased points", userID: 2, status: domain.AppUserSubscriptionCancelled, wantVIP: 0, wantPoints: 340},
	}
	for _, test := range tests {
		if err := db.Exec(`INSERT INTO video_user
			(id, subscription_status, vip_points, points_balance, frozen_points)
			VALUES (?, ?, 0, 280, 90)`, test.userID, test.status).Error; err != nil {
			t.Fatal(err)
		}
	}

	manager := &Manager{AppUserRepo: repository.NewAppUserRepo()}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := &repository.GenerationTaskPointState{
				UserID: test.userID, Score: 90, ScoreType: TaskScoreTypeMixed,
				VIPScore: 30, PointsScore: 60,
			}
			if err := repository.Transaction(context.Background(), func(ctx context.Context) error {
				return manager.releaseFrozenTaskPoints(ctx, state)
			}); err != nil {
				t.Fatal(err)
			}
			assertGenerationPointBalances(t, db, test.userID, test.wantVIP, test.wantPoints, 0)
		})
	}
}

func newGenerationPointsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "points.db")), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`CREATE TABLE video_user (
			id INTEGER PRIMARY KEY,
			subscription_status INTEGER NOT NULL DEFAULT 1,
			vip_points INTEGER NOT NULL DEFAULT 0,
			points_balance INTEGER NOT NULL DEFAULT 0,
			frozen_points INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)`,
		`CREATE TABLE video_user_points_ledger (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			direction INTEGER NOT NULL,
			points_change INTEGER NOT NULL,
			balance_before INTEGER NOT NULL,
			balance_after INTEGER NOT NULL,
			description TEXT,
			source_type INTEGER NOT NULL,
			order_code TEXT,
			points_id INTEGER NOT NULL DEFAULT 0,
			vip_id INTEGER NOT NULL DEFAULT 0,
			occurred_at DATETIME NOT NULL,
			admin_id INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			deleted_at DATETIME
		)`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}

	previousDB := config.DB
	config.DB = db
	t.Cleanup(func() {
		config.DB = previousDB
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func assertGenerationPointBalances(t *testing.T, db *gorm.DB, userID uint64, vip, points int64, frozen uint64) {
	t.Helper()
	var got struct {
		VIP    int64  `gorm:"column:vip_points"`
		Points int64  `gorm:"column:points_balance"`
		Frozen uint64 `gorm:"column:frozen_points"`
	}
	if err := db.Table("video_user").
		Select("vip_points", "points_balance", "frozen_points").
		Where("id = ?", userID).Take(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got.VIP != vip || got.Points != points || got.Frozen != frozen {
		t.Fatalf("balances = vip %d, points %d, frozen %d; want %d, %d, %d",
			got.VIP, got.Points, got.Frozen, vip, points, frozen)
	}
}
