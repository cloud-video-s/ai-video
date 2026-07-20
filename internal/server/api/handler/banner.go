package handler

import (
	"ai-video/internal/middleware"
	"ai-video/internal/pkg/errcode"
	"ai-video/internal/pkg/response"
	apiservice "ai-video/internal/server/api/server"

	"github.com/gin-gonic/gin"
)

type BannerHandler struct {
	svc *apiservice.ClientBannerService
}

func NewBannerHandler() *BannerHandler {
	return &BannerHandler{svc: apiservice.NewClientBannerService()}
}

func (h *BannerHandler) List(c *gin.Context) {
	var req apiservice.ClientBannerRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	applyBannerHeaders(c, &req)
	list, err := h.svc.List(c, middleware.GetAPIUserID(c), &req)
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, list)
}

func applyBannerHeaders(c *gin.Context, req *apiservice.ClientBannerRequest) {
	req.Country = middleware.GetAPIDeviceCountry(c)
	req.Channel = middleware.GetAPIChannelID(c)
	req.ChannelPackage = middleware.GetAPIChannelPackage(c)
	req.PackageCode = middleware.GetAPIAppPackage(c)
	req.PackageVersion = middleware.GetAPIAppVersion(c)
}
