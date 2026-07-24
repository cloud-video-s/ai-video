package app

import (
	"ai-video/internal/config"
	"ai-video/internal/gen/model"

	"gorm.io/gorm"
)

// SeedVIPSubscriptionLevelAdmin reconciles VIP-level APIs, menus and grants.
// It only manages admin metadata and never changes the database schema.
func SeedVIPSubscriptionLevelAdmin() error {
	return config.DB.Transaction(func(tx *gorm.DB) error {
		seeds := []templateAPISeed{
			{Path: "/admin/vip-subscription-levels", Method: "GET", Group: "VIP 等级管理", Description: "VIP 等级列表"},
			{Path: "/admin/vip-subscription-levels/:id", Method: "GET", Group: "VIP 等级管理", Description: "VIP 等级详情"},
			{Path: "/admin/vip-subscription-levels", Method: "POST", Group: "VIP 等级管理", Description: "新增 VIP 等级"},
			{Path: "/admin/vip-subscription-levels/:id", Method: "PUT", Group: "VIP 等级管理", Description: "编辑 VIP 等级"},
			{Path: "/admin/vip-subscription-levels/:id/status", Method: "PATCH", Group: "VIP 等级管理", Description: "切换 VIP 等级状态"},
			{Path: "/admin/vip-subscription-levels/:id", Method: "DELETE", Group: "VIP 等级管理", Description: "删除 VIP 等级"},
		}
		apis := make([]model.VideoAPI, 0, len(seeds))
		for _, seed := range seeds {
			api, err := upsertTemplateAPI(tx, seed)
			if err != nil {
				return err
			}
			apis = append(apis, *api)
		}

		root, err := upsertVIPSubscriptionMenu(tx, model.VideoMenu{
			ParentID: 0, Name: "订阅管理", Path: "/subscription", Icon: "Wallet", Sort: 4,
			Type: 0, Visible: 1, Status: 1,
		})
		if err != nil {
			return err
		}
		page, err := upsertVIPSubscriptionMenu(tx, model.VideoMenu{
			ParentID: root.ID, Name: "VIP 等级", Path: "/subscription/vip-levels",
			Component: "subscription/vip-levels/index", Icon: "Medal", Sort: 4, Type: 1,
			Permission: "subscription:vip-level:list", Visible: 1, Status: 1,
		})
		if err != nil {
			return err
		}
		if err := replaceMenuAPIs(tx, page, apis[0], apis[1]); err != nil {
			return err
		}

		buttons := []struct {
			menu model.VideoMenu
			apis []model.VideoAPI
		}{
			{menu: model.VideoMenu{ParentID: page.ID, Name: "新增 VIP 等级", Sort: 1, Type: 2, Permission: "subscription:vip-level:add", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[2]}},
			{menu: model.VideoMenu{ParentID: page.ID, Name: "编辑 VIP 等级", Sort: 2, Type: 2, Permission: "subscription:vip-level:edit", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[1], apis[3], apis[4]}},
			{menu: model.VideoMenu{ParentID: page.ID, Name: "删除 VIP 等级", Sort: 3, Type: 2, Permission: "subscription:vip-level:delete", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[5]}},
		}
		menus := []model.VideoMenu{*root, *page}
		for _, seed := range buttons {
			button, err := upsertVIPSubscriptionMenu(tx, seed.menu)
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
