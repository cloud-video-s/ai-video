package repository

import (
	"ai-video/internal/gen/model"
	"context"

	"gorm.io/gorm"
)

type AdminRepo struct{}

func NewAdminRepo() *AdminRepo {
	return &AdminRepo{}
}

func (d *AdminRepo) Create(ctx context.Context, user *model.VideoAdmin) error {
	return qFrom(ctx).VideoAdmin.WithContext(ctx).UnderlyingDB().Create(user).Error
}

func (d *AdminRepo) GetByID(ctx context.Context, id uint64) (*model.VideoAdmin, error) {
	var user model.VideoAdmin
	q := qFrom(ctx).VideoAdmin
	err := q.WithContext(ctx).Where(q.ID.Eq(id)).UnderlyingDB().Preload("Roles").First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (d *AdminRepo) GetByUsername(ctx context.Context, username string) (*model.VideoAdmin, error) {
	var user model.VideoAdmin
	q := qFrom(ctx).VideoAdmin
	err := q.WithContext(ctx).Where(q.Username.Eq(username)).UnderlyingDB().Preload("Roles").First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetTokenVersion returns the user's current token version (session revocation).
func (d *AdminRepo) GetTokenVersion(ctx context.Context, id uint64) (int64, error) {
	var user model.VideoAdmin
	q := qFrom(ctx).VideoAdmin
	err := q.WithContext(ctx).Select(q.TokenVersion).Where(q.ID.Eq(id)).UnderlyingDB().First(&user).Error
	if err != nil {
		return 0, err
	}
	return user.TokenVersion, nil
}

// Update writes only base columns; Roles is managed by SetRoles.
func (d *AdminRepo) Update(ctx context.Context, user *model.VideoAdmin) error {
	q := qFrom(ctx).VideoAdmin
	return q.WithContext(ctx).Where(q.ID.Eq(uint64(user.ID))).UnderlyingDB().
		Model(&model.VideoAdmin{}).
		Select("Nickname", "Email", "Phone", "Avatar", "Status", "Password", "TokenVersion").
		Updates(user).Error
}

// Delete soft-deletes the user, clears role associations, and mangles username.
func (d *AdminRepo) Delete(ctx context.Context, id uint64) error {
	return Transaction(ctx, func(ctx context.Context) error {
		q := qFrom(ctx).VideoAdmin
		if _, err := q.WithContext(ctx).Where(q.ID.Eq(id)).
			Update(q.Username, gorm.Expr("CONCAT('del#', id, '#', LEFT(username, 40))")); err != nil {
			return err
		}
		return dbFrom(ctx).Select("Roles").Delete(&model.VideoAdmin{ID: id}).Error
	})
}

func (d *AdminRepo) PageList(ctx context.Context, page, pageSize int, _ *QueryOptions) ([]model.VideoAdmin, int64, error) {
	q := qFrom(ctx).VideoAdmin
	dao := q.WithContext(ctx).Order(q.ID.Desc())
	total, err := dao.Count()
	if err != nil {
		return nil, 0, err
	}
	var users []model.VideoAdmin
	err = dao.UnderlyingDB().Preload("Roles").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&users).Error
	if err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

func (d *AdminRepo) SetRoles(ctx context.Context, userID uint64, roleIDs []uint) error {
	user := &model.VideoAdmin{ID: userID}
	var roles []model.VideoRole
	for _, id := range roleIDs {
		roles = append(roles, model.VideoRole{ID: uint64(id)})
	}
	return dbFrom(ctx).Model(user).Association("Roles").Replace(roles)
}
