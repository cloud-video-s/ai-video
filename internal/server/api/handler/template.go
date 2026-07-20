package handler

import (
	"errors"
	"net/http"

	"ai-video/internal/middleware"
	"ai-video/internal/pkg/errcode"
	"ai-video/internal/pkg/response"
	apiservice "ai-video/internal/server/api/server"

	"github.com/gin-gonic/gin"
)

type TemplateHandler struct {
	svc *apiservice.ClientTemplateService
}

func NewTemplateHandler() *TemplateHandler {
	return &TemplateHandler{svc: apiservice.NewClientTemplateService()}
}

func (h *TemplateHandler) List(c *gin.Context) {
	var req apiservice.ClientTemplateRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	req.PositionKey = "homeCategory"
	list, err := h.svc.List(c, middleware.GetAPIUserID(c), &req)
	if err != nil {
		if errors.Is(err, apiservice.ErrClientTemplateAudienceMismatch) {
			response.FailWithStatus(c, http.StatusBadRequest, errcode.ErrParam, err.Error())
			return
		}
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, list)
}

func (h *TemplateHandler) Categories(c *gin.Context) {
	var req apiservice.ClientTemplateRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	req.PositionKey = "homeCategory"
	list, err := h.svc.List(c, middleware.GetAPIUserID(c), &req)
	if err != nil {
		if errors.Is(err, apiservice.ErrClientTemplateAudienceMismatch) {
			response.FailWithStatus(c, http.StatusBadRequest, errcode.ErrParam, err.Error())
			return
		}
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, list)
}

func (h *TemplateHandler) Recommend(c *gin.Context) {
	var req apiservice.ClientTemplateRecommendRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	list, err := h.svc.Recommend(c, middleware.GetAPIUserID(c), &req)
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, list)
}
