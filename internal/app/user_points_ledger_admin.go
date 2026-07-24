package app

import (
	"ai-video/internal/config"
	"ai-video/internal/gen/model"

	"gorm.io/gorm"
)

// SeedUserPointsLedgerAdmin adds the read-only points-ledger query page and permissions.
func SeedUserPointsLedgerAdmin() error {
	return config.DB.Transaction(func(tx *gorm.DB) error {
		seeds := []templateAPISeed{
			{Path: "/admin/user-points-ledgers", Method: "GET", Group: "用户积分明细", Description: "积分明细列表与汇总"},
			{Path: "/admin/user-points-ledgers/:id", Method: "GET", Group: "用户积分明细", Description: "积分明细详情"},
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
			ParentID: 0, Name: "订阅管理", Path: "/subscription", Icon: "Wallet", Sort: 4, Type: 0, Visible: 1, Status: 1,
		})
		if err != nil {
			return err
		}
		page, err := upsertVIPSubscriptionMenu(tx, model.VideoMenu{
			ParentID: root.ID, Name: "积分明细", Path: "/subscription/points-ledger",
			Component: "subscription/points-ledger/index", Icon: "Tickets", Sort: 3, Type: 1,
			Permission: "subscription:points-ledger:list", Visible: 1, Status: 1,
		})
		if err != nil {
			return err
		}
		if err := replaceMenuAPIs(tx, page, apis...); err != nil {
			return err
		}

		var adminRole model.VideoRole
		if err := tx.Where("code = ?", "admin").First(&adminRole).Error; err != nil {
			return err
		}
		return grantRoleMenus(tx, &adminRole, *root, *page)
	})
}
