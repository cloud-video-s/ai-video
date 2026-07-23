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
