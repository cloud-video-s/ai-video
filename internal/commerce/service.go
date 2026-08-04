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
	orders        *repository.OrderRepo
	users         *repository.AppUserRepo
	ledgers       *repository.CommercePointsLedgerRepo
	vipProducts   *repository.VIPSubscriptionRepo
	pointProducts *repository.PointsPackageRepo
	packages      *repository.PackageRepo
	appleRootCAs  *x509.CertPool
}

// NewService constructs a commerce service with production repositories and
// the built-in Apple notification trust store.
func NewService() *Service {
	return &Service{
		orders: repository.NewOrderRepo(), users: repository.NewAppUserRepo(),
		ledgers: repository.NewCommercePointsLedgerRepo(), vipProducts: repository.NewVIPSubscriptionRepo(),
		pointProducts: repository.NewPointsPackageRepo(), packages: repository.NewPackageRepo(),
		appleRootCAs: defaultAppleRootCAs,
	}
}

// CreateOrderRequest identifies the user, product, and idempotency key used to
// snapshot a new order.
type CreateOrderRequest struct {
	UserID          uint64
	ProductType     uint32
	ProductID       uint64
	PaymentMethod   string
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
	Points      uint64
	Description string
}

// CreateOrder snapshots all mutable product values so historical orders remain
// accurate even when an administrator later edits the product.
func (s *Service) CreateOrder(ctx context.Context, req CreateOrderRequest) (*model.VideoOrder, error) {
	req.PaymentMethod = strings.TrimSpace(req.PaymentMethod)
	req.ClientRequestID = strings.TrimSpace(req.ClientRequestID)
	if req.UserID == 0 || req.ProductID == 0 || req.ClientRequestID == "" {
		return nil, errors.New("user, product and client request ID are required")
	}
	if req.PaymentMethod != domain.PaymentMethodAppleIAP {
		return nil, ErrUnsupportedPaymentMethod
	}
	if existing, err := s.orders.GetByClientRequestID(ctx, req.ClientRequestID); err == nil {
		if existing.UserID != req.UserID || existing.ProductType != req.ProductType || existing.ProductID != req.ProductID {
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
			ProductType: req.ProductType, ProductID: req.ProductID, PaymentMethod: req.PaymentMethod,
			Status: domain.OrderStatusPending,
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
			price, bonus := product.SubscriptionPrice, product.SubscriptionPoints
			if paidCount == 0 && !req.Renewal {
				price, bonus = product.FirstSubscriptionPrice, product.FirstBonusPoints
			}
			order.ProductCode, order.ProductName, order.Currency = product.SukCode, product.Name, strings.ToUpper(product.Currency)
			order.ProductAmount, order.PayableAmount, order.BonusPoints = price, req.PaidAmount, bonus
			order.VipLevel, order.VipDurationDays = uint(product.LevelID), product.VIPDurationDays
		case domain.OrderProductPointsPackage:
			product, err := s.pointProducts.GetByID(ctx, uint(req.ProductID))
			if err != nil {
				return err
			}
			if product.Status != 1 {
				return errors.New("points product is disabled")
			}
			order.ProductCode, order.ProductName, order.Currency = product.ProductCode, product.Name, strings.ToUpper(product.Currency)
			order.ProductAmount, order.PayableAmount, order.BonusPoints = product.SalePrice, product.SalePrice, product.Points
		default:
			return ErrUnsupportedProduct
		}
		expiresAt := time.Now().Add(30 * time.Minute)
		order.ExpiresAt = expiresAt
		if err := s.orders.Create(ctx, order); err != nil {
			return err
		}

		now := time.Now()
		updates := map[string]interface{}{
			"order_count":        user.OrderCount + 1,
			"order_amount_money": user.OrderAmountMoney + order.ProductAmount,
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
	if result.TransactionID == "" || result.PaidAmount < 0 {
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
		if used, err := s.orders.GetByPaymentTransaction(ctx, domain.PaymentMethodAppleIAP, result.TransactionID); err == nil && used.ID != order.ID {
			return ErrPaymentTransactionUsed
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		//user, err := s.users.GetByIDForUpdate(ctx, order.UserID)
		//if err != nil {
		//	return err
		//}
		//now :=
		paidAt := result.PurchaseDate
		if paidAt.IsZero() {
			paidAt = time.Now()
		}
		//if order.BonusPoints > 0 {
		//if order.ProductType == domain.OrderProductVIPSubscription {
		//	user.VipPoints = user.VipPoints + order.BonusPoints
		//} else {
		//	user.PointsBalance = user.PointsBalance + order.BonusPoints
		//}
		//before, after := user.VipPoints+user.PointsBalance-order.BonusPoints, user.VipPoints+user.PointsBalance
		//ledger := &model.VideoUserPointsLedger{
		//	UserID: user.ID, Direction: int8(domain.PointsDirectionIncome),
		//	PointsChange: int64(order.BonusPoints), BalanceBefore: before, BalanceAfter: after,
		//	SourceType: domain.PointsSourcePurchase, OrderCode: order.OrderNo,
		//	Description: "Subscription bonus points", OccurredAt: paidAt, CreatedAt: now,
		//}
		//if order.ProductType == domain.OrderProductPointsPackage {
		//	ledger.PointsID = order.ProductID
		//}
		//if order.ProductType == domain.OrderProductVIPSubscription {
		//	ledger.VipID = order.ProductID
		//}
		//if err = s.ledgers.Create(ctx, ledger); err != nil {
		//	return err
		//}
		//}

		//updates := map[string]interface{}{
		//	"vip_points":          user.VipPoints,
		//	"points_balance":      user.PointsBalance,
		//	"payment_count":       user.PaymentCount + 1,
		//	"actual_amount_money": user.ActualAmountMoney + result.PaidAmount,
		//	"last_paid_at":        paidAt, "payment_met": 1,
		//}
		//if user.FirstPaidAt == nil {
		//	updates["first_paid_at"], updates["first_payment_met"] = paidAt, 1
		//}
		//if order.ProductType == domain.OrderProductVIPSubscription {
		//	updates["subscription_payment_count"] = user.SubscriptionPaymentCount + 1
		//	applyVIPEntitlement(user, order, paidAt, result.SubscriptionExpiresAt, updates)
		//} else {
		//	updates["one_time_payment_count"] = user.OneTimePaymentCount + 1
		//}
		//if err := s.users.Update(ctx, user.ID, updates); err != nil {
		//	return err
		//}

		if err := s.orders.MarkPaid(ctx, order.ID, map[string]interface{}{
			"payment_method": domain.PaymentMethodAppleIAP, "third_order_no": result.TransactionID,
			"original_transaction_id": strings.TrimSpace(result.OriginalTransactionID), "paid_amount": result.PaidAmount,
			"payment_evidence": result.SignedTransaction, "paid_at": paidAt,
		}); err != nil {
			return err
		}
		order, err = s.orders.GetByOrderNo(ctx, orderNo, false)
		paidOrder = order
		return err
	})
	return paidOrder, err
}

func (s *Service) NotificationApplePayment(ctx context.Context, order *model.VideoOrder, summary *AppleNotificationV2Summary) (paidOrder *model.VideoOrder, err error) {
	defer func() {
		if err == nil {
			return
		}
		if _, cancelErr := s.cancelPendingAppleOrder(ctx, order.OrderNo, "Apple payment confirmation failed"); cancelErr != nil {
			err = errors.Join(err, fmt.Errorf("cancel pending Apple order: %w", cancelErr))
		}
	}()
	if order.Status != domain.OrderStatusPaid {
		return nil, repository.ErrOrderAlreadyPaid
	}
	summary.TransactionID = strings.TrimSpace(summary.TransactionID)
	err = repository.Transaction(ctx, func(ctx context.Context) error {
		order, err = s.orders.GetByOrderNo(ctx, order.OrderNo, true)
		if err != nil {
			return err
		}
		if used, err := s.orders.GetByPaymentTransaction(ctx, domain.PaymentMethodAppleIAP, summary.TransactionID); err == nil && used.ID != order.ID {
			return ErrPaymentTransactionUsed
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		user, err := s.users.GetByIDForUpdate(ctx, order.UserID)
		if err != nil {
			return err
		}
		now := time.Now()
		if order.BonusPoints > 0 {
			if order.ProductType == domain.OrderProductVIPSubscription {
				user.VipPoints = user.VipPoints + order.BonusPoints
			} else {
				user.PointsBalance = user.PointsBalance + order.BonusPoints
			}
			before, after := user.VipPoints+user.PointsBalance-order.BonusPoints, user.VipPoints+user.PointsBalance
			ledger := &model.VideoUserPointsLedger{
				UserID: user.ID, Direction: int8(domain.PointsDirectionIncome),
				PointsChange: int64(order.BonusPoints), BalanceBefore: before, BalanceAfter: after,
				SourceType: domain.PointsSourcePurchase, OrderCode: order.OrderNo,
				Description: "Subscription bonus points", OccurredAt: now, CreatedAt: now,
			}
			if order.ProductType == domain.OrderProductPointsPackage {
				ledger.PointsID = order.ProductID
			}
			if order.ProductType == domain.OrderProductVIPSubscription {
				ledger.VipID = order.ProductID
			}
			if err = s.ledgers.Create(ctx, ledger); err != nil {
				return err
			}
		}

		//updates := map[string]interface{}{
		//	"vip_points":          user.VipPoints,
		//	"points_balance":      user.PointsBalance,
		//	"payment_count":       user.PaymentCount + 1,
		//	"actual_amount_money": user.ActualAmountMoney + order.PaidAmount,
		//	"last_paid_at":        now, "payment_met": 1,
		//}
		//if user.FirstPaidAt == nil {
		//	updates["first_paid_at"], updates["first_payment_met"] = new, 1
		//}
		//if order.ProductType == domain.OrderProductVIPSubscription {
		//	updates["subscription_payment_count"] = user.SubscriptionPaymentCount + 1
		//	applyVIPEntitlement(user, order, new, order.SubscriptionExpiresAt, updates)
		//} else {
		//	updates["one_time_payment_count"] = user.OneTimePaymentCount + 1
		//}
		//if err := s.users.Update(ctx, user.ID, updates); err != nil {
		//	return err
		//}
		//
		//if err := s.orders.MarkPaid(ctx, order.ID, map[string]interface{}{
		//	"payment_method": domain.PaymentMethodAppleIAP, "third_order_no": summary.TransactionID,
		//	"original_transaction_id": strings.TrimSpace(result.OriginalTransactionID), "paid_amount": result.PaidAmount,
		//	"payment_evidence": result.SignedTransaction, "paid_at": paidAt,
		//}); err != nil {
		//	return err
		//}
		//order, err = s.orders.GetByOrderNo(ctx, order.OrderNo, false)
		paidOrder = order
		return err
	})
	return paidOrder, err
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
	if req.Points > uint64MaxInt64 {
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
		user.VipPoints = user.VipPoints - req.Points
		if user.VipPoints < 0 {
			user.PointsBalance += user.VipPoints
			user.VipPoints = 0
		}
		ledger := &model.VideoUserPointsLedger{
			UserID:    user.ID,
			Direction: int8(domain.PointsDirectionExpense), PointsChange: -int64(req.Points),
			BalanceBefore: user.PointsBalance, BalanceAfter: user.VipPoints + user.PointsBalance, SourceType: domain.PointsSourceModelConsume,
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
// verified Apple expiration takes precedence over the locally configured days.
func applyVIPEntitlement(user *model.VideoUser, order *model.VideoOrder, now time.Time, appleExpiresAt *time.Time, updates map[string]interface{}) {
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
	if appleExpiresAt != nil && appleExpiresAt.After(now) {
		expiresAt = *appleExpiresAt
	}
	updates["vip_level"], updates["vip_expires_at"] = level, expiresAt
	updates["user_type"] = domain.AppUserTypePaid
	updates["subscription_status"] = domain.AppUserSubscriptionSubscribed
	if user.VIPStartedAt == nil {
		updates["vip_started_at"] = now
	}
}
