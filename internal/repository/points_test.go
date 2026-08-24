package repository

import (
	"context"
	"testing"
	"time"

	"ai-video/internal/config"
	"ai-video/internal/gen/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestPointsRepoTreatsMissingPackageRelationsAsAllPackages(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:points-all-packages?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	statements := []string{
		`CREATE TABLE video_points (
			id INTEGER PRIMARY KEY, product_code TEXT NOT NULL, name TEXT NOT NULL,
			systems TEXT NOT NULL, user_types TEXT NOT NULL, resource_type TEXT NOT NULL,
			points INTEGER NOT NULL, currency TEXT NOT NULL, sale_price REAL NOT NULL,
			actual_revenue REAL NOT NULL, original_price REAL NOT NULL, icon TEXT,
			description TEXT, button_text TEXT, is_default INTEGER NOT NULL,
			status INTEGER NOT NULL, sort INTEGER NOT NULL, created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL, deleted_at DATETIME NULL
		)`,
		`CREATE TABLE video_points_app (points_id INTEGER NOT NULL, app_code TEXT NOT NULL, deleted_at DATETIME NULL)`,
		`CREATE TABLE video_points_package (points_id INTEGER NOT NULL, package_code TEXT NOT NULL, deleted_at DATETIME NULL)`,
		`CREATE TABLE video_points_version (points_id INTEGER NOT NULL, version_code TEXT NOT NULL, deleted_at DATETIME NULL)`,
		`CREATE TABLE video_points_country (points_id INTEGER NOT NULL, country_code TEXT NOT NULL, deleted_at DATETIME NULL)`,
		`CREATE TABLE video_points_channel (points_id INTEGER NOT NULL, channel_code TEXT NOT NULL, deleted_at DATETIME NULL)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now().Truncate(time.Second)
	for _, item := range []struct {
		id          int
		productCode string
		isDefault   int
	}{
		{id: 1, productCode: "global", isDefault: 1},
		{id: 2, productCode: "package-a", isDefault: 0},
		{id: 3, productCode: "package-b", isDefault: 0},
	} {
		if err := db.Exec(`INSERT INTO video_points (
			id, product_code, name, systems, user_types, resource_type, points,
			currency, sale_price, actual_revenue, original_price, icon, description,
			button_text, is_default, status, sort, created_at, updated_at
		) VALUES (?, ?, ?, '["ios"]', '[1,2]', 'credits', 100, 'USD', 1.99, 1.5, 2.99, '', '', 'Buy', ?, 1, 0, ?, ?)`,
			item.id, item.productCode, item.productCode, item.isDefault, now, now,
		).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Exec(`INSERT INTO video_points_package (points_id, package_code) VALUES (2, 'pkg-a'), (3, 'pkg-b')`).Error; err != nil {
		t.Fatal(err)
	}

	previousDB := config.DB
	config.DB = db
	t.Cleanup(func() { config.DB = previousDB })
	repo := NewPointsRepo()

	items, err := repo.ListForClient(context.Background(), ClientPointsTargets{PackageCode: "pkg-a"})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].ProductCode != "global" || items[1].ProductCode != "package-a" {
		t.Fatalf("ListForClient() products = %#v, want global and package-a", pointProductCodes(items))
	}
	if item, err := repo.GetAppleProduct(context.Background(), "global", "pkg-a"); err != nil || item.ProductCode != "global" {
		t.Fatalf("GetAppleProduct(global) = %#v, %v", item, err)
	}
	if item, err := repo.GetEnabledForPackage(context.Background(), 1, "pkg-a"); err != nil || item.ProductCode != "global" {
		t.Fatalf("GetEnabledForPackage(global) = %#v, %v", item, err)
	}
	if _, err := repo.GetEnabledForPackage(context.Background(), 3, "pkg-a"); err == nil {
		t.Fatal("GetEnabledForPackage(package-b, pkg-a) succeeded, want not found")
	}
}

func pointProductCodes(items []*model.VideoPoint) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, item.ProductCode)
	}
	return result
}
