package app

import (
	"ai-video/internal/config"
	"errors"

	"ai-video/internal/gen/model"

	"gorm.io/gorm"
)

type templateAPISeed struct {
	Path        string
	Method      string
	Group       string
	Description string
}

// SeedTemplateAdmin reconciles template-management menus, APIs and the
// super-admin grants for both fresh and existing installations.
func SeedTemplateAdmin() error {
	return config.DB.Transaction(func(tx *gorm.DB) error {
		seeds := []templateAPISeed{
			{Path: "/admin/template-types", Method: "GET", Group: "模板分类管理", Description: "模板分类列表"},
			{Path: "/admin/template-types/:id", Method: "GET", Group: "模板分类管理", Description: "模板分类详情"},
			{Path: "/admin/template-types", Method: "POST", Group: "模板分类管理", Description: "新增模板分类"},
			{Path: "/admin/template-types/:id", Method: "PUT", Group: "模板分类管理", Description: "编辑模板分类"},
			{Path: "/admin/template-types/:id", Method: "DELETE", Group: "模板分类管理", Description: "删除模板分类"},
			{Path: "/admin/templates", Method: "GET", Group: "模板管理", Description: "模板列表"},
			{Path: "/admin/templates/:id", Method: "GET", Group: "模板管理", Description: "模板详情"},
			{Path: "/admin/templates", Method: "POST", Group: "模板管理", Description: "新增模板"},
			{Path: "/admin/templates/:id", Method: "PUT", Group: "模板管理", Description: "编辑模板"},
			{Path: "/admin/templates/:id", Method: "DELETE", Group: "模板管理", Description: "删除模板"},
		}

		apis := make([]model.VideoAPI, 0, len(seeds))
		for _, seed := range seeds {
			api, err := upsertTemplateAPI(tx, seed)
			if err != nil {
				return err
			}
			apis = append(apis, *api)
		}
		var uploadAPIs []model.VideoAPI
		if err := tx.Where(map[string]interface{}{"group": "文件上传"}).Find(&uploadAPIs).Error; err != nil {
			return err
		}

		root, err := upsertTemplateMenu(tx, model.VideoMenu{
			ParentID: 0, Name: "模板管理", Path: "/template", Icon: "VideoCamera",
			Sort: 2, Type: 0, Visible: 1, Status: 1,
		})
		if err != nil {
			return err
		}

		typePage, err := upsertTemplateMenu(tx, model.VideoMenu{
			ParentID: root.ID, Name: "模板分类", Path: "/template/types",
			Component: "template/types/index", Icon: "CollectionTag", Sort: 2, Type: 1,
			Permission: "template:type:list", Visible: 1, Status: 1,
		})
		if err != nil {
			return err
		}
		if err := tx.Model(typePage).Association("APIs").Replace(apis[0], apis[1]); err != nil {
			return err
		}

		templatePage, err := upsertTemplateMenu(tx, model.VideoMenu{
			ParentID: root.ID, Name: "视频模板", Path: "/template/list",
			Component: "template/list/index", Icon: "Film", Sort: 3, Type: 1,
			Permission: "template:list", Visible: 1, Status: 1,
		})
		if err != nil {
			return err
		}
		if err := tx.Model(templatePage).Association("APIs").Replace(apis[5], apis[6]); err != nil {
			return err
		}

		buttonSeeds := []struct {
			menu model.VideoMenu
			apis []model.VideoAPI
		}{
			{menu: model.VideoMenu{ParentID: typePage.ID, Name: "新增模板分类", Sort: 1, Type: 2, Permission: "template:type:add", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[2]}},
			{menu: model.VideoMenu{ParentID: typePage.ID, Name: "编辑模板分类", Sort: 2, Type: 2, Permission: "template:type:edit", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[3]}},
			{menu: model.VideoMenu{ParentID: typePage.ID, Name: "删除模板分类", Sort: 3, Type: 2, Permission: "template:type:delete", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[4]}},
			{menu: model.VideoMenu{ParentID: templatePage.ID, Name: "新增模板", Sort: 1, Type: 2, Permission: "template:add", Visible: 1, Status: 1}, apis: append([]model.VideoAPI{apis[7]}, uploadAPIs...)},
			{menu: model.VideoMenu{ParentID: templatePage.ID, Name: "编辑模板", Sort: 2, Type: 2, Permission: "template:edit", Visible: 1, Status: 1}, apis: append([]model.VideoAPI{apis[8]}, uploadAPIs...)},
			{menu: model.VideoMenu{ParentID: templatePage.ID, Name: "删除模板", Sort: 3, Type: 2, Permission: "template:delete", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[9]}},
		}

		allMenus := []model.VideoMenu{*root, *typePage, *templatePage}
		for _, seed := range buttonSeeds {
			button, err := upsertTemplateMenu(tx, seed.menu)
			if err != nil {
				return err
			}
			if err := tx.Model(button).Association("APIs").Replace(seed.apis); err != nil {
				return err
			}
			allMenus = append(allMenus, *button)
		}

		var adminRole model.VideoRole
		if err := tx.Where("code = ?", "admin").First(&adminRole).Error; err != nil {
			return err
		}
		return tx.Model(&adminRole).Association("Menus").Append(allMenus)
	})
}

func upsertTemplateAPI(tx *gorm.DB, seed templateAPISeed) (*model.VideoAPI, error) {
	var api model.VideoAPI
	err := tx.Where("path = ? AND method = ?", seed.Path, seed.Method).First(&api).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		api = model.VideoAPI{Path: seed.Path, Method: seed.Method, Group: seed.Group, Description: seed.Description}
		if err := tx.Create(&api).Error; err != nil {
			return nil, err
		}
	case err != nil:
		return nil, err
	default:
		if err := tx.Model(&api).Updates(map[string]interface{}{
			"group": seed.Group, "description": seed.Description,
		}).Error; err != nil {
			return nil, err
		}
	}
	return &api, nil
}

func upsertTemplateMenu(tx *gorm.DB, desired model.VideoMenu) (*model.VideoMenu, error) {
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
		"parent_id": desired.ParentID, "name": desired.Name, "path": desired.Path,
		"component": desired.Component, "icon": desired.Icon, "sort": desired.Sort,
		"type": desired.Type, "visible": desired.Visible, "status": desired.Status,
	}).Error; err != nil {
		return nil, err
	}
	return &menu, nil
}
