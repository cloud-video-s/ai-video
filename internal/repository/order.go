package repository

import (
	"ai-video/internal/gen/query"
	"context"
	"errors"
	"time"

	"ai-video/internal/domain"
	"ai-video/internal/gen/model"

	"gorm.io/gen/field"
	"gorm.io/gorm/clause"
)

var (
	ErrOrderAlreadyPaid  = errors.New("order already paid")
	ErrOrderNotPending   = errors.New("order is not pending")
	ErrOrderNotCompleted = errors.New("order is not completed")
)

type OrderRepo struct{}

func NewOrderRepo() *OrderRepo { return &OrderRepo{} }

type OrderAdminFilter struct {
	UserID      uint64
	ProductType uint32
	ProductCode string
	Status      uint32
	PayType     uint32
	Keyword     string
	CreatedFrom *time.Time
	CreatedTo   *time.Time
}

type OrderAdminSummary struct {
	PaidOrderCount int64                       `gorm:"column:paid_order_count" json:"paid_order_count"`
	Amounts        []OrderAdminCurrencySummary `gorm:"-" json:"amounts"`
}

type OrderAdminCurrencySummary struct {
	Currency      string  `gorm:"column:currency" json:"currency"`
	PayableTotal  float64 `gorm:"column:payable_total" json:"payable_total"`
	PaidTotal     float64 `gorm:"column:paid_total" json:"paid_total"`
	RefundedTotal float64 `gorm:"column:refunded_total" json:"refunded_total"`
}

type OrderAdminRecord struct {
	Order model.VideoOrder
	User  *model.VideoUser
}

func (r *OrderRepo) Create(ctx context.Context, order *model.VideoOrder) error {
	// Pending orders do not have provider or completion timestamps yet. Omit
	// these nullable columns so MySQL does not store empty transaction IDs or
	// zero dates (which would also break the composite unique index).
	q := qFrom(ctx).VideoOrder
	return q.WithContext(ctx).Omit(
		q.ThirdOrderNo, q.OriginalTransactionID, q.PayTime, q.CancelledAt, q.CompletedAt,
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

func (r *OrderRepo) GetByPaymentTransaction(ctx context.Context, payType uint32, transactionID string) (*model.VideoOrder, error) {
	q := qFrom(ctx).VideoOrder
	return q.WithContext(ctx).Where(q.PayType.Eq(payType), q.ThirdOrderNo.Eq(transactionID)).First()
}

// GetByAppleOriginalTransactionID returns the newest Apple IAP order in an
// original transaction chain. Subscription renewals share the same original
// transaction ID, so ordering by ID keeps the lookup deterministic.
func (r *OrderRepo) GetByAppleOriginalTransactionID(ctx context.Context, originalTransactionID string) (*model.VideoOrder, error) {
	q := qFrom(ctx).VideoOrder
	return q.WithContext(ctx).Where(
		q.PayType.Eq(domain.PaymentMethodAppleIAP),
		q.OriginalTransactionID.Eq(originalTransactionID),
	).Order(q.ID.Desc()).First()
}

func (r *OrderRepo) GetByAppleOriginalTransactionIDs(ctx context.Context, originalTransactionIDs []string) (*model.VideoOrder, error) {
	q := qFrom(ctx).VideoOrder
	return q.WithContext(ctx).Where(
		q.ProductType.Eq(domain.OrderProductVIPSubscription),
		q.OriginalTransactionID.In(originalTransactionIDs...),
	).Order(q.ID.Desc()).First()
}

func (r *OrderRepo) CountPaidByProductType(ctx context.Context, userID uint64, productType uint32) (int64, error) {
	q := qFrom(ctx).VideoOrder
	return q.WithContext(ctx).Where(
		q.UserID.Eq(userID), q.ProductType.Eq(productType),
		q.Status.In(domain.OrderStatusPaid, domain.OrderStatusEnd),
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

func (r *OrderRepo) Update(ctx context.Context, id uint64, updates map[string]interface{}) error {
	q := qFrom(ctx).VideoOrder
	result, err := q.WithContext(ctx).Where(q.ID.Eq(id), q.Status.Eq(domain.OrderStatusPaid)).Updates(updates)
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		order, err := q.WithContext(ctx).Select(q.Status).Where(q.ID.Eq(id)).First()
		if err != nil {
			return err
		}
		if order.Status == domain.OrderStatusEnd {
			return ErrOrderNotCompleted
		}
		return ErrOrderAlreadyPaid
	}
	return nil
}

func (r *OrderRepo) CancelPending(ctx context.Context, id uint64, reason string, now time.Time) error {
	q := qFrom(ctx).VideoOrder
	result, err := q.WithContext(ctx).Where(q.ID.Eq(id), q.Status.Eq(domain.OrderStatusPending)).
		Updates(map[string]any{"status": domain.OrderStatusCancelled, "cancel_reason": reason, "cancelled_at": now})
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

// PageAdmin returns filtered orders together with a compact purchaser
// association. Purchasers are loaded separately and unscoped so historical
// orders can still identify a user that has since been soft-deleted.
func (r *OrderRepo) PageAdmin(ctx context.Context, page, pageSize int, filter *OrderAdminFilter) ([]OrderAdminRecord, int64, OrderAdminSummary, error) {
	// 1. 统计总数
	total, err := buildDao(ctx, filter).Count()
	if err != nil {
		return nil, 0, OrderAdminSummary{}, err
	}

	// 2. 汇总（无 GROUP BY）
	var summary OrderAdminSummary
	if err := buildDao(ctx, filter).Select(
		field.NewUnsafeFieldRaw("COALESCE(SUM(CASE WHEN video_order.status IN (3, 4) THEN 1 ELSE 0 END), 0)").As("paid_order_count"),
	).Scan(&summary); err != nil {
		return nil, 0, OrderAdminSummary{}, err
	}
	order := qFrom(ctx).VideoOrder
	// 3. 按货币分组汇总（仅分组列 + 聚合函数，符合 ONLY_FULL_GROUP_BY）
	amounts := make([]OrderAdminCurrencySummary, 0)
	if err := buildDao(ctx, filter).Select(
		order.Currency,
		field.NewUnsafeFieldRaw("COALESCE(SUM(video_order.payable_amount), 0)").As("payable_total"),
		field.NewUnsafeFieldRaw("COALESCE(SUM(video_order.paid_amount), 0)").As("paid_total"),
		field.NewUnsafeFieldRaw("COALESCE(SUM(video_order.refunded_amount), 0)").As("refunded_total"),
	).Group(order.Currency).Order(order.Currency.Asc()).Scan(&amounts); err != nil {
		return nil, 0, OrderAdminSummary{}, err
	}
	summary.Amounts = amounts

	// 4. 分页列表（无 GROUP BY）
	rows, err := buildDao(ctx, filter).Select(order.ALL).Order(order.CreatedAt.Desc(), order.ID.Desc()).
		Offset((page - 1) * pageSize).Limit(pageSize).Find()
	if err != nil {
		return nil, 0, OrderAdminSummary{}, err
	}
	records, err := r.loadAdminRecords(ctx, valuesOf(rows))
	return records, total, summary, err
}

// PageAdminList is the compact list-only variant used by the dashboard. It
// avoids running the generic order page's additional summary queries.
func (r *OrderRepo) PageAdminList(ctx context.Context, page, pageSize int, filter *OrderAdminFilter) ([]OrderAdminRecord, int64, error) {
	total, err := buildDao(ctx, filter).Count()
	if err != nil {
		return nil, 0, err
	}
	order := qFrom(ctx).VideoOrder
	rows, err := buildDao(ctx, filter).Select(order.ALL).
		Order(order.CreatedAt.Desc(), order.ID.Desc()).
		Offset((page - 1) * pageSize).Limit(pageSize).Find()
	if err != nil {
		return nil, 0, err
	}
	records, err := r.loadAdminRecords(ctx, valuesOf(rows))
	return records, total, err
}

func buildDao(ctx context.Context, filter *OrderAdminFilter) query.IVideoOrderDo {
	q := qFrom(ctx)
	order := q.VideoOrder
	user := q.VideoUser
	dao := order.WithContext(ctx)
	if filter == nil {
		return dao
	}
	if filter.UserID != 0 {
		dao = dao.Where(order.UserID.Eq(filter.UserID))
	}
	if filter.ProductType != 0 {
		dao = dao.Where(order.ProductType.Eq(filter.ProductType))
	}
	if filter.ProductCode != "" {
		dao = dao.Where(order.ProductCode.Eq(filter.ProductCode))
	}
	if filter.Status > 0 {
		dao = dao.Where(order.Status.Eq(filter.Status))
	}
	if filter.PayType > 0 {
		dao = dao.Where(order.PayType.Eq(filter.PayType))
	}
	if filter.CreatedFrom != nil {
		dao = dao.Where(order.CreatedAt.Gte(*filter.CreatedFrom))
	}
	if filter.CreatedTo != nil {
		dao = dao.Where(order.CreatedAt.Lt(*filter.CreatedTo))
	}
	if filter.Keyword != "" {
		keyword := "%" + filter.Keyword + "%"
		dao = dao.LeftJoin(user, user.ID.EqCol(order.UserID))
		conditions := []field.Expr{
			order.OrderNo.Like(keyword), order.ClientRequestID.Like(keyword),
			order.ProductCode.Like(keyword), order.ProductName.Like(keyword),
			order.ThirdOrderNo.Like(keyword), order.OriginalTransactionID.Like(keyword),
			user.Username.Like(keyword), user.LoginAccount.Like(keyword),
			user.Email.Like(keyword), user.Phone.Like(keyword), user.IMEI.Like(keyword),
		}
		dao = dao.Where(field.Or(conditions...))
	}
	return dao
}
func (r *OrderRepo) GetAdminDetail(ctx context.Context, id uint64) (*OrderAdminRecord, error) {
	q := qFrom(ctx).VideoOrder
	item, err := q.WithContext(ctx).Where(q.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}
	records, err := r.loadAdminRecords(ctx, []model.VideoOrder{*item})
	if err != nil {
		return nil, err
	}
	return &records[0], nil
}

func (r *OrderRepo) loadAdminRecords(ctx context.Context, orders []model.VideoOrder) ([]OrderAdminRecord, error) {
	records := make([]OrderAdminRecord, 0, len(orders))
	if len(orders) == 0 {
		return records, nil
	}
	userIDs := make([]uint64, 0, len(orders))
	for i := range orders {
		userIDs = append(userIDs, orders[i].UserID)
	}
	q := qFrom(ctx).VideoUser
	users, err := q.WithContext(ctx).Unscoped().Where(q.ID.In(userIDs...)).Find()
	if err != nil {
		return nil, err
	}
	userByID := make(map[uint64]*model.VideoUser, len(users))
	for _, user := range users {
		if user != nil {
			userByID[user.ID] = user
		}
	}
	for i := range orders {
		records = append(records, OrderAdminRecord{Order: orders[i], User: userByID[orders[i].UserID]})
	}
	return records, nil
}
