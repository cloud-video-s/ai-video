package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"ai-video/internal/config"
	"ai-video/internal/domain"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestAppUserRepoExpireDueSubscriptions(t *testing.T) {
	dsn := fmt.Sprintf("file:subscription-expiration-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	previousDB := config.DB
	config.DB = db
	t.Cleanup(func() {
		config.DB = previousDB
		if sqlDB, dbErr := db.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
	})

	if err := db.Exec(`CREATE TABLE video_user (
		id INTEGER PRIMARY KEY,
		subscription_status INTEGER NOT NULL,
		vip_expires_at DATETIME,
		points_balance INTEGER NOT NULL,
		updated_at DATETIME,
		deleted_at DATETIME
	)`).Error; err != nil {
		t.Fatalf("create isolated test table: %v", err)
	}

	now := time.Date(2026, 7, 31, 16, 30, 0, 0, time.UTC)
	rows := []struct {
		id        int
		status    uint8
		expiresAt *time.Time
		points    uint64
		deletedAt *time.Time
	}{
		{id: 1, status: domain.AppUserSubscriptionSubscribed, expiresAt: timePtr(now.Add(-time.Second)), points: 10},
		{id: 2, status: domain.AppUserSubscriptionSubscribed, expiresAt: timePtr(now), points: 20},
		{id: 3, status: domain.AppUserSubscriptionSubscribed, expiresAt: timePtr(now.Add(time.Second)), points: 30},
		{id: 4, status: domain.AppUserSubscriptionCancelled, expiresAt: timePtr(now.Add(-time.Hour)), points: 40},
		{id: 5, status: domain.AppUserSubscriptionSubscribed, expiresAt: nil, points: 50},
		{id: 6, status: domain.AppUserSubscriptionSubscribed, expiresAt: timePtr(now.Add(-time.Hour)), points: 60, deletedAt: timePtr(now.Add(-time.Minute))},
	}
	for _, row := range rows {
		if err := db.Exec(
			"INSERT INTO video_user (id, subscription_status, vip_expires_at, points_balance, deleted_at) VALUES (?, ?, ?, ?, ?)",
			row.id, row.status, row.expiresAt, row.points, row.deletedAt,
		).Error; err != nil {
			t.Fatalf("insert user %d: %v", row.id, err)
		}
	}

	affected, err := NewAppUserRepo().ExpireDueSubscriptions(context.Background(), now)
	if err != nil {
		t.Fatalf("expire due subscriptions: %v", err)
	}
	if affected != 2 {
		t.Fatalf("affected rows = %d, want 2", affected)
	}

	type state struct {
		ID                 int
		SubscriptionStatus uint8
		PointsBalance      uint64
	}
	var states []state
	if err := db.Table("video_user").Select("id", "subscription_status", "points_balance").Order("id").Scan(&states).Error; err != nil {
		t.Fatalf("read user states: %v", err)
	}
	want := []state{
		{ID: 1, SubscriptionStatus: domain.AppUserSubscriptionExpired, PointsBalance: 0},
		{ID: 2, SubscriptionStatus: domain.AppUserSubscriptionExpired, PointsBalance: 0},
		{ID: 3, SubscriptionStatus: domain.AppUserSubscriptionSubscribed, PointsBalance: 30},
		{ID: 4, SubscriptionStatus: domain.AppUserSubscriptionCancelled, PointsBalance: 40},
		{ID: 5, SubscriptionStatus: domain.AppUserSubscriptionSubscribed, PointsBalance: 50},
		{ID: 6, SubscriptionStatus: domain.AppUserSubscriptionSubscribed, PointsBalance: 60},
	}
	if len(states) != len(want) {
		t.Fatalf("state count = %d, want %d", len(states), len(want))
	}
	for i := range want {
		if states[i] != want[i] {
			t.Errorf("state[%d] = %+v, want %+v", i, states[i], want[i])
		}
	}
}

func timePtr(value time.Time) *time.Time { return &value }
