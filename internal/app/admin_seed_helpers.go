package app

import (
	"errors"
	"fmt"
	"strings"

	"ai-video/internal/gen/model"

	"gorm.io/gorm"
)

// templateAPISeed is shared by the independent admin seeders. It lives here
// so those seeders do not depend on any one feature's legacy seed file.
type templateAPISeed struct {
	Path        string
	Method      string
	Group       string
	Description string
}

// replaceMenuAPIs 显式替换 video_menu_api 数据，不解析任何 GORM 关联。
func replaceMenuAPIs(tx *gorm.DB, menu *model.VideoMenu, apis ...model.VideoAPI) error {
	if menu == nil || menu.ID == 0 {
		return fmt.Errorf("menu ID is required")
	}
	if err := tx.Unscoped().Where("video_menu_id = ?", menu.ID).
		Delete(&model.VideoMenuAPI{}).Error; err != nil {
		return err
	}
	rows := make([]model.VideoMenuAPI, 0, len(apis))
	seen := make(map[uint64]struct{}, len(apis))
	for _, api := range apis {
		if api.ID == 0 {
			return fmt.Errorf("API ID is required for menu %d", menu.ID)
		}
		if _, ok := seen[api.ID]; ok {
			continue
		}
		seen[api.ID] = struct{}{}
		rows = append(rows, model.VideoMenuAPI{VideoMenuID: menu.ID, VideoAPIID: api.ID})
	}
	if len(rows) == 0 {
		return nil
	}
	return tx.Create(&rows).Error
}

// grantRoleMenus 幂等授予角色菜单，只重建指定的映射，不影响角色已有的其他菜单。
func grantRoleMenus(tx *gorm.DB, role *model.VideoRole, menus ...model.VideoMenu) error {
	if role == nil || role.ID == 0 {
		return fmt.Errorf("role ID is required")
	}
	seen := make(map[uint64]struct{}, len(menus))
	for _, menu := range menus {
		if menu.ID == 0 {
			return fmt.Errorf("menu ID is required for role %d", role.ID)
		}
		if _, ok := seen[menu.ID]; ok {
			continue
		}
		seen[menu.ID] = struct{}{}
		if err := tx.Unscoped().Where("video_role_id = ? AND video_menu_id = ?", role.ID, menu.ID).
			Delete(&model.VideoRoleMenu{}).Error; err != nil {
			return err
		}
		if err := tx.Create(&model.VideoRoleMenu{VideoRoleID: role.ID, VideoMenuID: menu.ID}).Error; err != nil {
			return err
		}
	}
	return nil
}

// replaceAdminRoles 显式替换后台账号角色映射。
func replaceAdminRoles(tx *gorm.DB, admin *model.VideoAdmin, roles ...model.VideoRole) error {
	if admin == nil || admin.ID == 0 {
		return fmt.Errorf("admin ID is required")
	}
	if err := tx.Unscoped().Where("video_admin_id = ?", admin.ID).
		Delete(&model.VideoAdminRole{}).Error; err != nil {
		return err
	}
	rows := make([]model.VideoAdminRole, 0, len(roles))
	seen := make(map[uint64]struct{}, len(roles))
	for _, role := range roles {
		if role.ID == 0 {
			return fmt.Errorf("role ID is required for admin %d", admin.ID)
		}
		if _, ok := seen[role.ID]; ok {
			continue
		}
		seen[role.ID] = struct{}{}
		rows = append(rows, model.VideoAdminRole{VideoAdminID: admin.ID, VideoRoleID: role.ID})
	}
	if len(rows) == 0 {
		return nil
	}
	return tx.Create(&rows).Error
}

// grantAdminRoles 幂等追加账号角色，不移除账号已有的其他角色。
func grantAdminRoles(tx *gorm.DB, admin *model.VideoAdmin, roles ...model.VideoRole) error {
	if admin == nil || admin.ID == 0 {
		return fmt.Errorf("admin ID is required")
	}
	seen := make(map[uint64]struct{}, len(roles))
	for _, role := range roles {
		if role.ID == 0 {
			return fmt.Errorf("role ID is required for admin %d", admin.ID)
		}
		if _, ok := seen[role.ID]; ok {
			continue
		}
		seen[role.ID] = struct{}{}
		if err := tx.Unscoped().Where("video_admin_id = ? AND video_role_id = ?", admin.ID, role.ID).
			Delete(&model.VideoAdminRole{}).Error; err != nil {
			return err
		}
		if err := tx.Create(&model.VideoAdminRole{VideoAdminID: admin.ID, VideoRoleID: role.ID}).Error; err != nil {
			return err
		}
	}
	return nil
}

func upsertTemplateAPI(tx *gorm.DB, seed templateAPISeed) (*model.VideoAPI, error) {
	seed.Path = strings.TrimSpace(seed.Path)
	seed.Method = strings.ToUpper(strings.TrimSpace(seed.Method))
	seed.Group = strings.TrimSpace(seed.Group)
	seed.Description = strings.TrimSpace(seed.Description)

	var api model.VideoAPI
	err := tx.Unscoped().Where("path = ? AND method = ?", seed.Path, seed.Method).First(&api).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		api = model.VideoAPI{
			Path: seed.Path, Method: seed.Method, Group: seed.Group, Description: seed.Description,
		}
		if err := tx.Create(&api).Error; err != nil {
			return nil, err
		}
	case err != nil:
		return nil, err
	default:
		if err := tx.Unscoped().Model(&model.VideoAPI{}).Where("id = ?", api.ID).Updates(map[string]interface{}{
			"path": seed.Path, "method": seed.Method, "group": seed.Group,
			"description": seed.Description, "deleted_at": nil,
		}).Error; err != nil {
			return nil, err
		}
		api.Path, api.Method = seed.Path, seed.Method
		api.Group, api.Description = seed.Group, seed.Description
		api.DeletedAt = gorm.DeletedAt{}
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
	err := query.Unscoped().First(&menu).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := tx.Omit("ParentMenu", "ChildMenus", "APIs").Create(&desired).Error; err != nil {
			return nil, err
		}
		return &desired, nil
	}
	if err != nil {
		return nil, err
	}
	if err := tx.Unscoped().Model(&model.VideoMenu{}).Where("id = ?", menu.ID).Updates(map[string]interface{}{
		"parent_id": desired.ParentID, "name": desired.Name, "path": desired.Path,
		"component": desired.Component, "icon": desired.Icon, "sort": desired.Sort,
		"type": desired.Type, "permission": desired.Permission,
		"visible": desired.Visible, "status": desired.Status, "deleted_at": nil,
	}).Error; err != nil {
		return nil, err
	}
	desired.ID = menu.ID
	desired.CreatedAt = menu.CreatedAt
	menu = desired
	return &menu, nil
}
