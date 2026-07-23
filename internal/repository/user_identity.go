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
	q := qFrom(ctx).VideoUserIdentity
	dao := q.WithContext(ctx).Where(q.Provider.Eq(provider), q.ProviderSubject.Eq(subject))
	if lock {
		dao = dao.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	return dao.First()
}

func (r *UserIdentityRepo) GetByUserProvider(ctx context.Context, userID uint64, provider string, lock bool) (*model.VideoUserIdentity, error) {
	q := qFrom(ctx).VideoUserIdentity
	dao := q.WithContext(ctx).Where(q.UserID.Eq(userID), q.Provider.Eq(provider))
	if lock {
		dao = dao.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	return dao.First()
}

func (r *UserIdentityRepo) ListByUser(ctx context.Context, userID uint64) ([]model.VideoUserIdentity, error) {
	q := qFrom(ctx).VideoUserIdentity
	rows, err := q.WithContext(ctx).Where(q.UserID.Eq(userID)).Order(q.ID.Asc()).Find()
	return valuesOf(rows), err
}

func (r *UserIdentityRepo) UpdateProfile(ctx context.Context, item *model.VideoUserIdentity) error {
	q := qFrom(ctx).VideoUserIdentity
	_, err := q.WithContext(ctx).Where(q.ID.Eq(item.ID)).Select(
		q.Issuer, q.Audience, q.Email, q.EmailVerified, q.IsPrivateEmail, q.DisplayName,
		q.GivenName, q.FamilyName, q.AvatarURL, q.LastLoginAt, q.LastTokenIssuedAt,
	).Updates(item)
	return err
}

func (r *UserIdentityRepo) DeleteByUserProvider(ctx context.Context, userID uint64, provider string) error {
	q := qFrom(ctx).VideoUserIdentity
	_, err := q.WithContext(ctx).Where(q.UserID.Eq(userID), q.Provider.Eq(provider)).Delete()
	return err
}
