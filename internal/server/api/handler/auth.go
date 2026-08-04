package handler

import (
	"ai-video/internal/config"
	"errors"
	"net/http"
	"strings"

	"ai-video/internal/middleware"
	"ai-video/internal/pkg/errcode"
	"ai-video/internal/pkg/oidc"
	"ai-video/internal/pkg/response"
	apiservice "ai-video/internal/server/api/server"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	svc    *apiservice.AuthService
	points *apiservice.ClientPointsService
}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{svc: apiservice.NewAuthService(), points: apiservice.NewClientPointsService()}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req apiservice.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	result, err := h.svc.Login(c, &req, c.ClientIP())
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, result)
}

func (h *AuthHandler) AppleOrderLogin(c *gin.Context) {
	var req apiservice.AppleOrderLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, errcode.ErrParam, "Apple 支付订单参数错误: "+err.Error())
		return
	}
	req.OrderCode = strings.TrimSpace(req.OrderCode)
	if req.OrderCode == "" {
		response.FailWithStatus(c, http.StatusBadRequest, errcode.ErrParam, "order_code 不能为空")
		return
	}

	result, err := h.svc.LoginByAppleOrder(c.Request.Context(), &req)
	switch {
	case errors.Is(err, apiservice.ErrAppleOrderNotFound):
		response.FailWithStatus(c, http.StatusNotFound, errcode.ErrNotFound, err.Error())
		return
	case errors.Is(err, apiservice.ErrAppleOrderUserNotFound):
		response.FailWithStatus(c, http.StatusNotFound, errcode.ErrUserNotFound, err.Error())
		return
	case errors.Is(err, apiservice.ErrAppleOrderUserDisabled):
		response.FailWithStatus(c, http.StatusForbidden, errcode.ErrUserDisabled, err.Error())
		return
	case err != nil:
		response.FailWithStatus(c, http.StatusInternalServerError, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, result)
}

func (h *AuthHandler) ThirdBinding(c *gin.Context) {
	var req apiservice.ThirdPartyLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	if req.ThirdCode == "" && req.IDToken == "" {
		response.FailWithStatus(c, http.StatusBadRequest, errcode.ErrParam, "参数异常")
	}
	result, err := h.svc.ThirdPartyLogin(c, &req, c.ClientIP())
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
	case errors.Is(err, apiservice.ErrDeviceCodeNotConfigured):
		response.Fail(c, errcode.ErrRoleExist, err.Error())
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

func (h *AuthHandler) Refresh(c *gin.Context) {
	result, err := h.svc.Refresh(
		c.Request.Context(), middleware.GetAPIUserID(c), middleware.GetAPITokenVersion(c), bearerToken(c),
	)
	if errors.Is(err, apiservice.ErrAuthStateInvalid) {
		response.Unauthorized(c, err.Error())
		return
	}
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, result)
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

func (h *AuthHandler) GetPointsList(c *gin.Context) {
	var req apiservice.ClientPointsRequest
	if err := c.ShouldBind(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	data, err := h.points.GetPointsList(c, req)
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, data)
}

func bearerToken(c *gin.Context) string {
	parts := strings.SplitN(c.GetHeader("Authorization"), " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}
	return ""
}

func requestCountryHeader(c *gin.Context) string {
	if header := strings.TrimSpace(config.Cfg.GeoIP.CountryHeader); header != "" {
		return c.GetHeader(header)
	}
	return ""
}
