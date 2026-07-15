package handler

import (
	"strings"

	"ai-video/internal/app"
	"ai-video/internal/middleware"
	"ai-video/internal/pkg/errcode"
	"ai-video/internal/pkg/response"
	apiservice "ai-video/internal/server/api/server"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	svc *apiservice.AuthService
}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{svc: apiservice.NewAuthService()}
}

func (h *AuthHandler) DeviceRegister(c *gin.Context) {
	var req apiservice.DeviceRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	result, err := h.svc.RegisterDevice(c.Request.Context(), &req, c.ClientIP(), requestCountryHeader(c))
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, result)
}

func (h *AuthHandler) ReRegister(c *gin.Context) {
	var req apiservice.DeviceRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	result, err := h.svc.ReRegister(c.Request.Context(), &req, c.ClientIP(), requestCountryHeader(c))
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, result)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	if err := h.svc.Logout(bearerToken(c)); err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *AuthHandler) Profile(c *gin.Context) {
	user, err := h.svc.GetProfile(c.Request.Context(), middleware.GetAPIUserID(c))
	if err != nil {
		response.Fail(c, errcode.ErrUserNotFound, err.Error())
		return
	}
	response.OK(c, user)
}

func (h *AuthHandler) UpdateCountry(c *gin.Context) {
	var req apiservice.UpdateCountryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	user, err := h.svc.UpdateCountry(
		c.Request.Context(), middleware.GetAPIUserID(c), &req, c.ClientIP(), requestCountryHeader(c),
	)
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, user)
}

func bearerToken(c *gin.Context) string {
	parts := strings.SplitN(c.GetHeader("Authorization"), " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}
	return ""
}

func requestCountryHeader(c *gin.Context) string {
	if header := strings.TrimSpace(app.Cfg.GeoIP.CountryHeader); header != "" {
		return c.GetHeader(header)
	}
	return ""
}
