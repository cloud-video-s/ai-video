package handler

import (
	"ai-video/internal/pkg/errcode"
	"ai-video/internal/pkg/response"
	"ai-video/internal/repository"

	"github.com/gin-gonic/gin"
)

type DelayConfigHandler struct {
	repo *repository.DelayConfigRepo
}

func NewDelayConfigHandler() *DelayConfigHandler {
	return &DelayConfigHandler{repo: repository.NewDelayConfigRepo()}
}

// All returns every active delay config in stable display order.
func (h *DelayConfigHandler) All(c *gin.Context) {
	list, err := h.repo.ListAll(c.Request.Context())
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, list)
}
