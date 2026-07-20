package handler

import (
	"strconv"

	"ai-video/internal/pkg/errcode"
	"ai-video/internal/pkg/response"
	"ai-video/internal/pkg/utils"
	"ai-video/internal/server/admin/service"

	"github.com/gin-gonic/gin"
)

type CountryHandler struct {
	svc *service.CountryService
}

func NewCountryHandler() *CountryHandler {
	return &CountryHandler{svc: service.NewCountryService()}
}

func (h *CountryHandler) List(c *gin.Context) {
	var req service.ListCountryRequest
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

func (h *CountryHandler) ListOptions(c *gin.Context) {
	list, err := h.svc.ListOptions(c.Request.Context())
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, list)
}

func (h *CountryHandler) GetByID(c *gin.Context) {
	id, ok := countryResourceID(c)
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

func (h *CountryHandler) Create(c *gin.Context) {
	var req service.CountryPayload
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

func (h *CountryHandler) Update(c *gin.Context) {
	id, ok := countryResourceID(c)
	if !ok {
		return
	}
	var req service.CountryPayload
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

func (h *CountryHandler) UpdateStatus(c *gin.Context) {
	id, ok := countryResourceID(c)
	if !ok {
		return
	}
	var req service.CountryStatusPayload
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

func (h *CountryHandler) Delete(c *gin.Context) {
	id, ok := countryResourceID(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, nil)
}

func countryResourceID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Fail(c, errcode.ErrParam, "国家 ID 无效")
		return 0, false
	}
	return id, true
}
