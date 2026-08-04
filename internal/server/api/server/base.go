package service

import (
	"ai-video/internal/middleware"
	"ai-video/internal/pkg/utils"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func GetCtxAccountBaseRequest(ctx *gin.Context, req *AccountBaseRequest) {
	deviceCountry := middleware.GetAPIDeviceCountry(ctx)
	if strings.TrimSpace(deviceCountry) == "" && ctx.Request != nil {
		deviceCountry, _ = utils.GetCountryByIP(ctx.ClientIP())
	}
	req.ClientCountry = deviceCountry
	req.AppName = middleware.GetAPIAPPCode(ctx)
	req.AppPackage = middleware.GetAPIAppPackageCode(ctx)
	req.AppVersion = middleware.GetAPIAppVersion(ctx)
	req.PhoneModel = middleware.GetAPIPhoneModel(ctx)
	req.LoginType = middleware.GetAPILoginType(ctx)

}

type AccountBaseRequest struct {
	ClientCountry        string     `json:"client_country"`
	AppName              string     `json:"app_name"`
	AppPackage           string     `json:"app_package" `
	AppVersion           string     `json:"app_version"`
	PhoneModel           string     `json:"phone_model"`
	LoginType            uint32     `json:"login_type"`
	FirstOpenedAt        *time.Time `json:"first_opened_at"`
	LastOpenedAt         *time.Time `json:"last_opened_at"`
	AttributionClickedAt *time.Time `json:"attribution_clicked_at"`
}

type PageResult struct {
	Page       int64 `json:"page"`       // 当前页码（从1开始）
	PageSize   int64 `json:"pageSize"`   // 每页条数
	Total      int64 `json:"total"`      // 总记录数
	TotalPages int64 `json:"totalPages"` // 总页数
	List       any   `json:"list"`       // 当前页的数据列表
}

type BasePage struct {
	Page     int `query:"page" json:"page" form:"page" binding:"omitempty,min=1" default:"1"`
	PageSize int `query:"page_size" json:"page_size" form:"page_size" binding:"omitempty,min=1" default:"10"`
}

func GetPageResponse(page, pageSize, total int64, data any) (*PageResult, error) {
	if pageSize == 0 {
		pageSize = 10
	}
	if page == 0 {
		page = 1
	}
	// 计算总页数
	totalPages := (total + pageSize - 1) / pageSize
	return &PageResult{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
		List:       data,
	}, nil
}
