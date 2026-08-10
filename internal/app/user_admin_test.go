package app

import (
	"strings"
	"testing"
	"time"

	"ai-video/internal/gen/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type appUserMenuTestRow struct {
	ID         uint64 `gorm:"primaryKey;autoIncrement"`
	ParentID   uint64
	Name       string
	Path       string
	Component  string
	Icon       string
	Sort       uint64
	Type       uint8 `gorm:"default:1"`
	Permission string
	Visible    uint8
	Status     uint8
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  gorm.DeletedAt
}

func (appUserMenuTestRow) TableName() string { return model.TableNameVideoMenu }

type appUserRoleMenuTestRow struct {
	ID          uint64 `gorm:"primaryKey;autoIncrement"`
	VideoRoleID uint64
	VideoMenuID uint64
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt
}

func (appUserRoleMenuTestRow) TableName() string { return model.TableNameVideoRoleMenu }

type appUserMenuAPITestRow struct {
	ID          uint64 `gorm:"primaryKey;autoIncrement"`
	VideoMenuID uint64
	VideoAPIID  uint64
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt
}

func (appUserMenuAPITestRow) TableName() string { return model.TableNameVideoMenuAPI }

func newAppUserMenuTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "-") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open isolated database: %v", err)
	}
	if err := db.AutoMigrate(
		&appUserMenuTestRow{},
		&appUserRoleMenuTestRow{},
		&appUserMenuAPITestRow{},
	); err != nil {
		t.Fatalf("create isolated schema: %v", err)
	}
	return db
}

func TestUpsertAppUserMenuKeepsOneDirectory(t *testing.T) {
	db := newAppUserMenuTestDB(t)
	desired := model.VideoMenu{
		Name: "用户管理", Path: "/user", Icon: "UserFilled",
		Sort: 2, Type: 0, Visible: 1, Status: 1,
	}

	first, err := upsertAppUserMenu(db, desired)
	if err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	second, err := upsertAppUserMenu(db, desired)
	if err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("directory was duplicated: first ID %d, second ID %d", first.ID, second.ID)
	}

	var rows []appUserMenuTestRow
	if err := db.Where("path = ?", desired.Path).Find(&rows).Error; err != nil {
		t.Fatalf("list directories: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("got %d active directories, want 1", len(rows))
	}
	if rows[0].Type != 0 {
		t.Fatalf("directory type = %d, want 0", rows[0].Type)
	}
}

func TestRemoveDuplicateAppUserMenusRemovesStandaloneLegacyMenu(t *testing.T) {
	db := newAppUserMenuTestDB(t)
	root, err := upsertAppUserMenu(db, model.VideoMenu{
		Name: "用户管理", Path: "/user", Type: 0, Visible: 1, Status: 1,
	})
	if err != nil {
		t.Fatalf("upsert root: %v", err)
	}
	page, err := upsertAppUserMenu(db, model.VideoMenu{
		ParentID: root.ID, Name: "用户中心", Path: "/user/list", Type: 1,
		Permission: "user:user:list", Visible: 1, Status: 1,
	})
	if err != nil {
		t.Fatalf("upsert page: %v", err)
	}

	legacy := appUserMenuTestRow{
		Name: "用户中心管理", Path: "/legacy-user-center", Type: 1,
		Visible: 1, Status: 1,
	}
	if err := db.Create(&legacy).Error; err != nil {
		t.Fatalf("create legacy menu: %v", err)
	}
	child := appUserMenuTestRow{ParentID: legacy.ID, Name: "旧按钮", Type: 2, Status: 1}
	if err := db.Create(&child).Error; err != nil {
		t.Fatalf("create legacy child: %v", err)
	}
	if err := db.Create(&appUserRoleMenuTestRow{VideoRoleID: 7, VideoMenuID: legacy.ID}).Error; err != nil {
		t.Fatalf("create legacy role link: %v", err)
	}
	if err := db.Create(&appUserMenuAPITestRow{VideoMenuID: legacy.ID, VideoAPIID: 9}).Error; err != nil {
		t.Fatalf("create legacy API link: %v", err)
	}

	if err := removeDuplicateAppUserMenus(db, root.ID, page.ID); err != nil {
		t.Fatalf("remove duplicate menus: %v", err)
	}

	var activeLegacy int64
	if err := db.Model(&appUserMenuTestRow{}).Where("id = ?", legacy.ID).Count(&activeLegacy).Error; err != nil {
		t.Fatalf("count active legacy menu: %v", err)
	}
	if activeLegacy != 0 {
		t.Fatalf("legacy menu is still active")
	}
	var movedChild appUserMenuTestRow
	if err := db.First(&movedChild, child.ID).Error; err != nil {
		t.Fatalf("load moved child: %v", err)
	}
	if movedChild.ParentID != page.ID {
		t.Fatalf("child parent ID = %d, want %d", movedChild.ParentID, page.ID)
	}

	var roleLinks, apiLinks int64
	if err := db.Model(&appUserRoleMenuTestRow{}).
		Where("video_role_id = ? AND video_menu_id = ?", 7, page.ID).Count(&roleLinks).Error; err != nil {
		t.Fatalf("count moved role links: %v", err)
	}
	if err := db.Model(&appUserMenuAPITestRow{}).
		Where("video_api_id = ? AND video_menu_id = ?", 9, page.ID).Count(&apiLinks).Error; err != nil {
		t.Fatalf("count moved API links: %v", err)
	}
	if roleLinks != 1 || apiLinks != 1 {
		t.Fatalf("moved links = role:%d API:%d, want 1 each", roleLinks, apiLinks)
	}
}
