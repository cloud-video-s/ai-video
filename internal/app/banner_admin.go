package app

import (
	"errors"

	"ai-video/internal/model"

	"gorm.io/gorm"
)

type bannerAPISeed struct {
	Path        string
	Method      string
	Description string
}

// SeedBannerAdmin reconciles Banner API metadata, its admin page and RBAC permissions.
func SeedBannerAdmin() error {
	return DB.Transaction(func(tx *gorm.DB) error {
		seeds := []bannerAPISeed{
			{Path: "/admin/banners", Method: "GET", Description: "Banner 列表"},
			{Path: "/admin/banners/:id", Method: "GET", Description: "Banner 详情"},
			{Path: "/admin/banners", Method: "POST", Description: "新增 Banner"},
			{Path: "/admin/banners/:id", Method: "PUT", Description: "编辑 Banner"},
			{Path: "/admin/banners/:id", Method: "DELETE", Description: "删除 Banner"},
		}
		apis := make([]model.VideoAPI, 0, len(seeds))
		for _, seed := range seeds {
			api, err := upsertBannerAPI(tx, seed)
			if err != nil {
				return err
			}
			apis = append(apis, *api)
		}

		var uploadAPIs []model.VideoAPI
		if err := tx.Where(map[string]interface{}{"group": "文件上传"}).Find(&uploadAPIs).Error; err != nil {
			return err
		}
		var templateListAPI model.VideoAPI
		if err := tx.Where("path = ? AND method = ?", "/admin/templates", "GET").First(&templateListAPI).Error; err != nil {
			return err
		}
		root, err := upsertTemplateMenu(tx, model.VideoMenu{
			ParentID: 0, Name: "模板管理", Path: "/template", Icon: "VideoCamera",
			Sort: 2, Type: 0, Visible: 1, Status: 1,
		})
		if err != nil {
			return err
		}
		page, err := upsertBannerPermission(tx, model.VideoMenu{
			ParentID: root.ID, Name: "Banner 管理", Path: "/template/banners",
			Component: "template/banners/index", Icon: "PictureFilled", Sort: 4,
			Type: 1, Permission: "banner:list", Visible: 1, Status: 1,
		})
		if err != nil {
			return err
		}
		if err := tx.Model(page).Association("APIs").Replace(apis[0], apis[1], templateListAPI); err != nil {
			return err
		}
		items := []struct {
			menu model.VideoMenu
			apis []model.VideoAPI
		}{
			{menu: model.VideoMenu{ParentID: page.ID, Name: "新增 Banner", Type: 2, Permission: "banner:add", Sort: 1, Visible: 1, Status: 1}, apis: append([]model.VideoAPI{apis[2], templateListAPI}, uploadAPIs...)},
			{menu: model.VideoMenu{ParentID: page.ID, Name: "编辑 Banner", Type: 2, Permission: "banner:edit", Sort: 2, Visible: 1, Status: 1}, apis: append([]model.VideoAPI{apis[1], apis[3], templateListAPI}, uploadAPIs...)},
			{menu: model.VideoMenu{ParentID: page.ID, Name: "删除 Banner", Type: 2, Permission: "banner:delete", Sort: 3, Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[4]}},
		}
		menus := []model.VideoMenu{*root, *page}
		for _, item := range items {
			menu, err := upsertBannerPermission(tx, item.menu)
			if err != nil {
				return err
			}
			if err := tx.Model(menu).Association("APIs").Replace(item.apis); err != nil {
				return err
			}
			menus = append(menus, *menu)
		}

		var adminRole model.VideoRole
		if err := tx.Where("code = ?", model.SuperAdminRoleCode).First(&adminRole).Error; err != nil {
			return err
		}
		return tx.Model(&adminRole).Association("Menus").Append(menus)
	})
}

func upsertBannerAPI(tx *gorm.DB, seed bannerAPISeed) (*model.VideoAPI, error) {
	var api model.VideoAPI
	err := tx.Where("path = ? AND method = ?", seed.Path, seed.Method).First(&api).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		api = model.VideoAPI{Path: seed.Path, Method: seed.Method, Group: "Banner管理", Description: seed.Description}
		if err := tx.Create(&api).Error; err != nil {
			return nil, err
		}
	case err != nil:
		return nil, err
	default:
		if err := tx.Model(&api).Updates(map[string]interface{}{
			"group": "Banner管理", "description": seed.Description,
		}).Error; err != nil {
			return nil, err
		}
	}
	return &api, nil
}

func upsertBannerPermission(tx *gorm.DB, desired model.VideoMenu) (*model.VideoMenu, error) {
	var menu model.VideoMenu
	err := tx.Where("permission = ?", desired.Permission).First(&menu).Error
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
		"parent_id": desired.ParentID, "name": desired.Name, "type": desired.Type,
		"path": desired.Path, "component": desired.Component, "icon": desired.Icon,
		"sort": desired.Sort, "visible": desired.Visible, "status": desired.Status,
	}).Error; err != nil {
		return nil, err
	}
	return &menu, nil
}
