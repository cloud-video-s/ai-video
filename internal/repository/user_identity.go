package repository

import (
	"context"

	"ai-video/internal/gen/model"

	"gorm.io/gorm/clause"
)

type UserIdentityRepo struct {
	BaseRepo[model.VideoUserIdentity]
}

func NewUserIdentityRepo() *UserIdentityRepo { return &UserIdentityRepo{} }

func (r *UserIdentityRepo) GetByProviderSubject(ctx context.Context, provider, subject string, lock bool) (*model.VideoUserIdentity, error) {
	db := dbFrom(ctx).Where("provider = ? AND provider_subject = ?", provider, subject)
	if lock {
		db = db.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var item model.VideoUserIdentity
	if err := db.First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *UserIdentityRepo) GetByUserProvider(ctx context.Context, userID uint64, provider string, lock bool) (*model.VideoUserIdentity, error) {
	db := dbFrom(ctx).Where("user_id = ? AND provider = ?", userID, provider)
	if lock {
		db = db.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var item model.VideoUserIdentity
	if err := db.First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *UserIdentityRepo) ListByUser(ctx context.Context, userID uint64) ([]model.VideoUserIdentity, error) {
	var list []model.VideoUserIdentity
	if err := dbFrom(ctx).Where("user_id = ?", userID).Order("id ASC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *UserIdentityRepo) UpdateProfile(ctx context.Context, item *model.VideoUserIdentity) error {
	return r.BaseRepo.Update(ctx, item, "Issuer", "Audience", "Email", "EmailVerified", "IsPrivateEmail", "DisplayName", "GivenName", "FamilyName", "AvatarURL", "LastLoginAt", "LastTokenIssuedAt")
}

func (r *UserIdentityRepo) DeleteByUserProvider(ctx context.Context, userID uint64, provider string) error {
	return dbFrom(ctx).Where("user_id = ? AND provider = ?", userID, provider).Delete(&model.VideoUserIdentity{}).Error
}
