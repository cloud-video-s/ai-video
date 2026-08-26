package handler

import (
	"ai-video/internal/pkg/errcode"
	"ai-video/internal/pkg/response"
	apiservice "ai-video/internal/server/api/server"

	"github.com/gin-gonic/gin"
)

type ToolHandler struct {
	svc *apiservice.ClientToolService
}

func NewToolHandler() *ToolHandler {
	return &ToolHandler{svc: apiservice.NewClientToolService()}
}

func (h *ToolHandler) List(c *gin.Context) {
	list, err := h.svc.List(c.Request.Context())
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, list)
}
