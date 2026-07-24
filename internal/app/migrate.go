package app

import (
	"errors"
	"fmt"
	"strings"

	"ai-video/internal/config"
	"ai-video/internal/domain"
	"ai-video/internal/gen/model"
	"ai-video/internal/pkg/utils"

	"gorm.io/gorm"
)

// SeedData 幂等修复后台核心账号、角色、API、菜单和中间表数据。
// 该函数不会修改表结构，也不会在服务启动时自动执行。
func SeedData() error {
	createdDefaultAdmin := false
	err := config.DB.Transaction(func(tx *gorm.DB) error {
		apiByKey := make(map[string]model.VideoAPI)
		for _, desired := range defaultAPIs() {
			api, err := upsertTemplateAPI(tx, templateAPISeed{
				Path: desired.Path, Method: desired.Method,
				Group: desired.Group, Description: desired.Description,
			})
			if err != nil {
				return err
			}
			apiByKey[coreAPIKey(api.Method, api.Path)] = *api
		}

		adminRole, err := upsertSuperAdminRole(tx)
		if err != nil {
			return err
		}
		adminUser, created, err := upsertDefaultAdmin(tx)
		if err != nil {
			return err
		}
		createdDefaultAdmin = created
		if err := grantAdminRoles(tx, adminUser, *adminRole); err != nil {
			return err
		}

		menus, err := seedCoreMenus(tx, apiByKey)
		if err != nil {
			return err
		}
		return grantRoleMenus(tx, adminRole, menus...)
	})
	if err != nil {
		return err
	}
	if createdDefaultAdmin {
		config.Log.Warn("已创建默认管理员 admin/admin123，请登录后立即修改密码")
	}
	config.Log.Info("core admin metadata reconciled")
	return nil
}

// SeedAdminMetadata 是显式的完整后台元数据初始化入口。
// 调用者必须主动执行；admin-server 启动流程不会调用它。
func SeedAdminMetadata() error {
	seeders := []struct {
		name string
		fn   func() error
	}{
		{name: "core", fn: SeedData},
		{name: "video apps", fn: SeedVideoAppAdmin},
		{name: "packages", fn: SeedPackageAdmin},
		{name: "display positions", fn: SeedDisplayPositionAdmin},
		{name: "countries", fn: SeedCountryAdmin},
		{name: "channels", fn: SeedChannelAdmin},
		{name: "VIP subscriptions", fn: SeedVIPSubscriptionAdmin},
		{name: "VIP subscription levels", fn: SeedVIPSubscriptionLevelAdmin},
		{name: "points packages", fn: SeedPointsPackageAdmin},
		{name: "app users", fn: SeedAppUserAdmin},
		{name: "user attribution", fn: SeedUserAttributionAdmin},
		{name: "points ledgers", fn: SeedUserPointsLedgerAdmin},
		{name: "delay configs", fn: SeedDelayConfigAdmin},
		{name: "uploads", fn: SeedUploadAdmin},
	}
	for _, seeder := range seeders {
		if err := seeder.fn(); err != nil {
			return fmt.Errorf("seed %s: %w", seeder.name, err)
		}
	}
	return nil
}

func upsertSuperAdminRole(tx *gorm.DB) (*model.VideoRole, error) {
	var role model.VideoRole
	err := tx.Unscoped().Where("code = ?", domain.SuperAdminRoleCode).First(&role).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		role = model.VideoRole{
			Name: "超级管理员", Code: domain.SuperAdminRoleCode, Sort: 0,
			Status: 1, Remark: "超级管理员，拥有所有权限",
		}
		if err := tx.Omit("Menus").Create(&role).Error; err != nil {
			return nil, err
		}
		return &role, nil
	}
	if err != nil {
		return nil, err
	}
	if err := tx.Unscoped().Model(&model.VideoRole{}).Where("id = ?", role.ID).Updates(map[string]interface{}{
		"name": "超级管理员", "sort": 0, "status": 1,
		"remark": "超级管理员，拥有所有权限", "deleted_at": nil,
	}).Error; err != nil {
		return nil, err
	}
	role.Name, role.Sort, role.Status, role.Remark = "超级管理员", 0, 1, "超级管理员，拥有所有权限"
	role.DeletedAt = gorm.DeletedAt{}
	return &role, nil
}

func upsertDefaultAdmin(tx *gorm.DB) (*model.VideoAdmin, bool, error) {
	var admin model.VideoAdmin
	err := tx.Unscoped().Where("username = ?", "admin").First(&admin).Error
	if err == nil {
		if admin.DeletedAt.Valid {
			if err := tx.Unscoped().Model(&model.VideoAdmin{}).Where("id = ?", admin.ID).Updates(map[string]interface{}{
				"status": 1, "token_version": gorm.Expr("COALESCE(token_version, 0) + 1"), "deleted_at": nil,
			}).Error; err != nil {
				return nil, false, err
			}
			admin.Status = 1
			admin.TokenVersion++
			admin.DeletedAt = gorm.DeletedAt{}
		}
		return &admin, false, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}
	hashed, err := utils.HashPassword("admin123")
	if err != nil {
		return nil, false, err
	}
	admin = model.VideoAdmin{
		Username: "admin", Password: hashed, Nickname: "管理员", Status: 1,
	}
	if err := tx.Omit("Roles").Create(&admin).Error; err != nil {
		return nil, false, err
	}
	return &admin, true, nil
}

func seedCoreMenus(tx *gorm.DB, apiByKey map[string]model.VideoAPI) ([]model.VideoMenu, error) {
	allMenus := make([]model.VideoMenu, 0, 32)
	seed := func(desired model.VideoMenu, apiKeys ...string) (*model.VideoMenu, error) {
		menu, err := upsertTemplateMenu(tx, desired)
		if err != nil {
			return nil, err
		}
		apis := make([]model.VideoAPI, 0, len(apiKeys))
		for _, key := range apiKeys {
			api, ok := apiByKey[key]
			if !ok {
				return nil, fmt.Errorf("core API %s not found", key)
			}
			apis = append(apis, api)
		}
		if err := replaceMenuAPIs(tx, menu, apis...); err != nil {
			return nil, err
		}
		allMenus = append(allMenus, *menu)
		return menu, nil
	}
	key := func(method, path string) string { return coreAPIKey(method, path) }

	root, err := seed(model.VideoMenu{
		Name: "系统管理", Path: "/system", Icon: "Setting", Sort: 1,
		Type: 0, Visible: 1, Status: 1,
	})
	if err != nil {
		return nil, err
	}

	users, err := seed(model.VideoMenu{
		ParentID: root.ID, Name: "账号管理", Path: "/system/admin", Component: "system/admin/index",
		Icon: "User", Sort: 1, Type: 1, Permission: "system:user:list", Visible: 1, Status: 1,
	}, key("GET", "/admin/users"))
	if err != nil {
		return nil, err
	}
	if _, err = seedButton(seed, users.ID, "账号详情", 1, "system:user:query", key("GET", "/admin/users/:id")); err != nil {
		return nil, err
	}
	if _, err = seedButton(seed, users.ID, "新增账号", 2, "system:user:add", key("POST", "/admin/users")); err != nil {
		return nil, err
	}
	if _, err = seedButton(seed, users.ID, "编辑账号", 3, "system:user:edit", key("GET", "/admin/users/:id"), key("PUT", "/admin/users/:id")); err != nil {
		return nil, err
	}
	if _, err = seedButton(seed, users.ID, "删除账号", 4, "system:user:delete", key("DELETE", "/admin/users/:id")); err != nil {
		return nil, err
	}

	roles, err := seed(model.VideoMenu{
		ParentID: root.ID, Name: "角色管理", Path: "/system/role", Component: "system/role/index",
		Icon: "UserFilled", Sort: 2, Type: 1, Permission: "system:role:list", Visible: 1, Status: 1,
	}, key("GET", "/admin/roles"))
	if err != nil {
		return nil, err
	}
	if _, err = seedButton(seed, roles.ID, "角色详情", 1, "system:role:query", key("GET", "/admin/roles/:id")); err != nil {
		return nil, err
	}
	if _, err = seedButton(seed, roles.ID, "新增角色", 2, "system:role:add", key("POST", "/admin/roles")); err != nil {
		return nil, err
	}
	if _, err = seedButton(seed, roles.ID, "编辑与授权角色", 3, "system:role:edit",
		key("GET", "/admin/roles/:id"), key("PUT", "/admin/roles/:id"),
		key("PUT", "/admin/roles/:id/menus"), key("GET", "/admin/roles/:id/apis"), key("PUT", "/admin/roles/:id/apis")); err != nil {
		return nil, err
	}
	if _, err = seedButton(seed, roles.ID, "删除角色", 4, "system:role:delete", key("DELETE", "/admin/roles/:id")); err != nil {
		return nil, err
	}

	menus, err := seed(model.VideoMenu{
		ParentID: root.ID, Name: "菜单管理", Path: "/system/menu", Component: "system/menu/index",
		Icon: "Menu", Sort: 3, Type: 1, Permission: "system:menu:list", Visible: 1, Status: 1,
	})
	if err != nil {
		return nil, err
	}
	if _, err = seedButton(seed, menus.ID, "菜单详情", 1, "system:menu:query", key("GET", "/admin/menus/:id")); err != nil {
		return nil, err
	}
	if _, err = seedButton(seed, menus.ID, "新增菜单", 2, "system:menu:add", key("POST", "/admin/menus")); err != nil {
		return nil, err
	}
	if _, err = seedButton(seed, menus.ID, "编辑菜单", 3, "system:menu:edit", key("GET", "/admin/menus/:id"), key("PUT", "/admin/menus/:id")); err != nil {
		return nil, err
	}
	if _, err = seedButton(seed, menus.ID, "删除菜单", 4, "system:menu:delete", key("DELETE", "/admin/menus/:id")); err != nil {
		return nil, err
	}

	apis, err := seed(model.VideoMenu{
		ParentID: root.ID, Name: "API 管理", Path: "/system/api", Component: "system/api/index",
		Icon: "Connection", Sort: 4, Type: 1, Permission: "system:api:list", Visible: 1, Status: 1,
	}, key("GET", "/admin/apis"))
	if err != nil {
		return nil, err
	}
	if _, err = seedButton(seed, apis.ID, "API 详情", 1, "system:api:query", key("GET", "/admin/apis/:id")); err != nil {
		return nil, err
	}
	if _, err = seedButton(seed, apis.ID, "新增 API", 2, "system:api:add", key("POST", "/admin/apis")); err != nil {
		return nil, err
	}
	if _, err = seedButton(seed, apis.ID, "编辑 API", 3, "system:api:edit", key("GET", "/admin/apis/:id"), key("PUT", "/admin/apis/:id")); err != nil {
		return nil, err
	}
	if _, err = seedButton(seed, apis.ID, "删除 API", 4, "system:api:delete", key("DELETE", "/admin/apis/:id")); err != nil {
		return nil, err
	}

	configs, err := seed(model.VideoMenu{
		ParentID: root.ID, Name: "系统配置", Path: "/system/config", Component: "system/config/index",
		Icon: "Tools", Sort: 5, Type: 1, Permission: "system:config:list", Visible: 1, Status: 1,
	}, key("GET", "/admin/configs"))
	if err != nil {
		return nil, err
	}
	if _, err = seedButton(seed, configs.ID, "新增配置", 1, "system:config:add", key("POST", "/admin/configs")); err != nil {
		return nil, err
	}
	if _, err = seedButton(seed, configs.ID, "编辑配置", 2, "system:config:edit",
		key("PUT", "/admin/configs"), key("PUT", "/admin/configs/:id"), key("POST", "/admin/configs/refresh")); err != nil {
		return nil, err
	}
	if _, err = seedButton(seed, configs.ID, "删除配置", 3, "system:config:delete", key("DELETE", "/admin/configs/:id")); err != nil {
		return nil, err
	}

	logs, err := seed(model.VideoMenu{
		ParentID: root.ID, Name: "操作日志", Path: "/system/operlog", Component: "system/operlog/index",
		Icon: "Document", Sort: 6, Type: 1, Permission: "system:operlog:list", Visible: 1, Status: 1,
	}, key("GET", "/admin/operation-logs"), key("GET", "/admin/operation-logs/:id"))
	if err != nil {
		return nil, err
	}
	if _, err = seedButton(seed, logs.ID, "删除日志", 1, "system:operlog:delete", key("DELETE", "/admin/operation-logs/:id")); err != nil {
		return nil, err
	}
	if _, err = seedButton(seed, logs.ID, "清空日志", 2, "system:operlog:clear", key("DELETE", "/admin/operation-logs")); err != nil {
		return nil, err
	}
	return allMenus, nil
}

type coreMenuSeeder func(model.VideoMenu, ...string) (*model.VideoMenu, error)

func seedButton(seed coreMenuSeeder, parentID uint64, name string, sort uint64, permission string, apiKeys ...string) (*model.VideoMenu, error) {
	return seed(model.VideoMenu{
		ParentID: parentID, Name: name, Sort: sort, Type: 2,
		Permission: permission, Visible: 1, Status: 1,
	}, apiKeys...)
}

func coreAPIKey(method, path string) string {
	return strings.ToUpper(strings.TrimSpace(method)) + " " + strings.TrimSpace(path)
}

func defaultAPIs() []model.VideoAPI {
	return []model.VideoAPI{
		{Path: "/admin/users", Method: "GET", Group: "账号管理", Description: "账号列表"},
		{Path: "/admin/users", Method: "POST", Group: "账号管理", Description: "创建账号"},
		{Path: "/admin/users/:id", Method: "GET", Group: "账号管理", Description: "账号详情"},
		{Path: "/admin/users/:id", Method: "PUT", Group: "账号管理", Description: "更新账号"},
		{Path: "/admin/users/:id", Method: "DELETE", Group: "账号管理", Description: "删除账号"},
		{Path: "/admin/roles", Method: "GET", Group: "角色管理", Description: "角色列表"},
		{Path: "/admin/roles", Method: "POST", Group: "角色管理", Description: "创建角色"},
		{Path: "/admin/roles/:id", Method: "GET", Group: "角色管理", Description: "角色详情"},
		{Path: "/admin/roles/:id", Method: "PUT", Group: "角色管理", Description: "更新角色"},
		{Path: "/admin/roles/:id", Method: "DELETE", Group: "角色管理", Description: "删除角色"},
		{Path: "/admin/roles/:id/menus", Method: "PUT", Group: "角色管理", Description: "分配菜单"},
		{Path: "/admin/roles/:id/apis", Method: "GET", Group: "角色管理", Description: "角色 API 列表"},
		{Path: "/admin/roles/:id/apis", Method: "PUT", Group: "角色管理", Description: "分配角色 API"},
		{Path: "/admin/menus", Method: "POST", Group: "菜单管理", Description: "创建菜单"},
		{Path: "/admin/menus/:id", Method: "GET", Group: "菜单管理", Description: "菜单详情"},
		{Path: "/admin/menus/:id", Method: "PUT", Group: "菜单管理", Description: "更新菜单"},
		{Path: "/admin/menus/:id", Method: "DELETE", Group: "菜单管理", Description: "删除菜单"},
		{Path: "/admin/apis", Method: "GET", Group: "API 管理", Description: "API 列表"},
		{Path: "/admin/apis", Method: "POST", Group: "API 管理", Description: "创建 API"},
		{Path: "/admin/apis/:id", Method: "GET", Group: "API 管理", Description: "API 详情"},
		{Path: "/admin/apis/:id", Method: "PUT", Group: "API 管理", Description: "更新 API"},
		{Path: "/admin/apis/:id", Method: "DELETE", Group: "API 管理", Description: "删除 API"},
		{Path: "/admin/configs", Method: "GET", Group: "系统配置", Description: "配置列表"},
		{Path: "/admin/configs", Method: "POST", Group: "系统配置", Description: "新增配置"},
		{Path: "/admin/configs", Method: "PUT", Group: "系统配置", Description: "批量保存配置"},
		{Path: "/admin/configs/:id", Method: "PUT", Group: "系统配置", Description: "编辑配置"},
		{Path: "/admin/configs/:id", Method: "DELETE", Group: "系统配置", Description: "删除配置"},
		{Path: "/admin/configs/refresh", Method: "POST", Group: "系统配置", Description: "刷新配置缓存"},
		{Path: "/admin/operation-logs", Method: "GET", Group: "操作日志", Description: "日志列表"},
		{Path: "/admin/operation-logs/:id", Method: "GET", Group: "操作日志", Description: "日志详情"},
		{Path: "/admin/operation-logs/:id", Method: "DELETE", Group: "操作日志", Description: "删除日志"},
		{Path: "/admin/operation-logs", Method: "DELETE", Group: "操作日志", Description: "清空日志"},
	}
}
