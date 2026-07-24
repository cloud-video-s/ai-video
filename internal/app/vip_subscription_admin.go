package app

import (
	"ai-video/internal/config"
	"errors"

	"ai-video/internal/gen/model"

	"gorm.io/gorm"
)

type vipSubscriptionAPISeed struct{ Path, Method, Description string }

// SeedVIPSubscriptionAdmin reconciles VIP subscription APIs, menus and grants.
func SeedVIPSubscriptionAdmin() error {
	return config.DB.Transaction(func(tx *gorm.DB) error {
		seeds := []vipSubscriptionAPISeed{
			{Path: "/admin/vip-subscriptions", Method: "GET", Description: "VIP 订阅套餐列表"},
			{Path: "/admin/vip-subscriptions/:id", Method: "GET", Description: "VIP 订阅套餐详情"},
			{Path: "/admin/vip-subscriptions", Method: "POST", Description: "新增 VIP 订阅套餐"},
			{Path: "/admin/vip-subscriptions/:id", Method: "PUT", Description: "编辑 VIP 订阅套餐"},
			{Path: "/admin/vip-subscriptions/:id", Method: "DELETE", Description: "删除 VIP 订阅套餐"},
			{Path: "/admin/vip-subscriptions/:id/status", Method: "PATCH", Description: "切换 VIP 套餐状态"},
			{Path: "/admin/vip-subscriptions/:id/display", Method: "PATCH", Description: "切换 VIP 套餐显示模式"},
			{Path: "/admin/vip-subscriptions/:id/default", Method: "PATCH", Description: "设置默认 VIP 套餐"},
			{Path: "/admin/vip-subscriptions/:id/clone", Method: "POST", Description: "复制 VIP 订阅套餐"},
		}
		apis := make([]model.VideoAPI, 0, len(seeds))
		for _, seed := range seeds {
			api, err := upsertVIPSubscriptionAPI(tx, seed)
			if err != nil {
				return err
			}
			apis = append(apis, *api)
		}

		root, err := upsertVIPSubscriptionMenu(tx, model.VideoMenu{ParentID: 0, Name: "订阅管理", Path: "/subscription", Icon: "Wallet", Sort: 4, Type: 0, Visible: 1, Status: 1})
		if err != nil {
			return err
		}
		page, err := upsertVIPSubscriptionMenu(tx, model.VideoMenu{
			ParentID: root.ID, Name: "VIP 订阅", Path: "/subscription/vip", Component: "subscription/vip/index",
			Icon: "Present", Sort: 1, Type: 1, Permission: "subscription:vip:list", Visible: 1, Status: 1,
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
			{menu: model.VideoMenu{ParentID: page.ID, Name: "新增 VIP 套餐", Sort: 1, Type: 2, Permission: "subscription:vip:add", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[2], apis[8]}},
			{menu: model.VideoMenu{ParentID: page.ID, Name: "编辑 VIP 套餐", Sort: 2, Type: 2, Permission: "subscription:vip:edit", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[1], apis[3], apis[5], apis[6], apis[7]}},
			{menu: model.VideoMenu{ParentID: page.ID, Name: "删除 VIP 套餐", Sort: 3, Type: 2, Permission: "subscription:vip:delete", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[4]}},
		}
		menus := []model.VideoMenu{*root, *page}
		for _, seed := range buttonSeeds {
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

func upsertVIPSubscriptionAPI(tx *gorm.DB, seed vipSubscriptionAPISeed) (*model.VideoAPI, error) {
	var api model.VideoAPI
	err := tx.Where("path = ? AND method = ?", seed.Path, seed.Method).First(&api).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		api = model.VideoAPI{Path: seed.Path, Method: seed.Method, Group: "VIP 订阅管理", Description: seed.Description}
		if err := tx.Create(&api).Error; err != nil {
			return nil, err
		}
	case err != nil:
		return nil, err
	default:
		if err := tx.Model(&api).Updates(map[string]interface{}{"group": "VIP 订阅管理", "description": seed.Description}).Error; err != nil {
			return nil, err
		}
	}
	return &api, nil
}

func upsertVIPSubscriptionMenu(tx *gorm.DB, desired model.VideoMenu) (*model.VideoMenu, error) {
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
		"parent_id": desired.ParentID, "name": desired.Name, "path": desired.Path, "component": desired.Component,
		"icon": desired.Icon, "sort": desired.Sort, "type": desired.Type, "visible": desired.Visible, "status": desired.Status,
	}).Error; err != nil {
		return nil, err
	}
	return &menu, nil
}
