package handler

import (
	"strconv"

	"ai-video/internal/pkg/errcode"
	"ai-video/internal/pkg/response"
	"ai-video/internal/pkg/utils"
	"ai-video/internal/server/admin/service"

	"github.com/gin-gonic/gin"
)

type DelayConfigHandler struct {
	svc *service.DelayConfigService
}

func NewDelayConfigHandler() *DelayConfigHandler {
	return &DelayConfigHandler{svc: service.NewDelayConfigService()}
}

func (h *DelayConfigHandler) List(c *gin.Context) {
	pagination := utils.GetPagination(c)
	list, total, err := h.svc.List(c.Request.Context(), pagination.Page, pagination.PageSize, &service.ListDelayConfigRequest{
		Group: c.Query("group"), Keyword: c.Query("keyword"),
	})
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, gin.H{"list": list, "total": total, "page": pagination.Page, "size": pagination.PageSize})
}

func (h *DelayConfigHandler) ListGroups(c *gin.Context) {
	groups, err := h.svc.ListGroups(c.Request.Context())
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, groups)
}

func (h *DelayConfigHandler) GetByID(c *gin.Context) {
	id, ok := delayConfigID(c)
	if !ok {
		return
	}
	config, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, errcode.ErrNotFound, err.Error())
		return
	}
	response.OK(c, config)
}

func (h *DelayConfigHandler) Create(c *gin.Context) {
	var req service.CreateDelayConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.Create(c.Request.Context(), &req); err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *DelayConfigHandler) Update(c *gin.Context) {
	id, ok := delayConfigID(c)
	if !ok {
		return
	}
	var req service.UpdateDelayConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.Update(c.Request.Context(), id, &req); err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, nil)
}

type batchDelayConfigRequest struct {
	Items []service.DelayConfigValueItem `json:"items" binding:"required,min=1,dive"`
}

func (h *DelayConfigHandler) BatchUpdateValues(c *gin.Context) {
	var req batchDelayConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.BatchUpdateValues(c.Request.Context(), req.Items); err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *DelayConfigHandler) Delete(c *gin.Context) {
	id, ok := delayConfigID(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *DelayConfigHandler) Sync(c *gin.Context) {
	if err := h.svc.SyncFromFile(); err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, nil)
}

func delayConfigID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Fail(c, errcode.ErrParam, "配置 ID 参数错误")
		return 0, false
	}
	return uint(id), true
}
