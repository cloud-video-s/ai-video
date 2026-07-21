package app

import (
	"ai-video/internal/model"

	"gorm.io/gorm"
)

// SeedTemplateDisplayConfigAdmin installs the RBAC resources for curating
// concrete templates at client display positions.
func SeedTemplateDisplayConfigAdmin() error {
	return DB.Transaction(func(tx *gorm.DB) error {
		seeds := []templateAPISeed{
			{Path: "/admin/template-display-configs", Method: "GET", Group: "模板展示配置", Description: "模板展示配置列表"},
			{Path: "/admin/template-display-configs/:id", Method: "GET", Group: "模板展示配置", Description: "模板展示配置详情"},
			{Path: "/admin/template-display-configs", Method: "POST", Group: "模板展示配置", Description: "新增模板展示配置"},
			{Path: "/admin/template-display-configs/:id", Method: "PUT", Group: "模板展示配置", Description: "编辑模板展示配置"},
			{Path: "/admin/template-display-configs/:id", Method: "DELETE", Group: "模板展示配置", Description: "删除模板展示配置"},
		}
		apis := make([]model.VideoAPI, 0, len(seeds))
		for _, seed := range seeds {
			api, err := upsertTemplateAPI(tx, seed)
			if err != nil {
				return err
			}
			apis = append(apis, *api)
		}

		var root model.VideoMenu
		if err := tx.Where("path = ? AND type = ?", "/template", 0).First(&root).Error; err != nil {
			return err
		}
		page, err := upsertTemplateMenu(tx, model.VideoMenu{
			ParentID: root.ID, Name: "模板展示配置", Path: "/template/display-configs",
			Component: "template/display-configs/index", Icon: "Grid", Sort: 5, Type: 1,
			Permission: "template:display-config:list", Visible: 1, Status: 1,
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
			{menu: model.VideoMenu{ParentID: page.ID, Name: "新增模板展示配置", Sort: 1, Type: 2, Permission: "template:display-config:add", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[2]}},
			{menu: model.VideoMenu{ParentID: page.ID, Name: "编辑模板展示配置", Sort: 2, Type: 2, Permission: "template:display-config:edit", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[1], apis[3]}},
			{menu: model.VideoMenu{ParentID: page.ID, Name: "删除模板展示配置", Sort: 3, Type: 2, Permission: "template:display-config:delete", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[4]}},
		}
		menus := []model.VideoMenu{*page}
		for _, seed := range buttonSeeds {
			button, err := upsertTemplateMenu(tx, seed.menu)
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
