package handler

import (
	"ai-video/internal/pkg/errcode"
	"ai-video/internal/pkg/response"
	"ai-video/internal/pkg/utils"
	"ai-video/internal/server/admin/service"

	"github.com/gin-gonic/gin"
)

type TemplateComplaintHandler struct {
	svc *service.TemplateComplaintService
}

func NewTemplateComplaintHandler() *TemplateComplaintHandler {
	return &TemplateComplaintHandler{svc: service.NewTemplateComplaintService()}
}

func (h *TemplateComplaintHandler) List(c *gin.Context) {
	var req service.ListTemplateComplaintRequest
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

func (h *TemplateComplaintHandler) GetByID(c *gin.Context) {
	id, ok := templateResourceID(c, "投诉记录")
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
