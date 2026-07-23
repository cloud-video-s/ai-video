package middleware

import (
	"ai-video/internal/pkg/cache"
	"ai-video/internal/pkg/jwt"
	"ai-video/internal/pkg/response"
	"ai-video/internal/pkg/utils"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	HeaderUserIDKey      = "Video_user_id"
	HeaderDeviceCountry  = "Video_Device_Country"
	HeaderPhoneModel     = "Video_Phone_Model"
	HeaderAPPCode        = "Video_App_Code"
	HeaderAppPackageCode = "Video_App_Package_Code"
	HeaderAppVersion     = "Video_App_Version"
	HeaderChannelCode    = "Video_Channel_Code"
	HeaderTokenVersion   = "Video_Token_Version"
	HeaderLoginType      = "Video_Login_Type"
)

func ApiAuth(userRepo UserRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if strings.TrimSpace(authHeader) == "" {
			response.Unauthorized(c, "缺少 Authorization 头")
			return
		}
		headerAppCode := c.GetHeader(HeaderAPPCode)
		if headerAppCode == "" {
			response.Unauthorized(c, "缺少 Video_App_Code 头")
			return
		}
		headerAppPackageCode := c.GetHeader(HeaderAppPackageCode)
		if headerAppPackageCode == "" {
			response.Unauthorized(c, "缺少 Video_App_Package_Code 头")
			return
		}
		headerAppVersion := c.GetHeader(HeaderAppVersion)
		if headerAppVersion == "" {
			response.Unauthorized(c, "缺少 Video_App_Version 头")
			return
		}

		headerPhoneModel := c.GetHeader(HeaderPhoneModel)
		if headerPhoneModel == "" {
			response.Unauthorized(c, "缺少 Video_Phone_Model 头")
			return
		}

		headerChannelCode := c.GetHeader(HeaderChannelCode)
		if headerChannelCode == "" {
			response.Unauthorized(c, "缺少 Video_Channel_Code 头")
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
		imei, version, err := userRepo.GetAuthState(c.Request.Context(), claims.UserID)
		if err != nil || imei != claims.IMEI || version != claims.TokenVersion {
			response.Unauthorized(c, "登录状态已失效，请重新注册或登录")
			return
		}
		headerDeviceCountry := c.GetHeader(HeaderDeviceCountry)
		if headerDeviceCountry == "" {
			headerDeviceCountry, _ = utils.GetCountryByIP(c.ClientIP())
		}
		c.Set(HeaderUserIDKey, claims.UserID)
		c.Set(HeaderTokenVersion, claims.TokenVersion)
		c.Set(HeaderLoginType, claims.LoginType)
		c.Set(HeaderAPPCode, headerAppCode)
		c.Set(HeaderAppPackageCode, headerAppPackageCode)
		c.Set(HeaderAppVersion, headerAppVersion)
		c.Set(HeaderChannelCode, headerChannelCode)
		c.Set(HeaderDeviceCountry, headerDeviceCountry)
		c.Set(HeaderPhoneModel, headerPhoneModel)
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
	value, ok := c.Get(HeaderUserIDKey)
	if !ok {
		return 0
	}
	id, ok := value.(uint64)
	if !ok {
		return 0
	}
	return id
}

func GetAPITokenVersion(c *gin.Context) int64 {
	value, ok := c.Get(HeaderTokenVersion)
	if !ok {
		return 0
	}
	version, ok := value.(int64)
	if !ok {
		return 0
	}
	return version
}

func GetAPIAPPCode(c *gin.Context) string {
	value, ok := c.Get(HeaderAPPCode)
	if !ok {
		return ""
	}
	aPPCode, ok := value.(string)
	if !ok {
		return ""
	}
	return aPPCode
}

func GetAPIAppPackageCode(c *gin.Context) string {
	value, ok := c.Get(HeaderAppPackageCode)
	if !ok {
		return ""
	}
	appPackageCode, ok := value.(string)
	if !ok {
		return ""
	}
	return appPackageCode
}

func GetAPIAppVersion(c *gin.Context) string {
	value, ok := c.Get(HeaderAppVersion)
	if !ok {
		return ""
	}
	appVersion, ok := value.(string)
	if !ok {
		return ""
	}
	return appVersion
}

func GetAPIChannelCode(c *gin.Context) string {
	value, ok := c.Get(HeaderChannelCode)
	if !ok {
		return ""
	}
	channelCode, ok := value.(string)
	if !ok {
		return ""
	}
	return channelCode
}

func GetAPIDeviceCountry(c *gin.Context) string {
	value, ok := c.Get(HeaderDeviceCountry)
	if !ok {
		return ""
	}
	deviceCountry, ok := value.(string)
	if !ok {
		return ""
	}
	return deviceCountry
}

func GetAPIPhoneModel(c *gin.Context) string {
	value, ok := c.Get(HeaderPhoneModel)
	if !ok {
		return ""
	}
	phoneModel, ok := value.(string)
	if !ok {
		return ""
	}
	return phoneModel
}

func GetAPILoginType(c *gin.Context) uint32 {
	value, ok := c.Get(HeaderLoginType)
	if !ok {
		return 0
	}
	loginType, ok := value.(uint32)
	if !ok {
		return 0
	}
	return loginType
}
