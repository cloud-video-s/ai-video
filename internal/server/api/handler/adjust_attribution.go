package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"mime"
	"net/http"
	"strings"
	"time"

	"ai-video/internal/config"
	"ai-video/internal/middleware"
	"ai-video/internal/pkg/errcode"
	"ai-video/internal/pkg/response"
	apiservice "ai-video/internal/server/api/server"

	"github.com/gin-gonic/gin"
)

const adjustCallbackTokenHeader = "X-Adjust-Callback-Token"

var errAdjustCallbackTokenConflict = errors.New("conflicting Adjust callback tokens")

type AdjustAttributionHandler struct {
	service adjustAttributionService
}

type adjustAttributionService interface {
	ReportApp(ctx context.Context, userID uint64, req apiservice.AdjustAppReportRequest) (*apiservice.AdjustAppReportResult, error)
	Handle(ctx context.Context, input apiservice.AdjustCallbackInput) (*apiservice.AdjustCallbackResult, error)
}

func NewAdjustAttributionHandler() *AdjustAttributionHandler {
	return &AdjustAttributionHandler{service: apiservice.NewAdjustAttributionService()}
}

// ReportApp receives an Adjust SDK attribution snapshot whenever attribution
// changes for an authenticated APP user. Repeated reports are idempotent.
// The user ID is always taken from the Bearer token;
// callers cannot choose or override it in the JSON payload.
func (h *AdjustAttributionHandler) ReportApp(c *gin.Context) {
	limit := config.Cfg.Adjust.MaxBodyBytes
	if limit <= 0 {
		limit = 65536
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)

	var req apiservice.AdjustAppReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			response.FailWithStatus(c, http.StatusRequestEntityTooLarge, errcode.ErrParam, "Adjust APP report payload is too large")
			return
		}
		response.FailWithStatus(c, http.StatusBadRequest, errcode.ErrParam, "invalid Adjust APP report payload: "+err.Error())
		return
	}

	userID := middleware.GetAPIUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "invalid authenticated user")
		return
	}
	result, err := h.service.ReportApp(c.Request.Context(), userID, req)
	switch {
	case errors.Is(err, apiservice.ErrAdjustCallbackDisabled):
		response.FailWithStatus(c, http.StatusServiceUnavailable, errcode.ErrServer, err.Error())
		return
	case errors.Is(err, apiservice.ErrAdjustAppReportInvalid):
		response.FailWithStatus(c, http.StatusBadRequest, errcode.ErrParam, err.Error())
		return
	case errors.Is(err, apiservice.ErrAdjustAttributionConflict):
		response.FailWithStatus(c, http.StatusConflict, errcode.ErrParam, err.Error())
		return
	case err != nil:
		response.FailWithStatus(c, http.StatusInternalServerError, errcode.ErrServer, err.Error())
		return
	}

	config.Logger(c.Request.Context()).Infow("Adjust APP attribution report accepted",
		"user_id", userID, "adid", result.ADID,
		"fused", result.Fused, "applied", result.Applied, "status", result.Status,
	)
	response.OK(c, result)
}

func (h *AdjustAttributionHandler) Callback(c *gin.Context) {
	token, payload, err := bindAdjustCallback(c)
	config.Logger(c.Request.Context()).Infow("AdjustAttribution payload", payload)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			response.FailWithStatus(c, http.StatusRequestEntityTooLarge, errcode.ErrParam, "Adjust callback payload is too large")
			return
		}
		if errors.Is(err, errAdjustCallbackTokenConflict) {
			response.FailWithStatus(c, http.StatusUnauthorized, errcode.ErrTokenInvalid, "invalid Adjust callback token")
			return
		}
		response.FailWithStatus(c, http.StatusBadRequest, errcode.ErrParam, "invalid Adjust callback payload: "+err.Error())
		return
	}

	result, err := h.service.Handle(c.Request.Context(), apiservice.AdjustCallbackInput{
		Token: token, Payload: payload, ReceivedAt: time.Now(),
	})
	switch {
	case errors.Is(err, apiservice.ErrAdjustCallbackDisabled):
		response.FailWithStatus(c, http.StatusServiceUnavailable, errcode.ErrServer, err.Error())
		return
	case errors.Is(err, apiservice.ErrAdjustCallbackUnauthorized):
		response.FailWithStatus(c, http.StatusUnauthorized, errcode.ErrTokenInvalid, err.Error())
		return
	case errors.Is(err, apiservice.ErrAdjustCallbackInvalid):
		response.FailWithStatus(c, http.StatusBadRequest, errcode.ErrParam, err.Error())
		return
	case err != nil:
		response.FailWithStatus(c, http.StatusInternalServerError, errcode.ErrServer, err.Error())
		return
	}

	config.Logger(c.Request.Context()).Infow("Adjust attribution callback acknowledged",
		"duplicate", result.Duplicate, "matched", result.Matched,
		"applied", result.Applied, "status", result.Status,
	)
	c.JSON(http.StatusOK, gin.H{
		"code": 0, "message": "acknowledged", "data": result,
	})
}

func bindAdjustCallback(c *gin.Context) (string, map[string]any, error) {
	payload := make(map[string]any)
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			payload[key] = values[len(values)-1]
		}
	}

	if c.Request.Method != http.MethodGet {
		limit := config.Cfg.Adjust.MaxBodyBytes
		if limit <= 0 {
			limit = 65536
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		mediaType, _, _ := mime.ParseMediaType(c.GetHeader("Content-Type"))
		if mediaType == "application/json" {
			decoder := json.NewDecoder(c.Request.Body)
			decoder.UseNumber()
			var body map[string]any
			if err := decoder.Decode(&body); err != nil {
				return "", nil, err
			}
			if err := ensureJSONEOF(decoder); err != nil {
				return "", nil, err
			}
			maps.Copy(payload, body)
		} else {
			if err := c.Request.ParseForm(); err != nil {
				return "", nil, err
			}
			for key, values := range c.Request.PostForm {
				if len(values) > 0 {
					payload[key] = values[len(values)-1]
				}
			}
		}
	}

	parameterToken := callbackPayloadString(payload, "callback_token")
	headerToken := strings.TrimSpace(c.GetHeader(adjustCallbackTokenHeader))
	if headerToken != "" && parameterToken != "" && headerToken != parameterToken {
		return "", nil, errAdjustCallbackTokenConflict
	}
	token := parameterToken
	if headerToken != "" {
		token = headerToken
	}
	//for key := range payload {
	//	if strings.EqualFold(strings.TrimSpace(key), "callback_token") {
	//		delete(payload, key)
	//	}
	//}
	return token, payload, nil
}

func callbackPayloadString(payload map[string]any, name string) string {
	for key, value := range payload {
		if strings.EqualFold(strings.TrimSpace(key), name) {
			if text, ok := value.(string); ok {
				return strings.TrimSpace(text)
			}
		}
	}
	return ""
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var trailing any
	err := decoder.Decode(&trailing)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return errors.New("multiple JSON values are not allowed")
	}
	return err
}
