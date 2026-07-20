package upload

import (
	"errors"
	"net/http"
	"strconv"

	"ai-video/internal/pkg/errcode"
	"ai-video/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

type HTTPHandler struct {
	manager       *Manager
	recorder      CompletionRecorder
	ownerResolver func(*gin.Context) (UploadOwner, error)
}

type HTTPHandlerOption func(*HTTPHandler)

func WithCompletionRecording(recorder CompletionRecorder, resolver func(*gin.Context) (UploadOwner, error)) HTTPHandlerOption {
	return func(handler *HTTPHandler) {
		handler.recorder = recorder
		handler.ownerResolver = resolver
	}
}

func NewHTTPHandler(manager *Manager, options ...HTTPHandlerOption) *HTTPHandler {
	handler := &HTTPHandler{manager: manager}
	for _, option := range options {
		option(handler)
	}
	return handler
}

func (h *HTTPHandler) RegisterRoutes(group *gin.RouterGroup) {
	h.registerMediaRoutes(group.Group("/images"), MediaImage)
	h.registerMediaRoutes(group.Group("/videos"), MediaVideo)
}

func (h *HTTPHandler) registerMediaRoutes(group *gin.RouterGroup, kind MediaKind) {
	group.POST("/batches", h.initiate(kind))
	group.PUT("/:upload_id/chunks/:index", h.putChunk(kind))
	group.GET("/:upload_id", h.status(kind))
	group.POST("/:upload_id/complete", h.complete(kind))
}

func (h *HTTPHandler) initiate(kind MediaKind) gin.HandlerFunc {
	type request struct {
		Files []FileSpec `json:"files" binding:"required,min=1"`
	}
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
		var req request
		if err := c.ShouldBindJSON(&req); err != nil {
			response.FailWithStatus(c, http.StatusBadRequest, errcode.ErrParam, "参数错误: "+err.Error())
			return
		}
		owner, err := h.resolveOwner(c)
		if err != nil {
			handleHTTPError(c, err)
			return
		}
		sessions, err := h.manager.InitiateBatchForOwner(c.Request.Context(), kind, req.Files, owner)
		if err != nil {
			handleHTTPError(c, err)
			return
		}
		response.OK(c, gin.H{"uploads": sessions})
	}
}

func (h *HTTPHandler) putChunk(kind MediaKind) gin.HandlerFunc {
	return func(c *gin.Context) {
		index, err := strconv.Atoi(c.Param("index"))
		if err != nil || index < 0 {
			response.FailWithStatus(c, http.StatusBadRequest, errcode.ErrParam, "分片序号错误")
			return
		}
		uploadID := c.Param("upload_id")
		if _, _, err := h.requireAccess(c, uploadID, kind); err != nil {
			handleHTTPError(c, err)
			return
		}
		session, err := h.manager.PutChunk(
			c.Request.Context(), uploadID, index, c.Request.Body, c.GetHeader("X-Chunk-SHA256"),
		)
		if err != nil {
			handleHTTPError(c, err)
			return
		}
		response.OK(c, session)
	}
}

func (h *HTTPHandler) status(kind MediaKind) gin.HandlerFunc {
	return func(c *gin.Context) {
		session, _, err := h.requireAccess(c, c.Param("upload_id"), kind)
		if err != nil {
			handleHTTPError(c, err)
			return
		}
		response.OK(c, session)
	}
}

func (h *HTTPHandler) complete(kind MediaKind) gin.HandlerFunc {
	return func(c *gin.Context) {
		uploadID := c.Param("upload_id")
		_, owner, err := h.requireAccess(c, uploadID, kind)
		if err != nil {
			handleHTTPError(c, err)
			return
		}
		session, err := h.manager.Complete(c.Request.Context(), uploadID)
		if err != nil {
			handleHTTPError(c, err)
			return
		}
		if h.recorder != nil {
			if err := h.recorder.RecordCompleted(c.Request.Context(), CompletedUpload{Owner: owner, Session: *session}); err != nil {
				handleHTTPError(c, err)
				return
			}
		}
		response.OK(c, session)
	}
}

func (h *HTTPHandler) requireAccess(c *gin.Context, uploadID string, kind MediaKind) (*Session, UploadOwner, error) {
	session, err := h.manager.Status(c.Request.Context(), uploadID)
	if err != nil {
		return nil, UploadOwner{}, err
	}
	if session.Kind != kind {
		return nil, UploadOwner{}, ErrUploadKindMismatch
	}
	owner, err := h.resolveOwner(c)
	if err != nil {
		return nil, UploadOwner{}, err
	}
	if h.ownerResolver != nil && (session.UploaderType != owner.Type || session.UploaderID != owner.ID) {
		return nil, UploadOwner{}, ErrUploadNotFound
	}
	return session, owner, nil
}

func (h *HTTPHandler) resolveOwner(c *gin.Context) (UploadOwner, error) {
	if h.ownerResolver == nil {
		return UploadOwner{}, nil
	}
	owner, err := h.ownerResolver(c)
	if err != nil {
		return UploadOwner{}, err
	}
	if owner.ID == 0 || (owner.Type != UploaderAdmin && owner.Type != UploaderAPIUser) {
		return UploadOwner{}, uploadError(ErrInvalidRequest, "invalid upload owner")
	}
	return owner, nil
}

func handleHTTPError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrUploadNotFound):
		response.FailWithStatus(c, http.StatusNotFound, errcode.ErrNotFound, err.Error())
	case errors.Is(err, ErrUploadExpired):
		response.FailWithStatus(c, http.StatusGone, errcode.ErrParam, err.Error())
	case errors.Is(err, ErrFileTooLarge):
		response.FailWithStatus(c, http.StatusRequestEntityTooLarge, errcode.ErrParam, err.Error())
	case errors.Is(err, ErrMissingChunks), errors.Is(err, ErrChecksumMismatch):
		response.FailWithStatus(c, http.StatusConflict, errcode.ErrParam, err.Error())
	case errors.Is(err, ErrInvalidRequest), errors.Is(err, ErrUnsupportedType),
		errors.Is(err, ErrBatchTooLarge), errors.Is(err, ErrInvalidChunk), errors.Is(err, ErrUploadKindMismatch):
		response.FailWithStatus(c, http.StatusBadRequest, errcode.ErrParam, err.Error())
	default:
		response.FailWithStatus(c, http.StatusInternalServerError, errcode.ErrServer, err.Error())
	}
}
