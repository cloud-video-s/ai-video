package app

import (
	"errors"

	"ai-video/internal/config"
	"ai-video/internal/domain"
	"ai-video/internal/gen/model"

	"gorm.io/gorm"
)

// SeedChannelAdmin 幂等修复渠道管理 API、菜单权限和超级管理员菜单映射。
func SeedChannelAdmin() error {
	return config.DB.Transaction(func(tx *gorm.DB) error {
		seeds := []templateAPISeed{
			{Path: "/admin/channels", Method: "GET", Group: "渠道管理", Description: "渠道列表"},
			{Path: "/admin/channels/:id", Method: "GET", Group: "渠道管理", Description: "渠道详情"},
			{Path: "/admin/channels", Method: "POST", Group: "渠道管理", Description: "新增渠道"},
			{Path: "/admin/channels/:id", Method: "PUT", Group: "渠道管理", Description: "编辑渠道"},
			{Path: "/admin/channels/:id/status", Method: "PATCH", Group: "渠道管理", Description: "切换渠道状态"},
			{Path: "/admin/channels/:id", Method: "DELETE", Group: "渠道管理", Description: "删除渠道"},
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
			Name: "渠道管理", Path: "/channel", Icon: "Promotion", Sort: 3,
			Type: 0, Visible: 1, Status: 1,
		})
		if err != nil {
			return err
		}
		if err := replaceMenuAPIs(tx, root); err != nil {
			return err
		}

		page, err := upsertTemplateMenu(tx, model.VideoMenu{
			ParentID: root.ID, Name: "渠道列表", Path: "/channel/list",
			Component: "channel/list/index", Icon: "List", Sort: 1, Type: 1,
			Permission: "channel:list", Visible: 1, Status: 1,
		})
		if err != nil {
			return err
		}
		if err := replaceMenuAPIs(tx, page, apis[0], apis[1]); err != nil {
			return err
		}

		buttonSeeds := []struct {
			menu model.VideoMenu
			apis []model.VideoAPI
		}{
			{menu: model.VideoMenu{ParentID: page.ID, Name: "新增渠道", Sort: 1, Type: 2, Permission: "channel:add", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[2]}},
			{menu: model.VideoMenu{ParentID: page.ID, Name: "编辑渠道", Sort: 2, Type: 2, Permission: "channel:edit", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[1], apis[3], apis[4]}},
			{menu: model.VideoMenu{ParentID: page.ID, Name: "删除渠道", Sort: 3, Type: 2, Permission: "channel:delete", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[5]}},
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

		var role model.VideoRole
		err = tx.Where("code = ?", domain.SuperAdminRoleCode).First(&role).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		return grantRoleMenus(tx, &role, menus...)
	})
}
