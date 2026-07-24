package handler

import (
	"ai-video/internal/pkg/errcode"
	"ai-video/internal/pkg/response"
	"ai-video/internal/pkg/utils"
	"ai-video/internal/server/admin/service"

	"github.com/gin-gonic/gin"
)

type VIPSubscriptionLevelHandler struct {
	svc *service.VIPSubscriptionLevelService
}

func NewVIPSubscriptionLevelHandler() *VIPSubscriptionLevelHandler {
	return &VIPSubscriptionLevelHandler{svc: service.NewVIPSubscriptionLevelService()}
}

func (h *VIPSubscriptionLevelHandler) List(c *gin.Context) {
	var req service.ListVIPSubscriptionLevelRequest
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

func (h *VIPSubscriptionLevelHandler) ListOptions(c *gin.Context) {
	list, err := h.svc.ListOptions(c.Request.Context())
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, list)
}

func (h *VIPSubscriptionLevelHandler) GetByID(c *gin.Context) {
	id, ok := templateResourceID(c, "VIP 等级")
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

func (h *VIPSubscriptionLevelHandler) Create(c *gin.Context) {
	var req service.VIPSubscriptionLevelPayload
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

func (h *VIPSubscriptionLevelHandler) Update(c *gin.Context) {
	id, ok := templateResourceID(c, "VIP 等级")
	if !ok {
		return
	}
	var req service.VIPSubscriptionLevelPayload
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

func (h *VIPSubscriptionLevelHandler) UpdateStatus(c *gin.Context) {
	id, ok := templateResourceID(c, "VIP 等级")
	if !ok {
		return
	}
	var req service.VIPSubscriptionLevelStatusPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.UpdateStatus(c.Request.Context(), id, *req.Status); err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *VIPSubscriptionLevelHandler) Delete(c *gin.Context) {
	id, ok := templateResourceID(c, "VIP 等级")
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, nil)
}
