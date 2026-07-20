package app

import (
	"errors"

	"ai-video/internal/model"

	"gorm.io/gorm"
)

type packageAPISeed struct {
	Path        string
	Method      string
	Description string
}

// SeedPackageAdmin reconciles package-management APIs, menus and super-admin grants.
func SeedPackageAdmin() error {
	return DB.Transaction(func(tx *gorm.DB) error {
		seeds := []packageAPISeed{
			{Path: "/admin/packages", Method: "GET", Description: "包列表"},
			{Path: "/admin/packages/:id", Method: "GET", Description: "包详情"},
			{Path: "/admin/packages", Method: "POST", Description: "新增包"},
			{Path: "/admin/packages/:id", Method: "PUT", Description: "编辑包"},
			{Path: "/admin/packages/:id", Method: "DELETE", Description: "删除包"},
		}
		apis := make([]model.VideoAPI, 0, len(seeds))
		for _, seed := range seeds {
			api, err := upsertPackageAPI(tx, seed)
			if err != nil {
				return err
			}
			apis = append(apis, *api)
		}

		root, err := upsertPackageMenu(tx, model.VideoMenu{
			ParentID: 0, Name: "包管理", Path: "/package", Icon: "Box",
			Sort: 3, Type: 0, Visible: 1, Status: 1,
		})
		if err != nil {
			return err
		}
		page, err := upsertPackageMenu(tx, model.VideoMenu{
			ParentID: root.ID, Name: "安装包管理", Path: "/package/list",
			Component: "package/list/index", Icon: "Download", Sort: 1, Type: 1,
			Permission: "package:list", Visible: 1, Status: 1,
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
			{menu: model.VideoMenu{ParentID: page.ID, Name: "新增包", Sort: 1, Type: 2, Permission: "package:add", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[2]}},
			{menu: model.VideoMenu{ParentID: page.ID, Name: "编辑包", Sort: 2, Type: 2, Permission: "package:edit", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[1], apis[3]}},
			{menu: model.VideoMenu{ParentID: page.ID, Name: "删除包", Sort: 3, Type: 2, Permission: "package:delete", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[4]}},
		}
		menus := []model.VideoMenu{*root, *page}
		for _, seed := range buttonSeeds {
			button, err := upsertPackageMenu(tx, seed.menu)
			if err != nil {
				return err
			}
			if err := tx.Model(button).Association("APIs").Replace(seed.apis); err != nil {
				return err
			}
			menus = append(menus, *button)
		}

		var adminRole model.VideoRole
		if err := tx.Where("code = ?", model.SuperAdminRoleCode).First(&adminRole).Error; err != nil {
			return err
		}
		return tx.Model(&adminRole).Association("Menus").Append(menus)
	})
}

func upsertPackageAPI(tx *gorm.DB, seed packageAPISeed) (*model.VideoAPI, error) {
	var api model.VideoAPI
	err := tx.Where("path = ? AND method = ?", seed.Path, seed.Method).First(&api).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		api = model.VideoAPI{Path: seed.Path, Method: seed.Method, Group: "包管理", Description: seed.Description}
		if err := tx.Create(&api).Error; err != nil {
			return nil, err
		}
	case err != nil:
		return nil, err
	default:
		if err := tx.Model(&api).Updates(map[string]interface{}{
			"group": "包管理", "description": seed.Description,
		}).Error; err != nil {
			return nil, err
		}
	}
	return &api, nil
}

func upsertPackageMenu(tx *gorm.DB, desired model.VideoMenu) (*model.VideoMenu, error) {
	var menu model.VideoMenu
	query := tx
	if desired.Permission != "" {
		query = query.Where("permission = ?", desired.Permission)
	} else {
		query = query.Where("path = ? AND type = ?", desired.Path, desired.Type)
	}
	err := query.First(&menu).Error
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
