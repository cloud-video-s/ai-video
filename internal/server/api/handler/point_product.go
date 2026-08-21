package handler

import (
	"ai-video/internal/pkg/errcode"
	"ai-video/internal/pkg/response"
	apiservice "ai-video/internal/server/api/server"

	"github.com/gin-gonic/gin"
)

type PointProductHandler struct {
	svc *apiservice.ClientPointProductService
}

func NewPointProductHandler() *PointProductHandler {
	return &PointProductHandler{svc: apiservice.NewClientPointProductService()}
}

func (h *PointProductHandler) List(c *gin.Context) {
	list, err := h.svc.List(c)
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, list)
}
