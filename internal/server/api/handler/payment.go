package handler

import (
	"errors"

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
