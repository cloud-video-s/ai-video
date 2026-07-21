package repository

import (
	"context"

	"ai-video/internal/domain"
	"ai-video/internal/gen/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AppUserRepo struct{}

func NewAppUserRepo() *AppUserRepo { return &AppUserRepo{} }

type AppUserListFilter struct {
	Keyword            string
	DeviceCountry      string
	ChannelID          string
	AppVersion         string
	AppName            string
	LoginType          uint32
	UserType           uint32
	SubscriptionStatus uint32
	Activated          *uint32
	Registered         *bool
	PaymentMet         *bool
	Status             *int32
}

func (d *AppUserRepo) Create(ctx context.Context, user *model.VideoUser) error {
	return dbFrom(ctx).Create(user).Error
}

func (d *AppUserRepo) GetByID(ctx context.Context, id uint64) (*model.VideoUser, error) {
	var user model.VideoUser
	if err := dbFrom(ctx).First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (d *AppUserRepo) GetByIDForUpdate(ctx context.Context, id uint64) (*model.VideoUser, error) {
	var user model.VideoUser
	if err := dbFrom(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (d *AppUserRepo) GetByIMEI(ctx context.Context, imei string, lock bool) (*model.VideoUser, error) {
	var user model.VideoUser
	db := dbFrom(ctx).Where("imei = ?", imei)
	if lock {
		db = db.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	if err := db.Order("last_login_at DESC").First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (d *AppUserRepo) GetByProviderSubject(ctx context.Context, provider, subject string, lock bool) (*model.VideoUser, error) {
	loginType := domain.AppUserLoginGoogle
	if provider == domain.IdentityProviderApple {
		loginType = domain.AppUserLoginAppID
	}
	var user model.VideoUser
	db := dbFrom(ctx).Where("third_code = ? AND login_type = ?", subject, loginType)
	if lock {
		db = db.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	if err := db.First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (d *AppUserRepo) GetAuthState(ctx context.Context, id uint64) (string, int64, error) {
	var user model.VideoUser
	if err := dbFrom(ctx).Select("imei", "token_version").Where("id = ? AND status = 1", id).First(&user).Error; err != nil {
		return "", 0, err
	}
	return user.IMEI, user.TokenVersion, nil
}

func (d *AppUserRepo) Update(ctx context.Context, id uint64, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	return dbFrom(ctx).Model(&model.VideoUser{}).Where("id = ?", id).Updates(updates).Error
}

// IncrementTokenVersion atomically rotates the user's session version. API
// middleware compares this value with the JWT claim, so older tokens stop
// working immediately.
func (d *AppUserRepo) IncrementTokenVersion(ctx context.Context, id uint64) error {
	result := dbFrom(ctx).Model(&model.VideoUser{}).Where("id = ?", id).
		UpdateColumn("token_version", gorm.Expr("token_version + 1"))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (d *AppUserRepo) Delete(ctx context.Context, id uint64) error {
	return dbFrom(ctx).Delete(&model.VideoUser{}, id).Error
}

func (d *AppUserRepo) PageList(ctx context.Context, page, pageSize int, filter *AppUserListFilter) ([]model.VideoUser, int64, error) {
	db := dbFrom(ctx).Model(&model.VideoUser{})
	if filter != nil {
		if filter.DeviceCountry != "" {
			db = db.Where("device_country = ?", filter.DeviceCountry)
		}
		if filter.ChannelID != "" {
			db = db.Where("channel_id = ?", filter.ChannelID)
		}
		if filter.AppVersion != "" {
			db = db.Where("app_version = ?", filter.AppVersion)
		}
		if filter.AppName != "" {
			db = db.Where("app_name = ?", filter.AppName)
		}
		if filter.LoginType != 0 {
			db = db.Where("login_type = ?", filter.LoginType)
		}
		if filter.UserType != 0 {
			db = db.Where("user_type = ?", filter.UserType)
		}
		if filter.SubscriptionStatus != 0 {
			db = db.Where("subscription_status = ?", filter.SubscriptionStatus)
		}
		if filter.Activated != nil {
			db = db.Where("activated = ?", *filter.Activated)
		}
		if filter.Registered != nil {
			db = db.Where("registered = ?", *filter.Registered)
		}
		if filter.PaymentMet != nil {
			db = db.Where("payment_met = ?", *filter.PaymentMet)
		}
		if filter.Status != nil {
			db = db.Where("status = ?", *filter.Status)
		}
		if filter.Keyword != "" {
			keyword := "%" + filter.Keyword + "%"
			db = db.Where("username LIKE ? OR login_account LIKE ? OR imei LIKE ? OR email LIKE ? OR third_code LIKE ? OR app_name LIKE ?",
				keyword, keyword, keyword, keyword, keyword, keyword)
		}
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
