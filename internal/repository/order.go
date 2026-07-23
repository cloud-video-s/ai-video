package repository

import (
	"context"
	"errors"
	"time"

	"ai-video/internal/domain"
	"ai-video/internal/gen/model"

	"gorm.io/gorm/clause"
)

var (
	ErrOrderAlreadyPaid = errors.New("order already paid")
	ErrOrderNotPending  = errors.New("order is not pending")
)

type OrderRepo struct{}

func NewOrderRepo() *OrderRepo { return &OrderRepo{} }

func (r *OrderRepo) Create(ctx context.Context, order *model.VideoOrder) error {
	// Pending orders do not have provider or completion timestamps yet. Omit
	// these nullable columns so MySQL does not store empty transaction IDs or
	// zero dates (which would also break the composite unique index).
	return dbFrom(ctx).Omit(
		"ProviderTransactionID", "OriginalTransactionID", "PaidAt", "CancelledAt",
	).Create(order).Error
}

func (r *OrderRepo) GetByOrderNo(ctx context.Context, orderNo string, lock bool) (*model.VideoOrder, error) {
	var order model.VideoOrder
	db := dbFrom(ctx)
	if lock {
		db = db.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	if err := db.Where("order_no = ?", orderNo).First(&order).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *OrderRepo) GetByClientRequestID(ctx context.Context, requestID string) (*model.VideoOrder, error) {
	var order model.VideoOrder
	if err := dbFrom(ctx).Where("client_request_id = ?", requestID).First(&order).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *OrderRepo) GetByPaymentTransaction(ctx context.Context, method, transactionID string) (*model.VideoOrder, error) {
	var order model.VideoOrder
	if err := dbFrom(ctx).Where("payment_method = ? AND provider_transaction_id = ?", method, transactionID).First(&order).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *OrderRepo) CountPaidByProductType(ctx context.Context, userID uint64, productType string) (int64, error) {
	var count int64
	err := dbFrom(ctx).Model(&model.VideoOrder{}).
		Where("user_id = ? AND product_type = ? AND status = ?", userID, productType, domain.OrderStatusPaid).
		Count(&count).Error
	return count, err
}

func (r *OrderRepo) MarkPaid(ctx context.Context, id uint64, updates map[string]interface{}) error {
	updates["status"] = domain.OrderStatusPaid
	result := dbFrom(ctx).Model(&model.VideoOrder{}).
		Where("id = ? AND status = ?", id, domain.OrderStatusPending).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		var order model.VideoOrder
		if err := dbFrom(ctx).Select("status").First(&order, id).Error; err != nil {
			return err
		}
		if order.Status == domain.OrderStatusPaid {
			return ErrOrderAlreadyPaid
		}
		return ErrOrderNotPending
	}
	return nil
}

func (r *OrderRepo) CancelPending(ctx context.Context, id uint64, reason string, now time.Time) error {
	result := dbFrom(ctx).Model(&model.VideoOrder{}).
		Where("id = ? AND status = ?", id, domain.OrderStatusPending).
		Updates(map[string]interface{}{"status": domain.OrderStatusCancelled, "cancel_reason": reason, "cancelled_at": now})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrOrderNotPending
	}
	return nil
}

func (r *OrderRepo) PageByUser(ctx context.Context, userID uint64, page, pageSize int) ([]model.VideoOrder, int64, error) {
	db := dbFrom(ctx).Model(&model.VideoOrder{}).Where("user_id = ?", userID)
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.VideoOrder
	err := db.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}
