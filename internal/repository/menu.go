package repository

import (
	"context"

	"ai-video/internal/gen/model"
)

type MenuRepo struct{}

func NewMenuRepo() *MenuRepo {
	return &MenuRepo{}
}

func (d *MenuRepo) Create(ctx context.Context, menu *model.VideoMenu) error {
	return qFrom(ctx).VideoMenu.WithContext(ctx).UnderlyingDB().Create(menu).Error
}

func (d *MenuRepo) GetByID(ctx context.Context, id uint64) (*model.VideoMenu, error) {
	var menu model.VideoMenu
	q := qFrom(ctx).VideoMenu
	err := q.WithContext(ctx).Where(q.ID.Eq(id)).UnderlyingDB().Preload("APIs").First(&menu).Error
	if err != nil {
		return nil, err
	}
	return &menu, nil
}

// Update writes only base columns; APIs is managed by SetAPIs.
func (d *MenuRepo) Update(ctx context.Context, menu *model.VideoMenu) error {
	q := qFrom(ctx).VideoMenu
	return q.WithContext(ctx).Where(q.ID.Eq(uint64(menu.ID))).UnderlyingDB().
		Model(&model.VideoMenu{}).
		Select("ParentID", "Name", "Path", "Component", "Icon", "Sort", "Type", "Permission", "Visible", "Status").
		Updates(menu).Error
}

// Delete removes the menu and its API associations.
func (d *MenuRepo) Delete(ctx context.Context, id uint64) error {
	return dbFrom(ctx).Select("APIs").Delete(&model.VideoMenu{ID: id}).Error
}

func (d *MenuRepo) ListAll(ctx context.Context) ([]model.VideoMenu, error) {
	var menus []model.VideoMenu
	q := qFrom(ctx).VideoMenu
	if err := q.WithContext(ctx).Order(q.Sort.Asc(), q.ID.Asc()).Scan(&menus); err != nil {
		return nil, err
	}
	return menus, nil
}

func (d *MenuRepo) GetByIDs(ctx context.Context, ids []uint64) ([]model.VideoMenu, error) {
	var menus []model.VideoMenu
	if len(ids) == 0 {
		return menus, nil
	}
	converted := make([]uint64, 0, len(ids))
	for _, id := range ids {
		converted = append(converted, id)
	}
	q := qFrom(ctx).VideoMenu
	err := q.WithContext(ctx).Where(q.ID.In(converted...)).UnderlyingDB().
		Preload("APIs").
		Order("sort ASC, id ASC").
		Find(&menus).Error
	if err != nil {
		return nil, err
	}
	return menus, nil
}

func (d *MenuRepo) SetAPIs(ctx context.Context, menuID uint64, apiIDs []uint64) error {
	menu := &model.VideoMenu{ID: menuID}
	var apis []model.VideoAPI
	for _, id := range apiIDs {
		apis = append(apis, model.VideoAPI{ID: id})
	}
	return dbFrom(ctx).Model(menu).Association("APIs").Replace(apis)
}

func (d *MenuRepo) HasChildren(ctx context.Context, id uint64) (bool, error) {
	q := qFrom(ctx).VideoMenu
	total, err := q.WithContext(ctx).Where(q.ParentID.Eq(id)).Count()
	if err != nil {
		return false, err
	}
	return total > 0, nil
}

func BuildMenuTree(menus []model.VideoMenu, parentID uint64) []*model.VideoMenu {
	tree := make([]*model.VideoMenu, 0)
	for i := range menus {
		if menus[i].ParentID == parentID {
			node := &menus[i]
			//node.Children = BuildMenuTree(menus, node.ID)
			tree = append(tree, node)
		}
	}
	return tree
}
