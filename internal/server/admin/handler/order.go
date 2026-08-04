package handler

import (
	"ai-video/internal/pkg/errcode"
	"ai-video/internal/pkg/response"
	"ai-video/internal/pkg/utils"
	"ai-video/internal/server/admin/service"

	"github.com/gin-gonic/gin"
)

type OrderAdminHandler struct {
	svc *service.OrderAdminService
}

func NewOrderAdminHandler() *OrderAdminHandler {
	return &OrderAdminHandler{svc: service.NewOrderAdminService()}
}

func (h *OrderAdminHandler) List(c *gin.Context) {
	var req service.ListOrderRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	p := utils.GetPagination(c)
	items, total, summary, err := h.svc.List(c.Request.Context(), p.Page, p.PageSize, &req)
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, gin.H{
		"list": items, "total": total, "page": p.Page, "size": p.PageSize, "summary": summary,
	})
}

func (h *OrderAdminHandler) GetByID(c *gin.Context) {
	id, ok := templateResourceID(c, "订单")
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
