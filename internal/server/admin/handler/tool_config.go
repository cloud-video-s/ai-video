package handler

import (
	"strconv"

	"ai-video/internal/pkg/errcode"
	"ai-video/internal/pkg/response"
	"ai-video/internal/pkg/utils"
	"ai-video/internal/server/admin/service"

	"github.com/gin-gonic/gin"
)

type ToolConfigHandler struct {
	svc *service.ToolConfigService
}

func NewToolConfigHandler() *ToolConfigHandler {
	return &ToolConfigHandler{svc: service.NewToolConfigService()}
}

func (h *ToolConfigHandler) List(c *gin.Context) {
	var req service.ListToolConfigRequest
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

func (h *ToolConfigHandler) ListOptions(c *gin.Context) {
	list, err := h.svc.ListOptions(c.Request.Context())
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, list)
}

func (h *ToolConfigHandler) ListModelOptions(c *gin.Context) {
	var req service.ListToolModelOptionRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	list, err := h.svc.ListModelOptions(c.Request.Context(), req.ToolType)
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, list)
}

func (h *ToolConfigHandler) GetByID(c *gin.Context) {
	id, ok := toolConfigResourceID(c)
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

func (h *ToolConfigHandler) Create(c *gin.Context) {
	var req service.ToolConfigPayload
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

func (h *ToolConfigHandler) Update(c *gin.Context) {
	id, ok := toolConfigResourceID(c)
	if !ok {
		return
	}
	var req service.ToolConfigPayload
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

func (h *ToolConfigHandler) UpdateStatus(c *gin.Context) {
	id, ok := toolConfigResourceID(c)
	if !ok {
		return
	}
	var req service.ToolConfigStatusPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	item, err := h.svc.UpdateStatus(c.Request.Context(), id, req.Status)
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, item)
}

func (h *ToolConfigHandler) Delete(c *gin.Context) {
	id, ok := toolConfigResourceID(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, nil)
}

func toolConfigResourceID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Fail(c, errcode.ErrParam, "工具配置 ID 无效")
		return 0, false
	}
	return id, true
}
