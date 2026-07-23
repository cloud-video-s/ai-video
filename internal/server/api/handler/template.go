package handler

import (
	"errors"
	"net/http"
	"strconv"

	"ai-video/internal/middleware"
	"ai-video/internal/pkg/errcode"
	"ai-video/internal/pkg/response"
	apiservice "ai-video/internal/server/api/server"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type TemplateHandler struct {
	svc         *apiservice.ClientTemplateService
	favoriteSvc *apiservice.TemplateFavoriteService
}

func NewTemplateHandler() *TemplateHandler {
	return &TemplateHandler{
		svc: apiservice.NewClientTemplateService(), favoriteSvc: apiservice.NewTemplateFavoriteService(),
	}
}

func (h *TemplateHandler) List(c *gin.Context) {
	var req apiservice.ClientTemplateRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	list, err := h.svc.List(c, &req)
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
	list, err := h.svc.Categories(c, &req)
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
	list, err := h.svc.Recommend(c, &req)
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, list)
}

func (h *TemplateHandler) TemplateList(c *gin.Context) {
	var req apiservice.TemplateListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	list, err := h.svc.CategoryTemplateList(c, &req)
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

func (h *TemplateHandler) TemplateInfo(c *gin.Context) {
	var req apiservice.TemplateInfoRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	list, err := h.svc.ClientTemplateInfo(c, &req)
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, list)
}

func (h *TemplateHandler) Favorite(c *gin.Context) {
	h.setFavorite(c, true)
}

func (h *TemplateHandler) Unfavorite(c *gin.Context) {
	h.setFavorite(c, false)
}

func (h *TemplateHandler) setFavorite(c *gin.Context, favorited bool) {
	templateID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || templateID == 0 {
		response.FailWithStatus(c, http.StatusBadRequest, errcode.ErrParam, "模板 ID 无效")
		return
	}
	result, err := h.favoriteSvc.Set(c.Request.Context(), middleware.GetAPIUserID(c), templateID, favorited)
	if err == nil {
		response.OK(c, result)
		return
	}
	switch {
	case errors.Is(err, apiservice.ErrTemplateFavoriteBusy):
		response.FailWithStatus(c, http.StatusConflict, errcode.ErrParam, "收藏操作正在处理中")
	case errors.Is(err, gorm.ErrRecordNotFound):
		response.FailWithStatus(c, http.StatusNotFound, errcode.ErrNotFound, "模板不存在")
	default:
		response.Fail(c, errcode.ErrServer, err.Error())
	}
}
