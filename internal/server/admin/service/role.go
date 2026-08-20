package service

import (
	"ai-video/internal/config"
	"ai-video/internal/domain"
	"ai-video/internal/gen/model"
	"ai-video/internal/pkg/cache"
	"ai-video/internal/repository"
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

type RoleService struct {
	roleRepo *repository.RoleRepo
}

func NewRoleService() *RoleService {
	return &RoleService{roleRepo: repository.NewRoleRepo()}
}

type CreateRoleRequest struct {
	Name   string `json:"name" binding:"required"`
	Code   string `json:"code" binding:"required"`
	Sort   int64  `json:"sort"`
	Status *int8  `json:"status" binding:"omitempty,oneof=0 1"`
	Remark string `json:"remark"`
}

type UpdateRoleRequest struct {
	Name   *string `json:"name"`
	Sort   *int64  `json:"sort"`
	Status *int8   `json:"status" binding:"omitempty,oneof=0 1"`
	Remark *string `json:"remark"`
}

type SetRoleMenusRequest struct {
	MenuIDs *[]uint64 `json:"menu_ids" binding:"required"`
}

type SetRoleAPIsRequest struct {
	APIs *[]RoleAPIItem `json:"apis" binding:"required"`
}

type RoleAPIItem struct {
	Path   string `json:"path"`
	Method string `json:"method"`
}

func (s *RoleService) Create(ctx context.Context, req *CreateRoleRequest) error {
	name := strings.TrimSpace(req.Name)
	code := strings.TrimSpace(req.Code)
	if name == "" || code == "" {
		return errors.New("角色名称和编码不能为空")
	}
	if strings.EqualFold(code, domain.SuperAdminRoleCode) {
		return errors.New("该角色编码为系统保留，不可使用")
	}
	_, err := s.roleRepo.GetByCode(ctx, code)
	if err == nil {
		return errors.New("角色编码已存在")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	status := int8(1)
	if req.Status != nil {
		status = *req.Status
	}

	role := &model.VideoRole{
		Name:   name,
		Code:   code,
		Sort:   req.Sort,
		Status: uint8(status),
		Remark: strings.TrimSpace(req.Remark),
	}
	if err := s.roleRepo.Create(ctx, role); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return errors.New("角色编码已存在")
		}
		return err
	}
	return nil
}

func (s *RoleService) GetByID(ctx context.Context, id uint64) (*repository.RoleRecord, error) {
	return s.roleRepo.GetByID(ctx, id)
}

func (s *RoleService) Update(ctx context.Context, id uint64, req *UpdateRoleRequest) error {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		return notFoundOr(err, "角色不存在")
	}

	if err := ensureEditableRole(&role.VideoRole); err != nil {
		return err
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return errors.New("角色名称不能为空")
		}
		role.Name = name
	}
	if req.Sort != nil {
		role.Sort = *req.Sort
	}
	if req.Status != nil {
		role.Status = uint8(*req.Status)
	}
	if req.Remark != nil {
		role.Remark = strings.TrimSpace(*req.Remark)
	}
	if err := repository.Transaction(ctx, func(txCtx context.Context) error {
		if err := s.roleRepo.Update(txCtx, &role.VideoRole); err != nil {
			return err
		}
		return s.persistMenuPolicies(txCtx, id)
	}); err != nil {
		return err
	}
	return s.reloadPolicies()
}

func (s *RoleService) Delete(ctx context.Context, id uint64) error {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		return notFoundOr(err, "角色不存在")
	}
	if err := ensureEditableRole(&role.VideoRole); err != nil {
		return err
	}
	if err := repository.Transaction(ctx, func(txCtx context.Context) error {
		if err := s.roleRepo.Delete(txCtx, id); err != nil {
			return err
		}
		return s.roleRepo.ReplacePolicies(txCtx, role.Code, nil)
	}); err != nil {
		return err
	}
	return s.reloadPolicies()
}

func (s *RoleService) List(ctx context.Context, page, pageSize int, req *ListSortRequest) ([]model.VideoRole, int64, error) {
	return s.roleRepo.PageList(ctx, page, pageSize, &repository.QueryOptions{
		ListSort: req.listSort(),
	})
}

func (s *RoleService) ListAll(ctx context.Context) ([]model.VideoRole, error) {
	return s.roleRepo.ListAll(ctx)
}

// SetMenus assigns menus to a role and rebuilds the role's Casbin policies from
// the menus' associated APIs in the same database transaction.
func (s *RoleService) SetMenus(ctx context.Context, roleID uint64, menuIDs []uint64) error {
	if err := repository.Transaction(ctx, func(txCtx context.Context) error {
		role, err := s.roleRepo.GetByID(txCtx, roleID)
		if err != nil {
			return notFoundOr(err, "角色不存在")
		}
		if err := ensureEditableRole(&role.VideoRole); err != nil {
			return err
		}
		if err := s.roleRepo.SetMenus(txCtx, roleID, menuIDs); err != nil {
			return err
		}
		return s.persistMenuPolicies(txCtx, roleID)
	}); err != nil {
		return err
	}
	return s.reloadPolicies()
}

// persistMenuPolicies materializes the APIs associated with a role's menus.
// The caller may include it in the same transaction as a menu/role mutation.
func (s *RoleService) persistMenuPolicies(ctx context.Context, roleID uint64) error {
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return err
	}
	policies := make([]repository.RolePolicy, 0)
	if role.Status == 1 && len(role.Menus) > 0 {
		seen := make(map[string]struct{})
		for _, menu := range role.Menus {
			if menu.Status != 1 {
				continue
			}
			for _, api := range menu.APIs {
				path := strings.TrimSpace(api.Path)
				method := strings.ToUpper(strings.TrimSpace(api.Method))
				if path == "" || method == "" {
					continue
				}
				key := method + " " + path
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}
				policies = append(policies, repository.RolePolicy{Path: path, Method: method})
			}
		}
	}
	return s.roleRepo.ReplacePolicies(ctx, role.Code, policies)
}

func (s *RoleService) SetAPIs(ctx context.Context, roleID uint64, apis []RoleAPIItem) error {
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return notFoundOr(err, "角色不存在")
	}
	if err := ensureEditableRole(&role.VideoRole); err != nil {
		return err
	}
	policies := make([]repository.RolePolicy, 0, len(apis))
	seen := make(map[string]struct{}, len(apis))
	for _, api := range apis {
		path := strings.TrimSpace(api.Path)
		method := strings.ToUpper(strings.TrimSpace(api.Method))
		if path == "" || method == "" {
			continue
		}
		key := method + " " + path
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		policies = append(policies, repository.RolePolicy{Path: path, Method: method})
	}
	if err := s.roleRepo.ReplacePolicies(ctx, role.Code, policies); err != nil {
		return fmt.Errorf("保存权限策略失败: %w", err)
	}
	return s.reloadPolicies()
}

func (s *RoleService) GetAPIs(ctx context.Context, roleID uint64) ([]RoleAPIItem, error) {
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return nil, notFoundOr(err, "角色不存在")
	}

	if config.Enforcer == nil {
		return nil, errors.New("权限服务未初始化")
	}
	policies := config.Enforcer.GetFilteredPolicy(0, role.Code)
	items := make([]RoleAPIItem, 0, len(policies))
	for _, p := range policies {
		if len(p) >= 3 {
			items = append(items, RoleAPIItem{Path: p[1], Method: p[2]})
		}
	}
	return items, nil
}

func ensureEditableRole(role *model.VideoRole) error {
	if role != nil && role.Code == domain.SuperAdminRoleCode {
		return errors.New("系统内置超级管理员角色不可编辑或删除")
	}
	return nil
}

func (s *RoleService) reloadPolicies() error {
	if config.Enforcer == nil {
		return errors.New("权限服务未初始化")
	}
	if err := config.Enforcer.LoadPolicy(); err != nil {
		return fmt.Errorf("刷新权限策略失败: %w", err)
	}
	cache.ClearAllPermissionCache()
	return nil
}
