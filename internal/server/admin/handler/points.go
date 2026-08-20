package handler

import (
	"ai-video/internal/pkg/errcode"
	"ai-video/internal/pkg/response"
	"ai-video/internal/pkg/utils"
	"ai-video/internal/server/admin/service"

	"github.com/gin-gonic/gin"
)

type PointsHandler struct {
	svc *service.PointsService
}

func NewPointsHandler() *PointsHandler {
	return &PointsHandler{svc: service.NewPointsService()}
}

func (h *PointsHandler) List(c *gin.Context) {
	var req service.ListPointsRequest
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

func (h *PointsHandler) ListOptions(c *gin.Context) {
	list, err := h.svc.ListOptions(c.Request.Context())
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, list)
}

func (h *PointsHandler) GetByID(c *gin.Context) {
	id, ok := templateResourceID(c, "积分套餐")
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

func (h *PointsHandler) Create(c *gin.Context) {
	var req service.PointsPayload
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

func (h *PointsHandler) Update(c *gin.Context) {
	id, ok := templateResourceID(c, "积分套餐")
	if !ok {
		return
	}
	var req service.PointsPayload
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

func (h *PointsHandler) Delete(c *gin.Context) {
	id, ok := templateResourceID(c, "积分套餐")
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *PointsHandler) UpdateStatus(c *gin.Context) {
	id, ok := templateResourceID(c, "积分套餐")
	if !ok {
		return
	}
	var req service.PointsStatusPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.UpdateStatus(c.Request.Context(), id, req.Status); err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *PointsHandler) SetDefault(c *gin.Context) {
	id, ok := templateResourceID(c, "积分套餐")
	if !ok {
		return
	}
	if err := h.svc.SetDefault(c.Request.Context(), id); err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, nil)
}
