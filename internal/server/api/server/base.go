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
	req.AppName = middleware.GetAPIAppPackage(ctx)
	req.AppVersion = middleware.GetAPIAppVersion(ctx)
	req.ChannelID = middleware.GetAPIChannelID(ctx)
	req.PhoneModel = middleware.GetAPIPhoneModel(ctx)
	req.LoginType = middleware.GetAPILoginType(ctx)
	req.ChannelPackage = middleware.GetAPIChannelPackage(ctx)
	req.AppPackage = middleware.GetAPIAppPackage(ctx)
}

type AccountBaseRequest struct {
	ClientCountry        string     `json:"client_country"`
	ChannelID            string     `json:"channel_id"`
	AppVersion           string     `json:"app_version"`
	AppName              string     `json:"app_name"`
	PhoneModel           string     `json:"phone_model"`
	ChannelPackage       string     `json:"channel_package"`
	AppPackage           string     `json:"app_package" `
	LoginType            uint32     `json:"login_type"`
	FirstOpenedAt        *time.Time `json:"first_opened_at"`
	LastOpenedAt         *time.Time `json:"last_opened_at"`
	AttributionClickedAt *time.Time `json:"attribution_clicked_at"`
}
