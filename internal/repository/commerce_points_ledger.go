package repository

import (
	"context"

	"ai-video/internal/gen/model"
)

type CommercePointsLedgerRepo struct{}

func NewCommercePointsLedgerRepo() *CommercePointsLedgerRepo { return &CommercePointsLedgerRepo{} }

func (r *CommercePointsLedgerRepo) Create(ctx context.Context, item *model.VideoUserPointsLedger) error {
	return dbFrom(ctx).Create(item).Error
}

func (r *CommercePointsLedgerRepo) GetByIdempotencyKey(ctx context.Context, key string) (*model.VideoUserPointsLedger, error) {
	var item model.VideoUserPointsLedger
	if err := dbFrom(ctx).Where("idempotency_key = ?", key).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}
