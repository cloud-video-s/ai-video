package handler

import (
	"ai-video/internal/pkg/errcode"
	"ai-video/internal/pkg/response"
	"ai-video/internal/pkg/utils"
	"ai-video/internal/server/admin/service"

	"github.com/gin-gonic/gin"
)

type UserPointsLedgerHandler struct {
	svc *service.UserPointsLedgerService
}

func NewUserPointsLedgerHandler() *UserPointsLedgerHandler {
	return &UserPointsLedgerHandler{svc: service.NewUserPointsLedgerService()}
}

func (h *UserPointsLedgerHandler) List(c *gin.Context) {
	var req service.ListUserPointsLedgerRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	p := utils.GetPagination(c)
	list, total, summary, err := h.svc.List(c.Request.Context(), p.Page, p.PageSize, &req)
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, gin.H{"list": list, "total": total, "page": p.Page, "size": p.PageSize, "summary": summary})
}

func (h *UserPointsLedgerHandler) GetByID(c *gin.Context) {
	id, ok := templateResourceID(c, "积分明细")
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
