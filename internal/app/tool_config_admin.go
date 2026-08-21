package app

import (
	"ai-video/internal/config"
	"ai-video/internal/gen/model"

	"gorm.io/gorm"
)

// SeedToolConfigAdmin reconciles tool-config APIs, menus and default admin
// permissions. It is only reached by the explicit admin-seed command.
func SeedToolConfigAdmin() error {
	return config.DB.Transaction(func(tx *gorm.DB) error {
		seeds := []templateAPISeed{
			{Path: "/admin/tool-configs", Method: "GET", Group: "工具配置", Description: "工具配置列表"},
			{Path: "/admin/tool-configs/:id", Method: "GET", Group: "工具配置", Description: "工具配置详情"},
			{Path: "/admin/tool-configs", Method: "POST", Group: "工具配置", Description: "新增工具配置"},
			{Path: "/admin/tool-configs/:id", Method: "PUT", Group: "工具配置", Description: "编辑工具配置"},
			{Path: "/admin/tool-configs/:id/status", Method: "PATCH", Group: "工具配置", Description: "更新工具上下线状态"},
			{Path: "/admin/tool-configs/:id", Method: "DELETE", Group: "工具配置", Description: "删除工具配置"},
		}

		apis := make([]model.VideoAPI, 0, len(seeds))
		for _, seed := range seeds {
			api, err := upsertTemplateAPI(tx, seed)
			if err != nil {
				return err
			}
			apis = append(apis, *api)
		}

		root, err := upsertTemplateMenu(tx, model.VideoMenu{
			Name: "工具管理", Path: "/tool", Icon: "Tools", Sort: 6,
			Type: 0, Visible: 1, Status: 1,
		})
		if err != nil {
			return err
		}
		page, err := upsertTemplateMenu(tx, model.VideoMenu{
			ParentID: root.ID, Name: "工具配置", Path: "/tool/configs",
			Component: "tool/configs/index", Icon: "SetUp", Sort: 1, Type: 1,
			Permission: "tool:config:list", Visible: 1, Status: 1,
		})
		if err != nil {
			return err
		}
		if err := replaceMenuAPIs(tx, page, apis[0], apis[1]); err != nil {
			return err
		}

		var imageUploadAPIs []model.VideoAPI
		if err := tx.Where("path LIKE ?", "/admin/uploads/images/%").Find(&imageUploadAPIs).Error; err != nil {
			return err
		}
		addAPIs := append([]model.VideoAPI{apis[2]}, imageUploadAPIs...)
		editAPIs := append([]model.VideoAPI{apis[1], apis[3]}, imageUploadAPIs...)
		buttonSeeds := []struct {
			menu model.VideoMenu
			apis []model.VideoAPI
		}{
			{menu: model.VideoMenu{ParentID: page.ID, Name: "新增工具配置", Sort: 1, Type: 2, Permission: "tool:config:add", Visible: 1, Status: 1}, apis: addAPIs},
			{menu: model.VideoMenu{ParentID: page.ID, Name: "编辑工具配置", Sort: 2, Type: 2, Permission: "tool:config:edit", Visible: 1, Status: 1}, apis: editAPIs},
			{menu: model.VideoMenu{ParentID: page.ID, Name: "工具上下线", Sort: 3, Type: 2, Permission: "tool:config:status", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[4]}},
			{menu: model.VideoMenu{ParentID: page.ID, Name: "删除工具配置", Sort: 4, Type: 2, Permission: "tool:config:delete", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[5]}},
		}

		menus := []model.VideoMenu{*root, *page}
		for _, seed := range buttonSeeds {
			button, err := upsertTemplateMenu(tx, seed.menu)
			if err != nil {
				return err
			}
			if err := replaceMenuAPIs(tx, button, seed.apis...); err != nil {
				return err
			}
			menus = append(menus, *button)
		}

		var adminRole model.VideoRole
		if err := tx.Where("code = ?", "admin").First(&adminRole).Error; err != nil {
			return err
		}
		return grantRoleMenus(tx, &adminRole, menus...)
	})
}
