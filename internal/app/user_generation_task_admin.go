package app

import (
	"errors"

	"ai-video/internal/config"
	"ai-video/internal/domain"
	"ai-video/internal/gen/model"

	"gorm.io/gorm"
)

// SeedUserGenerationTaskAdmin reconciles the read-only generation-task page
// and its API permissions. It is only called by the explicit metadata seeder.
func SeedUserGenerationTaskAdmin() error {
	return config.DB.Transaction(func(tx *gorm.DB) error {
		seeds := []templateAPISeed{
			{Path: "/admin/user-generation-tasks", Method: "GET", Group: "用户生成任务", Description: "用户生成任务列表"},
			{Path: "/admin/user-generation-tasks/:id", Method: "GET", Group: "用户生成任务", Description: "用户生成任务详情与结果预览"},
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
			ParentID: 0, Name: "用户中心", Path: "/user", Icon: "UserFilled",
			Sort: 2, Type: 0, Visible: 1, Status: 1,
		})
		if err != nil {
			return err
		}
		page, err := upsertTemplateMenu(tx, model.VideoMenu{
			ParentID: root.ID, Name: "生成任务", Path: "/user/generation-tasks",
			Component: "user/generation-tasks/index", Icon: "Film", Sort: 4, Type: 1,
			Permission: "user:generation-task:list", Visible: 1, Status: 1,
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
