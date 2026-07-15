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
	CtxAdminIDKey   = "admin_id"
	CtxAdminnameKey = "username"
	CtxRoleCodesKey = "role_codes"
)

func AdminAuth() gin.HandlerFunc {
	userRepo := repository.NewAdminRepo()
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

		claims, err := jwt.ParseToken(tokenString)
		if err != nil {
			response.Unauthorized(c, "Token 无效或已过期")
			return
		}

		// 会话撤销：比对 token 版本与用户当前版本
		// 改密 / 禁用 / 改角色 / 删除用户后，旧 token 立即失效
		version, err := userRepo.GetTokenVersion(c.Request.Context(), claims.UserID)
		if err != nil || version != claims.TokenVersion {
			response.Unauthorized(c, "登录状态已失效，请重新登录")
			return
		}

		c.Set(CtxAdminIDKey, claims.UserID)
		c.Set(CtxAdminnameKey, claims.Username)
		c.Set(CtxRoleCodesKey, claims.RoleCodes)
		c.Next()
	}
}

func GetAdminID(c *gin.Context) uint {
	val, exists := c.Get(CtxAdminIDKey)
	if !exists {
		return 0
	}
	id, _ := val.(uint)
	return id
}

func GetUsername(c *gin.Context) string {
	val, exists := c.Get(CtxAdminnameKey)
	if !exists {
		return ""
	}
	name, _ := val.(string)
	return name
}

func GetRoleCodes(c *gin.Context) []string {
	val, exists := c.Get(CtxRoleCodesKey)
	if !exists {
		return nil
	}
	codes, _ := val.([]string)
	return codes
}
