package handler

import (
	"ai-video/internal/pkg/errcode"
	"ai-video/internal/pkg/response"
	"ai-video/internal/repository"

	"github.com/gin-gonic/gin"
)

type ConfigHandler struct {
	repo *repository.ConfigRepo
}

func NewConfigHandler() *ConfigHandler {
	return &ConfigHandler{repo: repository.NewConfigRepo()}
}

// Public returns is_public configs as a key→value map for the unauthenticated
// SPA bootstrap (e.g. site name / logo on the login page).
func (h *ConfigHandler) Public(c *gin.Context) {
	list, err := h.repo.ListPublic(c.Request.Context())
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	out := make(map[string]string, len(list))
	for i := range list {
		out[list[i].Key] = list[i].Value
	}
	response.OK(c, out)
}
