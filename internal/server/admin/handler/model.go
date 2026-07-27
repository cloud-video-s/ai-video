package handler

import (
	"ai-video/internal/pkg/errcode"
	"ai-video/internal/pkg/response"
	"ai-video/internal/pkg/utils"
	"ai-video/internal/server/admin/service"

	"github.com/gin-gonic/gin"
)

type ModelHandler struct{ svc *service.ModelService }

func NewModelHandler() *ModelHandler { return &ModelHandler{svc: service.NewModelService()} }

func (h *ModelHandler) List(c *gin.Context) {
	var req service.ListModelRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	p := utils.GetPagination(c)
	items, total, err := h.svc.List(c.Request.Context(), p.Page, p.PageSize, &req)
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, gin.H{"list": items, "total": total, "page": p.Page, "size": p.PageSize})
}

func (h *ModelHandler) Get(c *gin.Context) {
	id, ok := modelManagementID(c, "id", "模型")
	if !ok {
		return
	}
	item, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, errcode.ErrNotFound, err.Error())
		return
	}
	response.OK(c, item)
}

func (h *ModelHandler) Create(c *gin.Context) {
	var req service.ModelPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	item, err := h.svc.Create(c.Request.Context(), &req)
	if err != nil {
		response.Fail(c, errcode.ErrParam, err.Error())
		return
	}
	response.OK(c, item)
}

func (h *ModelHandler) Update(c *gin.Context) {
	id, ok := modelManagementID(c, "id", "模型")
	if !ok {
		return
	}
	var req service.ModelPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	item, err := h.svc.Update(c.Request.Context(), id, &req)
	if err != nil {
		response.Fail(c, errcode.ErrParam, err.Error())
		return
	}
	response.OK(c, item)
}

func (h *ModelHandler) Delete(c *gin.Context) {
	id, ok := modelManagementID(c, "id", "模型")
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.Fail(c, errcode.ErrParam, err.Error())
		return
	}
	response.OK(c, nil)
}
