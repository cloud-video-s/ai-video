package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ai-video/internal/gen/model"

	"gorm.io/gorm"
)

// AdminRecord 是后台账号的接口返回对象。角色通过 video_admin_role 显式查询，
// 不依赖生成模型中的 GORM 关联；序列化时不会返回密码和令牌版本。
type AdminRecord struct {
	model.VideoAdmin
	Roles []model.VideoRole `json:"roles"`
}

func (r AdminRecord) MarshalJSON() ([]byte, error) {
	type publicAdmin struct {
		ID        uint64            `json:"id"`
		Username  string            `json:"username"`
		Nickname  string            `json:"nickname"`
		Avatar    string            `json:"avatar"`
		Email     string            `json:"email"`
		Phone     string            `json:"phone"`
		Status    uint8             `json:"status"`
		Roles     []model.VideoRole `json:"roles"`
		CreatedAt time.Time         `json:"created_at"`
		UpdatedAt time.Time         `json:"updated_at"`
	}
	return json.Marshal(publicAdmin{
		ID: r.ID, Username: r.Username, Nickname: r.Nickname, Avatar: r.Avatar,
		Email: r.Email, Phone: r.Phone, Status: r.Status, Roles: r.Roles,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	})
}

type AdminRepo struct{}

func NewAdminRepo() *AdminRepo { return &AdminRepo{} }

func (d *AdminRepo) Create(ctx context.Context, user *model.VideoAdmin) error {
	q := qFrom(ctx).VideoAdmin
	return q.WithContext(ctx).UnderlyingDB().Omit("Roles").Create(user).Error
}

func (d *AdminRepo) GetByID(ctx context.Context, id uint64) (*AdminRecord, error) {
	q := qFrom(ctx).VideoAdmin
	user, err := q.WithContext(ctx).Where(q.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}
	records, err := d.loadRecords(ctx, []model.VideoAdmin{*user})
	if err != nil {
		return nil, err
	}
	return &records[0], nil
}

func (d *AdminRepo) GetByUsername(ctx context.Context, username string) (*AdminRecord, error) {
	q := qFrom(ctx).VideoAdmin
	user, err := q.WithContext(ctx).Where(q.Username.Eq(username)).First()
	if err != nil {
		return nil, err
	}
	records, err := d.loadRecords(ctx, []model.VideoAdmin{*user})
	if err != nil {
		return nil, err
	}
	return &records[0], nil
}

// GetTokenVersion 返回账号当前会话版本，用于撤销已签发的令牌。
func (d *AdminRepo) GetTokenVersion(ctx context.Context, id uint64) (int64, error) {
	q := qFrom(ctx).VideoAdmin
	user, err := q.WithContext(ctx).Select(q.TokenVersion).Where(q.ID.Eq(id)).First()
	if err != nil {
		return 0, err
	}
	return user.TokenVersion, nil
}

// Update 只更新账号基础字段，角色由 SetRoles 单独维护。
func (d *AdminRepo) Update(ctx context.Context, user *model.VideoAdmin) error {
	q := qFrom(ctx).VideoAdmin
	_, err := q.WithContext(ctx).Where(q.ID.Eq(user.ID)).Select(
		q.Nickname, q.Email, q.Phone, q.Avatar, q.Status, q.Password, q.TokenVersion,
	).Updates(user)
	return err
}

// Delete 显式清理账号角色映射后软删除账号，并释放原用户名。
func (d *AdminRepo) Delete(ctx context.Context, id uint64) error {
	return Transaction(ctx, func(txCtx context.Context) error {
		q := qFrom(txCtx)
		relation := q.VideoAdminRole
		if _, err := relation.WithContext(txCtx).Unscoped().
			Where(relation.VideoAdminID.Eq(id)).Delete(); err != nil {
			return err
		}
		admin := q.VideoAdmin
		if _, err := admin.WithContext(txCtx).Where(admin.ID.Eq(id)).
			Update(admin.Username, gorm.Expr("CONCAT('del#', id, '#', LEFT(username, 40))")); err != nil {
			return err
		}
		_, err := admin.WithContext(txCtx).Where(admin.ID.Eq(id)).Delete()
		return err
	})
}

func (d *AdminRepo) PageList(ctx context.Context, page, pageSize int, _ *QueryOptions) ([]AdminRecord, int64, error) {
	q := qFrom(ctx).VideoAdmin
	dao := q.WithContext(ctx).Order(q.ID.Desc())
	total, err := dao.Count()
	if err != nil {
		return nil, 0, err
	}
	rows, err := dao.Offset((page - 1) * pageSize).Limit(pageSize).Find()
	if err != nil {
		return nil, 0, err
	}
	users := make([]model.VideoAdmin, 0, len(rows))
	for _, row := range rows {
		if row != nil {
			users = append(users, *row)
		}
	}
	records, err := d.loadRecords(ctx, users)
	return records, total, err
}

// SetRoles 使用 video_admin_role 显式替换账号角色，不调用 GORM Association。
func (d *AdminRepo) SetRoles(ctx context.Context, userID uint64, roleIDs []uint) error {
	ids := make([]uint64, 0, len(roleIDs))
	seen := make(map[uint64]struct{}, len(roleIDs))
	for _, value := range roleIDs {
		id := uint64(value)
		if id == 0 {
			return fmt.Errorf("角色 ID 不能为 0")
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}

	q := qFrom(ctx)
	if len(ids) > 0 {
		count, err := q.VideoRole.WithContext(ctx).Where(q.VideoRole.ID.In(ids...)).Count()
		if err != nil {
			return err
		}
		if count != int64(len(ids)) {
			return fmt.Errorf("一个或多个角色不存在")
		}
	}

	return Transaction(ctx, func(txCtx context.Context) error {
		relation := qFrom(txCtx).VideoAdminRole
		if _, err := relation.WithContext(txCtx).Unscoped().
			Where(relation.VideoAdminID.Eq(userID)).Delete(); err != nil {
			return err
		}
		rows := make([]*model.VideoAdminRole, 0, len(ids))
		for _, roleID := range ids {
			rows = append(rows, &model.VideoAdminRole{VideoAdminID: userID, VideoRoleID: roleID})
		}
		if len(rows) == 0 {
			return nil
		}
		return relation.WithContext(txCtx).Create(rows...)
	})
}

func (d *AdminRepo) loadRecords(ctx context.Context, users []model.VideoAdmin) ([]AdminRecord, error) {
	records := make([]AdminRecord, len(users))
	if len(users) == 0 {
		return records, nil
	}
	adminIDs := make([]uint64, 0, len(users))
	for i := range users {
		records[i] = AdminRecord{VideoAdmin: users[i], Roles: []model.VideoRole{}}
		adminIDs = append(adminIDs, users[i].ID)
	}

	q := qFrom(ctx)
	relation := q.VideoAdminRole
	relations, err := relation.WithContext(ctx).Where(relation.VideoAdminID.In(adminIDs...)).Find()
	if err != nil {
		return nil, err
	}
	if len(relations) == 0 {
		return records, nil
	}
	roleIDs := make([]uint64, 0, len(relations))
	membership := make(map[uint64]map[uint64]struct{}, len(users))
	for _, row := range relations {
		if row == nil {
			continue
		}
		roleIDs = append(roleIDs, row.VideoRoleID)
		if membership[row.VideoAdminID] == nil {
			membership[row.VideoAdminID] = make(map[uint64]struct{})
		}
		membership[row.VideoAdminID][row.VideoRoleID] = struct{}{}
	}
	role := q.VideoRole
	roles, err := role.WithContext(ctx).Where(role.ID.In(roleIDs...)).
		Order(role.Sort.Asc(), role.ID.Asc()).Find()
	if err != nil {
		return nil, err
	}
	for i := range records {
		for _, item := range roles {
			if item == nil {
				continue
			}
			if _, ok := membership[records[i].ID][item.ID]; ok {
				records[i].Roles = append(records[i].Roles, *item)
			}
		}
	}
	return records, nil
}
