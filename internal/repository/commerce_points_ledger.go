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
