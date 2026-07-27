package app

import (
	"errors"

	"ai-video/internal/config"
	"ai-video/internal/domain"
	"ai-video/internal/gen/model"

	"gorm.io/gorm"
)

// SeedModelAdmin reconciles model-management API metadata, the two visible
// menus, the nested model-configuration permission, and the super-admin grant.
// It is only called by the explicit SeedAdminMetadata entrypoint.
func SeedModelAdmin() error {
	return config.DB.Transaction(func(tx *gorm.DB) error {
		seeds := []templateAPISeed{
			{Path: "/admin/platforms", Method: "GET", Group: "平台管理", Description: "平台列表"},
			{Path: "/admin/platforms/:id", Method: "GET", Group: "平台管理", Description: "平台详情"},
			{Path: "/admin/platforms", Method: "POST", Group: "平台管理", Description: "新增平台"},
			{Path: "/admin/platforms/:id", Method: "PUT", Group: "平台管理", Description: "编辑平台"},
			{Path: "/admin/platforms/:id", Method: "DELETE", Group: "平台管理", Description: "软删除平台"},
			{Path: "/admin/models", Method: "GET", Group: "模型管理", Description: "模型列表"},
			{Path: "/admin/models/:id", Method: "GET", Group: "模型管理", Description: "模型详情"},
			{Path: "/admin/models", Method: "POST", Group: "模型管理", Description: "新增模型"},
			{Path: "/admin/models/:id", Method: "PUT", Group: "模型管理", Description: "编辑模型"},
			{Path: "/admin/models/:id", Method: "DELETE", Group: "模型管理", Description: "软删除模型"},
			{Path: "/admin/models/:id/parameters", Method: "GET", Group: "模型配置", Description: "模型配置列表"},
			{Path: "/admin/models/:id/parameters", Method: "POST", Group: "模型配置", Description: "新增模型配置"},
			{Path: "/admin/models/:id/parameters/:parameter_id", Method: "PUT", Group: "模型配置", Description: "编辑模型配置"},
			{Path: "/admin/models/:id/parameters/:parameter_id", Method: "DELETE", Group: "模型配置", Description: "软删除模型配置"},
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
			Name: "模型管理", Path: "/model", Icon: "Cpu", Sort: 4,
			Type: 0, Visible: 1, Status: 1,
		})
		if err != nil {
			return err
		}
		if err := replaceMenuAPIs(tx, root); err != nil {
			return err
		}

		platformPage, err := upsertTemplateMenu(tx, model.VideoMenu{
			ParentID: root.ID, Name: "平台管理", Path: "/model/platforms",
			Component: "model/platforms/index", Icon: "Connection", Sort: 1, Type: 1,
			Permission: "platform:list", Visible: 1, Status: 1,
		})
		if err != nil {
			return err
		}
		if err := replaceMenuAPIs(tx, platformPage, apis[0], apis[1]); err != nil {
			return err
		}

		modelPage, err := upsertTemplateMenu(tx, model.VideoMenu{
			ParentID: root.ID, Name: "模型列表", Path: "/model/list",
			Component: "model/list/index", Icon: "SetUp", Sort: 2, Type: 1,
			Permission: "model:list", Visible: 1, Status: 1,
		})
		if err != nil {
			return err
		}
		if err := replaceMenuAPIs(tx, modelPage, apis[5], apis[6]); err != nil {
			return err
		}

		buttonSeeds := []struct {
			menu model.VideoMenu
			apis []model.VideoAPI
		}{
			{menu: model.VideoMenu{ParentID: platformPage.ID, Name: "新增平台", Sort: 1, Type: 2, Permission: "platform:add", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[2]}},
			{menu: model.VideoMenu{ParentID: platformPage.ID, Name: "编辑平台", Sort: 2, Type: 2, Permission: "platform:edit", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[1], apis[3]}},
			{menu: model.VideoMenu{ParentID: platformPage.ID, Name: "删除平台", Sort: 3, Type: 2, Permission: "platform:delete", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[4]}},
			{menu: model.VideoMenu{ParentID: modelPage.ID, Name: "新增模型", Sort: 1, Type: 2, Permission: "model:add", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[7]}},
			{menu: model.VideoMenu{ParentID: modelPage.ID, Name: "编辑模型", Sort: 2, Type: 2, Permission: "model:edit", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[6], apis[8]}},
			{menu: model.VideoMenu{ParentID: modelPage.ID, Name: "删除模型", Sort: 3, Type: 2, Permission: "model:delete", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[9]}},
			{menu: model.VideoMenu{ParentID: modelPage.ID, Name: "模型配置", Sort: 4, Type: 2, Permission: "model:config", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[10], apis[11], apis[12], apis[13]}},
		}
		menus := []model.VideoMenu{*root, *platformPage, *modelPage}
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
