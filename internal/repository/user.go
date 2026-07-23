package repository

import (
	"context"
	"strconv"
	"strings"

	"ai-video/internal/gen/model"

	"gorm.io/gen/field"
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
	IsFrozen           *bool
	IsBlacklisted      *bool
}

func (d *AppUserRepo) Create(ctx context.Context, user *model.VideoUser) error {
	return qFrom(ctx).VideoUser.WithContext(ctx).Create(user)
}

func (d *AppUserRepo) GetByID(ctx context.Context, id uint64) (*model.VideoUser, error) {
	q := qFrom(ctx).VideoUser
	return q.WithContext(ctx).Where(q.ID.Eq(id)).First()
}

// GetByLookup resolves an exact user ID, account/email or an email stored in
// the normalized identity table. It keeps legacy direct-email rows searchable.
func (d *AppUserRepo) GetByLookup(ctx context.Context, value string) (*model.VideoUser, error) {
	value = strings.TrimSpace(value)
	q := qFrom(ctx)
	user := q.VideoUser
	if id, err := strconv.ParseUint(value, 10, 64); err == nil && id != 0 {
		return user.WithContext(ctx).Where(user.ID.Eq(id)).First()
	}
	conditions := []field.Expr{user.LoginAccount.Eq(value), user.Email.Eq(value)}
	identity := q.VideoUserIdentity
	var identityUserIDs []uint64
	if err := identity.WithContext(ctx).Where(identity.Email.Eq(value)).Pluck(identity.UserID, &identityUserIDs); err != nil {
		return nil, err
	}
	if len(identityUserIDs) > 0 {
		conditions = append(conditions, user.ID.In(identityUserIDs...))
	}
	return user.WithContext(ctx).Where(field.Or(conditions...)).First()
}

func (d *AppUserRepo) GetByIDForUpdate(ctx context.Context, id uint64) (*model.VideoUser, error) {
	q := qFrom(ctx).VideoUser
	return q.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where(q.ID.Eq(id)).First()
}

func (d *AppUserRepo) GetByDeviceCode(ctx context.Context, deviceCode string, lock bool) (*model.VideoUser, error) {
	q := qFrom(ctx).VideoUser
	dao := q.WithContext(ctx).Where(q.DeviceCode.Eq(deviceCode))
	if lock {
		dao = dao.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	return dao.Order(q.LastLoginAt.Desc()).First()
}

func (d *AppUserRepo) GetByThirdCode(ctx context.Context, thirdCode string, lock bool) (*model.VideoUser, error) {
	q := qFrom(ctx).VideoUser
	dao := q.WithContext(ctx).Where(q.ThirdCode.Eq(thirdCode))
	if lock {
		dao = dao.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	return dao.First()
}

func (d *AppUserRepo) GetAuthState(ctx context.Context, id uint64) (string, int64, error) {
	q := qFrom(ctx).VideoUser
	user, err := q.WithContext(ctx).Where(q.ID.Eq(id)).
		Where(q.Status.Eq(1)).
		Where(q.IsFrozen.Eq(false)).
		Where(q.IsBlacklisted.Eq(false)).
		Select(q.DeviceCode, q.TokenVersion).First()
	if err != nil {
		return "", 0, err
	}
	return user.DeviceCode, user.TokenVersion, nil
}

func (d *AppUserRepo) Update(ctx context.Context, id uint64, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	q := qFrom(ctx).VideoUser
	_, err := q.WithContext(ctx).Where(q.ID.Eq(id)).Updates(updates)
	return err
}

// IncrementTokenVersion atomically rotates the user's session version. API
// middleware compares this value with the JWT claim, so older tokens stop
// working immediately.
func (d *AppUserRepo) IncrementTokenVersion(ctx context.Context, id uint64) error {
	q := qFrom(ctx).VideoUser
	result, err := q.WithContext(ctx).Where(q.ID.Eq(id)).UpdateColumn(q.TokenVersion, q.TokenVersion.Add(1))
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (d *AppUserRepo) Delete(ctx context.Context, id uint64) error {
	q := qFrom(ctx).VideoUser
	_, err := q.WithContext(ctx).Where(q.ID.Eq(id)).Delete()
	return err
}

func (d *AppUserRepo) PageList(ctx context.Context, page, pageSize int, filter *AppUserListFilter) ([]model.VideoUser, int64, error) {
	q := qFrom(ctx)
	user := q.VideoUser
	dao := user.WithContext(ctx)
	if filter != nil {
		if filter.DeviceCountry != "" {
			dao = dao.Where(user.ClientCountry.Eq(filter.DeviceCountry))
		}
		if filter.ChannelID != "" {
			dao = dao.Where(user.ChannelID.Eq(filter.ChannelID))
		}
		if filter.AppVersion != "" {
			dao = dao.Where(user.AppVersion.Eq(filter.AppVersion))
		}
		if filter.AppName != "" {
			dao = dao.Where(user.AppName.Eq(filter.AppName))
		}
		if filter.LoginType != 0 {
			dao = dao.Where(user.LoginType.Eq(uint8(filter.LoginType)))
		}
		if filter.UserType != 0 {
			dao = dao.Where(user.UserType.Eq(uint8(filter.UserType)))
		}
		if filter.SubscriptionStatus != 0 {
			dao = dao.Where(user.SubscriptionStatus.Eq(uint8(filter.SubscriptionStatus)))
		}
		if filter.Activated != nil {
			dao = dao.Where(user.Activated.Eq(*filter.Activated))
		}
		if filter.Registered != nil {
			dao = dao.Where(user.Registered.Eq(*filter.Registered))
		}
		if filter.PaymentMet != nil {
			dao = dao.Where(user.PaymentMet.Eq(*filter.PaymentMet))
		}
		if filter.Status != nil {
			dao = dao.Where(user.Status.Eq(int8(*filter.Status)))
		}
		if filter.IsFrozen != nil {
			dao = dao.Where(user.IsFrozen.Eq(*filter.IsFrozen))
		}
		if filter.IsBlacklisted != nil {
			dao = dao.Where(user.IsBlacklisted.Eq(*filter.IsBlacklisted))
		}
		if filter.Keyword != "" {
			keyword := "%" + filter.Keyword + "%"
			conditions := []field.Expr{
				user.Username.Like(keyword), user.LoginAccount.Like(keyword), user.IMEI.Like(keyword),
				user.Email.Like(keyword), user.Phone.Like(keyword), user.ThirdCode.Like(keyword), user.AppName.Like(keyword),
			}
			if id, err := strconv.ParseUint(strings.TrimSpace(filter.Keyword), 10, 64); err == nil {
				conditions = append(conditions, user.ID.Eq(id))
			}
			identity := q.VideoUserIdentity
			var identityUserIDs []uint64
			if err := identity.WithContext(ctx).Where(identity.Email.Like(keyword)).Pluck(identity.UserID, &identityUserIDs); err != nil {
				return nil, 0, err
			}
			if len(identityUserIDs) > 0 {
				conditions = append(conditions, user.ID.In(identityUserIDs...))
			}
			dao = dao.Where(field.Or(conditions...))
		}
	}

	total, err := dao.Count()
	if err != nil {
		return nil, 0, err
	}
	rows, err := dao.Order(user.ID.Desc()).Offset((page - 1) * pageSize).Limit(pageSize).Find()
	return valuesOf(rows), total, err
}
