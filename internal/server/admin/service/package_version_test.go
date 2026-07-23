package service

import (
	"context"
	"testing"

	"ai-video/internal/config"
	"ai-video/internal/gen/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestPackageAndVersionServiceCRUD(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:package-version-service?mode=memory&cache=shared&_time_format=sqlite"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), TranslateError: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	previous := config.DB
	config.DB = db
	t.Cleanup(func() { config.DB = previous })
	statements := []string{
		`CREATE TABLE video_app (
			id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, app_code TEXT NOT NULL,
			status INTEGER NOT NULL DEFAULT 1, sort INTEGER NOT NULL DEFAULT 0, description TEXT,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE video_package (
			id INTEGER PRIMARY KEY AUTOINCREMENT, package_name TEXT NOT NULL, package_code TEXT NOT NULL,
			app_code TEXT NOT NULL, description TEXT, sort INTEGER NOT NULL DEFAULT 0,
			status INTEGER NOT NULL DEFAULT 1, system_type INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE video_package_version (
			id INTEGER PRIMARY KEY AUTOINCREMENT, package_code TEXT NOT NULL, version_code TEXT NOT NULL,
			download_url TEXT NOT NULL, install_count INTEGER NOT NULL DEFAULT 0,
			download_count INTEGER NOT NULL DEFAULT 0, device_count INTEGER NOT NULL DEFAULT 0,
			description TEXT, status INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	app := model.VideoApp{Name: "AI Video", AppCode: "ai.video", Status: 1}
	if err := db.Omit("CreatedAt", "UpdatedAt").Create(&app).Error; err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	packageService := NewPackageService()
	createdPackage, err := packageService.Create(ctx, &PackagePayload{
		PackageName: "Android", PackageCode: "com.example.video", AppCode: app.AppCode,
		Description: "Android package", Sort: 10, Status: 1, SystemType: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if createdPackage.ID == 0 || createdPackage.AppCode != app.AppCode || createdPackage.SystemType != 2 {
		t.Fatalf("unexpected package: %#v", createdPackage)
	}
	if err := db.Exec("UPDATE video_package SET created_at = NULL, updated_at = NULL").Error; err != nil {
		t.Fatal(err)
	}
	list, total, err := packageService.List(ctx, 1, 20, &ListPackageRequest{AppCode: app.AppCode, PackageCode: createdPackage.PackageCode})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(list) != 1 || list[0].ID != createdPackage.ID {
		t.Fatalf("unexpected package list: total=%d list=%#v", total, list)
	}

	versionService := NewPackageVersionService()
	createdVersion, err := versionService.Create(ctx, &PackageVersionPayload{
		PackageCode: createdPackage.PackageCode, VersionCode: "1.2.0",
		DownloadURL: "https://example.com/video.apk", InstallCount: 1,
		DownloadCount: 2, DeviceCount: 3, Description: "Stable", Status: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if createdVersion.ID == 0 || createdVersion.VersionCode != "1.2.0" || createdVersion.DeviceCount != 3 {
		t.Fatalf("unexpected version: %#v", createdVersion)
	}
	if err := db.Exec("UPDATE video_package_version SET created_at = NULL, updated_at = NULL").Error; err != nil {
		t.Fatal(err)
	}
	resolved, err := packageService.repo.ResolveEnabledTargets(ctx, createdPackage.PackageCode, createdVersion.VersionCode)
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved) != 1 || resolved[0].ID != createdPackage.ID {
		t.Fatalf("enabled package version was not resolved: %#v", resolved)
	}
	if _, err := versionService.Create(ctx, &PackageVersionPayload{
		PackageCode: createdPackage.PackageCode, VersionCode: "1.2.0", DownloadURL: "/duplicate.apk", Status: 1,
	}); err == nil {
		t.Fatal("duplicate package version must be rejected")
	}
	updatedVersion, err := versionService.Update(ctx, createdVersion.ID, &PackageVersionPayload{
		PackageCode: createdPackage.PackageCode, VersionCode: "1.2.1",
		DownloadURL: "/downloads/video.apk", InstallCount: 4,
		DownloadCount: 5, DeviceCount: 6, Description: "Updated", Status: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updatedVersion.VersionCode != "1.2.1" || updatedVersion.Status != 2 || updatedVersion.DownloadCount != 5 {
		t.Fatalf("unexpected version update: %#v", updatedVersion)
	}
	if err := db.Exec("UPDATE video_package_version SET created_at = NULL, updated_at = NULL").Error; err != nil {
		t.Fatal(err)
	}
	resolved, err = packageService.repo.ResolveEnabledTargets(ctx, createdPackage.PackageCode, updatedVersion.VersionCode)
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved) != 0 {
		t.Fatalf("disabled package version must not be resolved: %#v", resolved)
	}
	status := uint32(2)
	versions, versionTotal, err := versionService.List(ctx, 1, 20, &ListPackageVersionRequest{
		PackageCode: createdPackage.PackageCode, VersionCode: "1.2.1", Status: &status,
	})
	if err != nil {
		t.Fatal(err)
	}
	if versionTotal != 1 || len(versions) != 1 || versions[0].ID != createdVersion.ID {
		t.Fatalf("unexpected version list: total=%d list=%#v", versionTotal, versions)
	}
	if err := packageService.Delete(ctx, createdPackage.ID); err == nil {
		t.Fatal("package with dependent versions must not be deleted")
	}
	if err := versionService.Delete(ctx, createdVersion.ID); err != nil {
		t.Fatal(err)
	}
}
