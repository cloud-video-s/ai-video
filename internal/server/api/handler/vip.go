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
	result, err := h.svc.Recommend(c)
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, result)
}
