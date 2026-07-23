package repository

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"ai-video/internal/config"
	"ai-video/internal/gen/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestVIPSubscriptionDefaultScopeAndExcludedChannelFilter(t *testing.T) {
	db := openVIPSubscriptionTestDB(t)
	packageA := createVIPTestPackage(t, db, 1, "com.example.a")
	packageB := createVIPTestPackage(t, db, 2, "com.example.b")
	channel := model.VideoChannel{
		ChannelID:   1,
		ChannelCode: "google_ads", ChannelName: "Google Ads", AdPlatform: "google",
		DeliveryPackage: "com.example.a", Status: 1,
	}
	if err := db.Omit("CreatedAt", "UpdatedAt").Create(&channel).Error; err != nil {
		t.Fatalf("create channel: %v", err)
	}

	repo := NewVIPSubscriptionRepo()
	ctx := context.Background()
	oldA := createVIPTestSubscription(t, repo, ctx, 1, packageA.ID, "sku_a_old", true, nil)
	newA := createVIPTestSubscription(t, repo, ctx, 2, packageA.ID, "sku_a_new", false, []uint64{channel.ChannelID})
	oldB := createVIPTestSubscription(t, repo, ctx, 3, packageB.ID, "sku_b_old", true, nil)

	newADetail, err := repo.GetDetail(ctx, newA.ID)
	if err != nil {
		t.Fatalf("load new package A subscription: %v", err)
	}
	if err := repo.SetDefault(ctx, newADetail); err != nil {
		t.Fatalf("set package A default: %v", err)
	}
	if err := db.Exec("UPDATE video_vip_subscription SET created_at = NULL, updated_at = NULL").Error; err != nil {
		t.Fatalf("normalize sqlite timestamps: %v", err)
	}

	assertVIPDefault(t, db, oldA.ID, false)
	assertVIPDefault(t, db, newA.ID, true)
	assertVIPDefault(t, db, oldB.ID, true)

	list, total, err := repo.PageList(ctx, 1, 20, &VIPSubscriptionListFilter{ExcludedChannelID: channel.ChannelID})
	if err != nil {
		t.Fatalf("filter excluded channel: %v", err)
	}
	if total != 1 || len(list) != 1 || list[0].ID != newA.ID {
		t.Fatalf("excluded-channel result total=%d ids=%v, want only %d", total, vipSubscriptionIDs(list), newA.ID)
	}
	packageACount, err := repo.PackageCount(ctx, packageA.ID)
	if err != nil {
		t.Fatalf("count package A subscriptions: %v", err)
	}
	if packageACount != 2 {
		t.Fatalf("package A subscription count=%d, want 2", packageACount)
	}
}

func openVIPSubscriptionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent), TranslateError: true})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.VideoPackage{}, &model.VideoDisplayPosition{}, &model.VideoChannel{},
		&model.VideoVipSubscription{}, &model.VideoVipSubscriptionPackage{},
		&model.VideoVipSubscriptionPosition{}, &model.VideoVipSubscriptionChannel{},
		&model.VideoVipSubscriptionExcludedChannel{},
	); err != nil {
		t.Fatalf("migrate VIP subscription schema: %v", err)
	}
	previous := config.DB
	config.DB = db
	t.Cleanup(func() {
		config.DB = previous
		if sqlDB, sqlErr := db.DB(); sqlErr == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func createVIPTestPackage(t *testing.T, db *gorm.DB, id uint64, code string) model.VideoPackage {
	t.Helper()
	item := model.VideoPackage{
		ID:          id,
		PackageName: code, PackageCode: code, AppCode: "video",
		SystemType: 2, Status: 1,
	}
	if err := db.Omit("CreatedAt", "UpdatedAt").Create(&item).Error; err != nil {
		t.Fatalf("create package %s: %v", code, err)
	}
	return item
}

func createVIPTestSubscription(t *testing.T, repo *VIPSubscriptionRepo, ctx context.Context, id, packageID uint64, productID string, isDefault bool, excludedChannelIDs []uint64) *model.VideoVipSubscription {
	t.Helper()
	item := &model.VideoVipSubscription{
		ID:       id,
		Platform: "android", ProductID: productID, Name: productID, VIPLevel: "monthly",
		PlanType: "normal", Currency: "USD", DisplayMode: 1, Status: 1,
		IsSubscription: true, IsDefault: isDefault,
	}
	if err := config.DB.WithContext(ctx).Omit("CreatedAt", "UpdatedAt").Create(item).Error; err != nil {
		t.Fatalf("create subscription %s: %v", productID, err)
	}
	if err := repo.ReplaceTargets(ctx, item, VIPSubscriptionTargets{
		PackageIDs: []uint64{packageID}, ExcludedChannelIDs: excludedChannelIDs,
	}); err != nil {
		t.Fatalf("replace subscription %s targets: %v", productID, err)
	}
	if err := config.DB.Exec("UPDATE video_vip_subscription SET created_at = NULL, updated_at = NULL WHERE id = ?", item.ID).Error; err != nil {
		t.Fatalf("normalize subscription %s timestamps: %v", productID, err)
	}
	return item
}

func assertVIPDefault(t *testing.T, db *gorm.DB, id uint64, want bool) {
	t.Helper()
	var item model.VideoVipSubscription
	if err := db.First(&item, id).Error; err != nil {
		t.Fatalf("load subscription %d: %v", id, err)
	}
	if item.IsDefault != want {
		t.Fatalf("subscription %d default=%v, want %v", id, item.IsDefault, want)
	}
}

func vipSubscriptionIDs(items []model.VideoVipSubscription) []uint64 {
	result := make([]uint64, len(items))
	for i := range items {
		result[i] = items[i].ID
	}
	return result
}
