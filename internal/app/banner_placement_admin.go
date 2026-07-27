package app

import (
	"errors"

	"ai-video/internal/config"
	"ai-video/internal/gen/model"

	"gorm.io/gorm"
)

// SeedBannerPlacementAdmin reconciles Banner-placement permissions and nests
// the existing Banner list beneath a Banner directory without changing any
// Banner data-model relationship.
func SeedBannerPlacementAdmin() error {
	return config.DB.Transaction(func(tx *gorm.DB) error {
		seeds := []templateAPISeed{
			{Path: "/admin/banner-placements", Method: "GET", Group: "Banner 位置管理", Description: "Banner 位置列表"},
			{Path: "/admin/banner-placements/:id", Method: "GET", Group: "Banner 位置管理", Description: "Banner 位置详情"},
			{Path: "/admin/banner-placements", Method: "POST", Group: "Banner 位置管理", Description: "新增 Banner 位置"},
			{Path: "/admin/banner-placements/:id", Method: "PUT", Group: "Banner 位置管理", Description: "编辑 Banner 位置"},
			{Path: "/admin/banner-placements/:id", Method: "DELETE", Group: "Banner 位置管理", Description: "删除 Banner 位置"},
		}
		apis := make([]model.VideoAPI, 0, len(seeds))
		for _, seed := range seeds {
			api, err := upsertTemplateAPI(tx, seed)
			if err != nil {
				return err
			}
			apis = append(apis, *api)
		}

		root, listPage, roleIDs, err := reconcileBannerMenuRoot(tx)
		if err != nil {
			return err
		}
		placementPage, err := upsertTemplateMenu(tx, model.VideoMenu{
			ParentID: root.ID, Name: "位置管理", Path: "/banner/placements",
			Component: "banner/placements/index", Icon: "Position", Sort: 2, Type: 1,
			Permission: "banner:placement:list", Visible: 1, Status: 1,
		})
		if err != nil {
			return err
		}
		if err := replaceMenuAPIs(tx, placementPage, apis[0], apis[1]); err != nil {
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
			{menu: model.VideoMenu{ParentID: placementPage.ID, Name: "新增 Banner 位置", Sort: 1, Type: 2, Permission: "banner:placement:add", Visible: 1, Status: 1}, apis: addAPIs},
			{menu: model.VideoMenu{ParentID: placementPage.ID, Name: "编辑 Banner 位置", Sort: 2, Type: 2, Permission: "banner:placement:edit", Visible: 1, Status: 1}, apis: editAPIs},
			{menu: model.VideoMenu{ParentID: placementPage.ID, Name: "删除 Banner 位置", Sort: 3, Type: 2, Permission: "banner:placement:delete", Visible: 1, Status: 1}, apis: []model.VideoAPI{apis[4]}},
		}
		placementMenus := []model.VideoMenu{*placementPage}
		for _, seed := range buttonSeeds {
			button, err := upsertTemplateMenu(tx, seed.menu)
			if err != nil {
				return err
			}
			if err := replaceMenuAPIs(tx, button, seed.apis...); err != nil {
				return err
			}
			placementMenus = append(placementMenus, *button)
		}

		for _, roleID := range roleIDs {
			if err := grantRoleMenus(tx, &model.VideoRole{ID: roleID}, *root, *listPage); err != nil {
				return err
			}
		}
		var adminRole model.VideoRole
		if err := tx.Where("code = ?", "admin").First(&adminRole).Error; err != nil {
			return err
		}
		return grantRoleMenus(tx, &adminRole, append([]model.VideoMenu{*root, *listPage}, placementMenus...)...)
	})
}

func reconcileBannerMenuRoot(tx *gorm.DB) (*model.VideoMenu, *model.VideoMenu, []uint64, error) {
	var root model.VideoMenu
	err := tx.Unscoped().Where("path = ? AND type = ?", "/banner", 0).First(&root).Error
	firstMigration := false
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = tx.Unscoped().Where("(permission = ? OR (path = ? AND type = ?))", "banner:list", "/banner/list", 1).
			Order("id ASC").First(&root).Error
		firstMigration = err == nil
	}
	if err != nil {
		return nil, nil, nil, err
	}

	menuIDs := []uint64{root.ID}
	var existingList model.VideoMenu
	if err := tx.Unscoped().Where("permission = ?", "banner:list").First(&existingList).Error; err == nil && existingList.ID != root.ID {
		menuIDs = append(menuIDs, existingList.ID)
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, nil, err
	}
	roleIDs, err := bannerMenuRoleIDs(tx, menuIDs)
	if err != nil {
		return nil, nil, nil, err
	}
	rootAPIs, err := bannerMenuAPIs(tx, root.ID)
	if err != nil {
		return nil, nil, nil, err
	}

	icon := root.Icon
	if icon == "" {
		icon = "Picture"
	}
	if err := tx.Unscoped().Model(&model.VideoMenu{}).Where("id = ?", root.ID).Updates(map[string]interface{}{
		"name": "Banner 管理", "path": "/banner", "component": "", "icon": icon,
		"type": 0, "permission": "", "visible": 1, "status": 1, "deleted_at": nil,
	}).Error; err != nil {
		return nil, nil, nil, err
	}
	root.Name, root.Path, root.Component, root.Icon = "Banner 管理", "/banner", "", icon
	root.Type, root.Permission, root.Visible, root.Status = 0, "", 1, 1
	root.DeletedAt = gorm.DeletedAt{}

	listPage, err := upsertTemplateMenu(tx, model.VideoMenu{
		ParentID: root.ID, Name: "Banner 列表", Path: "/banner/list",
		Component: "banner/list/index", Sort: 1, Type: 1,
		Permission: "banner:list", Visible: 1, Status: 1,
	})
	if err != nil {
		return nil, nil, nil, err
	}
	if firstMigration {
		if err := tx.Model(&model.VideoMenu{}).Where("parent_id = ? AND type = ?", root.ID, 2).
			Update("parent_id", listPage.ID).Error; err != nil {
			return nil, nil, nil, err
		}
	}
	if len(rootAPIs) > 0 {
		if err := replaceMenuAPIs(tx, listPage, rootAPIs...); err != nil {
			return nil, nil, nil, err
		}
	}
	if err := replaceMenuAPIs(tx, &root); err != nil {
		return nil, nil, nil, err
	}
	return &root, listPage, roleIDs, nil
}

func bannerMenuRoleIDs(tx *gorm.DB, menuIDs []uint64) ([]uint64, error) {
	var rows []model.VideoRoleMenu
	if err := tx.Where("video_menu_id IN ?", menuIDs).Find(&rows).Error; err != nil {
		return nil, err
	}
	seen := make(map[uint64]struct{}, len(rows))
	result := make([]uint64, 0, len(rows))
	for _, row := range rows {
		if _, ok := seen[row.VideoRoleID]; ok {
			continue
		}
		seen[row.VideoRoleID] = struct{}{}
		result = append(result, row.VideoRoleID)
	}
	return result, nil
}

func bannerMenuAPIs(tx *gorm.DB, menuID uint64) ([]model.VideoAPI, error) {
	var relations []model.VideoMenuAPI
	if err := tx.Where("video_menu_id = ?", menuID).Find(&relations).Error; err != nil {
		return nil, err
	}
	ids := make([]uint64, 0, len(relations))
	for _, relation := range relations {
		ids = append(ids, relation.VideoAPIID)
	}
	if len(ids) == 0 {
		return []model.VideoAPI{}, nil
	}
	var apis []model.VideoAPI
	if err := tx.Where("id IN ?", ids).Find(&apis).Error; err != nil {
		return nil, err
	}
	return apis, nil
}
