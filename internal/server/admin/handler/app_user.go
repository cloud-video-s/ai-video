package handler

import (
	"strconv"

	"ai-video/internal/pkg/errcode"
	"ai-video/internal/pkg/response"
	"ai-video/internal/pkg/utils"
	"ai-video/internal/server/admin/service"

	"github.com/gin-gonic/gin"
)

type AppUserHandler struct {
	svc *service.AppUserService
}

func NewAppUserHandler() *AppUserHandler {
	return &AppUserHandler{svc: service.NewAppUserService()}
}

func (h *AppUserHandler) Create(c *gin.Context) {
	var req service.CreateAppUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	user, err := h.svc.Create(c.Request.Context(), &req)
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, user)
}

func (h *AppUserHandler) GetByID(c *gin.Context) {
	id, ok := appUserID(c)
	if !ok {
		return
	}
	user, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, errcode.ErrNotFound, err.Error())
		return
	}
	response.OK(c, user)
}

func (h *AppUserHandler) Lookup(c *gin.Context) {
	user, err := h.svc.Lookup(c.Request.Context(), c.Query("query"))
	if err != nil {
		response.Fail(c, errcode.ErrNotFound, err.Error())
		return
	}
	response.OK(c, user)
}

func (h *AppUserHandler) GetCenter(c *gin.Context) {
	id, ok := appUserID(c)
	if !ok {
		return
	}
	detail, err := h.svc.GetCenter(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, detail)
}

func (h *AppUserHandler) SetFrozen(c *gin.Context) {
	id, ok := appUserID(c)
	if !ok {
		return
	}
	var req service.UserAccessStateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.SetFrozen(c.Request.Context(), id, req.Enabled); err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *AppUserHandler) SetBlacklisted(c *gin.Context) {
	id, ok := appUserID(c)
	if !ok {
		return
	}
	var req service.UserAccessStateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.SetBlacklisted(c.Request.Context(), id, req.Enabled); err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *AppUserHandler) BindPhone(c *gin.Context) {
	id, ok := appUserID(c)
	if !ok {
		return
	}
	var req service.BindUserPhoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.BindPhone(c.Request.Context(), id, req.Phone); err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *AppUserHandler) GrantVIP(c *gin.Context) {
	id, ok := appUserID(c)
	if !ok {
		return
	}
	var req service.GrantUserVIPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.GrantVIP(c.Request.Context(), id, &req); err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *AppUserHandler) ExtendVIP(c *gin.Context) {
	id, ok := appUserID(c)
	if !ok {
		return
	}
	var req service.ExtendUserVIPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.ExtendVIP(c.Request.Context(), id, req.Days); err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *AppUserHandler) TransferVIP(c *gin.Context) {
	id, ok := appUserID(c)
	if !ok {
		return
	}
	var req service.TransferUserVIPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.TransferVIP(c.Request.Context(), id, req.TargetUserID); err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *AppUserHandler) TerminateVIP(c *gin.Context) {
	id, ok := appUserID(c)
	if !ok {
		return
	}
	if err := h.svc.TerminateVIP(c.Request.Context(), id); err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *AppUserHandler) ClearDevice(c *gin.Context) {
	id, ok := appUserID(c)
	if !ok {
		return
	}
	if err := h.svc.ClearDevice(c.Request.Context(), id); err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *AppUserHandler) Update(c *gin.Context) {
	id, ok := appUserID(c)
	if !ok {
		return
	}
	var req service.UpdateAppUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	user, err := h.svc.Update(c.Request.Context(), id, &req)
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, user)
}

func (h *AppUserHandler) Delete(c *gin.Context) {
	id, ok := appUserID(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *AppUserHandler) List(c *gin.Context) {
	var req service.ListAppUserRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	p := utils.GetPagination(c)
	users, total, err := h.svc.List(c.Request.Context(), p.Page, p.PageSize, &req)
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, gin.H{"list": users, "total": total, "page": p.Page, "size": p.PageSize})
}

func appUserID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Fail(c, errcode.ErrParam, "用户 ID 参数错误")
		return 0, false
	}
	return id, true
}
