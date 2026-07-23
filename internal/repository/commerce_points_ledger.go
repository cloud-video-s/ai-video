package repository

import (
	"context"

	"ai-video/internal/gen/model"
)

type CommercePointsLedgerRepo struct{}

func NewCommercePointsLedgerRepo() *CommercePointsLedgerRepo { return &CommercePointsLedgerRepo{} }

func (r *CommercePointsLedgerRepo) Create(ctx context.Context, item *model.VideoUserPointsLedger) error {
	return qFrom(ctx).VideoUserPointsLedger.WithContext(ctx).Create(item)
}

func (r *CommercePointsLedgerRepo) GetByIdempotencyKey(ctx context.Context, key string) (*model.VideoUserPointsLedger, error) {
	q := qFrom(ctx).VideoUserPointsLedger
	return q.WithContext(ctx).Where(q.IdempotencyKey.Eq(key)).First()
}
