package middleware

import (
	"ai-video/internal/config"
	"ai-video/internal/domain"
	"ai-video/internal/pkg/monitor"
	"ai-video/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

func CasbinRBAC() gin.HandlerFunc {
	return func(c *gin.Context) {
		roleCodes := GetRoleCodes(c)
		if len(roleCodes) == 0 {
			response.Forbidden(c, "无法获取用户角色")
			return
		}

		obj := c.Request.URL.Path
		act := c.Request.Method

		// 多角色：任一角色是超管或拥有该权限即放行
		for _, rc := range roleCodes {
			if rc == domain.SuperAdminRoleCode {
				c.Next()
				return
			}
			ok, err := config.Enforcer.Enforce(rc, obj, act)
			if err != nil {
				monitor.Report(config.Logger(c.Request.Context()), monitor.KindError, "rbac", err,
					"role_code", rc, "object", obj, "action", act,
				)
				response.Forbidden(c, "权限校验异常")
				return
			}
			if ok {
				c.Next()
				return
			}
		}

		response.Forbidden(c, "无访问权限")
	}
}
