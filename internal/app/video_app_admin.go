package app

import (
	"errors"

	"ai-video/internal/config"
	"ai-video/internal/gen/model"

	"gorm.io/gorm"
)

func MigrateVideoAppTable(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	return db.AutoMigrate(&model.VideoApp{})
}

type videoAppAPISeed struct{ Path, Method, Description string }

func SeedVideoAppAdmin() error {
	return config.DB.Transaction(func(tx *gorm.DB) error {
		seeds := []videoAppAPISeed{
			{Path: "/admin/apps", Method: "GET", Description: "应用列表"},
			{Path: "/admin/apps/:id", Method: "GET", Description: "应用详情"},
			{Path: "/admin/apps", Method: "POST", Description: "新增应用"},
			{Path: "/admin/apps/:id", Method: "PUT", Description: "编辑应用"},
			{Path: "/admin/apps/:id", Method: "DELETE", Description: "删除应用"},
		}
		apis := make([]model.VideoAPI, 0, len(seeds))
		for _, seed := range seeds {
			api, err := upsertVideoAppAPI(tx, seed)
			if err != nil {
				return err
			}
			apis = append(apis, *api)
		}
		root, err := upsertPackageMenu(tx, model.VideoMenu{
			ParentID: 0, Name: "包与应用管理", Path: "/package", Icon: "Box",
			Sort: 3, Type: 0, Visible: 1, Status: 1,
		})
		if err != nil {
			return err
		}
		if err := tx.Model(&model.VideoMenu{}).Where("permission = ?", "package:list").Update("sort", 2).Error; err != nil {
			return err
		}
		page, err := upsertPackageMenu(tx, model.VideoMenu{
			ParentID: root.ID, Name: "应用管理", Path: "/package/apps",
			Component: "package/apps/index", Icon: "Grid", Sort: 1, Type: 1,
			Permission: "app:list", Visible: 1, Status: 1,
		})
		if err != nil {
			return err
		}
		if err := tx.Model(page).Association("APIs").Replace(apis[0], apis[1]); err != nil {
			return err
		}

		buttons := []struct {
			menu model.VideoMenu
			apis []model.VideoAPI
		}{
			{menu: model.VideoMenu{ParentID: page.ID, Name: "新增应用", Sort: 1, Type: 2, Permission: "app:add", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[2]}},
			{menu: model.VideoMenu{ParentID: page.ID, Name: "编辑应用", Sort: 2, Type: 2, Permission: "app:edit", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[1], apis[3]}},
			{menu: model.VideoMenu{ParentID: page.ID, Name: "删除应用", Sort: 3, Type: 2, Permission: "app:delete", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[4]}},
		}
		menus := []model.VideoMenu{*root, *page}
		for _, seed := range buttons {
			button, err := upsertPackageMenu(tx, seed.menu)
			if err != nil {
				return err
			}
			if err := tx.Model(button).Association("APIs").Replace(seed.apis); err != nil {
				return err
			}
			menus = append(menus, *button)
		}
		var role model.VideoRole
		if err := tx.Where("code = ?", "admin").First(&role).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		return tx.Model(&role).Association("Menus").Append(menus)
	})
}

func upsertVideoAppAPI(tx *gorm.DB, seed videoAppAPISeed) (*model.VideoAPI, error) {
	var api model.VideoAPI
	err := tx.Where("path = ? AND method = ?", seed.Path, seed.Method).First(&api).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		api = model.VideoAPI{Path: seed.Path, Method: seed.Method, Group: "应用管理", Description: seed.Description}
		if err := tx.Create(&api).Error; err != nil {
			return nil, err
		}
	case err != nil:
		return nil, err
	default:
		if err := tx.Model(&api).Updates(map[string]interface{}{"group": "应用管理", "description": seed.Description}).Error; err != nil {
			return nil, err
		}
	}
	return &api, nil
}
