package handler

import (
	"ai-video/internal/pkg/errcode"
	"ai-video/internal/pkg/response"
	"ai-video/internal/pkg/utils"
	"ai-video/internal/server/admin/service"

	"github.com/gin-gonic/gin"
)

type DisplayPositionHandler struct {
	svc *service.DisplayPositionService
}

func NewDisplayPositionHandler() *DisplayPositionHandler {
	return &DisplayPositionHandler{svc: service.NewDisplayPositionService()}
}

func (h *DisplayPositionHandler) List(c *gin.Context) {
	var req service.ListDisplayPositionRequest
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

func (h *DisplayPositionHandler) ListOptions(c *gin.Context) {
	list, err := h.svc.ListOptions(c.Request.Context())
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, list)
}

func (h *DisplayPositionHandler) GetByID(c *gin.Context) {
	id, ok := templateResourceID(c, "展示位置")
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

func (h *DisplayPositionHandler) Create(c *gin.Context) {
	var req service.DisplayPositionPayload
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

func (h *DisplayPositionHandler) Update(c *gin.Context) {
	id, ok := templateResourceID(c, "展示位置")
	if !ok {
		return
	}
	var req service.DisplayPositionPayload
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

func (h *DisplayPositionHandler) Delete(c *gin.Context) {
	id, ok := templateResourceID(c, "展示位置")
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, nil)
}
