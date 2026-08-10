package app

import (
	"errors"

	"ai-video/internal/config"
	"ai-video/internal/gen/model"

	"gorm.io/gorm"
)

type appUserAPISeed struct {
	Path, Method, Description string
}

// SeedAppUserAdmin reconciles the unified client-user center permissions.
func SeedAppUserAdmin() error {
	return config.DB.Transaction(func(tx *gorm.DB) error {
		seeds := []appUserAPISeed{
			{Path: "/admin/app-users", Method: "GET", Description: "客户端用户列表"},
			{Path: "/admin/app-users/:id", Method: "GET", Description: "客户端用户详情"},
			{Path: "/admin/app-users", Method: "POST", Description: "新增客户端用户"},
			{Path: "/admin/app-users/:id", Method: "PUT", Description: "编辑客户端用户"},
			{Path: "/admin/app-users/:id", Method: "DELETE", Description: "删除客户端用户"},
			{Path: "/admin/app-users/lookup", Method: "GET", Description: "按 ID 或邮箱查询用户"},
			{Path: "/admin/app-users/:id/center", Method: "GET", Description: "用户管理中心详情"},
			{Path: "/admin/app-users/:id/frozen", Method: "PATCH", Description: "冻结或解冻用户"},
			{Path: "/admin/app-users/:id/blacklisted", Method: "PATCH", Description: "设置用户黑名单"},
			{Path: "/admin/app-users/:id/phone", Method: "PUT", Description: "绑定用户手机号"},
			{Path: "/admin/app-users/:id/vip", Method: "POST", Description: "添加用户 VIP"},
			{Path: "/admin/app-users/:id/vip/extend", Method: "POST", Description: "延长用户 VIP"},
			{Path: "/admin/app-users/:id/vip/transfer", Method: "POST", Description: "转移用户 VIP"},
			{Path: "/admin/app-users/:id/vip", Method: "DELETE", Description: "终止用户 VIP"},
			{Path: "/admin/app-users/:id/device", Method: "DELETE", Description: "清除用户设备信息"},
		}
		apis := make([]model.VideoAPI, 0, len(seeds))
		for _, seed := range seeds {
			api, err := upsertAppUserAPI(tx, seed)
			if err != nil {
				return err
			}
			apis = append(apis, *api)
		}

		root, err := upsertAppUserMenu(tx, model.VideoMenu{
			ParentID: 0, Name: "用户管理", Path: "/user", Icon: "UserFilled",
			Sort: 2, Type: 0, Visible: 1, Status: 1,
		})
		if err != nil {
			return err
		}
		page, err := upsertAppUserMenu(tx, model.VideoMenu{
			ParentID: root.ID, Name: "用户中心", Path: "/user/list",
			Component: "user/list/index", Icon: "Avatar", Sort: 1, Type: 1,
			Permission: "user:user:list", Visible: 1, Status: 1,
		})
		if err != nil {
			return err
		}
		if err = removeDuplicateAppUserMenus(tx, root.ID, page.ID); err != nil {
			return err
		}
		if err = replaceMenuAPIs(tx, page, apis[0], apis[1], apis[5], apis[6]); err != nil {
			return err
		}

		buttonSeeds := []struct {
			menu model.VideoMenu
			apis []model.VideoAPI
		}{
			{menu: model.VideoMenu{ParentID: page.ID, Name: "新增客户端用户", Sort: 1, Type: 2, Permission: "user:user:add", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[2]}},
			{menu: model.VideoMenu{ParentID: page.ID, Name: "编辑客户端用户", Sort: 2, Type: 2, Permission: "user:user:edit", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[1], apis[3]}},
			{menu: model.VideoMenu{ParentID: page.ID, Name: "删除客户端用户", Sort: 3, Type: 2, Permission: "user:user:delete", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[4]}},
			{menu: model.VideoMenu{ParentID: page.ID, Name: "用户状态与会员操作", Sort: 4, Type: 2, Permission: "user:user:manage", Visible: 1, Status: 1}, apis: apis[7:]},
		}
		menus := []model.VideoMenu{*root, *page}
		for _, seed := range buttonSeeds {
			button, err := upsertAppUserMenu(tx, seed.menu)
			if err != nil {
				return err
			}
			if err = replaceMenuAPIs(tx, button, seed.apis...); err != nil {
				return err
			}
			menus = append(menus, *button)
		}
		// Keep existing attribution and points-ledger records/routes, but group
		// their pages under the same user-center module in the admin navigation.
		if err := tx.Model(&model.VideoMenu{}).
			Where("permission IN ?", []string{"attribution:list", "subscription:points-ledger:list"}).
			Update("parent_id", root.ID).Error; err != nil {
			return err
		}
		var adminRole model.VideoRole
		if err := tx.Where("code = ?", "admin").First(&adminRole).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		return grantRoleMenus(tx, &adminRole, menus...)
	})
}

func upsertAppUserAPI(tx *gorm.DB, seed appUserAPISeed) (*model.VideoAPI, error) {
	var api model.VideoAPI
	err := tx.Where("path = ? AND method = ?", seed.Path, seed.Method).First(&api).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		api = model.VideoAPI{Path: seed.Path, Method: seed.Method, Group: "用户管理中心", Description: seed.Description}
		if err := tx.Create(&api).Error; err != nil {
			return nil, err
		}
	case err != nil:
		return nil, err
	default:
		if err := tx.Model(&api).Updates(map[string]interface{}{"group": "用户管理中心", "description": seed.Description}).Error; err != nil {
			return nil, err
		}
	}
	return &api, nil
}

func upsertAppUserMenu(tx *gorm.DB, desired model.VideoMenu) (*model.VideoMenu, error) {
	requestedType := desired.Type
	var menu model.VideoMenu
	var err error
	if desired.Path != "" {
		// Visible nodes are identified by their stable route. Matching by a
		// renamed permission creates a second menu for the same page.
		err = tx.Where("path = ?", desired.Path).First(&menu).Error
	} else if desired.Type == 2 && desired.Name != "" {
		// Button permissions can be renamed as well; keep the existing button
		// under the same page instead of leaving an obsolete duplicate behind.
		err = tx.Where("parent_id = ? AND name = ? AND type = ?", desired.ParentID, desired.Name, desired.Type).
			First(&menu).Error
	} else if desired.Permission != "" {
		err = tx.Where("permission = ?", desired.Permission).First(&menu).Error
	} else {
		return nil, errors.New("menu path or permission is required")
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := tx.Omit("ParentMenu", "ChildMenus", "APIs").Create(&desired).Error; err != nil {
			return nil, err
		}
		// Type has a database default of 1, so GORM omits the zero value during
		// Create unless the directory type is restored explicitly.
		if requestedType == 0 {
			if err := tx.Model(&model.VideoMenu{}).Where("id = ?", desired.ID).Update("type", 0).Error; err != nil {
				return nil, err
			}
			desired.Type = requestedType
		}
		return &desired, nil
	}
	if err != nil {
		return nil, err
	}
	if err := tx.Model(&menu).Updates(map[string]interface{}{
		"parent_id": desired.ParentID, "name": desired.Name, "path": desired.Path,
		"component": desired.Component, "icon": desired.Icon, "sort": desired.Sort,
		"type": desired.Type, "permission": desired.Permission,
		"visible": desired.Visible, "status": desired.Status,
	}).Error; err != nil {
		return nil, err
	}
	menu.ParentID, menu.Name, menu.Path = desired.ParentID, desired.Name, desired.Path
	menu.Component, menu.Icon, menu.Sort = desired.Component, desired.Icon, desired.Sort
	menu.Type, menu.Permission = desired.Type, desired.Permission
	menu.Visible, menu.Status = desired.Visible, desired.Status
	return &menu, nil
}

// removeDuplicateAppUserMenus removes obsolete visible user-center nodes left
// by older seed runs. Their children and grants are moved to the canonical
// nodes first so deleting the duplicate does not revoke access.
func removeDuplicateAppUserMenus(tx *gorm.DB, rootID, pageID uint64) error {
	if err := mergeDuplicateAppUserMenus(tx, rootID, "path = ?", "/user"); err != nil {
		return err
	}
	if err := mergeDuplicateAppUserMenus(tx, pageID, "path = ?", "/user/list"); err != nil {
		return err
	}
	return mergeDuplicateAppUserMenus(tx, pageID, "parent_id = ? AND name = ?", 0, "用户中心管理")
}

func mergeDuplicateAppUserMenus(tx *gorm.DB, keepID uint64, where string, args ...interface{}) error {
	var duplicates []model.VideoMenu
	if err := tx.Where(where, args...).Where("id <> ?", keepID).Find(&duplicates).Error; err != nil {
		return err
	}
	for _, duplicate := range duplicates {
		if err := tx.Model(&model.VideoMenu{}).Where("parent_id = ?", duplicate.ID).
			Update("parent_id", keepID).Error; err != nil {
			return err
		}
		if err := moveAppUserMenuRelations(tx, duplicate.ID, keepID); err != nil {
			return err
		}
		if err := tx.Delete(&model.VideoMenu{}, duplicate.ID).Error; err != nil {
			return err
		}
	}
	return nil
}

func moveAppUserMenuRelations(tx *gorm.DB, fromID, toID uint64) error {
	var roleIDs []uint64
	if err := tx.Model(&model.VideoRoleMenu{}).Where("video_menu_id = ?", fromID).
		Pluck("video_role_id", &roleIDs).Error; err != nil {
		return err
	}
	for _, roleID := range roleIDs {
		link := model.VideoRoleMenu{VideoRoleID: roleID, VideoMenuID: toID}
		if err := tx.Where("video_role_id = ? AND video_menu_id = ?", roleID, toID).
			FirstOrCreate(&link).Error; err != nil {
			return err
		}
	}

	var apiIDs []uint64
	if err := tx.Model(&model.VideoMenuAPI{}).Where("video_menu_id = ?", fromID).
		Pluck("video_api_id", &apiIDs).Error; err != nil {
		return err
	}
	for _, apiID := range apiIDs {
		link := model.VideoMenuAPI{VideoMenuID: toID, VideoAPIID: apiID}
		if err := tx.Where("video_menu_id = ? AND video_api_id = ?", toID, apiID).
			FirstOrCreate(&link).Error; err != nil {
			return err
		}
	}

	if err := tx.Unscoped().Where("video_menu_id = ?", fromID).
		Delete(&model.VideoRoleMenu{}).Error; err != nil {
		return err
	}
	return tx.Unscoped().Where("video_menu_id = ?", fromID).
		Delete(&model.VideoMenuAPI{}).Error
}
