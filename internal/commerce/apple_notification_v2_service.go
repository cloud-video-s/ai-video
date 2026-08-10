package commerce

import (
	"ai-video/internal/config"
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"ai-video/internal/domain"
	"ai-video/internal/gen/model"
	"ai-video/internal/repository"

	"gorm.io/gorm"
)

// HandleAppleServerNotification is kept for callers using the previous name.
func (s *Service) HandleAppleServerNotification(ctx context.Context, signedPayload string) (*AppleNotificationSummary, error) {
	return s.HandleAppleServerNotificationV2(ctx, signedPayload)
}

// HandleAppleServerNotificationV2 verifies and processes an App Store Server
// Notifications V2 signedPayload. Business transaction failures are returned
// to the HTTP layer so Apple can retry the notification.
func (s *Service) HandleAppleServerNotificationV2(ctx context.Context, signedPayload string) (*AppleNotificationV2Summary, error) {
	decoded, err := decodeAppleNotificationV2Payload(signedPayload, s.appleRootCAs)
	if err != nil {
		return nil, err
	}
	config.Log.Debug("Apple server notification",
		"summary", decoded,
		"notification_type", decoded.NotificationType,
		"subtype", decoded.Subtype,
		"notification_uuid", decoded.NotificationUUID,
		"bundle_id", decoded.BundleID,
		"environment", decoded.Environment,
		"transaction_id", decoded.TransactionID,
	)
	if err = s.validateAppleNotificationV2App(ctx, decoded); err != nil {
		return nil, err
	}
	summary := newAppleNotificationV2Summary(decoded)

	order, err := s.findAppleNotificationV2Order(ctx, decoded)
	if err != nil {
		return nil, err
	}
	if order != nil {
		summary.AffectedUserID = order.UserID
		summary.AffectedOrderNo = order.OrderNo
	}

	switch decoded.NotificationType {
	case AppleNotificationRefund, AppleNotificationRevoke:
		summary.Action = "refund_or_revoke"
		if order != nil && (order.Status == domain.OrderStatusPaid || order.Status == domain.OrderStatusEnd || order.Status == domain.OrderStatusRefunded) {
			revokedAt := time.Now()
			if decoded.RevocationDate > 0 {
				revokedAt = time.UnixMilli(decoded.RevocationDate)
			}
			if err := s.revokePaidAppleOrder(ctx, order, revokedAt); err != nil {
				return nil, err
			}
			summary.Processed = true
			summary.Message = "revoked benefits"
		} else {
			summary.Message = "no matching paid order to revoke"
		}

	case AppleNotificationRefundReversed:
		summary.Action = "refund_reversed"
		summary.Processed = true
		summary.Message = "refund reversal acknowledged; entitlement restoration requires reconciliation"

	case AppleNotificationSubscribed:
		summary.Action = "subscribe"
		completed, processErr := s.processAppleSubscriptionTransaction(
			ctx, order, decoded, decoded.Subtype == AppleSubtypeResubscribe,
		)
		if processErr != nil {
			return nil, processErr
		}
		if completed == nil {
			// An initial purchase cannot be associated with a user until the
			// authenticated client confirmation creates its deterministic order.
			summary.Processed = true
			summary.Message = "initial subscription awaiting authenticated client confirmation"
			break
		}
		setAppleNotificationSummaryOrder(summary, completed)
		summary.Processed = true
		summary.Message = "subscription order completed"

	case AppleNotificationOneTimeCharge:
		summary.Action = "one_time_charge"
		summary.Processed = true
		summary.Message = "one-time charge acknowledged"

	case AppleNotificationDidRenew:
		summary.Action = "renew"
		completed, processErr := s.processAppleSubscriptionTransaction(ctx, order, decoded, true)
		if processErr != nil {
			return nil, processErr
		}
		if completed == nil {
			// Returning an error asks Apple to retry. A local original order may
			// appear shortly afterwards through client confirmation or recovery.
			return nil, fmt.Errorf("Apple renewal %s has no local original order", decoded.TransactionID)
		}
		setAppleNotificationSummaryOrder(summary, completed)
		summary.Processed = true
		summary.Message = "renewal order completed"

	case AppleNotificationRenewalExtended:
		summary.Action = "extend_renewal"
		if order != nil && order.ProductType == domain.OrderProductVIPSubscription {
			if err := s.extendVIPFromAppleNotificationV2(ctx, order, decoded); err != nil {
				return nil, err
			}
			summary.Message = "subscription expiration extended without a new charge"
		} else {
			summary.Message = "no local subscription to extend"
		}
		summary.Processed = true

	case AppleNotificationDidChangeRenewalStatus:
		summary.Action = "update_renewal_status"
		if order != nil && order.ProductType == domain.OrderProductVIPSubscription {
			if err := s.reflectAppleRenewalStatusChange(ctx, order, decoded.Subtype, decoded.ExpiresDate); err != nil {
				return nil, err
			}
			summary.Processed = true
			summary.Message = "reflected auto-renewal status change"
		} else {
			summary.Processed = true
			summary.Message = "no subscription order to update"
		}

	case AppleNotificationDidFailToRenew:
		summary.Action = "renew_failed"
		if order != nil && order.ProductType == domain.OrderProductVIPSubscription &&
			decoded.Subtype != AppleSubtypeGracePeriod && decoded.ExpiresDate > 0 &&
			!time.UnixMilli(decoded.ExpiresDate).After(time.Now()) {
			if err := s.expireVIPFromAppleNotificationV2(ctx, order, decoded.ExpiresDate); err != nil {
				return nil, err
			}
			summary.Message = "renewal failed and subscription is expired"
		} else {
			summary.Message = "renewal failure acknowledged; current entitlement retained"
		}
		summary.Processed = true

	case AppleNotificationExpired, AppleNotificationGracePeriodExpired:
		summary.Action = "expire"
		if order != nil && order.ProductType == domain.OrderProductVIPSubscription {
			if err := s.expireVIPFromAppleNotificationV2(ctx, order, decoded.ExpiresDate); err != nil {
				return nil, err
			}
			summary.Processed = true
			summary.Message = "subscription marked expired"
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

// newAppleNotificationV2Summary copies only verified metadata into the
// response/logging view and intentionally omits all compact JWS values.
func newAppleNotificationV2Summary(decoded *DecodedAppleNotificationV2) *AppleNotificationV2Summary {
	return &AppleNotificationV2Summary{
		NotificationType:    decoded.NotificationType,
		Subtype:             decoded.Subtype,
		NotificationUUID:    decoded.NotificationUUID,
		BundleID:            decoded.BundleID,
		Environment:         decoded.Environment,
		OriginalTransaction: decoded.OriginalTransaction,
		TransactionID:       decoded.TransactionID,
		ProductID:           decoded.ProductID,
		Version:             decoded.Version,
		SignedDate:          decoded.SignedDate,
		AppAppleID:          decoded.AppAppleID,
		SubscriptionStatus:  decoded.SubscriptionStatus,
	}
}

// processAppleSubscriptionTransaction completes the order for the signed
// transaction. Renewals and resubscriptions create a new local order from the
// original order's user/product association; initial purchases wait for the
// authenticated client endpoint when no local order exists yet.
func (s *Service) processAppleSubscriptionTransaction(
	ctx context.Context,
	order *model.VideoOrder,
	decoded *DecodedAppleNotificationV2,
	createFromOriginal bool,
) (*model.VideoOrder, error) {
	if decoded == nil || strings.TrimSpace(decoded.TransactionID) == "" ||
		strings.TrimSpace(decoded.OriginalTransaction) == "" ||
		strings.TrimSpace(decoded.ProductID) == "" || decoded.PurchaseDate <= 0 || decoded.Price < 0 {
		return nil, fmt.Errorf("%w: incomplete subscription transaction", ErrAppleEvidenceInvalid)
	}

	requestID := appleClientRequestID(decoded.TransactionID)
	renewal := createFromOriginal || strings.EqualFold(decoded.TransactionReason, "RENEWAL") ||
		decoded.TransactionID != decoded.OriginalTransaction
	orderType := orderTypeForRenewal(decoded.NotificationType == AppleNotificationDidRenew)
	isTransactionOrder := order != nil &&
		(order.ThirdOrderNo == decoded.TransactionID || order.ClientRequestID == requestID)
	if !isTransactionOrder {
		if order == nil || !createFromOriginal {
			return nil, nil
		}
		if order.ProductType != domain.OrderProductVIPSubscription {
			return nil, ErrPaymentMismatch
		}
		productID := order.ProductID
		if order.ProductCode != decoded.ProductID {
			resolvedID, err := s.resolveAppleProduct(ctx, 1, decoded.ProductID, decoded.BundleID)
			if err != nil {
				return nil, err
			}
			productID = resolvedID
		}
		created, err := s.CreateOrder(ctx, CreateOrderRequest{
			UserID: order.UserID, ProductType: domain.OrderProductVIPSubscription,
			ProductID: productID, PayType: domain.PaymentMethodAppleIAP,
			ClientRequestID: requestID, Renewal: renewal,
			PaidAmount: appleNotificationPaidAmount(decoded.Price),
		})
		if err != nil {
			return nil, err
		}
		order = created
	}

	if order.PayType != domain.PaymentMethodAppleIAP ||
		order.ProductType != domain.OrderProductVIPSubscription ||
		order.ProductCode != decoded.ProductID {
		return nil, ErrPaymentMismatch
	}
	var expiresAt *time.Time
	if decoded.ExpiresDate > 0 {
		value := time.UnixMilli(decoded.ExpiresDate)
		expiresAt = &value
	}

	switch order.Status {
	case domain.OrderStatusPending:
		currency := decoded.Currency
		if currency == "" {
			currency = order.Currency
		}
		paid, err := s.ConfirmApplePayment(ctx, order.OrderNo, ApplePaymentResult{
			TransactionID: decoded.TransactionID, OriginalTransactionID: decoded.OriginalTransaction,
			ProductCode: decoded.ProductID, OrderType: orderType, Currency: currency,
			PaidAmount:        appleNotificationPaidAmount(decoded.Price),
			SignedTransaction: decoded.SignedTransaction,
			PurchaseDate:      time.UnixMilli(decoded.PurchaseDate), SubscriptionExpiresAt: expiresAt,
		})
		if err != nil {
			return nil, err
		}
		order = paid
	case domain.OrderStatusPaid:
		// Continue below and recover a crash between payment and fulfillment.
	case domain.OrderStatusEnd:
		return order, nil
	default:
		return nil, fmt.Errorf("Apple transaction %s belongs to non-actionable order status %d", decoded.TransactionID, order.Status)
	}

	if err := s.fulfillAppleOrder(ctx, order, expiresAt); err != nil {
		return nil, err
	}
	return s.orders.GetByOrderNo(ctx, order.OrderNo, false)
}

func appleNotificationPaidAmount(price int64) float64 {
	return math.Round((float64(price)/1000)*100) / 100
}

func setAppleNotificationSummaryOrder(summary *AppleNotificationV2Summary, order *model.VideoOrder) {
	if summary == nil || order == nil {
		return
	}
	summary.AffectedUserID = order.UserID
	summary.AffectedOrderNo = order.OrderNo
}

// validateAppleNotificationV2App ensures the signed Bundle ID belongs to a
// configured iOS package and enforces appAppleId for Production notifications.
func (s *Service) validateAppleNotificationV2App(ctx context.Context, decoded *DecodedAppleNotificationV2) error {
	if decoded == nil || decoded.BundleID == "" {
		return ErrAppleBundleMismatch
	}
	if decoded.Environment == "Production" && decoded.AppAppleID <= 0 {
		return fmt.Errorf("%w: production notification appAppleId is required", ErrAppleEvidenceInvalid)
	}
	item, err := s.packages.GetByCode(ctx, decoded.BundleID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAppleBundleMismatch
		}
		return err
	}
	// Disabled iOS packages can still receive refunds and expirations for
	// historical purchases; existence and platform are the trust boundary.
	if int(item.SystemType) != domain.SystemTypeIos {
		return ErrAppleBundleMismatch
	}
	return nil
}

// findAppleNotificationV2Order resolves the local order by transaction ID,
// deterministic request ID, then original transaction ID in that order.
func (s *Service) findAppleNotificationV2Order(ctx context.Context, decoded *DecodedAppleNotificationV2) (*model.VideoOrder, error) {
	if decoded.TransactionID != "" {
		order, err := s.orders.GetByPaymentTransaction(ctx, domain.PaymentMethodAppleIAP, decoded.TransactionID)
		if err == nil {
			return order, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	if decoded.OriginalTransaction != "" {
		order, err := s.orders.GetByAppleOriginalTransactionID(ctx, decoded.OriginalTransaction)
		if err == nil {
			return order, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	return nil, nil
}

// cancelPendingAppleOrder locks and conditionally cancels an Apple order. The
// boolean is false when a concurrent request already moved it out of pending.
func (s *Service) cancelPendingAppleOrder(ctx context.Context, orderNo, reason string) (bool, error) {
	orderNo = strings.TrimSpace(orderNo)
	if orderNo == "" {
		return false, nil
	}
	cancelled := false
	err := repository.Transaction(ctx, func(ctx context.Context) error {
		order, err := s.orders.GetByOrderNo(ctx, orderNo, true)
		if err != nil {
			return err
		}
		if order.PayType != domain.PaymentMethodAppleIAP || order.Status != domain.OrderStatusPending {
			return nil
		}
		if err := s.orders.CancelPending(ctx, order.ID, strings.TrimSpace(reason), time.Now()); err != nil {
			return err
		}
		cancelled = true
		return nil
	})
	return cancelled, err
}

// revokePaidAppleOrder atomically reverses available purchase benefits and
// marks the paid order refunded. Repeated notifications are idempotent.
func (s *Service) revokePaidAppleOrder(ctx context.Context, order *model.VideoOrder, revokedAt time.Time) error {
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
		if lockedOrder.Status != domain.OrderStatusPaid && lockedOrder.Status != domain.OrderStatusEnd {
			return fmt.Errorf("order %s is not paid, cannot revoke", lockedOrder.OrderNo)
		}
		user, err := s.users.GetByIDForUpdate(ctx, lockedOrder.UserID)
		if err != nil {
			return err
		}
		now := time.Now()
		updates := map[string]any{}
		updates["refund_amount_money"] = user.RefundAmountMoney + lockedOrder.PaidAmount
		deductedPoints := int64(0)
		sourceType := uint32(domain.PointsSourceOther)
		if lockedOrder.ProductType == domain.OrderProductVIPSubscription {
			deductedPoints = user.VipPoints
			sourceType = uint32(domain.PointsSourceSubscriptionGift)
			updates["vip_points"] = 0
			updates["vip_expires_at"] = nil
			updates["vip_started_at"] = nil
			updates["user_type"] = domain.AppUserTypeFree
			updates["subscription_status"] = domain.AppUserSubscriptionNotSubscribed
		} else if user.PointsBalance >= lockedOrder.BonusPoints {
			deductedPoints = lockedOrder.BonusPoints
			sourceType = uint32(domain.PointsSourcePurchase)
			updates["points_balance"] = user.PointsBalance - lockedOrder.BonusPoints
		}
		if deductedPoints > 0 {
			before := user.PointsBalance + user.VipPoints
			after := before - deductedPoints
			ledger := &model.VideoUserPointsLedger{
				UserID: user.ID, OrderCode: lockedOrder.OrderNo,
				Direction:     int8(domain.PointsDirectionExpense),
				PointsChange:  -deductedPoints,
				BalanceBefore: uint64(before), BalanceAfter: uint64(after),
				SourceType:  sourceType,
				Description: "revoke purchase bonus points",
				OccurredAt:  revokedAt, CreatedAt: now,
			}
			if lockedOrder.ProductType == domain.OrderProductVIPSubscription {
				ledger.VipID = lockedOrder.ProductID
			} else if lockedOrder.ProductType == domain.OrderProductPointsPackage {
				ledger.PointsID = lockedOrder.ProductID
			}
			if err := s.ledgers.Create(ctx, ledger); err != nil && !errors.Is(err, gorm.ErrDuplicatedKey) {
				return err
			}
		}
		if len(updates) > 0 {
			if err := s.users.Update(ctx, user.ID, updates); err != nil {
				return err
			}
		}
		orderUpdates := map[string]any{
			"status":          domain.OrderStatusRefunded,
			"refunded_amount": lockedOrder.PaidAmount,
			"cancelled_at":    revokedAt,
			"cancel_reason":   "Apple refund or revoke notification",
		}
		q := repository.QFrom(ctx).VideoOrder
		_, err = q.WithContext(ctx).Where(q.ID.Eq(lockedOrder.ID)).Updates(orderUpdates)
		return err
	})
}

// extendVIPFromAppleNotificationV2 applies a non-charging extension only when
// Apple's signed expiration is strictly newer than the current entitlement.
func (s *Service) extendVIPFromAppleNotificationV2(ctx context.Context, order *model.VideoOrder, decoded *DecodedAppleNotificationV2) error {
	if decoded.ExpiresDate <= 0 {
		return nil
	}
	newExpires := time.UnixMilli(decoded.ExpiresDate)
	return repository.Transaction(ctx, func(ctx context.Context) error {
		user, err := s.users.GetByIDForUpdate(ctx, order.UserID)
		if err != nil {
			return err
		}
		// Apple may retry a notification or deliver an older renewal after a
		// newer one. Only a strictly newer expiration may mutate entitlement
		// state or increment the renewal counter.
		if user.VipExpiresAt != nil && !newExpires.After(*user.VipExpiresAt) {
			return nil
		}
		updates := map[string]any{}
		applyVIPEntitlement(user, order, time.Now(), &newExpires, updates)
		return s.users.Update(ctx, user.ID, updates)
	})
}

// reflectAppleRenewalStatusChange maps Apple's auto-renewal subtype to the
// local subscription state without changing the current expiration.
func (s *Service) reflectAppleRenewalStatusChange(ctx context.Context, order *model.VideoOrder, subtype string, notificationExpiresDate int64) error {
	return repository.Transaction(ctx, func(ctx context.Context) error {
		user, err := s.users.GetByIDForUpdate(ctx, order.UserID)
		if err != nil {
			return err
		}
		if notificationExpiresDate > 0 && user.VipExpiresAt != nil &&
			user.VipExpiresAt.After(time.UnixMilli(notificationExpiresDate)) {
			return nil
		}
		updates := map[string]interface{}{}
		switch subtype {
		case AppleSubtypeAutoRenewDisabled:
			updates["subscription_status"] = domain.AppUserSubscriptionCancelled
		case AppleSubtypeAutoRenewEnabled, AppleSubtypeBillingRecovery:
			if user.VipExpiresAt != nil && user.VipExpiresAt.After(time.Now()) {
				updates["subscription_status"] = domain.AppUserSubscriptionSubscribed
				updates["user_type"] = domain.AppUserTypePaid
			}
		}
		if len(updates) == 0 {
			return nil
		}
		return s.users.Update(ctx, user.ID, updates)
	})
}

// expireVIPFromAppleNotificationV2 cancels an expired subscription unless the
// user already holds an entitlement newer than the notification.
func (s *Service) expireVIPFromAppleNotificationV2(ctx context.Context, order *model.VideoOrder, notificationExpiresDate int64) error {
	return repository.Transaction(ctx, func(ctx context.Context) error {
		user, err := s.users.GetByIDForUpdate(ctx, order.UserID)
		if err != nil {
			return err
		}
		if notificationExpiresDate > 0 && user.VipExpiresAt != nil &&
			user.VipExpiresAt.After(time.UnixMilli(notificationExpiresDate)) {
			return nil
		}
		now := time.Now()
		expiresAt := now
		if notificationExpiresDate > 0 {
			expiresAt = time.UnixMilli(notificationExpiresDate)
		}
		if expiresAt.After(now) {
			return nil
		}
		hasNewerGift, err := s.ledgers.HasSubscriptionGiftSince(ctx, user.ID, expiresAt, order.OrderNo)
		if err != nil {
			return err
		}
		if hasNewerGift {
			return nil
		}
		beforeBalance := user.VipPoints + user.PointsBalance
		expiredPoints := user.VipPoints
		updates := map[string]any{
			"subscription_status": domain.AppUserSubscriptionExpired,
			"user_type":           domain.AppUserTypeFree,
			"vip_points":          uint64(0),
			"vip_expires_at":      nil,
		}
		if expiredPoints > 0 {
			ledger := &model.VideoUserPointsLedger{
				UserID: user.ID, OrderCode: order.OrderNo,
				Direction:     int8(domain.PointsDirectionExpense),
				PointsChange:  -expiredPoints,
				BalanceBefore: uint64(beforeBalance), BalanceAfter: uint64(user.PointsBalance),
				SourceType:  uint32(domain.PointsSourceExpireDeduct),
				VipID:       order.ProductID,
				Description: "subscription expired points deduction",
				OccurredAt:  expiresAt, CreatedAt: now,
			}
			if err := s.ledgers.Create(ctx, ledger); err != nil {
				return err
			}
		}
		return s.users.Update(ctx, user.ID, updates)
	})
}
