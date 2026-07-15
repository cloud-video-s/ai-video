package repository

import (
	"context"

	"ai-video/internal/model"

	"gorm.io/gorm/clause"
)

type AppUserRepo struct{}

func NewAppUserRepo() *AppUserRepo {
	return &AppUserRepo{}
}

type AppUserListFilter struct {
	Keyword            string
	DeviceCountry      string
	IPCountry          string
	ChannelID          string
	AppVersion         string
	LoginType          string
	UserType           string
	SubscriptionStatus string
	Activated          *bool
	Registered         *bool
	PaymentMet         *bool
}

func (d *AppUserRepo) Create(ctx context.Context, user *model.VideoUser) error {
	return qFrom(ctx).VideoUser.WithContext(ctx).UnderlyingDB().Create(user).Error
}

func (d *AppUserRepo) GetByID(ctx context.Context, id uint64) (*model.VideoUser, error) {
	var user model.VideoUser
	q := qFrom(ctx).VideoUser
	err := q.WithContext(ctx).Where(q.ID.Eq(id)).UnderlyingDB().First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (d *AppUserRepo) GetTokenVersion(ctx context.Context, id uint64) (int, error) {
	var user model.VideoUser
	q := qFrom(ctx).VideoUser
	err := q.WithContext(ctx).Select(q.TokenVersion).Where(q.ID.Eq(id), q.Status.Eq(1)).UnderlyingDB().First(&user).Error
	if err != nil {
		return 0, err
	}
	return user.TokenVersion, nil
}

func (d *AppUserRepo) GetLatestByPhoneCode(ctx context.Context, phoneCode string, lock bool) (*model.VideoUser, error) {
	var user model.VideoUser
	q := qFrom(ctx).VideoUser
	db := q.WithContext(ctx).Where(q.PhoneCode.Eq(phoneCode)).Order(q.RegistrationNo.Desc(), q.ID.Desc()).UnderlyingDB()
	if lock {
		db = db.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	if err := db.First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (d *AppUserRepo) Update(ctx context.Context, id uint64, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	q := qFrom(ctx).VideoUser
	return q.WithContext(ctx).Where(q.ID.Eq(id)).UnderlyingDB().
		Model(&model.VideoUser{}).
		Updates(updates).Error
}

func (d *AppUserRepo) Delete(ctx context.Context, id uint64) error {
	q := qFrom(ctx).VideoUser
	return q.WithContext(ctx).Where(q.ID.Eq(id)).UnderlyingDB().Delete(&model.VideoUser{}).Error
}

func (d *AppUserRepo) PageList(ctx context.Context, page, pageSize int, filter *AppUserListFilter) ([]model.VideoUser, int64, error) {
	q := qFrom(ctx).VideoUser
	dao := q.WithContext(ctx)
	if filter != nil {
		if filter.DeviceCountry != "" {
			dao = dao.Where(q.DeviceCountry.Eq(filter.DeviceCountry))
		}
		if filter.IPCountry != "" {
			dao = dao.Where(q.IPCountry.Eq(filter.IPCountry))
		}
		if filter.ChannelID != "" {
			dao = dao.Where(q.ChannelID.Eq(filter.ChannelID))
		}
		if filter.AppVersion != "" {
			dao = dao.Where(q.AppVersion.Eq(filter.AppVersion))
		}
		if filter.LoginType != "" {
			dao = dao.Where(q.LoginType.Eq(filter.LoginType))
		}
		if filter.UserType != "" {
			dao = dao.Where(q.UserType.Eq(filter.UserType))
		}
		if filter.SubscriptionStatus != "" {
			dao = dao.Where(q.SubscriptionStatus.Eq(filter.SubscriptionStatus))
		}
		if filter.Activated != nil {
			dao = dao.Where(q.Activated.Is(*filter.Activated))
		}
		if filter.Registered != nil {
			dao = dao.Where(q.Registered.Is(*filter.Registered))
		}
		if filter.PaymentMet != nil {
			dao = dao.Where(q.PaymentMet.Is(*filter.PaymentMet))
		}
	}

	db := dao.UnderlyingDB().Model(&model.VideoUser{})
	if filter != nil && filter.Keyword != "" {
		keyword := "%" + filter.Keyword + "%"
		db = db.Where(
			"username LIKE ? OR login_account LIKE ? OR phone_code LIKE ? OR email LIKE ?",
			keyword, keyword, keyword, keyword,
		)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var users []model.VideoUser
	if err := db.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&users).Error; err != nil {
		return nil, 0, err
	}
	return users, total, nil
}
