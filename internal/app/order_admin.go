package app

import (
	"errors"

	"ai-video/internal/config"
	"ai-video/internal/domain"
	"ai-video/internal/gen/model"

	"gorm.io/gorm"
)

// SeedOrderAdmin reconciles the standalone operations directory, its
// read-only order page, API metadata and the super-admin grant. It is only
// called by the explicit SeedAdminMetadata entrypoint.
func SeedOrderAdmin() error {
	return config.DB.Transaction(func(tx *gorm.DB) error {
		seeds := []templateAPISeed{
			{Path: "/admin/orders", Method: "GET", Group: "订单管理", Description: "订单列表、筛选与金额汇总"},
			{Path: "/admin/orders/:id", Method: "GET", Group: "订单管理", Description: "订单详情与下单人信息"},
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
			ParentID: 0, Name: "运营管理", Path: "/operation", Icon: "Operation",
			Sort: 5, Type: 0, Visible: 1, Status: 1,
		})
		if err != nil {
			return err
		}
		page, err := upsertTemplateMenu(tx, model.VideoMenu{
			ParentID: root.ID, Name: "订单管理", Path: "/operation/orders",
			Component: "operation/orders/index", Icon: "ShoppingCart", Sort: 1, Type: 1,
			Permission: "operation:order:list", Visible: 1, Status: 1,
		})
		if err != nil {
			return err
		}
		if err := replaceMenuAPIs(tx, page, apis...); err != nil {
			return err
		}

		var role model.VideoRole
		err = tx.Where("code = ?", domain.SuperAdminRoleCode).First(&role).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		return grantRoleMenus(tx, &role, *root, *page)
	})
}
