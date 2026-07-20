package handler

import (
	"ai-video/internal/pkg/errcode"
	"ai-video/internal/pkg/response"
	"ai-video/internal/pkg/utils"
	"ai-video/internal/repository"

	"github.com/gin-gonic/gin"
)

type UploadHandler struct{ repo *repository.UploadRepo }

func NewUploadHandler() *UploadHandler { return &UploadHandler{repo: repository.NewUploadRepo()} }

type adminUploadListRequest struct {
	UserType        int8   `form:"user_type" binding:"omitempty,oneof=1 2"`
	UserID          uint64 `form:"user_id"`
	MediaType       string `form:"media_type" binding:"omitempty,oneof=image video"`
	FileType        string `form:"file_type" binding:"max=32"`
	StorageProvider string `form:"storage_provider" binding:"omitempty,oneof=local aliyun_oss"`
	Keyword         string `form:"keyword" binding:"max=255"`
}

func (h *UploadHandler) List(c *gin.Context) {
	var req adminUploadListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	pagination := utils.GetPagination(c)
	list, total, err := h.repo.PageList(c.Request.Context(), pagination.Page, pagination.PageSize, &repository.UploadListFilter{
		UserType: req.UserType, UserID: req.UserID, MediaType: req.MediaType,
		FileType: req.FileType, StorageProvider: req.StorageProvider, Keyword: req.Keyword,
	})
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, gin.H{"list": list, "total": total, "page": pagination.Page, "size": pagination.PageSize})
}
