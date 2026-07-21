package handler

import (
	"ai-video/internal/pkg/errcode"
	"ai-video/internal/pkg/response"
	"ai-video/internal/pkg/utils"
	"ai-video/internal/server/admin/service"

	"github.com/gin-gonic/gin"
)

type TemplateDisplayConfigHandler struct {
	svc *service.TemplateDisplayConfigService
}

func NewTemplateDisplayConfigHandler() *TemplateDisplayConfigHandler {
	return &TemplateDisplayConfigHandler{svc: service.NewTemplateDisplayConfigService()}
}

func (h *TemplateDisplayConfigHandler) List(c *gin.Context) {
	var req service.ListTemplateDisplayConfigRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	p := utils.GetPagination(c)
	list, total, err := h.svc.List(c.Request.Context(), p.Page, p.PageSize, &req)
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, gin.H{"list": list, "total": total, "page": p.Page, "size": p.PageSize})
}

func (h *TemplateDisplayConfigHandler) GetByID(c *gin.Context) {
	id, ok := templateResourceID(c, "模板展示配置")
	if !ok {
		return
	}
	item, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, errcode.ErrNotFound, err.Error())
		return
	}
	response.OK(c, item)
}

func (h *TemplateDisplayConfigHandler) Create(c *gin.Context) {
	var req service.TemplateDisplayConfigPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	item, err := h.svc.Create(c.Request.Context(), &req)
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, item)
}

func (h *TemplateDisplayConfigHandler) Update(c *gin.Context) {
	id, ok := templateResourceID(c, "模板展示配置")
	if !ok {
		return
	}
	var req service.TemplateDisplayConfigPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	item, err := h.svc.Update(c.Request.Context(), id, &req)
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, item)
}

func (h *TemplateDisplayConfigHandler) Delete(c *gin.Context) {
	id, ok := templateResourceID(c, "模板展示配置")
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, nil)
}
