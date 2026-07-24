package app

import (
	"ai-video/internal/config"
	"ai-video/internal/gen/model"

	"gorm.io/gorm"
)

// SeedPointsPackageAdmin reconciles points-package APIs, menus and grants.
func SeedPointsPackageAdmin() error {
	return config.DB.Transaction(func(tx *gorm.DB) error {
		seeds := []vipSubscriptionAPISeed{
			{Path: "/admin/points-packages", Method: "GET", Description: "积分套餐列表"},
			{Path: "/admin/points-packages/:id", Method: "GET", Description: "积分套餐详情"},
			{Path: "/admin/points-packages", Method: "POST", Description: "新增积分套餐"},
			{Path: "/admin/points-packages/:id", Method: "PUT", Description: "编辑积分套餐"},
			{Path: "/admin/points-packages/:id", Method: "DELETE", Description: "删除积分套餐"},
			{Path: "/admin/points-packages/:id/status", Method: "PATCH", Description: "切换积分套餐状态"},
			{Path: "/admin/points-packages/:id/default", Method: "PATCH", Description: "设置默认积分套餐"},
		}
		apis := make([]model.VideoAPI, 0, len(seeds))
		for _, seed := range seeds {
			api, err := upsertPointsPackageAPI(tx, seed)
			if err != nil {
				return err
			}
			apis = append(apis, *api)
		}

		root, err := upsertVIPSubscriptionMenu(tx, model.VideoMenu{
			ParentID: 0, Name: "订阅管理", Path: "/subscription", Icon: "Wallet", Sort: 4, Type: 0, Visible: 1, Status: 1,
		})
		if err != nil {
			return err
		}
		page, err := upsertVIPSubscriptionMenu(tx, model.VideoMenu{
			ParentID: root.ID, Name: "积分套餐", Path: "/subscription/points", Component: "subscription/points/index",
			Icon: "Coin", Sort: 2, Type: 1, Permission: "subscription:points:list", Visible: 1, Status: 1,
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
			{menu: model.VideoMenu{ParentID: page.ID, Name: "新增积分套餐", Sort: 1, Type: 2, Permission: "subscription:points:add", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[2]}},
			{menu: model.VideoMenu{ParentID: page.ID, Name: "编辑积分套餐", Sort: 2, Type: 2, Permission: "subscription:points:edit", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[1], apis[3], apis[5], apis[6]}},
			{menu: model.VideoMenu{ParentID: page.ID, Name: "删除积分套餐", Sort: 3, Type: 2, Permission: "subscription:points:delete", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[4]}},
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

func upsertPointsPackageAPI(tx *gorm.DB, seed vipSubscriptionAPISeed) (*model.VideoAPI, error) {
	seedAPI := templateAPISeed{Path: seed.Path, Method: seed.Method, Group: "积分套餐管理", Description: seed.Description}
	return upsertTemplateAPI(tx, seedAPI)
}
