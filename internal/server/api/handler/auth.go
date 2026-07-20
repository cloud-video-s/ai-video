package handler

import (
	"errors"
	"net/http"
	"strings"

	"ai-video/internal/app"
	"ai-video/internal/middleware"
	"ai-video/internal/pkg/errcode"
	"ai-video/internal/pkg/oidc"
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

func (h *AuthHandler) Login(c *gin.Context) {
	var req apiservice.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	result, err := h.svc.Login(c, &req, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, result)
}

func (h *AuthHandler) ReRegister(c *gin.Context) {
	var req apiservice.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	result, err := h.svc.ReRegister(c, &req, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, result)
}

func (h *AuthHandler) GoogleLogin(c *gin.Context) { h.thirdPartyLogin(c, "google") }
func (h *AuthHandler) AppleLogin(c *gin.Context)  { h.thirdPartyLogin(c, "apple") }

func (h *AuthHandler) thirdPartyLogin(c *gin.Context, provider string) {
	var req apiservice.ThirdPartyLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	result, err := h.svc.ThirdPartyLogin(c, provider, &req, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		h.handleIdentityError(c, err)
		return
	}
	response.OK(c, result)
}

func (h *AuthHandler) ListIdentities(c *gin.Context) {
	list, err := h.svc.ListIdentities(c.Request.Context(), middleware.GetAPIUserID(c))
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, list)
}

func (h *AuthHandler) BindGoogle(c *gin.Context) { h.bindIdentity(c, "google") }
func (h *AuthHandler) BindApple(c *gin.Context)  { h.bindIdentity(c, "apple") }

func (h *AuthHandler) bindIdentity(c *gin.Context, provider string) {
	var req apiservice.BindIdentityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	item, err := h.svc.BindIdentity(c.Request.Context(), middleware.GetAPIUserID(c), provider, &req)
	if err != nil {
		h.handleIdentityError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *AuthHandler) UnbindIdentity(c *gin.Context) {
	if err := h.svc.UnbindIdentity(c.Request.Context(), middleware.GetAPIUserID(c), c.Param("provider")); err != nil {
		response.Fail(c, errcode.ErrParam, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *AuthHandler) handleIdentityError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, oidc.ErrInvalidToken):
		response.FailWithStatus(c, http.StatusUnauthorized, errcode.ErrTokenInvalid, "第三方身份凭证无效")
	case errors.Is(err, apiservice.ErrIdentityProviderNotConfigured):
		response.FailWithStatus(c, http.StatusServiceUnavailable, errcode.ErrServer, err.Error())
	default:
		response.Fail(c, errcode.ErrServer, err.Error())
	}
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
	user.LoginType = middleware.GetAPILoginType(c)
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
