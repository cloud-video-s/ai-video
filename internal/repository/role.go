package repository

import (
	"context"

	"ai-video/internal/gen/model"

	"gorm.io/gorm"
)

type RoleRepo struct{}

func NewRoleRepo() *RoleRepo {
	return &RoleRepo{}
}

func (d *RoleRepo) Create(ctx context.Context, role *model.VideoRole) error {
	return qFrom(ctx).VideoRole.WithContext(ctx).UnderlyingDB().Create(role).Error
}

func (d *RoleRepo) GetByID(ctx context.Context, id uint64) (*model.VideoRole, error) {
	var role model.VideoRole
	q := qFrom(ctx).VideoRole
	err := q.WithContext(ctx).Where(q.ID.Eq(id)).UnderlyingDB().Preload("Menus").First(&role).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (d *RoleRepo) GetByCode(ctx context.Context, code string) (*model.VideoRole, error) {
	var role model.VideoRole
	q := qFrom(ctx).VideoRole
	err := q.WithContext(ctx).Where(q.Code.Eq(code)).UnderlyingDB().First(&role).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// Update writes only base columns; Menus is managed by SetMenus.
func (d *RoleRepo) Update(ctx context.Context, role *model.VideoRole) error {
	q := qFrom(ctx).VideoRole
	return q.WithContext(ctx).Where(q.ID.Eq(role.ID)).UnderlyingDB().
		Model(&model.VideoRole{}).
		Select("Name", "Sort", "Status", "Remark").
		Updates(role).Error
}

// Delete soft-deletes the role, clears menu associations, and mangles code.
func (d *RoleRepo) Delete(ctx context.Context, id uint64) error {
	return Transaction(ctx, func(ctx context.Context) error {
		q := qFrom(ctx).VideoRole
		if _, err := q.WithContext(ctx).Where(q.ID.Eq(id)).
			Update(q.Code, gorm.Expr("CONCAT('del#', id, '#', LEFT(code, 40))")); err != nil {
			return err
		}
		return dbFrom(ctx).Select("Menus").Delete(&model.VideoRole{ID: id}).Error
	})
}

func (d *RoleRepo) PageList(ctx context.Context, page, pageSize int, _ *QueryOptions) ([]model.VideoRole, int64, error) {
	q := qFrom(ctx).VideoRole
	dao := q.WithContext(ctx).Order(q.Sort.Asc(), q.ID.Asc())
	total, err := dao.Count()
	if err != nil {
		return nil, 0, err
	}
	var roles []model.VideoRole
	if err := dao.Offset((page - 1) * pageSize).Limit(pageSize).Scan(&roles); err != nil {
		return nil, 0, err
	}
	return roles, total, nil
}

func (d *RoleRepo) ListAll(ctx context.Context) ([]model.VideoRole, error) {
	var roles []model.VideoRole
	q := qFrom(ctx).VideoRole
	if err := q.WithContext(ctx).Where(q.Status.Eq(1)).Order(q.Sort.Asc()).Scan(&roles); err != nil {
		return nil, err
	}
	return roles, nil
}

func (d *RoleRepo) SetMenus(ctx context.Context, roleID uint64, menuIDs []uint64) error {
	role := &model.VideoRole{ID: roleID}
	var menus []model.VideoMenu
	for _, id := range menuIDs {
		menus = append(menus, model.VideoMenu{ID: id})
	}
	return dbFrom(ctx).Model(role).Association("Menus").Replace(menus)
}

func (d *RoleRepo) GetMenusByRoleID(ctx context.Context, roleID uint64) ([]model.VideoMenu, error) {
	role := &model.VideoRole{ID: roleID}
	var menus []model.VideoMenu
	if err := dbFrom(ctx).Model(role).Association("Menus").Find(&menus); err != nil {
		return nil, err
	}
	return menus, nil
}
