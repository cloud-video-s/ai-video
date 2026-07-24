package app

import (
	"ai-video/internal/config"
	"errors"

	"ai-video/internal/gen/model"

	"gorm.io/gorm"
)

type uploadAPISeed struct {
	Path        string
	Method      string
	Description string
}

func SeedUploadAdmin() error {
	return config.DB.Transaction(func(tx *gorm.DB) error {
		seeds := []uploadAPISeed{
			{Path: "/admin/uploads", Method: "GET", Description: "查询上传记录"},
			{Path: "/admin/uploads/images/batches", Method: "POST", Description: "批量初始化图片上传"},
			{Path: "/admin/uploads/images/:upload_id/chunks/:index", Method: "PUT", Description: "上传图片分片"},
			{Path: "/admin/uploads/images/:upload_id", Method: "GET", Description: "查询图片上传进度"},
			{Path: "/admin/uploads/images/:upload_id/complete", Method: "POST", Description: "完成图片上传"},
			{Path: "/admin/uploads/videos/batches", Method: "POST", Description: "批量初始化视频上传"},
			{Path: "/admin/uploads/videos/:upload_id/chunks/:index", Method: "PUT", Description: "上传视频分片"},
			{Path: "/admin/uploads/videos/:upload_id", Method: "GET", Description: "查询视频上传进度"},
			{Path: "/admin/uploads/videos/:upload_id/complete", Method: "POST", Description: "完成视频上传"},
		}
		apis := make([]model.VideoAPI, 0, len(seeds))
		for _, seed := range seeds {
			var api model.VideoAPI
			err := tx.Where("path = ? AND method = ?", seed.Path, seed.Method).First(&api).Error
			switch {
			case errors.Is(err, gorm.ErrRecordNotFound):
				api = model.VideoAPI{Path: seed.Path, Method: seed.Method, Group: "文件上传", Description: seed.Description}
				if err := tx.Create(&api).Error; err != nil {
					return err
				}
			case err != nil:
				return err
			default:
				if err := tx.Model(&api).Updates(map[string]interface{}{
					"group": "文件上传", "description": seed.Description,
				}).Error; err != nil {
					return err
				}
			}
			apis = append(apis, api)
		}

		var root model.VideoMenu
		if err := tx.Where("path = ? AND type = ?", "/system", 0).First(&root).Error; err != nil {
			return err
		}
		var permission model.VideoMenu
		err := tx.Where("permission = ?", "system:upload").First(&permission).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			permission = model.VideoMenu{
				ParentID: root.ID, Name: "文件上传", Type: 2, Permission: "system:upload",
				Sort: 99, Visible: 0, Status: 1,
			}
			if err := tx.Create(&permission).Error; err != nil {
				return err
			}
		case err != nil:
			return err
		default:
			if err := tx.Model(&permission).Updates(map[string]interface{}{
				"parent_id": root.ID, "name": "文件上传", "type": 2,
				"sort": 99, "visible": 0, "status": 1,
			}).Error; err != nil {
				return err
			}
		}
		if err := replaceMenuAPIs(tx, &permission, apis...); err != nil {
			return err
		}

		var adminRole model.VideoRole
		if err := tx.Where("code = ?", "admin").First(&adminRole).Error; err != nil {
			return err
		}
		return grantRoleMenus(tx, &adminRole, permission)
	})
}
