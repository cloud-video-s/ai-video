package app

import (
	"errors"

	"ai-video/internal/config"
	"ai-video/internal/domain"
	"ai-video/internal/gen/model"

	"gorm.io/gorm"
)

// SeedTemplateComplaintAdmin reconciles the read-only complaint-management
// page and its API permissions. It is only called by the explicit admin
// metadata seeder and never during application startup.
func SeedTemplateComplaintAdmin() error {
	return config.DB.Transaction(func(tx *gorm.DB) error {
		seeds := []templateAPISeed{
			{Path: "/admin/template-complaints", Method: "GET", Group: "投诉管理", Description: "模板投诉列表与筛选"},
			{Path: "/admin/template-complaints/:id", Method: "GET", Group: "投诉管理", Description: "模板投诉详情"},
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
			ParentID: root.ID, Name: "投诉管理", Path: "/operation/complaints",
			Component: "operation/complaints/index", Icon: "WarningFilled", Sort: 3, Type: 1,
			Permission: "operation:template-complaint:list", Visible: 1, Status: 1,
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
