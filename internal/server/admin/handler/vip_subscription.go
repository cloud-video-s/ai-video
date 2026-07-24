package handler

import (
	"ai-video/internal/pkg/errcode"
	"ai-video/internal/pkg/response"
	"ai-video/internal/pkg/utils"
	"ai-video/internal/server/admin/service"

	"github.com/gin-gonic/gin"
)

type VIPSubscriptionHandler struct {
	svc *service.VIPSubscriptionService
}

func NewVIPSubscriptionHandler() *VIPSubscriptionHandler {
	return &VIPSubscriptionHandler{svc: service.NewVIPSubscriptionService()}
}

func (h *VIPSubscriptionHandler) List(c *gin.Context) {
	var req service.ListVIPSubscriptionRequest
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

func (h *VIPSubscriptionHandler) GetByID(c *gin.Context) {
	id, ok := templateResourceID(c, "VIP 订阅套餐")
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
func (h *VIPSubscriptionHandler) Create(c *gin.Context) {
	var req service.VIPSubscriptionPayload
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
func (h *VIPSubscriptionHandler) Update(c *gin.Context) {
	id, ok := templateResourceID(c, "VIP 订阅套餐")
	if !ok {
		return
	}
	var req service.VIPSubscriptionPayload
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
func (h *VIPSubscriptionHandler) Delete(c *gin.Context) {
	id, ok := templateResourceID(c, "VIP 订阅套餐")
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, nil)
}
func (h *VIPSubscriptionHandler) UpdateStatus(c *gin.Context) {
	id, ok := templateResourceID(c, "VIP 订阅套餐")
	if !ok {
		return
	}
	var req service.VIPSubscriptionStatusPayload
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
func (h *VIPSubscriptionHandler) UpdateDisplayMode(c *gin.Context) {
	id, ok := templateResourceID(c, "VIP 订阅套餐")
	if !ok {
		return
	}
	var req service.VIPSubscriptionDisplayPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.UpdateDisplayMode(c.Request.Context(), id, *req.DisplayMode); err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, nil)
}
func (h *VIPSubscriptionHandler) SetDefault(c *gin.Context) {
	id, ok := templateResourceID(c, "VIP 订阅套餐")
	if !ok {
		return
	}
	if err := h.svc.SetDefault(c.Request.Context(), id); err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, nil)
}
func (h *VIPSubscriptionHandler) Clone(c *gin.Context) {
	id, ok := templateResourceID(c, "VIP 订阅套餐")
	if !ok {
		return
	}
	var req service.CloneVIPSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	item, err := h.svc.Clone(c.Request.Context(), id, &req)
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, item)
}
