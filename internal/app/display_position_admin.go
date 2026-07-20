package app

import (
	"ai-video/internal/model"

	"gorm.io/gorm"
)

// SeedDisplayPositionAdmin reconciles display-position permissions and adds
// the management page beneath the existing template-management directory.
func SeedDisplayPositionAdmin() error {
	return DB.Transaction(func(tx *gorm.DB) error {
		seeds := []templateAPISeed{
			{Path: "/admin/display-positions", Method: "GET", Group: "展示位置管理", Description: "展示位置列表"},
			{Path: "/admin/display-positions/:id", Method: "GET", Group: "展示位置管理", Description: "展示位置详情"},
			{Path: "/admin/display-positions", Method: "POST", Group: "展示位置管理", Description: "新增展示位置"},
			{Path: "/admin/display-positions/:id", Method: "PUT", Group: "展示位置管理", Description: "编辑展示位置"},
			{Path: "/admin/display-positions/:id", Method: "DELETE", Group: "展示位置管理", Description: "删除展示位置"},
		}

		apis := make([]model.VideoAPI, 0, len(seeds))
		for _, seed := range seeds {
			api, err := upsertTemplateAPI(tx, seed)
			if err != nil {
				return err
			}
			apis = append(apis, *api)
		}

		var root model.VideoMenu
		if err := tx.Where("path = ? AND type = ?", "/template", 0).First(&root).Error; err != nil {
			return err
		}
		page, err := upsertTemplateMenu(tx, model.VideoMenu{
			ParentID: root.ID, Name: "展示位置", Path: "/template/positions",
			Component: "template/positions/index", Icon: "Position", Sort: 1, Type: 1,
			Permission: "template:position:list", Visible: 1, Status: 1,
		})
		if err != nil {
			return err
		}
		if err := tx.Model(page).Association("APIs").Replace(apis[0], apis[1]); err != nil {
			return err
		}
		var imageUploadAPIs []model.VideoAPI
		if err := tx.Where("path LIKE ?", "/admin/uploads/images/%").Find(&imageUploadAPIs).Error; err != nil {
			return err
		}
		addAPIs := append([]model.VideoAPI{apis[2]}, imageUploadAPIs...)
		editAPIs := append([]model.VideoAPI{apis[1], apis[3]}, imageUploadAPIs...)

		buttonSeeds := []struct {
			menu model.VideoMenu
			apis []model.VideoAPI
		}{
			{menu: model.VideoMenu{ParentID: page.ID, Name: "新增展示位置", Sort: 1, Type: 2, Permission: "template:position:add", Visible: 1, Status: 1}, apis: addAPIs},
			{menu: model.VideoMenu{ParentID: page.ID, Name: "编辑展示位置", Sort: 2, Type: 2, Permission: "template:position:edit", Visible: 1, Status: 1}, apis: editAPIs},
			{menu: model.VideoMenu{ParentID: page.ID, Name: "删除展示位置", Sort: 3, Type: 2, Permission: "template:position:delete", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[4]}},
		}
		menus := []model.VideoMenu{*page}
		for _, seed := range buttonSeeds {
			button, err := upsertTemplateMenu(tx, seed.menu)
			if err != nil {
				return err
			}
			if err := tx.Model(button).Association("APIs").Replace(seed.apis); err != nil {
				return err
			}
			menus = append(menus, *button)
		}

		var adminRole model.VideoRole
		if err := tx.Where("code = ?", model.SuperAdminRoleCode).First(&adminRole).Error; err != nil {
			return err
		}
		return tx.Model(&adminRole).Association("Menus").Append(menus)
	})
}
