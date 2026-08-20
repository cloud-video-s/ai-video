package repository

import (
	"context"
	"fmt"
	"strings"

	"ai-video/internal/gen/model"

	"gorm.io/gen/field"
	"gorm.io/gorm"
)

// RoleRecord 是角色详情对象，菜单通过 video_role_menu 显式查询。
type RoleRecord struct {
	model.VideoRole
	Menus []MenuRecord `json:"menus"`
}

type RoleRepo struct{}

func NewRoleRepo() *RoleRepo { return &RoleRepo{} }

// RolePolicy is the persisted Casbin policy derived from a role's menus.
type RolePolicy struct {
	Path   string
	Method string
}

func (d *RoleRepo) Create(ctx context.Context, role *model.VideoRole) error {
	q := qFrom(ctx).VideoRole
	return q.WithContext(ctx).UnderlyingDB().Omit("Menus").Create(role).Error
}

func (d *RoleRepo) GetByID(ctx context.Context, id uint64) (*RoleRecord, error) {
	q := qFrom(ctx).VideoRole
	role, err := q.WithContext(ctx).Where(q.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}
	menus, err := d.GetMenusByRoleID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &RoleRecord{VideoRole: *role, Menus: menus}, nil
}

func (d *RoleRepo) GetByCode(ctx context.Context, code string) (*model.VideoRole, error) {
	q := qFrom(ctx).VideoRole
	return q.WithContext(ctx).Where(q.Code.Eq(code)).First()
}

// Update 只更新角色基础字段，菜单由 SetMenus 单独维护。
func (d *RoleRepo) Update(ctx context.Context, role *model.VideoRole) error {
	q := qFrom(ctx).VideoRole
	_, err := q.WithContext(ctx).Where(q.ID.Eq(role.ID)).Select(
		q.Name, q.Sort, q.Status, q.Remark,
	).Updates(role)
	return err
}

// Delete 清理账号角色、角色菜单映射后软删除角色，并释放原角色编码。
func (d *RoleRepo) Delete(ctx context.Context, id uint64) error {
	return Transaction(ctx, func(txCtx context.Context) error {
		q := qFrom(txCtx)
		adminRole := q.VideoAdminRole
		var adminIDs []uint64
		if err := adminRole.WithContext(txCtx).Where(adminRole.VideoRoleID.Eq(id)).
			Pluck(adminRole.VideoAdminID, &adminIDs); err != nil {
			return err
		}
		adminIDs = uniqueUint64(adminIDs)
		if len(adminIDs) > 0 {
			admin := q.VideoAdmin
			if _, err := admin.WithContext(txCtx).Where(admin.ID.In(adminIDs...)).
				Update(admin.TokenVersion, gorm.Expr("token_version + 1")); err != nil {
				return err
			}
		}
		if _, err := adminRole.WithContext(txCtx).Unscoped().
			Where(adminRole.VideoRoleID.Eq(id)).Delete(); err != nil {
			return err
		}
		roleMenu := q.VideoRoleMenu
		if _, err := roleMenu.WithContext(txCtx).Unscoped().
			Where(roleMenu.VideoRoleID.Eq(id)).Delete(); err != nil {
			return err
		}
		role := q.VideoRole
		stored, err := role.WithContext(txCtx).Select(role.Code).Where(role.ID.Eq(id)).First()
		if err != nil {
			return err
		}
		codeRunes := []rune(stored.Code)
		if len(codeRunes) > 40 {
			codeRunes = codeRunes[:40]
		}
		deletedCode := fmt.Sprintf("del#%d#%s", id, string(codeRunes))
		if _, err := role.WithContext(txCtx).Where(role.ID.Eq(id)).
			Update(role.Code, deletedCode); err != nil {
			return err
		}
		_, err = role.WithContext(txCtx).Where(role.ID.Eq(id)).Delete()
		return err
	})
}

func (d *RoleRepo) PageList(ctx context.Context, page, pageSize int, opts *QueryOptions) ([]model.VideoRole, int64, error) {
	q := qFrom(ctx).VideoRole
	listSort := ListSort{}
	if opts != nil {
		listSort = opts.ListSort
	}
	order := orderForList(listSort, map[string]field.OrderExpr{"id": q.ID, "sort": q.Sort}, q.ID, q.Sort.Asc(), q.ID.Asc())
	dao := q.WithContext(ctx).Order(order...)
	total, err := dao.Count()
	if err != nil {
		return nil, 0, err
	}
	rows, err := dao.Offset((page - 1) * pageSize).Limit(pageSize).Find()
	if err != nil {
		return nil, 0, err
	}
	roles := make([]model.VideoRole, 0, len(rows))
	for _, row := range rows {
		if row != nil {
			roles = append(roles, *row)
		}
	}
	return roles, total, nil
}

func (d *RoleRepo) ListAll(ctx context.Context) ([]model.VideoRole, error) {
	q := qFrom(ctx).VideoRole
	rows, err := q.WithContext(ctx).Where(q.Status.Eq(1)).Order(q.Sort.Asc(), q.ID.Asc()).Find()
	if err != nil {
		return nil, err
	}
	roles := make([]model.VideoRole, 0, len(rows))
	for _, row := range rows {
		if row != nil {
			roles = append(roles, *row)
		}
	}
	return roles, nil
}

// SetMenus 使用 video_role_menu 显式替换角色菜单，不调用 GORM Association。
func (d *RoleRepo) SetMenus(ctx context.Context, roleID uint64, menuIDs []uint64) error {
	ids := uniqueUint64(menuIDs)
	for _, id := range ids {
		if id == 0 {
			return fmt.Errorf("菜单 ID 不能为 0")
		}
	}
	q := qFrom(ctx)
	if len(ids) > 0 {
		count, err := q.VideoMenu.WithContext(ctx).Where(q.VideoMenu.ID.In(ids...)).Count()
		if err != nil {
			return err
		}
		if count != int64(len(ids)) {
			return fmt.Errorf("一个或多个菜单不存在")
		}
	}
	return Transaction(ctx, func(txCtx context.Context) error {
		relation := qFrom(txCtx).VideoRoleMenu
		if _, err := relation.WithContext(txCtx).Unscoped().
			Where(relation.VideoRoleID.Eq(roleID)).Delete(); err != nil {
			return err
		}
		rows := make([]*model.VideoRoleMenu, 0, len(ids))
		for _, menuID := range ids {
			rows = append(rows, &model.VideoRoleMenu{VideoRoleID: roleID, VideoMenuID: menuID})
		}
		if len(rows) == 0 {
			return nil
		}
		return relation.WithContext(txCtx).Create(rows...)
	})
}

func (d *RoleRepo) GetMenusByRoleID(ctx context.Context, roleID uint64) ([]MenuRecord, error) {
	relation := qFrom(ctx).VideoRoleMenu
	var menuIDs []uint64
	if err := relation.WithContext(ctx).Where(relation.VideoRoleID.Eq(roleID)).
		Pluck(relation.VideoMenuID, &menuIDs); err != nil {
		return nil, err
	}
	return NewMenuRepo().GetByIDs(ctx, uniqueUint64(menuIDs))
}

func (d *RoleRepo) GetRoleIDsByMenuID(ctx context.Context, menuID uint64) ([]uint64, error) {
	relation := qFrom(ctx).VideoRoleMenu
	var roleIDs []uint64
	err := relation.WithContext(ctx).Where(relation.VideoMenuID.Eq(menuID)).
		Pluck(relation.VideoRoleID, &roleIDs)
	return uniqueUint64(roleIDs), err
}

func (d *RoleRepo) GetAdminIDsByRoleID(ctx context.Context, roleID uint64) ([]uint64, error) {
	relation := qFrom(ctx).VideoAdminRole
	var adminIDs []uint64
	err := relation.WithContext(ctx).Where(relation.VideoRoleID.Eq(roleID)).
		Pluck(relation.VideoAdminID, &adminIDs)
	return uniqueUint64(adminIDs), err
}

// ReplacePolicies atomically replaces only the p-policies owned by roleCode.
// Menu mappings remain the source of truth; casbin_rule is their materialized
// representation for request-time authorization.
func (d *RoleRepo) ReplacePolicies(ctx context.Context, roleCode string, policies []RolePolicy) error {
	roleCode = strings.TrimSpace(roleCode)
	if roleCode == "" {
		return fmt.Errorf("角色编码不能为空")
	}
	return Transaction(ctx, func(txCtx context.Context) error {
		db := dbFrom(txCtx)
		if err := db.Where("ptype = ? AND v0 = ?", "p", roleCode).
			Delete(&model.CasbinRule{}).Error; err != nil {
			return err
		}
		rows := make([]model.CasbinRule, 0, len(policies))
		seen := make(map[string]struct{}, len(policies))
		for _, policy := range policies {
			path := strings.TrimSpace(policy.Path)
			method := strings.ToUpper(strings.TrimSpace(policy.Method))
			if path == "" || method == "" {
				continue
			}
			key := method + " " + path
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			rows = append(rows, model.CasbinRule{Ptype: "p", V0: roleCode, V1: path, V2: method})
		}
		if len(rows) == 0 {
			return nil
		}
		return db.Create(&rows).Error
	})
}
