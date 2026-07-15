package repository

import (
	"ai-video/internal/gen/model"
	"context"
)

type UserRepo struct{}

func NewUserRepo() *UserRepo {
	return &UserRepo{}
}

func (d *UserRepo) GetByPhoneCode(ctx context.Context, phoneCode string) (*model.VideoUser, error) {
	q := qFrom(ctx).VideoUser
	user, err := q.WithContext(ctx).Where(q.PhoneCode.Eq(phoneCode)).Order(q.LastLoginAt.Desc()).First()
	if err != nil {
		return nil, err
	}
	return user, nil
}
