package handler

import (
	"errors"
	"net/http"
	"strconv"

	"ai-video/internal/pkg/errcode"
	"ai-video/internal/pkg/response"
	"ai-video/internal/pkg/utils"
	"ai-video/internal/server/admin/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AIModelHandler struct{ svc *service.AIModelService }

func NewAIModelHandler() *AIModelHandler { return &AIModelHandler{svc: service.NewAIModelService()} }

func (h *AIModelHandler) List(c *gin.Context) {
	var request service.ListAIModelRequest
	if err := c.ShouldBindQuery(&request); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	pagination := utils.GetPagination(c)
	items, total, err := h.svc.List(c.Request.Context(), pagination.Page, pagination.PageSize, &request)
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, gin.H{"list": items, "total": total, "page": pagination.Page, "size": pagination.PageSize})
}

func (h *AIModelHandler) Get(c *gin.Context) {
	id, ok := aiModelID(c)
	if !ok {
		return
	}
	item, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		aiModelError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *AIModelHandler) Create(c *gin.Context) {
	var request service.AIModelPayload
	if err := c.ShouldBindJSON(&request); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	item, err := h.svc.Create(c.Request.Context(), &request)
	if err != nil {
		response.Fail(c, errcode.ErrParam, err.Error())
		return
	}
	response.OK(c, item)
}

func (h *AIModelHandler) Update(c *gin.Context) {
	id, ok := aiModelID(c)
	if !ok {
		return
	}
	var request service.AIModelPayload
	if err := c.ShouldBindJSON(&request); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	item, err := h.svc.Update(c.Request.Context(), id, &request)
	if err != nil {
		aiModelError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *AIModelHandler) Delete(c *gin.Context) {
	id, ok := aiModelID(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		aiModelError(c, err)
		return
	}
	response.OK(c, nil)
}

func aiModelID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Fail(c, errcode.ErrParam, "模型 ID 无效")
		return 0, false
	}
	return id, true
}

func aiModelError(c *gin.Context, err error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.FailWithStatus(c, http.StatusNotFound, errcode.ErrNotFound, "模型配置不存在")
		return
	}
	response.Fail(c, errcode.ErrParam, err.Error())
}
