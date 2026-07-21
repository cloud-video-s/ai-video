package app

import (
	"ai-video/internal/config"
	"ai-video/internal/gen/model"

	"gorm.io/gorm"
)

type delayConfigAPISeed struct {
	Path        string
	Method      string
	Description string
}

// SeedDelayConfigAdmin adds the delay-config menu and API metadata to existing
// installations without re-running the one-shot base RBAC seed.
func SeedDelayConfigAdmin() error {
	return config.DB.Transaction(func(tx *gorm.DB) error {
		apiSeeds := []delayConfigAPISeed{
			{Path: "/admin/delay-configs", Method: "GET", Description: "延迟配置列表"},
			{Path: "/admin/delay-configs/groups", Method: "GET", Description: "延迟配置分组"},
			{Path: "/admin/delay-configs/:id", Method: "GET", Description: "延迟配置详情"},
			{Path: "/admin/delay-configs", Method: "POST", Description: "新增延迟配置"},
			{Path: "/admin/delay-configs/:id", Method: "PUT", Description: "编辑延迟配置"},
			{Path: "/admin/delay-configs/values", Method: "PUT", Description: "批量保存延迟配置"},
			{Path: "/admin/delay-configs/:id", Method: "DELETE", Description: "删除延迟配置"},
			{Path: "/admin/delay-configs/sync", Method: "POST", Description: "同步延迟配置文件"},
		}
		apis := make([]model.VideoAPI, 0, len(apiSeeds))
		for _, seed := range apiSeeds {
			var api model.VideoAPI
			err := tx.Where("path = ? AND method = ?", seed.Path, seed.Method).First(&api).Error
			if err == gorm.ErrRecordNotFound {
				api = model.VideoAPI{Path: seed.Path, Method: seed.Method, Description: seed.Description}
				api.Group = "OB延迟配置"
				if err := tx.Create(&api).Error; err != nil {
					return err
				}
			} else if err != nil {
				return err
			} else if err := tx.Model(&api).Updates(map[string]interface{}{
				"group": "OB延迟配置", "description": seed.Description,
			}).Error; err != nil {
				return err
			}
			apis = append(apis, api)
		}

		var root model.VideoMenu
		if err := tx.Where("path = ? AND type = ?", "/system", 0).First(&root).Error; err != nil {
			return err
		}
		page, err := upsertDelayConfigMenu(tx, model.VideoMenu{
			ParentID:   root.ID,
			Name:       "OB延迟配置",
			Path:       "/system/delay-config",
			Component:  "system/delay-config/index",
			Icon:       "Timer",
			Sort:       6,
			Type:       1,
			Permission: "system:delay-config:list",
			Visible:    1,
			Status:     1,
		})
		if err != nil {
			return err
		}
		if err := tx.Model(page).Association("APIs").Replace(apis[0], apis[1], apis[2]); err != nil {
			return err
		}

		buttons := []struct {
			menu model.VideoMenu
			apis []model.VideoAPI
		}{
			{menu: model.VideoMenu{ParentID: page.ID, Name: "新增延迟配置", Sort: 1, Type: 2, Permission: "system:delay-config:add", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[3]}},
			{menu: model.VideoMenu{ParentID: page.ID, Name: "编辑延迟配置", Sort: 2, Type: 2, Permission: "system:delay-config:edit", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[4], apis[5]}},
			{menu: model.VideoMenu{ParentID: page.ID, Name: "删除延迟配置", Sort: 3, Type: 2, Permission: "system:delay-config:delete", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[6]}},
			{menu: model.VideoMenu{ParentID: page.ID, Name: "同步延迟配置", Sort: 4, Type: 2, Permission: "system:delay-config:sync", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[7]}},
		}
		allMenus := []model.VideoMenu{*page}
		for _, item := range buttons {
			button, err := upsertDelayConfigMenu(tx, item.menu)
			if err != nil {
				return err
			}
			if err := tx.Model(button).Association("APIs").Replace(item.apis); err != nil {
				return err
			}
			allMenus = append(allMenus, *button)
		}

		var adminRole model.VideoRole
		if err := tx.Where("code = ?", "admin").First(&adminRole).Error; err != nil {
			return err
		}
		return tx.Model(&adminRole).Association("Menus").Append(allMenus)
	})
}

func upsertDelayConfigMenu(tx *gorm.DB, desired model.VideoMenu) (*model.VideoMenu, error) {
	var menu model.VideoMenu
	err := tx.Where("permission = ?", desired.Permission).First(&menu).Error
	if err == gorm.ErrRecordNotFound {
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
