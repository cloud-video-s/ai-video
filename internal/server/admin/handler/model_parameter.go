package handler

import (
	"ai-video/internal/pkg/errcode"
	"ai-video/internal/pkg/response"
	"ai-video/internal/server/admin/service"

	"github.com/gin-gonic/gin"
)

type ModelParameterHandler struct {
	svc *service.ModelParameterService
}

func NewModelParameterHandler() *ModelParameterHandler {
	return &ModelParameterHandler{svc: service.NewModelParameterService()}
}

func (h *ModelParameterHandler) List(c *gin.Context) {
	modelID, ok := modelManagementID(c, "id", "模型")
	if !ok {
		return
	}
	items, err := h.svc.List(c.Request.Context(), modelID)
	if err != nil {
		response.Fail(c, errcode.ErrParam, err.Error())
		return
	}
	response.OK(c, items)
}

func (h *ModelParameterHandler) Create(c *gin.Context) {
	modelID, ok := modelManagementID(c, "id", "模型")
	if !ok {
		return
	}
	var req service.ModelParameterPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	item, err := h.svc.Create(c.Request.Context(), modelID, &req)
	if err != nil {
		response.Fail(c, errcode.ErrParam, err.Error())
		return
	}
	response.OK(c, item)
}

func (h *ModelParameterHandler) Update(c *gin.Context) {
	modelID, ok := modelManagementID(c, "id", "模型")
	if !ok {
		return
	}
	parameterID, ok := modelManagementID(c, "parameter_id", "模型配置")
	if !ok {
		return
	}
	var req service.ModelParameterPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	item, err := h.svc.Update(c.Request.Context(), modelID, parameterID, &req)
	if err != nil {
		response.Fail(c, errcode.ErrParam, err.Error())
		return
	}
	response.OK(c, item)
}

func (h *ModelParameterHandler) Delete(c *gin.Context) {
	modelID, ok := modelManagementID(c, "id", "模型")
	if !ok {
		return
	}
	parameterID, ok := modelManagementID(c, "parameter_id", "模型配置")
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), modelID, parameterID); err != nil {
		response.Fail(c, errcode.ErrParam, err.Error())
		return
	}
	response.OK(c, nil)
}
