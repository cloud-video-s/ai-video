package repository

import (
	"context"
	"reflect"
	"sort"
	"testing"

	"ai-video/internal/config"
	"ai-video/internal/gen/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestBannerListForClientAppliesAllTargetDimensions(t *testing.T) {
	db := openBannerTargetTestDB(t, "banner-client-targets")
	seedBannerTargetTestData(t, db)
	repo := NewBannerRepo()
	ctx := context.Background()

	tests := []struct {
		name    string
		targets ClientBannerTargets
		want    []string
	}{
		{
			name: "matching position country package and selected version",
			targets: ClientBannerTargets{
				PositionKey: "home", CountryCode: "CN", AppCode: "ai-video", PackageCode: "com.example.video",
				VersionCode: "1.0.0", SubscriptionStatus: 1,
			},
			want: []string{"Exact versions", "Global", "Nonmember", "Package all versions"},
		},
		{
			name: "second explicitly selected version",
			targets: ClientBannerTargets{
				PositionKey: "home", CountryCode: "CN", AppCode: "ai-video", PackageCode: "com.example.video",
				VersionCode: "2.0.0", SubscriptionStatus: 1,
			},
			want: []string{"Exact versions", "Global", "Nonmember", "Package all versions"},
		},
		{
			name: "unselected version still matches package all versions",
			targets: ClientBannerTargets{
				PositionKey: "home", CountryCode: "CN", AppCode: "ai-video", PackageCode: "com.example.video",
				VersionCode: "9.9.9", SubscriptionStatus: 1,
			},
			want: []string{"Global", "Nonmember", "Package all versions"},
		},
		{
			name: "specified position excludes other positions",
			targets: ClientBannerTargets{
				PositionKey: "detail", CountryCode: "CN", AppCode: "ai-video", PackageCode: "com.example.video",
				VersionCode: "1.0.0", SubscriptionStatus: 1,
			},
			want: []string{"Detail only", "Global", "Nonmember"},
		},
		{
			name: "specified country excludes other countries",
			targets: ClientBannerTargets{
				PositionKey: "home", CountryCode: "US", AppCode: "ai-video", PackageCode: "com.example.video",
				VersionCode: "1.0.0", SubscriptionStatus: 1,
			},
			want: []string{"Global", "Nonmember", "US only"},
		},
		{
			name: "member audience",
			targets: ClientBannerTargets{
				PositionKey: "home", CountryCode: "CN", AppCode: "ai-video", PackageCode: "com.example.video",
				VersionCode: "1.0.0", SubscriptionStatus: 2,
			},
			want: []string{"Exact versions", "Global", "Member", "Package all versions"},
		},
		{
			name: "missing app only receives global app targets",
			targets: ClientBannerTargets{
				PositionKey: "home", CountryCode: "CN", SubscriptionStatus: 1,
			},
			want: []string{"Global", "Nonmember"},
		},
		{
			name: "different app only receives global app targets",
			targets: ClientBannerTargets{
				PositionKey: "home", CountryCode: "CN", AppCode: "other-app",
				PackageCode: "com.example.video", VersionCode: "1.0.0", SubscriptionStatus: 1,
			},
			want: []string{"Global", "Nonmember"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			list, err := repo.ListForClient(ctx, tt.targets)
			if err != nil {
				t.Fatal(err)
			}
			got := make([]string, len(list))
			for i := range list {
				got[i] = list[i].Name
			}
			sort.Strings(got)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("banner names = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBannerReplaceTargetsWithEmptySelectionsRemovesAllBindings(t *testing.T) {
	db := openBannerTargetTestDB(t, "banner-empty-targets")
	seedBannerTargetTestData(t, db)
	repo := NewBannerRepo()
	ctx := context.Background()
	item := model.VideoBanner{ID: 2}

	if err := repo.ReplaceTargets(ctx, &item, BannerTargetIDs{}); err != nil {
		t.Fatal(err)
	}
	assertActiveBannerRelationCount(t, db, &model.VideoBannerDisplayPosition{}, item.ID, 0)
	assertActiveBannerRelationCount(t, db, &model.VideoBannerCountry{}, item.ID, 0)
	assertActiveBannerRelationCount(t, db, &model.VideoBannerApp{}, item.ID, 0)

	list, err := repo.ListForClient(ctx, ClientBannerTargets{
		PositionKey: "unknown", CountryCode: "JP", AppCode: "unknown-app", PackageCode: "com.unknown",
		VersionCode: "0.0.1", SubscriptionStatus: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	matched := false
	for i := range list {
		if list[i].ID == item.ID {
			matched = true
			break
		}
	}
	if !matched {
		t.Fatal("banner with empty target selections must be delivered globally")
	}
}

func TestBannerLoadAppTargetsGroupsVersionsAndRepresentsAllVersionsAsEmpty(t *testing.T) {
	db := openBannerTargetTestDB(t, "banner-load-app-targets")
	seedBannerTargetTestData(t, db)
	targets, err := NewBannerRepo().LoadAppTargets(context.Background(), []uint64{2, 3})
	if err != nil {
		t.Fatal(err)
	}
	if got := targets[2]; len(got) != 1 || got[0].AppCode != "ai-video" || got[0].PackageCode != "com.example.video" ||
		!reflect.DeepEqual(got[0].VersionCodes, []string{"1.0.0", "2.0.0"}) {
		t.Fatalf("explicit version targets = %#v", got)
	}
	if got := targets[3]; len(got) != 1 || got[0].PackageCode != "com.example.video" || len(got[0].VersionCodes) != 0 {
		t.Fatalf("all-version target = %#v", got)
	}
}

func openBannerTargetTestDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared&_time_format=sqlite"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), TranslateError: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	previous := config.DB
	config.DB = db
	t.Cleanup(func() { config.DB = previous })
	statements := []string{
		`CREATE TABLE video_banner (
			id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, cover_image TEXT NOT NULL,
			remark TEXT, sort INTEGER NOT NULL DEFAULT 0, jump_type INTEGER NOT NULL DEFAULT 1,
			jump_url TEXT, template_id INTEGER, status INTEGER NOT NULL DEFAULT 1,
			subscription_status INTEGER NOT NULL DEFAULT 3,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE video_banner_display_position (
			id INTEGER PRIMARY KEY AUTOINCREMENT, banner_id INTEGER NOT NULL, position_key TEXT NOT NULL,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE video_display_position (
			id INTEGER PRIMARY KEY AUTOINCREMENT, position_name TEXT NOT NULL, position_key TEXT NOT NULL,
			description TEXT, cover_image TEXT NOT NULL, sort INTEGER NOT NULL DEFAULT 0,
			status INTEGER NOT NULL DEFAULT 1, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE video_banner_country (
			id INTEGER PRIMARY KEY AUTOINCREMENT, banner_id INTEGER NOT NULL, country_code TEXT NOT NULL,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE video_country (
			id INTEGER PRIMARY KEY AUTOINCREMENT, code TEXT NOT NULL, name_zh TEXT NOT NULL,
			language TEXT NOT NULL DEFAULT '', status INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE video_banner_app (
			id INTEGER PRIMARY KEY AUTOINCREMENT, banner_id INTEGER NOT NULL, app_code TEXT NOT NULL,
			package_code TEXT, version_code TEXT NOT NULL,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
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
		`CREATE TABLE video_template (id INTEGER PRIMARY KEY AUTOINCREMENT, status INTEGER NOT NULL DEFAULT 1, deleted_at DATETIME)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func seedBannerTargetTestData(t *testing.T, db *gorm.DB) {
	t.Helper()
	positions := []model.VideoDisplayPosition{
		{ID: 1, PositionName: "Home", PositionKey: "home", CoverImage: "/home.jpg", Status: 1},
		{ID: 2, PositionName: "Detail", PositionKey: "detail", CoverImage: "/detail.jpg", Status: 1},
	}
	countries := []model.VideoCountry{
		{ID: 1, Code: "CN", NameZh: "中国", Status: 1},
		{ID: 2, Code: "US", NameZh: "美国", Status: 1},
	}
	app := model.VideoApp{ID: 1, Name: "AI Video", AppCode: "ai-video", Status: 1}
	packages := []model.VideoPackage{
		{ID: 1, PackageName: "Main", PackageCode: "com.example.video", AppCode: app.AppCode, Status: 1},
		{ID: 2, PackageName: "Other", PackageCode: "com.example.other", AppCode: app.AppCode, Status: 1},
	}
	for _, values := range []interface{}{&positions, &countries, &app, &packages} {
		if err := db.Create(values).Error; err != nil {
			t.Fatal(err)
		}
	}
	banners := []model.VideoBanner{
		{ID: 1, Name: "Global", CoverImage: "/global.jpg", JumpType: 1, JumpURL: "/global", Status: 1, SubscriptionStatus: 3},
		{ID: 2, Name: "Exact versions", CoverImage: "/exact.jpg", JumpType: 1, JumpURL: "/exact", Status: 1, SubscriptionStatus: 3},
		{ID: 3, Name: "Package all versions", CoverImage: "/package-all.jpg", JumpType: 1, JumpURL: "/package-all", Status: 1, SubscriptionStatus: 3},
		{ID: 4, Name: "Detail only", CoverImage: "/detail.jpg", JumpType: 1, JumpURL: "/detail", Status: 1, SubscriptionStatus: 3},
		{ID: 5, Name: "US only", CoverImage: "/us.jpg", JumpType: 1, JumpURL: "/us", Status: 1, SubscriptionStatus: 3},
		{ID: 6, Name: "Other package", CoverImage: "/other.jpg", JumpType: 1, JumpURL: "/other", Status: 1, SubscriptionStatus: 3},
		{ID: 7, Name: "Member", CoverImage: "/member.jpg", JumpType: 1, JumpURL: "/member", Status: 1, SubscriptionStatus: 2},
		{ID: 8, Name: "Nonmember", CoverImage: "/nonmember.jpg", JumpType: 1, JumpURL: "/nonmember", Status: 1, SubscriptionStatus: 1},
	}
	if err := db.Create(&banners).Error; err != nil {
		t.Fatal(err)
	}
	positionRows := []model.VideoBannerDisplayPosition{
		{BannerID: 2, PositionKey: "home"}, {BannerID: 3, PositionKey: "home"}, {BannerID: 4, PositionKey: "detail"},
	}
	countryRows := []model.VideoBannerCountry{
		{BannerID: 2, CountryCode: "CN"}, {BannerID: 3, CountryCode: "CN"},
		{BannerID: 4, CountryCode: "CN"}, {BannerID: 5, CountryCode: "US"},
	}
	appRows := []model.VideoBannerApp{
		{BannerID: 2, AppCode: app.AppCode, PackageCode: "com.example.video", VersionCode: "1.0.0"},
		{BannerID: 2, AppCode: app.AppCode, PackageCode: "com.example.video", VersionCode: "2.0.0"},
		{BannerID: 3, AppCode: app.AppCode, PackageCode: "com.example.video", VersionCode: ""},
		{BannerID: 4, AppCode: app.AppCode, PackageCode: "com.example.video", VersionCode: ""},
		{BannerID: 6, AppCode: app.AppCode, PackageCode: "com.example.other", VersionCode: ""},
	}
	for _, values := range []interface{}{&positionRows, &countryRows, &appRows} {
		if err := db.Create(values).Error; err != nil {
			t.Fatal(err)
		}
	}
}

func assertActiveBannerRelationCount(t *testing.T, db *gorm.DB, value interface{}, bannerID uint64, want int64) {
	t.Helper()
	var got int64
	if err := db.Model(value).Where("banner_id = ?", bannerID).Count(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("active relation count for %T = %d, want %d", value, got, want)
	}
}
