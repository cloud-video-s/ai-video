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
	q := qFrom(ctx).VideoOrder
	return q.WithContext(ctx).Omit(
		q.ProviderTransactionID, q.OriginalTransactionID, q.PaidAt, q.CancelledAt,
	).Create(order)
}

func (r *OrderRepo) GetByOrderNo(ctx context.Context, orderNo string, lock bool) (*model.VideoOrder, error) {
	q := qFrom(ctx).VideoOrder
	dao := q.WithContext(ctx).Where(q.OrderNo.Eq(orderNo))
	if lock {
		dao = dao.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	return dao.First()
}

func (r *OrderRepo) GetByClientRequestID(ctx context.Context, requestID string) (*model.VideoOrder, error) {
	q := qFrom(ctx).VideoOrder
	return q.WithContext(ctx).Where(q.ClientRequestID.Eq(requestID)).First()
}

func (r *OrderRepo) GetByPaymentTransaction(ctx context.Context, method, transactionID string) (*model.VideoOrder, error) {
	q := qFrom(ctx).VideoOrder
	return q.WithContext(ctx).Where(q.PaymentMethod.Eq(method), q.ProviderTransactionID.Eq(transactionID)).First()
}

func (r *OrderRepo) CountPaidByProductType(ctx context.Context, userID uint64, productType string) (int64, error) {
	q := qFrom(ctx).VideoOrder
	return q.WithContext(ctx).Where(
		q.UserID.Eq(userID), q.ProductType.Eq(productType), q.Status.Eq(domain.OrderStatusPaid),
	).Count()
}

func (r *OrderRepo) MarkPaid(ctx context.Context, id uint64, updates map[string]interface{}) error {
	updates["status"] = domain.OrderStatusPaid
	q := qFrom(ctx).VideoOrder
	result, err := q.WithContext(ctx).Where(q.ID.Eq(id), q.Status.Eq(domain.OrderStatusPending)).Updates(updates)
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		order, err := q.WithContext(ctx).Select(q.Status).Where(q.ID.Eq(id)).First()
		if err != nil {
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
	q := qFrom(ctx).VideoOrder
	result, err := q.WithContext(ctx).Where(q.ID.Eq(id), q.Status.Eq(domain.OrderStatusPending)).
		Updates(map[string]interface{}{"status": domain.OrderStatusCancelled, "cancel_reason": reason, "cancelled_at": now})
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return ErrOrderNotPending
	}
	return nil
}

func (r *OrderRepo) PageByUser(ctx context.Context, userID uint64, page, pageSize int) ([]model.VideoOrder, int64, error) {
	q := qFrom(ctx).VideoOrder
	dao := q.WithContext(ctx).Where(q.UserID.Eq(userID))
	total, err := dao.Count()
	if err != nil {
		return nil, 0, err
	}
	rows, err := dao.Order(q.ID.Desc()).Offset((page - 1) * pageSize).Limit(pageSize).Find()
	return valuesOf(rows), total, err
}
