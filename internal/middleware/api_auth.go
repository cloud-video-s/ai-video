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
	CtxUserIDKey    = "user_id"
	CtxPhoneCodeKey = "phone_code"
)

func ApiAuth() gin.HandlerFunc {
	userRepo := repository.NewAppUserRepo()
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, "缺少 Authorization 头")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Unauthorized(c, "Authorization 格式错误")
			return
		}

		tokenString := parts[1]
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

		c.Set(CtxUserIDKey, claims.UserID)
		c.Set(CtxPhoneCodeKey, claims.PhoneCode)
		c.Next()
	}
}

func GetAPIUserID(c *gin.Context) uint64 {
	value, ok := c.Get(CtxUserIDKey)
	if !ok {
		return 0
	}
	id, _ := value.(uint64)
	return id
}
