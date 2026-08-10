package repository

import (
	"context"
	"strings"
	"time"

	"ai-video/internal/domain"
	"ai-video/internal/gen/model"
)

type CommercePointsLedgerRepo struct{}

func NewCommercePointsLedgerRepo() *CommercePointsLedgerRepo { return &CommercePointsLedgerRepo{} }

func (r *CommercePointsLedgerRepo) Create(ctx context.Context, item *model.VideoUserPointsLedger) error {
	return qFrom(ctx).VideoUserPointsLedger.WithContext(ctx).Create(item)
}

// HasSubscriptionGiftSince reports whether a different subscription order
// granted points at or after the supplied expiration. It is a fallback guard
// for out-of-order Apple expiration notifications when entitlement timestamps
// alone no longer identify the newer purchase.
func (r *CommercePointsLedgerRepo) HasSubscriptionGiftSince(
	ctx context.Context,
	userID uint64,
	since time.Time,
	excludedOrderCode string,
) (bool, error) {
	q := qFrom(ctx).VideoUserPointsLedger
	dao := q.WithContext(ctx).Where(
		q.UserID.Eq(userID),
		q.SourceType.Eq(uint32(domain.PointsSourceSubscriptionGift)),
		q.PointsChange.Gt(0),
		q.OccurredAt.Gte(since),
	)
	if excludedOrderCode = strings.TrimSpace(excludedOrderCode); excludedOrderCode != "" {
		dao = dao.Where(q.OrderCode.Neq(excludedOrderCode))
	}
	total, err := dao.Count()
	return total > 0, err
}
