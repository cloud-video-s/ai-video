package app

import (
	"ai-video/internal/config"
	"errors"

	"ai-video/internal/gen/model"

	"gorm.io/gorm"
)

type appUserAPISeed struct {
	Path        string
	Method      string
	Description string
}

// SeedAppUserAdmin reconciles client-user management permissions and menus.
func SeedAppUserAdmin() error {
	return config.DB.Transaction(func(tx *gorm.DB) error {
		seeds := []appUserAPISeed{
			{Path: "/admin/app-users", Method: "GET", Description: "客户端用户列表"},
			{Path: "/admin/app-users/:id", Method: "GET", Description: "客户端用户详情"},
			{Path: "/admin/app-users", Method: "POST", Description: "新增客户端用户"},
			{Path: "/admin/app-users/:id", Method: "PUT", Description: "编辑客户端用户"},
			{Path: "/admin/app-users/:id", Method: "DELETE", Description: "删除客户端用户"},
		}
		apis := make([]model.VideoAPI, 0, len(seeds))
		for _, seed := range seeds {
			api, err := upsertAppUserAPI(tx, seed)
			if err != nil {
				return err
			}
			apis = append(apis, *api)
		}

		var root model.VideoMenu
		if err := tx.Where("path = ? AND type = ?", "/system", 0).First(&root).Error; err != nil {
			return err
		}
		page, err := upsertAppUserMenu(tx, model.VideoMenu{
			ParentID: root.ID, Name: "客户端用户", Path: "/system/app-user",
			Component: "system/app-user/index", Icon: "Avatar", Sort: 2, Type: 1,
			Permission: "system:app-user:list", Visible: 1, Status: 1,
		})
		if err != nil {
			return err
		}
		if err := tx.Model(page).Association("APIs").Replace(apis[0], apis[1]); err != nil {
			return err
		}

		buttonSeeds := []struct {
			menu model.VideoMenu
			apis []model.VideoAPI
		}{
			{menu: model.VideoMenu{ParentID: page.ID, Name: "新增客户端用户", Sort: 1, Type: 2, Permission: "system:app-user:add", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[2]}},
			{menu: model.VideoMenu{ParentID: page.ID, Name: "编辑客户端用户", Sort: 2, Type: 2, Permission: "system:app-user:edit", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[1], apis[3]}},
			{menu: model.VideoMenu{ParentID: page.ID, Name: "删除客户端用户", Sort: 3, Type: 2, Permission: "system:app-user:delete", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[4]}},
		}
		menus := []model.VideoMenu{*page}
		for _, seed := range buttonSeeds {
			button, err := upsertAppUserMenu(tx, seed.menu)
			if err != nil {
				return err
			}
			if err := tx.Model(button).Association("APIs").Replace(seed.apis); err != nil {
				return err
			}
			menus = append(menus, *button)
		}

		var adminRole model.VideoRole
		if err := tx.Where("code = ?", "admin").First(&adminRole).Error; err != nil {
			return err
		}
		return tx.Model(&adminRole).Association("Menus").Append(menus)
	})
}

func upsertAppUserAPI(tx *gorm.DB, seed appUserAPISeed) (*model.VideoAPI, error) {
	var api model.VideoAPI
	err := tx.Where("path = ? AND method = ?", seed.Path, seed.Method).First(&api).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		api = model.VideoAPI{Path: seed.Path, Method: seed.Method, Group: "客户端用户管理", Description: seed.Description}
		if err := tx.Create(&api).Error; err != nil {
			return nil, err
		}
	case err != nil:
		return nil, err
	default:
		if err := tx.Model(&api).Updates(map[string]interface{}{
			"group": "客户端用户管理", "description": seed.Description,
		}).Error; err != nil {
			return nil, err
		}
	}
	return &api, nil
}

func upsertAppUserMenu(tx *gorm.DB, desired model.VideoMenu) (*model.VideoMenu, error) {
	var menu model.VideoMenu
	err := tx.Where("permission = ?", desired.Permission).First(&menu).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := tx.Create(&desired).Error; err != nil {
			return nil, err
		}
		return &desired, nil
	}
	if err != nil {
		return nil, err
	}
	if err := tx.Model(&menu).Updates(map[string]interface{}{
		"parent_id": desired.ParentID, "name": desired.Name, "path": desired.Path,
		"component": desired.Component, "icon": desired.Icon, "sort": desired.Sort,
		"type": desired.Type, "visible": desired.Visible, "status": desired.Status,
	}).Error; err != nil {
		return nil, err
	}
	return &menu, nil
}
