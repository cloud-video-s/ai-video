package middleware

import (
	"ai-video/internal/pkg/cache"
	"ai-video/internal/pkg/jwt"
	"ai-video/internal/pkg/response"
	"ai-video/internal/repository"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	CtxUserIDKey        = "user_id"
	CtxPhoneCodeKey     = "phone_code"
	CtxAPPVersionKey    = "app_version"
	CtxDeviceCountryKey = "device_country"
	CtxChannelIDKey     = "channel_id"
	CtxPhoneModelKey    = "phone_model"
)

func ApiAuth() gin.HandlerFunc {
	userRepo := repository.NewAppUserRepo()
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if strings.TrimSpace(authHeader) == "" {
			response.Unauthorized(c, "缺少 Authorization 头")
			return
		}

		tokenString, ok := extractBearerToken(authHeader)
		if !ok {
			response.Unauthorized(c, "Authorization 格式错误")
			return
		}

		if cache.IsTokenBlacklisted(tokenString) {
			response.Unauthorized(c, "Token 已失效，请重新登录")
			return
		}

		claims, err := jwt.ParseApiToken(tokenString)
		if err != nil {
			response.Unauthorized(c, "Token 无效或已过期")
			return
		}
		version, err := userRepo.GetTokenVersion(c.Request.Context(), claims.UserID)
		if err != nil || version != claims.TokenVersion {
			response.Unauthorized(c, "登录状态已失效，请重新注册或登录")
			return
		}

		deviceCountry := c.GetHeader("Video_Device_Country")
		appVersion := c.GetHeader("Video_App_Version")
		channelID := c.GetHeader("Video_Channel_ID")
		phoneModel := c.GetHeader("Video_Phone_Model")
		c.Set(CtxPhoneModelKey, phoneModel)
		c.Set(CtxChannelIDKey, channelID)
		c.Set(CtxAPPVersionKey, appVersion)
		c.Set(CtxDeviceCountryKey, deviceCountry)
		c.Set(CtxUserIDKey, claims.UserID)
		c.Set(CtxPhoneCodeKey, claims.PhoneCode)
		c.Next()
	}
}

func extractBearerToken(header string) (string, bool) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

func GetAPIUserID(c *gin.Context) uint64 {
	value, ok := c.Get(CtxUserIDKey)
	if !ok {
		return 0
	}
	id, _ := value.(uint64)
	return id
}
