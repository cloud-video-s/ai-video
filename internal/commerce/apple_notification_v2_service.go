package commerce

import (
	"context"
	"errors"
	"fmt"
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
	if err := s.validateAppleNotificationV2App(ctx, decoded); err != nil {
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
		if order != nil && (order.Status == domain.OrderStatusPaid || order.Status == domain.OrderStatusRefunded) {
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
		err = NewService().NotificationApplePaymentSuccess(ctx, order, summary)
		if err != nil {
			return nil, err
		}
	case AppleNotificationDidRenew, AppleNotificationRenewalExtended:
		summary.Action = "renew"
		if order != nil && order.ProductType == domain.OrderProductVIPSubscription {
			if err := s.extendVIPFromAppleNotificationV2(ctx, order, decoded); err != nil {
				return nil, err
			}
			summary.Processed = true
			summary.Message = "extended subscription from renewal notification"
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
			if err := s.reflectAppleRenewalStatusChange(ctx, order, decoded.Subtype); err != nil {
				return nil, err
			}
			summary.Processed = true
			summary.Message = "reflected auto-renewal status change"
		} else {
			summary.Processed = true
			summary.Message = "no subscription order to update"
		}

	case AppleNotificationDidFailToRenew:
		summary.Action = "cancel_failed_payment"
		summary.Processed = true
		switch {
		case order == nil:
			summary.Message = "no matching pending Apple order to cancel"
		case order.Status == domain.OrderStatusCancelled:
			summary.Message = "pending Apple order was already cancelled"
		case order.Status != domain.OrderStatusPending:
			summary.Message = "matching Apple order is not pending; left unchanged"
		default:
			cancelled, err := s.cancelPendingAppleOrder(ctx, order.OrderNo, "Apple DID_FAIL_TO_RENEW notification")
			if err != nil {
				return nil, err
			}
			if cancelled {
				summary.Message = "cancelled pending Apple order"
			} else {
				summary.Message = "matching Apple order is no longer pending; left unchanged"
			}
		}

	case AppleNotificationExpired, AppleNotificationGracePeriodExpired:
		summary.Action = "expire_or_fail"
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

		// A payment failure can arrive before a provider transaction ID has
		// been persisted. The purchase flow's deterministic request ID lets the
		// callback find that pending order without trusting client-owned fields.
		order, err = s.orders.GetByClientRequestID(ctx, appleClientRequestID(decoded.TransactionID))
		if err == nil {
			if order.PaymentMethod == domain.PaymentMethodAppleIAP {
				return order, nil
			}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
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
		if order.PaymentMethod != domain.PaymentMethodAppleIAP || order.Status != domain.OrderStatusPending {
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
		if lockedOrder.Status != domain.OrderStatusPaid {
			return fmt.Errorf("order %s is not paid, cannot revoke", lockedOrder.OrderNo)
		}
		user, err := s.users.GetByIDForUpdate(ctx, lockedOrder.UserID)
		if err != nil {
			return err
		}
		now := time.Now()
		updates := map[string]any{}
		updates["refund_amount_money"] = user.RefundAmountMoney + lockedOrder.PaidAmount
		if order.ProductType == domain.OrderProductVIPSubscription {
			updates["vip_points"] = 0
			updates["vip_expires_at"] = 0
			updates["vip_started_at"] = 0
			updates["user_type"] = 1
			updates["subscription_status"] = 1
		} else {
			updates["points_balance"] = user.PointsBalance - order.BonusPoints
		}
		if lockedOrder.BonusPoints > 0 && user.PointsBalance >= lockedOrder.BonusPoints {
			before := user.PointsBalance + user.VipPoints
			after := before - lockedOrder.BonusPoints
			ledger := &model.VideoUserPointsLedger{
				UserID: user.ID, OrderCode: lockedOrder.OrderNo,
				Direction:     int8(domain.PointsDirectionExpense),
				PointsChange:  -int64(lockedOrder.BonusPoints),
				BalanceBefore: before, BalanceAfter: after,
				SourceType:  domain.PointsSourceModelRefund,
				Description: "revoke purchase bonus points",
				OccurredAt:  revokedAt, CreatedAt: now,
			}
			if lockedOrder.ProductType == domain.OrderProductPointsPackage {
				ledger.PointsID = lockedOrder.ProductID
			}
			if err := s.ledgers.Create(ctx, ledger); err != nil && !errors.Is(err, gorm.ErrDuplicatedKey) {
				return err
			}
			//user.PointsBalance = after
			//updates["points_balance"] = after
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

// extendVIPFromAppleNotificationV2 applies a renewal only when Apple's signed
// expiration is strictly newer than the user's current entitlement.
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
		updates := map[string]interface{}{
			"subscription_status":        domain.AppUserSubscriptionSubscribed,
			"user_type":                  domain.AppUserTypePaid,
			"vip_expires_at":             newExpires,
			"subscription_payment_count": user.SubscriptionPaymentCount + 1,
		}
		return s.users.Update(ctx, user.ID, updates)
	})
}

// reflectAppleRenewalStatusChange maps Apple's auto-renewal subtype to the
// local subscription state without changing the current expiration.
func (s *Service) reflectAppleRenewalStatusChange(ctx context.Context, order *model.VideoOrder, subtype string) error {
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
