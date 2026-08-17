package response

import (
	"errors"
	"net/http"
	"strings"

	"ai-video/internal/pkg/errcode"
	"ai-video/internal/pkg/i18n"
	"ai-video/internal/pkg/monitor"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

func OKWithMessage(c *gin.Context, msg string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: msg,
		Data:    data,
	})
}

func Fail(c *gin.Context, code int, msg string) {
	recordServerError(c, code, http.StatusOK, msg)
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: localizedError(c, code, http.StatusOK, msg),
	})
}

func FailWithStatus(c *gin.Context, httpStatus int, code int, msg string) {
	recordServerError(c, code, httpStatus, msg)
	c.JSON(httpStatus, Response{
		Code:    code,
		Message: localizedError(c, code, httpStatus, msg),
	})
}

func Unauthorized(c *gin.Context, msg string) {
	c.JSON(http.StatusUnauthorized, Response{
		Code:    401,
		Message: localizedError(c, http.StatusUnauthorized, http.StatusUnauthorized, msg),
	})
	c.Abort()
}

func Forbidden(c *gin.Context, msg string) {
	c.JSON(http.StatusForbidden, Response{
		Code:    403,
		Message: localizedError(c, http.StatusForbidden, http.StatusForbidden, msg),
	})
	c.Abort()
}

func NotFound(c *gin.Context, msg string) {
	c.JSON(http.StatusNotFound, Response{
		Code:    404,
		Message: localizedError(c, http.StatusNotFound, http.StatusNotFound, msg),
	})
	c.Abort()
}

func localizedError(c *gin.Context, code, httpStatus int, fallback string) string {
	if !i18n.IsAPI(c) {
		return fallback
	}
	recordAPIError(c, fallback)
	return i18n.ErrorMessage(i18n.LocaleEnUS, code, httpStatus)
}

func recordServerError(c *gin.Context, code, httpStatus int, original string) {
	if strings.TrimSpace(original) == "" || (code != errcode.ErrServer && httpStatus < http.StatusInternalServerError) {
		return
	}
	monitor.MarkHTTPError(c, errors.New(original))
	// The API sanitizer records the same private error below. Admin requests do
	// not pass through it, so attach the error here for their request logs.
	if !i18n.IsAPI(c) {
		recordAPIError(c, original)
	}
}

func recordAPIError(c *gin.Context, original string) {
	if strings.TrimSpace(original) == "" {
		return
	}
	_ = c.Error(errors.New(original)).SetType(gin.ErrorTypePrivate)
}
