package commerce

import (
	"context"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"ai-video/internal/config"
	"ai-video/internal/domain"
	"ai-video/internal/gen/model"
	"ai-video/internal/repository"

	"gorm.io/gorm"
)

var (
	ErrUnsupportedProduct       = errors.New("unsupported product type")
	ErrUnsupportedPaymentMethod = errors.New("unsupported payment method")
	ErrOrderAlreadyPaid         = errors.New("order has already been paid")
	ErrOrderNotCancellable      = errors.New("order cannot be cancelled")
	ErrPaymentTransactionUsed   = errors.New("payment transaction already belongs to another order")
	ErrPaymentMismatch          = errors.New("verified payment does not match order")
	ErrInsufficientPoints       = errors.New("insufficient points balance")
)

// Service coordinates commerce repositories and keeps order, payment,
// entitlement, and points-ledger mutations inside explicit transactions.
type Service struct {
	orders         *repository.OrderRepo
	users          *repository.AppUserRepo
	ledgers        *repository.CommercePointsLedgerRepo
	vipProducts    *repository.VIPSubscriptionRepo
	pointProducts  *repository.PointsPackageRepo
	packages       *repository.PackageRepo
	tasks          *repository.UserGenerationTaskRepo
	appleRootCAs   *x509.CertPool
	appleServerAPI appleTransactionInfoProvider
}

// NewService constructs a commerce service with production repositories and
// the built-in Apple notification trust store.
func NewService() *Service {
	return &Service{
		orders: repository.NewOrderRepo(), users: repository.NewAppUserRepo(),
		ledgers: repository.NewCommercePointsLedgerRepo(), vipProducts: repository.NewVIPSubscriptionRepo(),
		pointProducts: repository.NewPointsPackageRepo(), packages: repository.NewPackageRepo(),
		tasks:          repository.NewUserGenerationTaskRepo(),
		appleRootCAs:   defaultAppleRootCAs,
		appleServerAPI: newAppleServerAPIClient(config.Cfg.AppStore),
	}
}

// CreateOrderRequest identifies the user, product, and idempotency key used to
// snapshot a new order.
type CreateOrderRequest struct {
	UserID          uint64
	ProductType     uint32
	ProductID       uint64
	PayType         uint32
	ClientRequestID string
	Renewal         bool
	PaidAmount      float64
}

// ApplePaymentResult contains transaction fields that have already passed
// Apple evidence verification and can be persisted during fulfillment.
type ApplePaymentResult struct {
	TransactionID         string
	OriginalTransactionID string
	ProductCode           string
	OrderType             uint32
	Currency              string
	PaidAmount            float64
	SignedTransaction     string
	PurchaseDate          time.Time
	SubscriptionExpiresAt *time.Time
}

// ConsumePointsRequest describes one atomic deduction from a user's available
// VIP points and ordinary points balances.
type ConsumePointsRequest struct {
	UserID      uint64
	Points      int64
	Description string
}

// CreateOrder snapshots all mutable product values so historical orders remain
// accurate even when an administrator later edits the product.
func (s *Service) CreateOrder(ctx context.Context, req CreateOrderRequest) (*model.VideoOrder, error) {
	req.ClientRequestID = strings.TrimSpace(req.ClientRequestID)
	orderType := orderTypeForRenewal(req.Renewal)
	if req.UserID == 0 || req.ProductID == 0 || req.ClientRequestID == "" {
		return nil, errors.New("user, product and client request ID are required")
	}
	if req.PaidAmount < 0 {
		return nil, errors.New("paid amount cannot be negative")
	}
	if req.PayType != domain.PaymentMethodAppleIAP && req.PayType != domain.PaymentMethodGooglePlay {
		return nil, ErrUnsupportedPaymentMethod
	}
	if existing, err := s.orders.GetByClientRequestID(ctx, req.ClientRequestID); err == nil {
		if existing.UserID != req.UserID || existing.ProductType != req.ProductType ||
			existing.ProductID != req.ProductID || existing.PayType != req.PayType {
			return nil, errors.New("client request ID was used for a different order")
		}
		return existing, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	var created *model.VideoOrder
	err := repository.Transaction(ctx, func(ctx context.Context) error {
		user, err := s.users.GetByIDForUpdate(ctx, req.UserID)
		if err != nil {
			return err
		}
		order := &model.VideoOrder{
			OrderNo: newOrderNo(), ClientRequestID: req.ClientRequestID, UserID: req.UserID,
			ProductType: req.ProductType, ProductID: req.ProductID, PayType: req.PayType,
			Status: domain.OrderStatusPending, OrderType: orderType,
		}
		switch req.ProductType {
		case domain.OrderProductVIPSubscription:
			product, err := s.vipProducts.GetByID(ctx, uint(req.ProductID))
			if err != nil {
				return err
			}
			if product.Status != 1 {
				return errors.New("VIP product is disabled")
			}
			paidCount, err := s.orders.CountPaidByProductType(ctx, req.UserID, req.ProductType)
			if err != nil {
				return err
			}

			// Whether the transaction belongs to an Apple renewal chain controls
			// the order type, but it does not describe whether this user has ever
			// paid for a subscription in this service. The local paid-order history
			// is authoritative for the first-subscription snapshot.
			price, revenue, bonus := subscriptionOrderTerms(product, paidCount > 0)
			payableAmount := price
			if req.PaidAmount > 0 {
				payableAmount = req.PaidAmount
			}
			order.ProductCode, order.ProductName, order.Currency = product.SukCode, product.Name, strings.ToUpper(product.Currency)
			order.ProductAmount, order.PayableAmount, order.ActualAmountMoney, order.BonusPoints = price, payableAmount, revenue, int64(bonus)
			order.VipLevel, order.VipDurationDays = uint(product.LevelID), product.VIPDurationDays
		case domain.OrderProductPointsPackage:
			product, err := s.pointProducts.GetByID(ctx, uint(req.ProductID))
			if err != nil {
				return err
			}
			if product.Status != 1 {
				return errors.New("points product is disabled")
			}
			payableAmount := product.SalePrice
			if req.PaidAmount > 0 {
				payableAmount = req.PaidAmount
			}
			order.ProductCode, order.ProductName, order.Currency = product.ProductCode, product.Name, strings.ToUpper(product.Currency)
			order.ProductAmount, order.PayableAmount, order.ActualAmountMoney, order.BonusPoints = product.SalePrice, payableAmount, product.ActualRevenue, int64(product.Points)
		default:
			return ErrUnsupportedProduct
		}
		expiresAt := time.Now().Add(7 * time.Minute)
		order.ExpiresAt = expiresAt
		order.CompletedAt = time.Now().UTC()
		if err := s.orders.Create(ctx, order); err != nil {
			return err
		}

		now := time.Now()
		updates := map[string]any{
			"order_count": user.OrderCount + 1,
		}
		if user.FirstOrderCreatedAt == nil {
			updates["first_order_created_at"] = now
		}
		if err := s.users.Update(ctx, user.ID, updates); err != nil {
			return err
		}
		created = order
		return nil
	})
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			if existing, lookupErr := s.orders.GetByClientRequestID(ctx, req.ClientRequestID); lookupErr == nil {
				return existing, nil
			}
		}
		return nil, err
	}
	return created, nil
}

func subscriptionOrderTerms(product *model.VideoVipSubscription, hasPaidSubscription bool) (price, revenue float64, bonus uint64) {
	if !hasPaidSubscription {
		return product.FirstSubscriptionPrice, product.FirstSubscriptionRevenue, product.FirstBonusPoints
	}
	return product.SubscriptionPrice, product.SubscriptionRevenue, product.SubscriptionPoints
}

// ConfirmApplePayment must be called only after the Apple signed transaction
// has been verified. Row locks, a conditional status update and the provider
// transaction unique index jointly prevent duplicate payment or fulfillment.
func (s *Service) ConfirmApplePayment(ctx context.Context, orderNo string, result ApplePaymentResult) (paidOrder *model.VideoOrder, err error) {
	defer func() {
		if err == nil {
			return
		}
		if _, cancelErr := s.cancelPendingAppleOrder(ctx, orderNo, "Apple payment confirmation failed"); cancelErr != nil {
			err = errors.Join(err, fmt.Errorf("cancel pending Apple order: %w", cancelErr))
		}
	}()

	result.TransactionID = strings.TrimSpace(result.TransactionID)
	if result.OrderType == 0 {
		result.OrderType = domain.OrderTypeNewPurchase
	}
	if result.TransactionID == "" || result.PaidAmount < 0 ||
		(result.OrderType != domain.OrderTypeNewPurchase && result.OrderType != domain.OrderTypeRenewal) {
		return nil, errors.New("invalid Apple payment result")
	}
	err = repository.Transaction(ctx, func(ctx context.Context) error {
		order, err := s.orders.GetByOrderNo(ctx, orderNo, true)
		if err != nil {
			return err
		}
		if order.Status == domain.OrderStatusPaid {
			if order.ThirdOrderNo == result.TransactionID {
				paidOrder = order
				return nil
			}
			return ErrOrderAlreadyPaid
		}
		if order.Status != domain.OrderStatusPending {
			return repository.ErrOrderNotPending
		}
		if strings.TrimSpace(result.ProductCode) != order.ProductCode ||
			strings.ToUpper(strings.TrimSpace(result.Currency)) != order.Currency ||
			math.Abs(result.PaidAmount-order.PayableAmount) > 0.005 {
			order.PaidAmount = result.PaidAmount
		}

		payAt := result.PurchaseDate
		if payAt.IsZero() {
			payAt = time.Now()
		}
		if err := s.orders.MarkPaid(ctx, order.ID, map[string]any{
			"pay_type": domain.PaymentMethodAppleIAP, "third_order_no": result.TransactionID,
			"original_transaction_id": strings.TrimSpace(result.OriginalTransactionID), "paid_amount": result.PaidAmount,
			"payment_evidence": result.SignedTransaction, "pay_time": payAt, "order_type": result.OrderType,
		}); err != nil {
			return err
		}
		order, err = s.orders.GetByOrderNo(ctx, orderNo, false)
		paidOrder = order
		return err
	})
	return paidOrder, err
}

func orderTypeForRenewal(renewal bool) uint32 {
	if renewal {
		return domain.OrderTypeRenewal
	}
	return domain.OrderTypeNewPurchase
}

// NotificationApplePaymentSuccess completes a paid Apple order. The summary is
// retained for source compatibility; verified expiration is supplied by the
// purchase or notification path through fulfillAppleOrder.
func (s *Service) NotificationApplePaymentSuccess(ctx context.Context, order *model.VideoOrder, _ *AppleNotificationV2Summary) error {
	return s.fulfillAppleOrder(ctx, order, nil)
}

// fulfillAppleOrder grants the order snapshot exactly once and then moves the
// order from paid to completed. The order row lock is the idempotency boundary
// for points, counters, and subscription entitlement.
func (s *Service) fulfillAppleOrder(ctx context.Context, order *model.VideoOrder, appleExpiresAt *time.Time) error {
	if order == nil || strings.TrimSpace(order.OrderNo) == "" {
		return errors.New("Apple order is required for fulfillment")
	}
	return repository.Transaction(ctx, func(ctx context.Context) error {
		lockedOrder, err := s.orders.GetByOrderNo(ctx, order.OrderNo, true)
		if err != nil {
			return err
		}
		if lockedOrder.Status == domain.OrderStatusEnd {
			return nil
		}
		if lockedOrder.Status != domain.OrderStatusPaid {
			return fmt.Errorf("order %s is not paid", lockedOrder.OrderNo)
		}

		user, err := s.users.GetByIDForUpdate(ctx, lockedOrder.UserID)
		if err != nil {
			return err
		}
		now := time.Now()
		creditedBonusPoints := lockedOrder.BonusPoints
		if lockedOrder.ProductType == domain.OrderProductVIPSubscription &&
			lockedOrder.OrderType == domain.OrderTypeRenewal {
			// Renewal replaces the previous period's available subscription
			// points. If expiration already ran, VipPoints is already zero and
			// no duplicate deduction ledger is written.
			if user.VipPoints > 0 {
				beforeClear := user.VipPoints + user.PointsBalance
				clearLedger := &model.VideoUserPointsLedger{
					UserID: user.ID, Direction: int8(domain.PointsDirectionExpense),
					PointsChange: -user.VipPoints, BalanceBefore: uint64(beforeClear),
					BalanceAfter: uint64(user.PointsBalance), SourceType: uint32(domain.PointsSourceExpireDeduct),
					OrderCode: lockedOrder.OrderNo, VipID: lockedOrder.ProductID,
					Description: "subscription renewal old points deduction", OccurredAt: now, CreatedAt: now,
				}
				if err := s.ledgers.Create(ctx, clearLedger); err != nil {
					return err
				}
			}
			user.VipPoints = 0

			frozenVIPPoints, err := s.tasks.SumActiveVIPScore(ctx, user.ID)
			if err != nil {
				return err
			}
			if creditedBonusPoints > 0 {
				// Active tasks still own their frozen points. Withholding the same
				// amount from the new available grant prevents those old-period
				// reservations from becoming spendable twice.
				deductedFrozen := frozenVIPPoints
				if deductedFrozen > uint64(creditedBonusPoints) {
					deductedFrozen = uint64(creditedBonusPoints)
				}
				creditedBonusPoints -= int64(deductedFrozen)
			}
		}

		beforeBalance := user.VipPoints + user.PointsBalance
		if lockedOrder.ProductType == domain.OrderProductVIPSubscription {
			user.VipPoints += creditedBonusPoints
		} else {
			user.PointsBalance += creditedBonusPoints
		}
		afterBalance := user.VipPoints + user.PointsBalance

		if lockedOrder.BonusPoints > 0 {
			sourceType := uint32(domain.PointsSourcePurchase)
			description := "Purchase bonus points"
			ledger := &model.VideoUserPointsLedger{
				UserID: user.ID, Direction: int8(domain.PointsDirectionIncome),
				// PointsChange intentionally records the complete order gift;
				// balances reflect the amount made available after frozen VIP
				// reservations are offset.
				PointsChange: lockedOrder.BonusPoints, BalanceBefore: uint64(beforeBalance), BalanceAfter: uint64(afterBalance),
				SourceType: sourceType, OrderCode: lockedOrder.OrderNo,
				Description: description, OccurredAt: now, CreatedAt: now,
			}
			if lockedOrder.ProductType == domain.OrderProductVIPSubscription {
				ledger.SourceType = uint32(domain.PointsSourceSubscriptionGift)
				ledger.Description = "Subscription bonus points"
				ledger.VipID = lockedOrder.ProductID
			} else {
				ledger.PointsID = lockedOrder.ProductID
			}
			if err := s.ledgers.Create(ctx, ledger); err != nil {
				return err
			}
		}

		paidAt := lockedOrder.PayTime
		if paidAt.IsZero() {
			paidAt = now
		}
		updates := map[string]any{
			"vip_points":          user.VipPoints,
			"points_balance":      user.PointsBalance,
			"payment_count":       user.PaymentCount + 1,
			"actual_amount_money": user.ActualAmountMoney + lockedOrder.ActualAmountMoney,
			"last_paid_at":        paidAt,
			"order_amount_money":  user.OrderAmountMoney + lockedOrder.PaidAmount,
		}
		if user.FirstPaidAt == nil {
			updates["first_paid_at"] = paidAt
		}
		if user.FirstPaymentMet == 0 {
			updates["first_payment_met"] = 1
		}
		if user.PaymentMet == 0 {
			updates["payment_met"] = 1
		}
		if lockedOrder.ProductType == domain.OrderProductVIPSubscription {
			updates["subscription_payment_count"] = user.SubscriptionPaymentCount + 1
			applyVIPEntitlement(user, lockedOrder, now, appleExpiresAt, updates)
		} else {
			updates["one_time_payment_count"] = user.OneTimePaymentCount + 1
		}
		if err := s.users.Update(ctx, user.ID, updates); err != nil {
			return err
		}
		return s.orders.Update(ctx, lockedOrder.ID, map[string]any{
			"status": domain.OrderStatusEnd, "completed_at": now,
		})
	})
}

// CancelOrder cancels a pending order owned by the requested user. Paid,
// refunded, or already cancelled orders are left unchanged and return an error.
func (s *Service) CancelOrder(ctx context.Context, userID uint64, orderNo, reason string) error {
	return repository.Transaction(ctx, func(ctx context.Context) error {
		order, err := s.orders.GetByOrderNo(ctx, strings.TrimSpace(orderNo), true)
		if err != nil {
			return err
		}
		if order.UserID != userID {
			return gorm.ErrRecordNotFound
		}
		if err := s.orders.CancelPending(ctx, order.ID, strings.TrimSpace(reason), time.Now()); err != nil {
			if errors.Is(err, repository.ErrOrderNotPending) {
				return ErrOrderNotCancellable
			}
			return err
		}
		return nil
	})
}

// ConsumePoints locks the user, deducts VIP points before ordinary points, and
// writes the corresponding expense ledger in the same database transaction.
func (s *Service) ConsumePoints(ctx context.Context, req ConsumePointsRequest) (*model.VideoUserPointsLedger, error) {
	if uint64(req.Points) > uint64MaxInt64 {
		return nil, errors.New("points value exceeds supported range")
	}
	var created *model.VideoUserPointsLedger
	err := repository.Transaction(ctx, func(ctx context.Context) error {
		user, err := s.users.GetByIDForUpdate(ctx, req.UserID)
		if err != nil {
			return err
		}
		if user.VipPoints+user.PointsBalance < req.Points {
			return ErrInsufficientPoints
		}
		now := time.Now()
		beforeBalance := user.VipPoints + user.PointsBalance
		user.VipPoints = user.VipPoints - int64(req.Points)
		if user.VipPoints < 0 {
			user.PointsBalance += user.VipPoints
			user.VipPoints = 0
		}
		ledger := &model.VideoUserPointsLedger{
			UserID:    user.ID,
			Direction: int8(domain.PointsDirectionExpense), PointsChange: -int64(req.Points),
			BalanceBefore: uint64(beforeBalance), BalanceAfter: uint64(user.VipPoints + user.PointsBalance), SourceType: domain.PointsSourceModelConsume,
			Description: strings.TrimSpace(req.Description),
			OccurredAt:  now, CreatedAt: now,
		}
		if err = s.ledgers.Create(ctx, ledger); err != nil {
			return err
		}
		if err = s.users.Update(ctx, user.ID, map[string]any{"points_balance": user.PointsBalance, "vip_points": user.VipPoints}); err != nil {
			return err
		}
		created = ledger
		return nil
	})
	return created, err
}

// uint64MaxInt64 is the largest unsigned value that can be represented by the
// signed ledger PointsChange field.
const uint64MaxInt64 = ^uint64(0) >> 1

// newOrderNo generates a timestamp-prefixed order number with 48 bits of
// cryptographically secure randomness.
func newOrderNo() string {
	random := make([]byte, 6)
	if _, err := rand.Read(random); err != nil {
		panic(err)
	}
	return time.Now().UTC().Format("20060102150405") + hex.EncodeToString(random)
}

// parseVIPLevel converts a configured level value and falls back to level 1
// when the value is empty, invalid, or zero.
func parseVIPLevel(value string) uint {
	parsed, err := strconv.ParseUint(strings.TrimSpace(value), 10, 32)
	if err != nil || parsed == 0 {
		return 1
	}
	return uint(parsed)
}

// applyVIPEntitlement calculates the effective VIP level and expiration. A
// verified Apple expiration is authoritative, but an older or duplicate
// transaction must not overwrite a newer entitlement or a later cancellation.
func applyVIPEntitlement(user *model.VideoUser, order *model.VideoOrder, now time.Time, appleExpiresAt *time.Time, updates map[string]interface{}) {
	if appleExpiresAt != nil {
		if user.VipExpiresAt != nil && !appleExpiresAt.After(*user.VipExpiresAt) {
			return
		}
		level := order.VipLevel
		if level == 0 {
			level = 1
		}
		updates["vip_level"] = level
		updates["vip_expires_at"] = *appleExpiresAt
		if user.VIPStartedAt == nil {
			updates["vip_started_at"] = now
		}
		if appleExpiresAt.After(now) {
			updates["user_type"] = domain.AppUserTypePaid
			updates["subscription_status"] = domain.AppUserSubscriptionSubscribed
		} else {
			updates["user_type"] = domain.AppUserTypeFree
			updates["subscription_status"] = domain.AppUserSubscriptionCancelled
		}
		return
	}

	base := now
	if user.VipExpiresAt != nil && user.VipExpiresAt.After(now) {
		base = *user.VipExpiresAt
	}
	days := order.VipDurationDays
	if days == 0 {
		days = 30
	}
	level := order.VipLevel
	if level == 0 {
		level = 1
	}
	expiresAt := base.AddDate(0, 0, int(days))
	updates["vip_level"], updates["vip_expires_at"] = level, expiresAt
	updates["user_type"] = domain.AppUserTypePaid
	updates["subscription_status"] = domain.AppUserSubscriptionSubscribed
	if user.VIPStartedAt == nil {
		updates["vip_started_at"] = now
	}
}
