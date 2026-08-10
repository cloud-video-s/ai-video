package middleware

import (
	"strings"

	"ai-video/internal/pkg/cache"
	"ai-video/internal/pkg/jwt"
	"ai-video/internal/pkg/response"
	"ai-video/internal/pkg/utils"

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
	HeaderSystemType     = "Video_System_Type"

	apiRequestMetadataGinKey = "api_request_metadata"
)

func ApiAuth(userRepo UserRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if strings.TrimSpace(authHeader) == "" {
			response.Unauthorized(c, "缺少 Authorization 头")
			return
		}
		headerAppCode, ok := requiredHeader(c, HeaderAPPCode)
		if !ok {
			return
		}
		headerAppPackageCode, ok := requiredHeader(c, HeaderAppPackageCode)
		if !ok {
			return
		}
		headerAppVersion, ok := requiredHeader(c, HeaderAppVersion)
		if !ok {
			return
		}
		headerPhoneModel, ok := requiredHeader(c, HeaderPhoneModel)
		if !ok {
			return
		}
		headerChannelCode, ok := requiredHeader(c, HeaderChannelCode)
		if !ok {
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
			response.Unauthorized(c, err.Error())
			return
		}
		deviceCode, version, err := userRepo.GetAuthState(c.Request.Context(), claims.UserID)
		if err != nil {
			response.Unauthorized(c, err.Error())
			return
		}
		if deviceCode != claims.DeviceCode || version != claims.TokenVersion {
			response.Unauthorized(c, "登录状态已失效，请重新注册或登录")
			return
		}
		headerDeviceCountry := strings.TrimSpace(c.GetHeader(HeaderDeviceCountry))
		if headerDeviceCountry == "" {
			headerDeviceCountry, _ = utils.GetCountryByIP(c.ClientIP())
		}
		setAPIRequestMetadata(c, APIRequestMetadata{
			UserID:         claims.UserID,
			TokenVersion:   claims.TokenVersion,
			LoginType:      claims.LoginType,
			AppCode:        headerAppCode,
			AppPackageCode: headerAppPackageCode,
			AppVersion:     headerAppVersion,
			ChannelCode:    headerChannelCode,
			DeviceCountry:  strings.TrimSpace(headerDeviceCountry),
			PhoneModel:     headerPhoneModel,
			SystemType:     getClientSystemType(c),
		})
		c.Next()
	}
}

func ApiHeader() gin.HandlerFunc {
	return func(c *gin.Context) {
		headerAppCode, ok := requiredHeader(c, HeaderAPPCode)
		if !ok {
			return
		}
		headerAppPackageCode, ok := requiredHeader(c, HeaderAppPackageCode)
		if !ok {
			return
		}
		headerAppVersion, ok := requiredHeader(c, HeaderAppVersion)
		if !ok {
			return
		}
		headerPhoneModel, ok := requiredHeader(c, HeaderPhoneModel)
		if !ok {
			return
		}
		headerChannelCode, ok := requiredHeader(c, HeaderChannelCode)
		if !ok {
			return
		}

		headerDeviceCountry := strings.TrimSpace(c.GetHeader(HeaderDeviceCountry))
		if headerDeviceCountry == "" {
			headerDeviceCountry, _ = utils.GetCountryByIP(c.ClientIP())
		}
		setAPIRequestMetadata(c, APIRequestMetadata{
			AppCode:        headerAppCode,
			AppPackageCode: headerAppPackageCode,
			AppVersion:     headerAppVersion,
			ChannelCode:    headerChannelCode,
			DeviceCountry:  strings.TrimSpace(headerDeviceCountry),
			PhoneModel:     headerPhoneModel,
			SystemType:     getClientSystemType(c),
		})
		c.Next()
	}
}

func requiredHeader(c *gin.Context, name string) (string, bool) {
	value := strings.TrimSpace(c.GetHeader(name))
	if value == "" {
		response.Unauthorized(c, "缺少 "+name+" 头")
		return "", false
	}
	return value, true
}

func setAPIRequestMetadata(c *gin.Context, metadata APIRequestMetadata) {
	c.Set(apiRequestMetadataGinKey, metadata)
	c.Set(HeaderUserIDKey, metadata.UserID)
	c.Set(HeaderTokenVersion, metadata.TokenVersion)
	c.Set(HeaderLoginType, metadata.LoginType)
	c.Set(HeaderAPPCode, metadata.AppCode)
	c.Set(HeaderAppPackageCode, metadata.AppPackageCode)
	c.Set(HeaderAppVersion, metadata.AppVersion)
	c.Set(HeaderChannelCode, metadata.ChannelCode)
	c.Set(HeaderDeviceCountry, metadata.DeviceCountry)
	c.Set(HeaderPhoneModel, metadata.PhoneModel)
	c.Set(HeaderSystemType, metadata.SystemType)

	// Gin values do not propagate when handlers pass c.Request.Context() into
	// lower layers, so install the same metadata on the standard Go context.
	if c.Request != nil {
		ctx := withAPIRequestMetadata(c.Request.Context(), metadata)
		c.Request = c.Request.WithContext(ctx)
	}
}

func apiRequestMetadata(c *gin.Context) (APIRequestMetadata, bool) {
	if c == nil {
		return APIRequestMetadata{}, false
	}
	if value, ok := c.Get(apiRequestMetadataGinKey); ok {
		if metadata, ok := value.(APIRequestMetadata); ok {
			return metadata, true
		}
	}
	if c.Request != nil {
		return APIRequestMetadataFromContext(c.Request.Context())
	}
	return APIRequestMetadata{}, false
}

func ginValue[T any](c *gin.Context, key string) (zero T) {
	if c == nil {
		return zero
	}
	value, ok := c.Get(key)
	if !ok {
		return zero
	}
	typed, ok := value.(T)
	if !ok {
		return zero
	}
	return typed
}

func extractBearerToken(header string) (string, bool) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

func GetAPIUserID(c *gin.Context) uint64 {
	if metadata, ok := apiRequestMetadata(c); ok {
		return metadata.UserID
	}
	return ginValue[uint64](c, HeaderUserIDKey)
}

func GetAPITokenVersion(c *gin.Context) int64 {
	if metadata, ok := apiRequestMetadata(c); ok {
		return metadata.TokenVersion
	}
	return ginValue[int64](c, HeaderTokenVersion)
}

func GetAPIAPPCode(c *gin.Context) string {
	if metadata, ok := apiRequestMetadata(c); ok {
		return metadata.AppCode
	}
	return ginValue[string](c, HeaderAPPCode)
}

func GetAPIAppPackageCode(c *gin.Context) string {
	if metadata, ok := apiRequestMetadata(c); ok {
		return metadata.AppPackageCode
	}
	return ginValue[string](c, HeaderAppPackageCode)
}

func GetAPIAppVersion(c *gin.Context) string {
	if metadata, ok := apiRequestMetadata(c); ok {
		return metadata.AppVersion
	}
	return ginValue[string](c, HeaderAppVersion)
}

func GetAPIChannelCode(c *gin.Context) string {
	if metadata, ok := apiRequestMetadata(c); ok {
		return metadata.ChannelCode
	}
	return ginValue[string](c, HeaderChannelCode)
}

func GetAPIDeviceCountry(c *gin.Context) string {
	if metadata, ok := apiRequestMetadata(c); ok {
		return metadata.DeviceCountry
	}
	return ginValue[string](c, HeaderDeviceCountry)
}

func GetAPIPhoneModel(c *gin.Context) string {
	if metadata, ok := apiRequestMetadata(c); ok {
		return metadata.PhoneModel
	}
	return ginValue[string](c, HeaderPhoneModel)
}

func GetAPILoginType(c *gin.Context) uint32 {
	if metadata, ok := apiRequestMetadata(c); ok {
		return metadata.LoginType
	}
	return ginValue[uint32](c, HeaderLoginType)
}

func GetAPISystemType(c *gin.Context) int {
	if metadata, ok := apiRequestMetadata(c); ok {
		return metadata.SystemType
	}
	return ginValue[int](c, HeaderSystemType)
}

func getClientSystemType(c *gin.Context) int {
	// Prefer an explicit client header, while accepting both the numeric API
	// value and its readable name for compatibility.
	switch strings.ToLower(strings.TrimSpace(c.GetHeader(HeaderSystemType))) {
	case "1", "ios":
		return 1
	case "2", "android":
		return 2
	}

	switch strings.ToLower(getClientOS(c)) {
	case "ios":
		return 1
	case "android":
		return 2
	default:
		return 0
	}
}

func getClientOS(c *gin.Context) string {
	// 1. 优先读取现代浏览器客户端提示（最准确）
	if platform := c.GetHeader("Sec-CH-UA-Platform"); platform != "" {
		// 注意：该值可能带双引号，如 "Windows"，需要去除
		return strings.Trim(platform, `"`)
	}
	// 2. 回退到传统 User-Agent 解析
	ua := c.GetHeader("User-Agent")
	return parseUA(ua)
}

// 极简回退解析（仅演示，生产环境建议用第三方库）
func parseUA(ua string) string {
	if ua == "" {
		return "Unknown"
	}
	ua = strings.ToLower(ua)
	switch {
	// Mobile user agents also contain Linux or Mac OS, so the more specific
	// checks must run first.
	case strings.Contains(ua, "android"):
		return "Android"
	case strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad") || strings.Contains(ua, "ipod"):
		return "iOS"
	case strings.Contains(ua, "windows"):
		return "Windows"
	case strings.Contains(ua, "mac os"):
		return "macOS"
	case strings.Contains(ua, "linux"):
		return "Linux"
	default:
		return "Other"
	}
}
