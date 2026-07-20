package app

import (
	"ai-video/internal/model"

	"gorm.io/gorm"
)

func SeedUserAttributionAdmin() error {
	return DB.Transaction(func(tx *gorm.DB) error {
		seeds := []templateAPISeed{
			{Path: "/admin/user-attributions", Method: "GET", Group: "用户归因", Description: "归因列表"},
			{Path: "/admin/user-attributions/:id", Method: "GET", Group: "用户归因", Description: "归因详情"},
			{Path: "/admin/user-attributions/:id", Method: "PUT", Group: "用户归因", Description: "编辑归因"},
			{Path: "/admin/user-attributions/:id/events", Method: "POST", Group: "用户归因", Description: "记录回传或扣除"},
			{Path: "/admin/user-attributions/sync", Method: "POST", Group: "用户归因", Description: "同步已有用户"},
		}
		apis := make([]model.VideoAPI, 0, len(seeds))
		for _, seed := range seeds {
			api, err := upsertTemplateAPI(tx, seed)
			if err != nil {
				return err
			}
			apis = append(apis, *api)
		}

		page, err := upsertTemplateMenu(tx, model.VideoMenu{
			ParentID: 0, Name: "用户归因", Path: "/attribution/list",
			Component: "attribution/list/index", Icon: "Aim", Sort: 5, Type: 1,
			Permission: "attribution:list", Visible: 1, Status: 1,
		})
		if err != nil {
			return err
		}
		if err := tx.Model(page).Association("APIs").Replace(apis[0], apis[1]); err != nil {
			return err
		}
		edit, err := upsertTemplateMenu(tx, model.VideoMenu{
			ParentID: page.ID, Name: "编辑归因", Sort: 1, Type: 2,
			Permission: "attribution:edit", Visible: 1, Status: 1,
		})
		if err != nil {
			return err
		}
		if err := tx.Model(edit).Association("APIs").Replace(apis[2], apis[3]); err != nil {
			return err
		}
		syncMenu, err := upsertTemplateMenu(tx, model.VideoMenu{
			ParentID: page.ID, Name: "同步用户归因", Sort: 2, Type: 2,
			Permission: "attribution:sync", Visible: 1, Status: 1,
		})
		if err != nil {
			return err
		}
		if err := tx.Model(syncMenu).Association("APIs").Replace(apis[4]); err != nil {
			return err
		}

		var adminRole model.VideoRole
		if err := tx.Where("code = ?", model.SuperAdminRoleCode).First(&adminRole).Error; err != nil {
			return err
		}
		return tx.Model(&adminRole).Association("Menus").Append(page, edit, syncMenu)
	})
}
