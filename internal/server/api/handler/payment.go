package handler

import (
	"ai-video/internal/config"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"

	"ai-video/internal/commerce"
	"ai-video/internal/middleware"
	"ai-video/internal/pkg/errcode"
	"ai-video/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

type PaymentHandler struct {
	service *commerce.Service
}

func NewPaymentHandler() *PaymentHandler {
	return &PaymentHandler{service: commerce.NewService()}
}

// ConfirmApple receives the StoreKit result after an app purchase. The
// authenticated package header is used as the expected Apple bundle ID.
func (h *PaymentHandler) ConfirmApple(c *gin.Context) {
	var req commerce.ApplePurchaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ErrParam, "苹果支付参数错误: "+err.Error())
		return
	}
	result, err := h.service.ConfirmApplePurchase(
		c.Request.Context(), middleware.GetAPIUserID(c), middleware.GetAPIAppPackageCode(c), req,
	)
	if err != nil {
		if isApplePaymentInputError(err) {
			response.Fail(c, errcode.ErrParam, err.Error())
			return
		}
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, result)
}

// AppleServerNotification 接收 App Store Server Notifications V2 的 Webhook 回调。
// 该端点为公开端点（无需鉴权），Apple 服务器会异步推送退款、续费、订阅过期等事件。
// 为防止重复处理，所有业务逻辑均基于幂等键设计。
// 参考文档：https://developer.apple.com/documentation/appstoreservernotifications
func (h *PaymentHandler) AppleServerNotification(c *gin.Context) {
	params, _ := GetAllParams(c)
	config.Log.Infow("request",
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
		"query", c.Request.URL.RawQuery,
		"body", c.Request.Body,
		"status", c.Writer.Status(),
		"params", params,
		"ip", c.ClientIP(),
		"latency", time.Now().Unix(),
		"errors", c.Errors.ByType(gin.ErrorTypePrivate).String(),
	)
	var body commerce.AppleNotificationRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.FailWithStatus(c, 400, errcode.ErrParam, "invalid notification body: "+err.Error())
		return
	}
	summary, err := h.service.HandleAppleServerNotification(c.Request.Context(), body.SignedPayload)
	if err != nil {
		if errors.Is(err, commerce.ErrAppleEvidenceInvalid) ||
			errors.Is(err, commerce.ErrAppleSignatureInvalid) {
			response.FailWithStatus(c, 400, errcode.ErrParam, err.Error())
			return
		}
		response.FailWithStatus(c, 500, errcode.ErrServer, err.Error())
		return
	}
	// App Store 要求 Webhook 必须在 200-299 范围内返回才视为成功，否则会指数退避重试。
	c.JSON(200, gin.H{
		"code":    0,
		"message": "acknowledged",
		"data":    summary,
	})
}

func isApplePaymentInputError(err error) bool {
	return errors.Is(err, commerce.ErrAppleEvidenceInvalid) ||
		errors.Is(err, commerce.ErrAppleSignatureInvalid) ||
		errors.Is(err, commerce.ErrAppleUnsignedProduction) ||
		errors.Is(err, commerce.ErrAppleBundleMismatch) ||
		errors.Is(err, commerce.ErrAppleProductNotFound) ||
		errors.Is(err, commerce.ErrAppleProductAmbiguous) ||
		errors.Is(err, commerce.ErrApplePurchaseInactive) ||
		errors.Is(err, commerce.ErrApplePurchaseRevoked) ||
		errors.Is(err, commerce.ErrPaymentMismatch) ||
		errors.Is(err, commerce.ErrPaymentTransactionUsed)
}

// GetAllParams 从 Gin Context 中提取所有参数（路径、查询、表单、JSON Body）
// 注意：调用后会读取 Body，若后续还需绑定结构体，请先缓存 Body 或使用 c.ShouldBind 前调用
func GetAllParams(c *gin.Context) (map[string]any, error) {
	params := make(map[string]any)

	// 1. 路径参数（例如 /user/:id）
	for _, param := range c.Params {
		params[param.Key] = param.Value
	}

	// 2. 查询参数（URL 中的 ?key=value）
	query := c.Request.URL.Query()
	for key, values := range query {
		if len(values) == 1 {
			params[key] = values[0]
		} else {
			params[key] = values
		}
	}

	// 3. 表单参数（application/x-www-form-urlencoded 或 multipart/form-data）
	contentType := c.ContentType()

	if strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
		// 解析表单
		if err := c.Request.ParseForm(); err != nil {
			return nil, err
		}
		for key, values := range c.Request.Form {
			mergeParam(params, key, values)
		}
	}

	if strings.HasPrefix(contentType, "multipart/form-data") {
		// 解析 multipart 表单（内存限制 10MB，可根据需要调整）
		if err := c.Request.ParseMultipartForm(10 << 20); err != nil {
			return nil, err
		}
		// 普通字段
		for key, values := range c.Request.Form {
			mergeParam(params, key, values)
		}
		// 文件字段（记录文件名）
		if c.Request.MultipartForm != nil {
			for key, fileHeaders := range c.Request.MultipartForm.File {
				var names []string
				for _, fh := range fileHeaders {
					names = append(names, fh.Filename)
				}
				if len(names) == 1 {
					params[key] = names[0]
				} else {
					params[key] = names
				}
			}
		}
	}

	// 4. JSON Body（application/json）
	if strings.HasPrefix(contentType, "application/json") {
		// 读取 Body（Gin 提供了 c.GetRawData()，但会消耗 Body）
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			return nil, err
		}
		// 恢复 Body，以便后续其他绑定（如 c.ShouldBindJSON）使用
		c.Request.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

		var jsonData map[string]any
		if err := json.Unmarshal(bodyBytes, &jsonData); err == nil {
			// 合并 JSON（这里让 JSON 覆盖同名键，可根据需求调整）
			for key, value := range jsonData {
				params[key] = value
			}
		}
		// 如果 JSON 解析失败，可选择忽略或返回错误
	}

	return params, nil
}

// mergeParam 辅助函数，将表单值合并到 params 中（如果键已存在，转为切片）
func mergeParam(params map[string]interface{}, key string, values []string) {
	if existing, ok := params[key]; ok {
		// 已存在，转为切片合并
		switch v := existing.(type) {
		case string:
			params[key] = []string{v, values[0]}
		case []string:
			params[key] = append(v, values...)
		}
	} else {
		if len(values) == 1 {
			params[key] = values[0]
		} else {
			params[key] = values
		}
	}
}
