package rbactest

import (
	"context"
	"reflect"
	"sort"
	"strings"
	"testing"

	"ai-video/internal/config"
	"ai-video/internal/domain"
	"ai-video/internal/gen/model"
	"ai-video/internal/repository"
	"ai-video/internal/server/admin/service"

	"github.com/casbin/casbin/v2"
	casbinmodel "github.com/casbin/casbin/v2/model"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/gin-gonic/gin/binding"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const testRBACModel = `[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && keyMatch2(r.obj, p.obj) && r.act == p.act || r.sub == "admin"
`

func setupRBAC(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "-") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	statements := []string{
		`CREATE TABLE video_role (
			id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, code TEXT NOT NULL UNIQUE,
			sort INTEGER NOT NULL DEFAULT 0, status INTEGER NOT NULL DEFAULT 1, remark TEXT NOT NULL DEFAULT '',
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME NULL
		)`,
		`CREATE TABLE video_admin (
			id INTEGER PRIMARY KEY AUTOINCREMENT, username TEXT NOT NULL UNIQUE, password TEXT NOT NULL,
			nickname TEXT NOT NULL DEFAULT '', avatar TEXT NOT NULL DEFAULT '', email TEXT NOT NULL DEFAULT '',
			phone TEXT NOT NULL DEFAULT '', status INTEGER NOT NULL DEFAULT 1, token_version INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME NULL
		)`,
		`CREATE TABLE video_menu (
			id INTEGER PRIMARY KEY AUTOINCREMENT, parent_id INTEGER NOT NULL DEFAULT 0, name TEXT NOT NULL,
			path TEXT NOT NULL DEFAULT '', component TEXT NOT NULL DEFAULT '', icon TEXT NOT NULL DEFAULT '',
			sort INTEGER NOT NULL DEFAULT 0, type INTEGER NOT NULL DEFAULT 1, permission TEXT NOT NULL DEFAULT '',
			visible INTEGER NOT NULL DEFAULT 1, status INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME NULL
		)`,
		`CREATE TABLE video_api (
			id INTEGER PRIMARY KEY AUTOINCREMENT, path TEXT NOT NULL, method TEXT NOT NULL,
			"group" TEXT NOT NULL DEFAULT '', description TEXT NOT NULL DEFAULT '',
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME NULL
		)`,
		`CREATE TABLE video_admin_role (
			id INTEGER PRIMARY KEY AUTOINCREMENT, video_admin_id INTEGER NOT NULL, video_role_id INTEGER NOT NULL,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME NULL
		)`,
		`CREATE TABLE video_role_menu (
			id INTEGER PRIMARY KEY AUTOINCREMENT, video_role_id INTEGER NOT NULL, video_menu_id INTEGER NOT NULL,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME NULL
		)`,
		`CREATE TABLE video_menu_api (
			id INTEGER PRIMARY KEY AUTOINCREMENT, video_menu_id INTEGER NOT NULL, video_api_id INTEGER NOT NULL,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME NULL
		)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	adapter, err := gormadapter.NewAdapterByDB(db)
	if err != nil {
		t.Fatal(err)
	}
	rbacModel, err := casbinmodel.NewModelFromString(testRBACModel)
	if err != nil {
		t.Fatal(err)
	}
	enforcer, err := casbin.NewSyncedEnforcer(rbacModel, adapter)
	if err != nil {
		t.Fatal(err)
	}
	if err := enforcer.LoadPolicy(); err != nil {
		t.Fatal(err)
	}
	previousDB, previousEnforcer := config.DB, config.Enforcer
	config.DB, config.Enforcer = db, enforcer
	t.Cleanup(func() {
		config.DB, config.Enforcer = previousDB, previousEnforcer
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func createRoleMenuAPI(t *testing.T, db *gorm.DB, roleCode, permission, path string) (uint64, uint64) {
	t.Helper()
	var roleCount, menuCount, apiCount int64
	db.Model(&model.VideoRole{}).Count(&roleCount)
	db.Model(&model.VideoMenu{}).Count(&menuCount)
	db.Model(&model.VideoAPI{}).Count(&apiCount)
	role := model.VideoRole{ID: uint64(roleCount + 1), Name: roleCode, Code: roleCode, Status: 1}
	menu := model.VideoMenu{ID: uint64(menuCount + 1), Name: permission, Permission: permission, Type: 2, Visible: 1, Status: 1}
	api := model.VideoAPI{ID: uint64(apiCount + 1), Path: path, Method: "GET", Group: "test"}
	if err := db.Omit("Menus").Create(&role).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Omit("ParentMenu", "ChildMenus", "APIs").Create(&menu).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&api).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.VideoMenuAPI{VideoMenuID: menu.ID, VideoAPIID: api.ID}).Error; err != nil {
		t.Fatal(err)
	}
	return role.ID, menu.ID
}

func TestRoleMenusAutomaticallyReplaceAPIPolicies(t *testing.T) {
	db := setupRBAC(t)
	roleID, menuID := createRoleMenuAPI(t, db, "editor", "system:user:list", "/admin/users")
	svc := service.NewRoleService()

	if err := svc.SetMenus(context.Background(), roleID, []uint64{menuID}); err != nil {
		t.Fatal(err)
	}
	allowed, err := config.Enforcer.Enforce("editor", "/admin/users", "GET")
	if err != nil || !allowed {
		t.Fatalf("menu grant was not materialized: allowed=%v err=%v", allowed, err)
	}

	if err := svc.SetMenus(context.Background(), roleID, []uint64{}); err != nil {
		t.Fatal(err)
	}
	allowed, err = config.Enforcer.Enforce("editor", "/admin/users", "GET")
	if err != nil || allowed {
		t.Fatalf("cleared menus still authorize API: allowed=%v err=%v", allowed, err)
	}
	var mappings, policies int64
	if err := db.Model(&model.VideoRoleMenu{}).Where("video_role_id = ?", roleID).Count(&mappings).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.CasbinRule{}).Where("ptype = ? AND v0 = ?", "p", "editor").Count(&policies).Error; err != nil {
		t.Fatal(err)
	}
	if mappings != 0 || policies != 0 {
		t.Fatalf("clear left stale data: mappings=%d policies=%d", mappings, policies)
	}
}

func TestAccountSupportsMultipleRolesAndPermissionUnion(t *testing.T) {
	db := setupRBAC(t)
	roleA, menuA := createRoleMenuAPI(t, db, "auditor", "audit:list", "/admin/audit")
	roleB, menuB := createRoleMenuAPI(t, db, "operator", "operation:edit", "/admin/operation")
	roleService := service.NewRoleService()
	if err := roleService.SetMenus(context.Background(), roleA, []uint64{menuA}); err != nil {
		t.Fatal(err)
	}
	if err := roleService.SetMenus(context.Background(), roleB, []uint64{menuB}); err != nil {
		t.Fatal(err)
	}
	admin := model.VideoAdmin{ID: 1, Username: "multi-role", Password: "not-used", Status: 1}
	if err := db.Omit("Roles").Create(&admin).Error; err != nil {
		t.Fatal(err)
	}
	if err := repository.NewAdminRepo().SetRoles(context.Background(), admin.ID, []uint{uint(roleA), uint(roleB)}); err != nil {
		t.Fatal(err)
	}

	record, err := repository.NewAdminRepo().GetByID(context.Background(), admin.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(record.Roles) != 2 {
		t.Fatalf("roles=%d, want 2", len(record.Roles))
	}
	permissions, err := service.NewMenuService().GetUserPermissions(context.Background(), admin.ID)
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(permissions)
	want := []string{"audit:list", "operation:edit"}
	if !reflect.DeepEqual(permissions, want) {
		t.Fatalf("permissions=%v, want %v", permissions, want)
	}
}

func TestSuperAdminRoleIsImmutableAndHasAllPermissions(t *testing.T) {
	db := setupRBAC(t)
	role := model.VideoRole{ID: 1, Name: "超级管理员", Code: domain.SuperAdminRoleCode, Status: 1}
	if err := db.Omit("Menus").Create(&role).Error; err != nil {
		t.Fatal(err)
	}
	admin := model.VideoAdmin{ID: 1, Username: "root", Password: "not-used", Status: 1}
	if err := db.Omit("Roles").Create(&admin).Error; err != nil {
		t.Fatal(err)
	}
	if err := repository.NewAdminRepo().SetRoles(context.Background(), admin.ID, []uint{uint(role.ID)}); err != nil {
		t.Fatal(err)
	}

	name := "changed"
	if err := service.NewRoleService().Update(context.Background(), role.ID, &service.UpdateRoleRequest{Name: &name}); err == nil {
		t.Fatal("super-admin update unexpectedly succeeded")
	}
	if err := service.NewRoleService().SetMenus(context.Background(), role.ID, []uint64{}); err == nil {
		t.Fatal("super-admin menu update unexpectedly succeeded")
	}
	if err := service.NewRoleService().SetAPIs(context.Background(), role.ID, nil); err == nil {
		t.Fatal("super-admin API update unexpectedly succeeded")
	}
	if err := service.NewRoleService().Delete(context.Background(), role.ID); err == nil {
		t.Fatal("super-admin delete unexpectedly succeeded")
	}
	permissions, err := service.NewMenuService().GetUserPermissions(context.Background(), admin.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(permissions, []string{"*"}) {
		t.Fatalf("permissions=%v, want wildcard", permissions)
	}
}

func TestDeletingRoleRevokesAffectedAdminSessions(t *testing.T) {
	db := setupRBAC(t)
	roleID, menuID := createRoleMenuAPI(t, db, "temporary", "temporary:list", "/admin/temporary")
	if err := service.NewRoleService().SetMenus(context.Background(), roleID, []uint64{menuID}); err != nil {
		t.Fatal(err)
	}
	admin := model.VideoAdmin{ID: 1, Username: "temporary-admin", Password: "not-used", Status: 1, TokenVersion: 7}
	if err := db.Omit("Roles").Create(&admin).Error; err != nil {
		t.Fatal(err)
	}
	if err := repository.NewAdminRepo().SetRoles(context.Background(), admin.ID, []uint{uint(roleID)}); err != nil {
		t.Fatal(err)
	}
	if err := service.NewRoleService().Delete(context.Background(), roleID); err != nil {
		t.Fatal(err)
	}
	version, err := repository.NewAdminRepo().GetTokenVersion(context.Background(), admin.ID)
	if err != nil {
		t.Fatal(err)
	}
	if version != 8 {
		t.Fatalf("token version=%d, want 8", version)
	}
	record, err := repository.NewAdminRepo().GetByID(context.Background(), admin.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(record.Roles) != 0 {
		t.Fatalf("roles=%v, want none", record.Roles)
	}
}

func TestEmptyMenuArrayIsValidButMissingFieldIsRejected(t *testing.T) {
	empty := []uint64{}
	if err := binding.Validator.ValidateStruct(service.SetRoleMenusRequest{MenuIDs: &empty}); err != nil {
		t.Fatalf("empty menu array should clear permissions: %v", err)
	}
	if err := binding.Validator.ValidateStruct(service.SetRoleMenusRequest{}); err == nil {
		t.Fatal("missing menu_ids should be rejected")
	}
}
