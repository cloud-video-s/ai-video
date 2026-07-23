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

// All returns the client configuration as a flat key-value object.
func (h *DelayConfigHandler) All(c *gin.Context) {
	values, err := h.repo.ListValues(c.Request.Context())
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, values)
}
