package repository

import (
	"context"
	"fmt"

	"ai-video/internal/gen/model"
)

// MenuRecord 是菜单接口对象。APIs 和 Children 都由普通查询显式装配。
type MenuRecord struct {
	model.VideoMenu
	APIs     []model.VideoAPI `json:"apis"`
	Children []*MenuRecord    `json:"children"`
}

type MenuRepo struct{}

func NewMenuRepo() *MenuRepo { return &MenuRepo{} }

func (d *MenuRepo) Create(ctx context.Context, menu *model.VideoMenu) error {
	q := qFrom(ctx).VideoMenu
	return q.WithContext(ctx).UnderlyingDB().Omit("ParentMenu", "ChildMenus", "APIs").Create(menu).Error
}

func (d *MenuRepo) GetByID(ctx context.Context, id uint64) (*MenuRecord, error) {
	q := qFrom(ctx).VideoMenu
	menu, err := q.WithContext(ctx).Where(q.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}
	records, err := d.loadRecords(ctx, []model.VideoMenu{*menu})
	if err != nil {
		return nil, err
	}
	return &records[0], nil
}

// Update 只更新菜单基础字段，API 映射由 SetAPIs 单独维护。
func (d *MenuRepo) Update(ctx context.Context, menu *model.VideoMenu) error {
	q := qFrom(ctx).VideoMenu
	_, err := q.WithContext(ctx).Where(q.ID.Eq(menu.ID)).Select(
		q.ParentID, q.Name, q.Path, q.Component, q.Icon, q.Sort,
		q.Type, q.Permission, q.Visible, q.Status,
	).Updates(menu)
	return err
}

// Delete 显式清理角色菜单、菜单 API 映射后软删除菜单。
func (d *MenuRepo) Delete(ctx context.Context, id uint64) error {
	return Transaction(ctx, func(txCtx context.Context) error {
		q := qFrom(txCtx)
		menuAPI := q.VideoMenuAPI
		if _, err := menuAPI.WithContext(txCtx).Unscoped().
			Where(menuAPI.VideoMenuID.Eq(id)).Delete(); err != nil {
			return err
		}
		roleMenu := q.VideoRoleMenu
		if _, err := roleMenu.WithContext(txCtx).Unscoped().
			Where(roleMenu.VideoMenuID.Eq(id)).Delete(); err != nil {
			return err
		}
		menu := q.VideoMenu
		_, err := menu.WithContext(txCtx).Where(menu.ID.Eq(id)).Delete()
		return err
	})
}

func (d *MenuRepo) ListAll(ctx context.Context) ([]MenuRecord, error) {
	q := qFrom(ctx).VideoMenu
	rows, err := q.WithContext(ctx).Order(q.Sort.Asc(), q.ID.Asc()).Find()
	if err != nil {
		return nil, err
	}
	menus := make([]model.VideoMenu, 0, len(rows))
	for _, row := range rows {
		if row != nil {
			menus = append(menus, *row)
		}
	}
	return d.loadRecords(ctx, menus)
}

func (d *MenuRepo) GetByIDs(ctx context.Context, ids []uint64) ([]MenuRecord, error) {
	if len(ids) == 0 {
		return []MenuRecord{}, nil
	}
	q := qFrom(ctx).VideoMenu
	rows, err := q.WithContext(ctx).Where(q.ID.In(ids...)).Order(q.Sort.Asc(), q.ID.Asc()).Find()
	if err != nil {
		return nil, err
	}
	menus := make([]model.VideoMenu, 0, len(rows))
	for _, row := range rows {
		if row != nil {
			menus = append(menus, *row)
		}
	}
	return d.loadRecords(ctx, menus)
}

// SetAPIs 使用 video_menu_api 显式替换菜单关联的 API。
func (d *MenuRepo) SetAPIs(ctx context.Context, menuID uint64, apiIDs []uint64) error {
	ids := uniqueUint64(apiIDs)
	for _, id := range ids {
		if id == 0 {
			return fmt.Errorf("API ID 不能为 0")
		}
	}
	q := qFrom(ctx)
	if len(ids) > 0 {
		count, err := q.VideoAPI.WithContext(ctx).Where(q.VideoAPI.ID.In(ids...)).Count()
		if err != nil {
			return err
		}
		if count != int64(len(ids)) {
			return fmt.Errorf("一个或多个 API 不存在")
		}
	}
	return Transaction(ctx, func(txCtx context.Context) error {
		relation := qFrom(txCtx).VideoMenuAPI
		if _, err := relation.WithContext(txCtx).Unscoped().
			Where(relation.VideoMenuID.Eq(menuID)).Delete(); err != nil {
			return err
		}
		rows := make([]*model.VideoMenuAPI, 0, len(ids))
		for _, apiID := range ids {
			rows = append(rows, &model.VideoMenuAPI{VideoMenuID: menuID, VideoAPIID: apiID})
		}
		if len(rows) == 0 {
			return nil
		}
		return relation.WithContext(txCtx).Create(rows...)
	})
}

func (d *MenuRepo) HasChildren(ctx context.Context, id uint64) (bool, error) {
	q := qFrom(ctx).VideoMenu
	total, err := q.WithContext(ctx).Where(q.ParentID.Eq(id)).Count()
	return total > 0, err
}

// WouldCreateCycle 判断把 menuID 移到 parentID 下是否会形成循环层级。
func (d *MenuRepo) WouldCreateCycle(ctx context.Context, menuID, parentID uint64) (bool, error) {
	if parentID == 0 {
		return false, nil
	}
	if menuID != 0 && menuID == parentID {
		return true, nil
	}
	seen := make(map[uint64]struct{})
	current := parentID
	q := qFrom(ctx).VideoMenu
	for current != 0 {
		if menuID != 0 && current == menuID {
			return true, nil
		}
		if _, ok := seen[current]; ok {
			return true, nil
		}
		seen[current] = struct{}{}
		item, err := q.WithContext(ctx).Select(q.ParentID).Where(q.ID.Eq(current)).First()
		if err != nil {
			return false, err
		}
		current = item.ParentID
	}
	return false, nil
}

func (d *MenuRepo) GetMenuIDsByAPIID(ctx context.Context, apiID uint64) ([]uint64, error) {
	relation := qFrom(ctx).VideoMenuAPI
	var ids []uint64
	err := relation.WithContext(ctx).Where(relation.VideoAPIID.Eq(apiID)).
		Pluck(relation.VideoMenuID, &ids)
	return uniqueUint64(ids), err
}

func (d *MenuRepo) loadRecords(ctx context.Context, menus []model.VideoMenu) ([]MenuRecord, error) {
	records := make([]MenuRecord, len(menus))
	if len(menus) == 0 {
		return records, nil
	}
	menuIDs := make([]uint64, 0, len(menus))
	for i := range menus {
		records[i] = MenuRecord{
			VideoMenu: menus[i], APIs: []model.VideoAPI{}, Children: []*MenuRecord{},
		}
		menuIDs = append(menuIDs, menus[i].ID)
	}

	q := qFrom(ctx)
	relation := q.VideoMenuAPI
	relations, err := relation.WithContext(ctx).Where(relation.VideoMenuID.In(menuIDs...)).Find()
	if err != nil {
		return nil, err
	}
	if len(relations) == 0 {
		return records, nil
	}
	apiIDs := make([]uint64, 0, len(relations))
	membership := make(map[uint64]map[uint64]struct{}, len(menus))
	for _, row := range relations {
		if row == nil {
			continue
		}
		apiIDs = append(apiIDs, row.VideoAPIID)
		if membership[row.VideoMenuID] == nil {
			membership[row.VideoMenuID] = make(map[uint64]struct{})
		}
		membership[row.VideoMenuID][row.VideoAPIID] = struct{}{}
	}
	api := q.VideoAPI
	apis, err := api.WithContext(ctx).Where(api.ID.In(apiIDs...)).
		Order(api.Group.Asc(), api.ID.Asc()).Find()
	if err != nil {
		return nil, err
	}
	for i := range records {
		for _, item := range apis {
			if item == nil {
				continue
			}
			if _, ok := membership[records[i].ID][item.ID]; ok {
				records[i].APIs = append(records[i].APIs, *item)
			}
		}
	}
	return records, nil
}

func BuildMenuTree(menus []MenuRecord, parentID uint64) []*MenuRecord {
	nodes := make(map[uint64]*MenuRecord, len(menus))
	order := make([]uint64, 0, len(menus))
	for i := range menus {
		node := menus[i]
		node.Children = []*MenuRecord{}
		nodes[node.ID] = &node
		order = append(order, node.ID)
	}
	tree := make([]*MenuRecord, 0)
	for _, id := range order {
		node := nodes[id]
		if node.ParentID == parentID {
			tree = append(tree, node)
			continue
		}
		parent := nodes[node.ParentID]
		if parent != nil && parent.ID != node.ID {
			parent.Children = append(parent.Children, node)
			continue
		}
		// 权限数据缺少父菜单时仍展示为根节点，避免页面完全不可达。
		if parentID == 0 {
			tree = append(tree, node)
		}
	}
	return tree
}

func uniqueUint64(values []uint64) []uint64 {
	result := make([]uint64, 0, len(values))
	seen := make(map[uint64]struct{}, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
