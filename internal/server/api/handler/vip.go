package handler

import (
	"ai-video/internal/pkg/errcode"
	"ai-video/internal/pkg/response"
	service "ai-video/internal/server/api/server"

	"github.com/gin-gonic/gin"
)

type VipHandler struct {
	svc *service.ClientVipService
}

func NewVipHandler() *VipHandler {
	return &VipHandler{
		svc: service.NewClientVipService(),
	}
}

func (h *VipHandler) Recommend(c *gin.Context) {
	var req service.VipRecommendRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	result, err := h.svc.VipRecommend(c, &req)
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, result)
}

func (h *VipHandler) VipList(c *gin.Context) {
	var req service.VipVipListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	result, err := h.svc.VipList(c, &req)
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, result)
}
