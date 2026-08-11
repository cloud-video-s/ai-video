package handler

import (
	"errors"
	"net/http"
	"strings"

	"ai-video/internal/pkg/errcode"
	"ai-video/internal/pkg/response"
	"ai-video/internal/pkg/upload"
	"ai-video/internal/pkg/uploadruntime"
	"ai-video/internal/server/admin/service"

	"github.com/gin-gonic/gin"
)

const configFileMultipartOverhead = int64(1 << 20)

type ConfigFileHandler struct {
	svc *service.ConfigFileService
}

func NewConfigFileHandler(storage upload.Storage) *ConfigFileHandler {
	return &ConfigFileHandler{svc: service.NewConfigFileService(storage)}
}

func (h *ConfigFileHandler) Upload(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, service.ConfigFileMaxSize+configFileMultipartOverhead)
	if err := c.Request.ParseMultipartForm(configFileMultipartOverhead); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			response.FailWithStatus(c, http.StatusRequestEntityTooLarge, errcode.ErrParam, "配置文件最大允许 5 MB")
			return
		}
		response.FailWithStatus(c, http.StatusBadRequest, errcode.ErrParam, "multipart 表单错误: "+err.Error())
		return
	}
	if c.Request.MultipartForm != nil {
		defer c.Request.MultipartForm.RemoveAll()
	}

	configKey := strings.TrimSpace(c.Request.FormValue("config_key"))
	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, errcode.ErrParam, "请选择要上传的配置文件")
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, errcode.ErrParam, "读取上传文件失败")
		return
	}
	defer file.Close()

	result, err := h.svc.Store(c.Request.Context(), configKey, fileHeader.Filename, file)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrConfigFileTooLarge):
			response.FailWithStatus(c, http.StatusRequestEntityTooLarge, errcode.ErrParam, err.Error())
		case errors.Is(err, service.ErrInvalidConfigFile), errors.Is(err, service.ErrUnsupportedConfigFile):
			response.FailWithStatus(c, http.StatusBadRequest, errcode.ErrParam, err.Error())
		default:
			response.FailWithStatus(c, http.StatusInternalServerError, errcode.ErrServer, err.Error())
		}
		return
	}
	response.OK(c, gin.H{
		"original_name": result.OriginalName,
		"content_type":  result.ContentType,
		"size":          result.Size,
		"file_path":     result.FilePath,
		"file_url":      result.FileURL,
		"preview_url":   uploadruntime.PublicURL(result.FileURL),
	})
}
