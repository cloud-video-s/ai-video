package app

import (
	"errors"

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

func upsertTemplateAPI(tx *gorm.DB, seed templateAPISeed) (*model.VideoAPI, error) {
	var api model.VideoAPI
	err := tx.Where("path = ? AND method = ?", seed.Path, seed.Method).First(&api).Error
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
