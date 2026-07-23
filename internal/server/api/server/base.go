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
