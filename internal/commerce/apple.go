package commerce

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"
	"time"

	"ai-video/internal/config"
	"ai-video/internal/domain"
	"ai-video/internal/gen/model"
	"ai-video/internal/repository"

	"gorm.io/gorm"
)

var (
	ErrAppleEvidenceInvalid    = errors.New("invalid Apple transaction evidence")
	ErrAppleSignatureInvalid   = errors.New("Apple transaction signature verification failed")
	ErrAppleUnsignedProduction = errors.New("unsigned Apple transaction is not allowed in production")
	ErrAppleBundleMismatch     = errors.New("Apple transaction bundle does not match request package")
	ErrAppleProductNotFound    = errors.New("Apple product is not configured for this package")
	ErrAppleProductAmbiguous   = errors.New("Apple product is configured as more than one product type")
	ErrApplePurchaseInactive   = errors.New("Apple subscription is inactive or expired")
	ErrApplePurchaseRevoked    = errors.New("Apple transaction has been revoked")
)

// App Store Server Notifications V2 通知类型常量
// https://developer.apple.com/documentation/appstoreservernotifications/notificationtype
const (
	AppleNotificationConsumptionRequest           = "CONSUMPTION_REQUEST"
	AppleNotificationDidChangeRenewalPref         = "DID_CHANGE_RENEWAL_PREF"
	AppleNotificationDidChangeRenewalStatus       = "DID_CHANGE_RENEWAL_STATUS"
	AppleNotificationDidFailToRenew               = "DID_FAIL_TO_RENEW"
	AppleNotificationDidRenew                     = "DID_RENEW"
	AppleNotificationExpired                      = "EXPIRED"
	AppleNotificationGracePeriodExpired           = "GRACE_PERIOD_EXPIRED"
	AppleNotificationOfferRedeemed                = "OFFER_REDEEMED"
	AppleNotificationOneTimeCharge                = "ONE_TIME_CHARGE"
	AppleNotificationPriceIncrease                = "PRICE_INCREASE"
	AppleNotificationRefund                       = "REFUND"
	AppleNotificationRefundDeclined               = "REFUND_DECLINED"
	AppleNotificationRefundReversed               = "REFUND_REVERSED"
	AppleNotificationRenewalExtension             = "RENEWAL_EXTENSION"
	AppleNotificationRenewalExtensionFailed       = "RENEWAL_EXTENSION_FAILED"
	AppleNotificationRevoke                       = "REVOKE"
	AppleNotificationSubscribed                   = "SUBSCRIBED"
	AppleNotificationTypeSummary                  = "SUMMARY"
	AppleNotificationTest                         = "TEST"
	AppleNotificationExternalPurchaseTokenSigned  = "EXTERNAL_PURCHASE_TOKEN_SIGNED"
	AppleNotificationExternalPurchaseTokenRevoked = "EXTERNAL_PURCHASE_TOKEN_REVOKED"
)

// App Store Server Notifications V2 子类型常量
// https://developer.apple.com/documentation/appstoreservernotifications/subtype
const (
	AppleSubtypeInitialBuy        = "INITIAL_BUY"
	AppleSubtypeResubscribe       = "RESUBSCRIBE"
	AppleSubtypeDowngrade         = "DOWNGRADE"
	AppleSubtypeUpgrade           = "UPGRADE"
	AppleSubtypeAutoRenewDisabled = "AUTO_RENEW_DISABLED"
	AppleSubtypeAutoRenewEnabled  = "AUTO_RENEW_ENABLED"
	AppleSubtypeBillingRecovery   = "BILLING_RECOVERY"
	AppleSubtypeBillingRetry      = "BILLING_RETRY"
	AppleSubtypePriceIncrease     = "PRICE_INCREASE"
	AppleSubtypeGracePeriod       = "GRACE_PERIOD"
	AppleSubtypePending           = "PENDING"
	AppleSubtypeAccepted          = "ACCEPTED"
	AppleSubtypeSummary           = "SUMMARY"
	AppleSubtypeFailure           = "FAILURE"
	AppleSubtypeVoluntary         = "VOLUNTARY"
	AppleSubtypeProductForSale    = "PRODUCT_FOR_SALE"
	AppleSubtypeProductNotForSale = "PRODUCT_NOT_FOR_SALE"
)

// AppleNotificationRequest App Store Server Notifications V2 的请求体
type AppleNotificationRequest struct {
	SignedPayload string `json:"signedPayload" binding:"required"`
}

// AppleNotificationSummary 处理后返回的摘要
type AppleNotificationSummary struct {
	NotificationType    string `json:"notification_type"`
	Subtype             string `json:"subtype,omitempty"`
	NotificationUUID    string `json:"notification_uuid,omitempty"`
	BundleID            string `json:"bundle_id,omitempty"`
	Environment         string `json:"environment,omitempty"`
	OriginalTransaction string `json:"original_transaction_id,omitempty"`
	TransactionID       string `json:"transaction_id,omitempty"`
	ProductID           string `json:"product_id,omitempty"`
	Processed           bool   `json:"processed"`
	AffectedUserID      uint64 `json:"affected_user_id,omitempty"`
	AffectedOrderNo     string `json:"affected_order_no,omitempty"`
	Action              string `json:"action,omitempty"`
	Message             string `json:"message,omitempty"`
}

// ApplePurchaseRequest mirrors the StoreKit result returned by the app. The
// signedTransactionInfo field may be a compact JWS in production. A decoded
// JSON value is accepted only for Sandbox while the server is not in release
// mode, matching the development payload supplied by the client.
type ApplePurchaseRequest struct {
	BundleID              string     `json:"bundleID" binding:"required,max=191"`
	ExpirationDate        *time.Time `json:"expirationDate"`
	IsActive              bool       `json:"isActive"`
	OriginalTransactionID string     `json:"originalTransactionID" binding:"required,max=191"`
	ProductID             string     `json:"productID" binding:"required,max=191"`
	PurchaseDate          time.Time  `json:"purchaseDate" binding:"required"`
	RevocationDate        *time.Time `json:"revocationDate"`
	SignedTransactionInfo string     `json:"signedTransactionInfo" binding:"required"`
	Source                string     `json:"source" binding:"omitempty,max=64"`
	TransactionID         string     `json:"transactionID" binding:"required,max=191"`
}

type appleSignedTransaction struct {
	TransactionID         string `json:"transactionId"`
	OriginalTransactionID string `json:"originalTransactionId"`
	BundleID              string `json:"bundleId"`
	ProductID             string `json:"productId"`
	PurchaseDate          int64  `json:"purchaseDate"`
	OriginalPurchaseDate  int64  `json:"originalPurchaseDate"`
	ExpiresDate           int64  `json:"expiresDate"`
	RevocationDate        int64  `json:"revocationDate"`
	RevocationReason      string `json:"revocationReason"`
	Quantity              int64  `json:"quantity"`
	Type                  string `json:"type"`
	SignedDate            int64  `json:"signedDate"`
	Environment           string `json:"environment"`
	TransactionReason     string `json:"transactionReason"`
	Price                 int64  `json:"price"`
	Currency              string `json:"currency"`
	OfferType             int64  `json:"offerType"`
	OfferIdentifier       string `json:"offerIdentifier"`
}

type verifiedAppleTransaction struct {
	appleSignedTransaction
	PurchaseAt   time.Time
	ExpiresAt    *time.Time
	RevokedAt    *time.Time
	PaidAmount   float64
	EvidenceMode string
}

type ApplePurchaseResponse struct {
	OrderNo               string     `json:"order_no"`
	Status                string     `json:"status"`
	ProductType           string     `json:"product_type"`
	ProductID             uint64     `json:"product_id"`
	ProductCode           string     `json:"product_code"`
	TransactionID         string     `json:"transaction_id"`
	OriginalTransactionID string     `json:"original_transaction_id"`
	Currency              string     `json:"currency"`
	PaidAmount            float64    `json:"paid_amount"`
	PurchaseDate          time.Time  `json:"purchase_date"`
	ExpirationDate        *time.Time `json:"expiration_date,omitempty"`
	IsActive              bool       `json:"is_active"`
	Environment           string     `json:"environment"`
	EvidenceMode          string     `json:"evidence_mode"`
}

// DecodedAppleNotification 解码后的苹果服务器通知
type DecodedAppleNotification struct {
	NotificationType    string
	Subtype             string
	NotificationUUID    string
	BundleID            string
	Environment         string
	SignedRenewalInfo   string
	SignedTransaction   string
	OriginalTransaction string
	TransactionID       string
	ProductID           string
	PurchaseDate        int64
	ExpiresDate         int64
	RevocationDate      int64
	RevocationReason    string
	TransactionReason   string
}

// DecodeAppleNotificationPayload 从 Apple JWS 通知的 signedPayload 中解码业务字段
func DecodeAppleNotificationPayload(signedPayload string) (*DecodedAppleNotification, error) {
	signedPayload = strings.TrimSpace(signedPayload)
	if signedPayload == "" || strings.Count(signedPayload, ".") != 2 {
		return nil, ErrAppleEvidenceInvalid
	}
	parts := strings.Split(signedPayload, ".")
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrAppleSignatureInvalid
	}
	var payload struct {
		NotificationType  string `json:"notificationType"`
		Subtype           string `json:"subtype"`
		NotificationUUID  string `json:"notificationUUID"`
		BundleID          string `json:"bundleId"`
		Environment       string `json:"environment"`
		SignedRenewalInfo string `json:"signedRenewalInfo"`
		SignedTransaction string `json:"signedTransactionInfo"`
	}
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return nil, fmt.Errorf("%w: invalid notification JSON", ErrAppleEvidenceInvalid)
	}
	result := &DecodedAppleNotification{
		NotificationType:  payload.NotificationType,
		Subtype:           payload.Subtype,
		NotificationUUID:  payload.NotificationUUID,
		BundleID:          payload.BundleID,
		Environment:       payload.Environment,
		SignedRenewalInfo: payload.SignedRenewalInfo,
		SignedTransaction: payload.SignedTransaction,
	}
	if payload.SignedTransaction != "" && strings.Count(payload.SignedTransaction, ".") == 2 {
		txParts := strings.Split(payload.SignedTransaction, ".")
		txPayload, txErr := base64.RawURLEncoding.DecodeString(txParts[1])
		if txErr == nil {
			var tx appleSignedTransaction
			if json.Unmarshal(txPayload, &tx) == nil {
				result.OriginalTransaction = tx.OriginalTransactionID
				result.TransactionID = tx.TransactionID
				result.ProductID = tx.ProductID
				result.PurchaseDate = tx.PurchaseDate
				result.ExpiresDate = tx.ExpiresDate
				result.RevocationDate = tx.RevocationDate
				result.RevocationReason = tx.RevocationReason
				result.TransactionReason = tx.TransactionReason
			}
		}
	}
	return result, nil
}

// HandleAppleServerNotification 处理 App Store Server Notifications V2 Webhook。
// 执行退款、续期、撤销、订阅过期等异步事件的业务补偿逻辑。返回处理摘要。
func (s *Service) HandleAppleServerNotification(ctx context.Context, signedPayload string) (*AppleNotificationSummary, error) {
	decoded, err := DecodeAppleNotificationPayload(signedPayload)
	if err != nil {
		return nil, err
	}
	summary := &AppleNotificationSummary{
		NotificationType:    decoded.NotificationType,
		Subtype:             decoded.Subtype,
		NotificationUUID:    decoded.NotificationUUID,
		BundleID:            decoded.BundleID,
		Environment:         decoded.Environment,
		OriginalTransaction: decoded.OriginalTransaction,
		TransactionID:       decoded.TransactionID,
		ProductID:           decoded.ProductID,
	}

	var order *model.VideoOrder
	var lookupErr error
	if decoded.TransactionID != "" {
		order, lookupErr = s.orders.GetByPaymentTransaction(ctx, domain.PaymentMethodAppleIAP, decoded.TransactionID)
	}
	if order == nil && decoded.OriginalTransaction != "" {
		q := repository.QFrom(ctx).VideoOrder
		order, lookupErr = q.WithContext(ctx).
			Where(q.OriginalTransactionID.Eq(decoded.OriginalTransaction)).
			Order(q.ID.Desc()).First()
	}
	if lookupErr != nil && !errors.Is(lookupErr, gorm.ErrRecordNotFound) {
		return nil, lookupErr
	}
	if order != nil {
		summary.AffectedUserID = order.UserID
		summary.AffectedOrderNo = order.OrderNo
	}

	switch decoded.NotificationType {
	case AppleNotificationRefund, AppleNotificationRefundReversed, AppleNotificationRevoke:
		summary.Action = "refund_or_revoke"
		if order != nil && (order.Status == domain.OrderStatusPaid || order.Status == domain.OrderStatusRefunded) {
			revokedAt := time.Now()
			if decoded.RevocationDate > 0 {
				revokedAt = time.UnixMilli(decoded.RevocationDate)
			}
			if innerErr := s.revokePaidOrder(ctx, order, revokedAt); innerErr != nil {
				summary.Message = innerErr.Error()
			} else {
				summary.Processed = true
				summary.Message = "revoked benefits"
			}
		} else {
			summary.Message = "no matching paid order to revoke"
		}

	case AppleNotificationSubscribed, AppleNotificationOneTimeCharge:
		summary.Action = "subscribe_or_purchase"
		if order == nil && decoded.TransactionID != "" && decoded.OriginalTransaction != "" {
			summary.Message = "silent notification, awaiting client confirmApple endpoint"
			summary.Processed = true
		} else if order != nil && order.Status == domain.OrderStatusPaid {
			summary.Message = "order already fulfilled"
			summary.Processed = true
		} else {
			summary.Message = "pending client confirmation"
		}

	case AppleNotificationDidRenew:
		summary.Action = "renew"
		if order != nil && order.ProductType == domain.OrderProductVIPSubscription {
			if innerErr := s.extendVIPFromNotification(ctx, order, decoded); innerErr == nil {
				summary.Processed = true
				summary.Message = "extended subscription from renewal notification"
			} else {
				summary.Message = innerErr.Error()
			}
		} else if order == nil {
			summary.Message = "renewal without local order, awaiting client confirmApple"
			summary.Processed = true
		} else {
			summary.Message = "renewal not actionable for non-subscription order"
			summary.Processed = true
		}

	case AppleNotificationDidChangeRenewalStatus:
		summary.Action = "update_renewal_status"
		if order != nil && order.ProductType == domain.OrderProductVIPSubscription {
			if innerErr := s.reflectRenewalStatusChange(ctx, order, decoded.Subtype); innerErr == nil {
				summary.Processed = true
				summary.Message = "reflected auto-renewal status change"
			} else {
				summary.Message = innerErr.Error()
			}
		} else {
			summary.Processed = true
			summary.Message = "no subscription order to update"
		}

	case AppleNotificationExpired, AppleNotificationGracePeriodExpired, AppleNotificationDidFailToRenew:
		summary.Action = "expire_or_fail"
		if order != nil && order.ProductType == domain.OrderProductVIPSubscription {
			if innerErr := s.expireVIPSubscription(ctx, order); innerErr == nil {
				summary.Processed = true
				summary.Message = "subscription marked expired"
			} else {
				summary.Message = innerErr.Error()
			}
		} else {
			summary.Processed = true
			summary.Message = "no local subscription to expire"
		}

	case AppleNotificationTest:
		summary.Action = "test"
		summary.Processed = true
		summary.Message = "App Store Connect test notification acknowledged"

	default:
		summary.Action = "acknowledged"
		summary.Processed = true
		summary.Message = fmt.Sprintf("notification type %s acknowledged without side effects", decoded.NotificationType)
	}

	return summary, nil
}

func (s *Service) revokePaidOrder(ctx context.Context, order *model.VideoOrder, revokedAt time.Time) error {
	if order.Status == domain.OrderStatusRefunded {
		return nil
	}
	return repository.Transaction(ctx, func(ctx context.Context) error {
		lockedOrder, err := s.orders.GetByOrderNo(ctx, order.OrderNo, true)
		if err != nil {
			return err
		}
		if lockedOrder.Status == domain.OrderStatusRefunded {
			return nil
		}
		if lockedOrder.Status != domain.OrderStatusPaid {
			return fmt.Errorf("order %s is not paid, cannot revoke", lockedOrder.OrderNo)
		}
		user, err := s.users.GetByIDForUpdate(ctx, lockedOrder.UserID)
		if err != nil {
			return err
		}
		now := time.Now()
		updates := map[string]interface{}{}
		if lockedOrder.BonusPoints > 0 && user.PointsBalance >= lockedOrder.BonusPoints {
			before := user.PointsBalance
			after := user.PointsBalance - lockedOrder.BonusPoints
			key := "refund:" + lockedOrder.OrderNo
			ledger := &model.VideoUserPointsLedger{
				UserID: user.ID, OrderID: lockedOrder.ID,
				Direction:     int8(domain.PointsDirectionExpense),
				PointsChange:  -int64(lockedOrder.BonusPoints),
				BalanceBefore: before, BalanceAfter: after,
				SourceType: domain.PointsSourceRefund, BusinessID: lockedOrder.OrderNo,
				IdempotencyKey: key, Description: "revoke purchase bonus points",
				OccurredAt: revokedAt, CreatedAt: now,
			}
			if lockedOrder.ProductType == domain.OrderProductPointsPackage {
				ledger.PointsPackageID = &lockedOrder.ProductID
			}
			if ledgerErr := s.ledgers.Create(ctx, ledger); ledgerErr != nil {
				if !errors.Is(ledgerErr, gorm.ErrDuplicatedKey) {
					return ledgerErr
				}
			}
			user.PointsBalance = after
			updates["points_balance"] = after
		}
		if lockedOrder.PaidAmount > 0 {
			updates["refund_amount_money"] = user.RefundAmountMoney + lockedOrder.PaidAmount
		}
		if lockedOrder.ProductType == domain.OrderProductVIPSubscription &&
			user.VipExpiresAt != nil && user.VipExpiresAt.After(now) {
			updates["vip_expires_at"] = revokedAt
			if revokedAt.Before(now) {
				updates["subscription_status"] = domain.AppUserSubscriptionCancelled
				if user.PointsBalance == 0 {
					updates["user_type"] = domain.AppUserTypeFree
				}
			}
		}
		if len(updates) > 0 {
			if updateErr := s.users.Update(ctx, user.ID, updates); updateErr != nil {
				return updateErr
			}
		}
		orderUpdates := map[string]interface{}{
			"status":          domain.OrderStatusRefunded,
			"refunded_amount": lockedOrder.PaidAmount,
			"cancelled_at":    revokedAt,
			"cancel_reason":   "Apple refund or revoke notification",
		}
		q := repository.QFrom(ctx).VideoOrder
		_, updateErr := q.WithContext(ctx).Where(q.ID.Eq(lockedOrder.ID)).Updates(orderUpdates)
		return updateErr
	})
}

func (s *Service) extendVIPFromNotification(ctx context.Context, order *model.VideoOrder, decoded *DecodedAppleNotification) error {
	if decoded.ExpiresDate <= 0 {
		return nil
	}
	newExpires := time.UnixMilli(decoded.ExpiresDate)
	return repository.Transaction(ctx, func(ctx context.Context) error {
		user, err := s.users.GetByIDForUpdate(ctx, order.UserID)
		if err != nil {
			return err
		}
		updates := map[string]interface{}{
			"subscription_status": domain.AppUserSubscriptionSubscribed,
			"user_type":           domain.AppUserTypePaid,
		}
		if user.VipExpiresAt == nil || newExpires.After(*user.VipExpiresAt) {
			updates["vip_expires_at"] = newExpires
		}
		updates["subscription_payment_count"] = user.SubscriptionPaymentCount + 1
		return s.users.Update(ctx, user.ID, updates)
	})
}

func (s *Service) reflectRenewalStatusChange(ctx context.Context, order *model.VideoOrder, subtype string) error {
	return repository.Transaction(ctx, func(ctx context.Context) error {
		user, err := s.users.GetByIDForUpdate(ctx, order.UserID)
		if err != nil {
			return err
		}
		updates := map[string]interface{}{}
		switch subtype {
		case AppleSubtypeAutoRenewDisabled:
			updates["subscription_status"] = domain.AppUserSubscriptionCancelled
		case AppleSubtypeAutoRenewEnabled, AppleSubtypeBillingRecovery:
			updates["subscription_status"] = domain.AppUserSubscriptionSubscribed
			updates["user_type"] = domain.AppUserTypePaid
		}
		if len(updates) == 0 {
			return nil
		}
		return s.users.Update(ctx, user.ID, updates)
	})
}

func (s *Service) expireVIPSubscription(ctx context.Context, order *model.VideoOrder) error {
	return repository.Transaction(ctx, func(ctx context.Context) error {
		user, err := s.users.GetByIDForUpdate(ctx, order.UserID)
		if err != nil {
			return err
		}
		updates := map[string]interface{}{
			"subscription_status": domain.AppUserSubscriptionCancelled,
		}
		now := time.Now()
		if user.VipExpiresAt == nil || !user.VipExpiresAt.After(now) {
			updates["vip_expires_at"] = now
		}
		if user.PointsBalance == 0 && (user.VipExpiresAt == nil || !user.VipExpiresAt.After(now)) {
			updates["user_type"] = domain.AppUserTypeFree
		}
		return s.users.Update(ctx, user.ID, updates)
	})
}

// ConfirmApplePurchase verifies the StoreKit result, resolves the SKU under
// the authenticated app package, creates an order and fulfills it exactly once.
func (s *Service) ConfirmApplePurchase(ctx context.Context, userID uint64, expectedBundle string, req ApplePurchaseRequest) (*ApplePurchaseResponse, error) {
	if userID == 0 {
		return nil, errors.New("authenticated user is required")
	}
	expectedBundle = strings.TrimSpace(expectedBundle)
	verified, err := verifyApplePurchase(req, expectedBundle, config.Cfg.Server.Mode != "release")
	if err != nil {
		return nil, err
	}

	if existing, lookupErr := s.orders.GetByPaymentTransaction(ctx, domain.PaymentMethodAppleIAP, verified.TransactionID); lookupErr == nil {
		if existing.UserID != userID || existing.ProductCode != verified.ProductID {
			return nil, ErrPaymentTransactionUsed
		}
		return applePurchaseResponse(existing, verified), nil
	} else if !errors.Is(lookupErr, gorm.ErrRecordNotFound) {
		return nil, lookupErr
	}
	if verified.RevokedAt != nil {
		return nil, ErrApplePurchaseRevoked
	}
	if isSubscriptionType(verified.Type) {
		if !req.IsActive || verified.ExpiresAt == nil || !verified.ExpiresAt.After(time.Now()) {
			return nil, ErrApplePurchaseInactive
		}
	}

	productType, productID, err := s.resolveAppleProduct(ctx, verified.ProductID, expectedBundle)
	if err != nil {
		return nil, err
	}
	if productType == domain.OrderProductVIPSubscription && !isSubscriptionType(verified.Type) {
		return nil, ErrPaymentMismatch
	}
	if productType == domain.OrderProductPointsPackage && isSubscriptionType(verified.Type) {
		return nil, ErrPaymentMismatch
	}

	order, err := s.CreateOrder(ctx, CreateOrderRequest{
		UserID: userID, ProductType: productType, ProductID: productID,
		PaymentMethod:   domain.PaymentMethodAppleIAP,
		ClientRequestID: appleClientRequestID(verified.TransactionID),
		Renewal: strings.EqualFold(verified.TransactionReason, "RENEWAL") ||
			verified.TransactionID != verified.OriginalTransactionID,
	})
	if err != nil {
		return nil, err
	}
	paid, err := s.ConfirmApplePayment(ctx, order.OrderNo, ApplePaymentResult{
		TransactionID: verified.TransactionID, OriginalTransactionID: verified.OriginalTransactionID,
		ProductCode: verified.ProductID, Currency: verified.Currency, PaidAmount: verified.PaidAmount,
		SignedTransaction: req.SignedTransactionInfo, PurchaseDate: verified.PurchaseAt,
		SubscriptionExpiresAt: verified.ExpiresAt,
	})
	if err != nil {
		return nil, err
	}
	return applePurchaseResponse(paid, verified), nil
}

func (s *Service) resolveAppleProduct(ctx context.Context, productCode, packageCode string) (string, uint64, error) {
	vip, vipErr := s.vipProducts.GetAppleProduct(ctx, productCode, packageCode)
	points, pointsErr := s.pointProducts.GetAppleProduct(ctx, productCode, packageCode)
	vipFound := vipErr == nil
	pointsFound := pointsErr == nil
	if vipErr != nil && !errors.Is(vipErr, gorm.ErrRecordNotFound) {
		return "", 0, vipErr
	}
	if pointsErr != nil && !errors.Is(pointsErr, gorm.ErrRecordNotFound) {
		return "", 0, pointsErr
	}
	if vipFound && pointsFound {
		return "", 0, ErrAppleProductAmbiguous
	}
	if vipFound {
		return domain.OrderProductVIPSubscription, vip.ID, nil
	}
	if pointsFound {
		return domain.OrderProductPointsPackage, points.ID, nil
	}
	return "", 0, ErrAppleProductNotFound
}

func verifyApplePurchase(req ApplePurchaseRequest, expectedBundle string, allowUnsignedSandbox bool) (*verifiedAppleTransaction, error) {
	req.BundleID = strings.TrimSpace(req.BundleID)
	req.TransactionID = strings.TrimSpace(req.TransactionID)
	req.OriginalTransactionID = strings.TrimSpace(req.OriginalTransactionID)
	req.ProductID = strings.TrimSpace(req.ProductID)
	evidence := strings.TrimSpace(req.SignedTransactionInfo)
	if expectedBundle == "" || req.BundleID == "" || req.TransactionID == "" || req.ProductID == "" || evidence == "" || req.PurchaseDate.IsZero() {
		return nil, ErrAppleEvidenceInvalid
	}

	var signed appleSignedTransaction
	mode := "jws"
	if strings.Count(evidence, ".") == 2 && !strings.HasPrefix(evidence, "{") {
		if err := verifyAppleJWS(evidence, &signed); err != nil {
			return nil, err
		}
	} else {
		mode = "sandbox_json"
		if err := json.Unmarshal([]byte(evidence), &signed); err != nil {
			return nil, fmt.Errorf("%w: signedTransactionInfo is neither JWS nor JSON", ErrAppleEvidenceInvalid)
		}
		if !allowUnsignedSandbox || !strings.EqualFold(signed.Environment, "Sandbox") {
			return nil, ErrAppleUnsignedProduction
		}
	}

	if req.BundleID != expectedBundle || signed.BundleID != expectedBundle {
		return nil, ErrAppleBundleMismatch
	}
	if signed.TransactionID != req.TransactionID || signed.OriginalTransactionID != req.OriginalTransactionID || signed.ProductID != req.ProductID {
		return nil, ErrAppleEvidenceInvalid
	}
	if signed.Quantity <= 0 || signed.Price < 0 || strings.TrimSpace(signed.Currency) == "" || signed.PurchaseDate <= 0 {
		return nil, ErrAppleEvidenceInvalid
	}
	purchaseAt := time.UnixMilli(signed.PurchaseDate)
	if durationAbs(purchaseAt.Sub(req.PurchaseDate)) > 2*time.Second || purchaseAt.After(time.Now().Add(5*time.Minute)) {
		return nil, ErrAppleEvidenceInvalid
	}
	result := &verifiedAppleTransaction{
		appleSignedTransaction: signed,
		PurchaseAt:             purchaseAt, PaidAmount: math.Round((float64(signed.Price)/1000)*100) / 100,
		EvidenceMode: mode,
	}
	result.Currency = strings.ToUpper(strings.TrimSpace(result.Currency))
	if signed.ExpiresDate > 0 {
		expiresAt := time.UnixMilli(signed.ExpiresDate)
		result.ExpiresAt = &expiresAt
		if req.ExpirationDate != nil && durationAbs(expiresAt.Sub(*req.ExpirationDate)) > 2*time.Second {
			return nil, ErrAppleEvidenceInvalid
		}
	} else if req.ExpirationDate != nil {
		return nil, ErrAppleEvidenceInvalid
	}
	if signed.RevocationDate > 0 {
		revokedAt := time.UnixMilli(signed.RevocationDate)
		result.RevokedAt = &revokedAt
		if req.RevocationDate == nil || durationAbs(revokedAt.Sub(*req.RevocationDate)) > 2*time.Second {
			return nil, ErrAppleEvidenceInvalid
		}
	} else if req.RevocationDate != nil {
		return nil, ErrAppleEvidenceInvalid
	}
	return result, nil
}

func verifyAppleJWS(compact string, target *appleSignedTransaction) error {
	parts := strings.Split(compact, ".")
	if len(parts) != 3 {
		return ErrAppleSignatureInvalid
	}
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return ErrAppleSignatureInvalid
	}
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ErrAppleSignatureInvalid
	}
	var header struct {
		Alg string   `json:"alg"`
		X5C []string `json:"x5c"`
	}
	if json.Unmarshal(headerJSON, &header) != nil || header.Alg != "ES256" || len(header.X5C) == 0 {
		return ErrAppleSignatureInvalid
	}
	if err := json.Unmarshal(payloadJSON, target); err != nil {
		return ErrAppleSignatureInvalid
	}
	certificates := make([]*x509.Certificate, 0, len(header.X5C))
	for _, encoded := range header.X5C {
		der, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return ErrAppleSignatureInvalid
		}
		certificate, err := x509.ParseCertificate(der)
		if err != nil {
			return ErrAppleSignatureInvalid
		}
		certificates = append(certificates, certificate)
	}
	intermediates := x509.NewCertPool()
	for _, certificate := range certificates[1:] {
		intermediates.AddCert(certificate)
	}
	verifyTime := time.Now()
	if target.SignedDate > 0 {
		verifyTime = time.UnixMilli(target.SignedDate)
	}
	if _, err := certificates[0].Verify(x509.VerifyOptions{
		Intermediates: intermediates, CurrentTime: verifyTime,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	}); err != nil {
		return fmt.Errorf("%w: %v", ErrAppleSignatureInvalid, err)
	}
	publicKey, ok := certificates[0].PublicKey.(*ecdsa.PublicKey)
	if !ok {
		return ErrAppleSignatureInvalid
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || len(signature) != 64 {
		return ErrAppleSignatureInvalid
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	r := new(big.Int).SetBytes(signature[:32])
	s := new(big.Int).SetBytes(signature[32:])
	if !ecdsa.Verify(publicKey, digest[:], r, s) {
		return ErrAppleSignatureInvalid
	}
	return nil
}

func appleClientRequestID(transactionID string) string {
	digest := sha256.Sum256([]byte(strings.TrimSpace(transactionID)))
	return "apple:" + base64.RawURLEncoding.EncodeToString(digest[:])
}

func applePurchaseResponse(order *model.VideoOrder, transaction *verifiedAppleTransaction) *ApplePurchaseResponse {
	active := transaction.RevokedAt == nil
	if isSubscriptionType(transaction.Type) {
		active = active && transaction.ExpiresAt != nil && transaction.ExpiresAt.After(time.Now())
	}
	return &ApplePurchaseResponse{
		OrderNo: order.OrderNo, Status: order.Status, ProductType: order.ProductType,
		ProductID: order.ProductID, ProductCode: order.ProductCode,
		TransactionID: transaction.TransactionID, OriginalTransactionID: transaction.OriginalTransactionID,
		Currency: transaction.Currency, PaidAmount: transaction.PaidAmount,
		PurchaseDate: transaction.PurchaseAt, ExpirationDate: transaction.ExpiresAt,
		IsActive: active, Environment: transaction.Environment, EvidenceMode: transaction.EvidenceMode,
	}
}

func isSubscriptionType(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return strings.Contains(value, "subscription") || strings.Contains(value, "auto-renewable")
}

func durationAbs(value time.Duration) time.Duration {
	if value < 0 {
		return -value
	}
	return value
}
