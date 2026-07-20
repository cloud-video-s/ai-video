package response

import (
	"net/http"

	"ai-video/internal/pkg/i18n"

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
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: localizedError(c, code, http.StatusOK, msg),
	})
}

func FailWithStatus(c *gin.Context, httpStatus int, code int, msg string) {
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
	return i18n.ErrorMessage(i18n.Locale(c), code, httpStatus)
}
