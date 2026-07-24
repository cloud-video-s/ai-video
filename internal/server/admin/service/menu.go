package service

import (
	"context"
	"errors"

	"ai-video/internal/domain"
	"ai-video/internal/gen/model"
	"ai-video/internal/pkg/cache"
	"ai-video/internal/repository"

	"gorm.io/gorm"
)

type MenuService struct {
	menuRepo *repository.MenuRepo
	roleRepo *repository.RoleRepo
}

func NewMenuService() *MenuService {
	return &MenuService{menuRepo: repository.NewMenuRepo(), roleRepo: repository.NewRoleRepo()}
}

type CreateMenuRequest struct {
	ParentID   uint64   `json:"parent_id"`
	Name       string   `json:"name" binding:"required"`
	Path       string   `json:"path"`
	Component  string   `json:"component"`
	Icon       string   `json:"icon"`
	Sort       uint64   `json:"sort"`
	Type       uint32   `json:"type" binding:"oneof=0 1 2"`
	Permission string   `json:"permission"`
	Visible    *uint8   `json:"visible" binding:"omitempty,oneof=0 1"`
	Status     *uint8   `json:"status" binding:"omitempty,oneof=0 1"`
	APIIDs     []uint64 `json:"api_ids"`
}

type UpdateMenuRequest struct {
	ParentID   *uint64   `json:"parent_id"`
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	Component  string    `json:"component"`
	Icon       string    `json:"icon"`
	Sort       uint64    `json:"sort"`
	Type       uint32    `json:"type" binding:"oneof=0 1 2"`
	Permission string    `json:"permission"`
	Visible    *uint8    `json:"visible" binding:"omitempty,oneof=0 1"`
	Status     *uint8    `json:"status" binding:"omitempty,oneof=0 1"`
	APIIDs     *[]uint64 `json:"api_ids"`
}

func (s *MenuService) Create(ctx context.Context, req *CreateMenuRequest) error {
	if err := s.validateParent(ctx, 0, req.ParentID); err != nil {
		return err
	}
	visible, status := uint8(1), uint8(1)
	if req.Visible != nil {
		visible = *req.Visible
	}
	if req.Status != nil {
		status = *req.Status
	}
	menu := &model.VideoMenu{
		ParentID: req.ParentID, Name: req.Name, Path: req.Path, Component: req.Component,
		Icon: req.Icon, Sort: req.Sort, Type: uint8(req.Type), Permission: req.Permission,
		Visible: visible, Status: status,
	}
	if err := repository.Transaction(ctx, func(txCtx context.Context) error {
		if err := s.menuRepo.Create(txCtx, menu); err != nil {
			return err
		}
		return s.menuRepo.SetAPIs(txCtx, menu.ID, req.APIIDs)
	}); err != nil {
		return err
	}
	cache.ClearAllPermissionCache()
	return nil
}

func (s *MenuService) GetByID(ctx context.Context, id uint64) (*repository.MenuRecord, error) {
	item, err := s.menuRepo.GetByID(ctx, id)
	if err != nil {
		return nil, notFoundOr(err, "菜单不存在")
	}
	return item, nil
}

func (s *MenuService) Update(ctx context.Context, id uint64, req *UpdateMenuRequest) error {
	menu, err := s.menuRepo.GetByID(ctx, id)
	if err != nil {
		return notFoundOr(err, "菜单不存在")
	}
	if req.ParentID != nil {
		if err := s.validateParent(ctx, id, *req.ParentID); err != nil {
			return err
		}
		menu.ParentID = *req.ParentID
	}
	if req.Name != "" {
		menu.Name = req.Name
	}
	menu.Path = req.Path
	menu.Component = req.Component
	menu.Icon = req.Icon
	menu.Sort = req.Sort
	menu.Type = uint8(req.Type)
	menu.Permission = req.Permission
	if req.Visible != nil {
		menu.Visible = *req.Visible
	}
	if req.Status != nil {
		menu.Status = *req.Status
	}

	roleIDs, err := s.roleRepo.GetRoleIDsByMenuID(ctx, id)
	if err != nil {
		return err
	}
	if err := repository.Transaction(ctx, func(txCtx context.Context) error {
		if err := s.menuRepo.Update(txCtx, &menu.VideoMenu); err != nil {
			return err
		}
		if req.APIIDs != nil {
			return s.menuRepo.SetAPIs(txCtx, id, *req.APIIDs)
		}
		return nil
	}); err != nil {
		return err
	}
	if err := s.syncRoles(ctx, roleIDs); err != nil {
		return err
	}
	cache.ClearAllPermissionCache()
	return nil
}

func (s *MenuService) Delete(ctx context.Context, id uint64) error {
	if _, err := s.menuRepo.GetByID(ctx, id); err != nil {
		return notFoundOr(err, "菜单不存在")
	}
	has, err := s.menuRepo.HasChildren(ctx, id)
	if err != nil {
		return err
	}
	if has {
		return errors.New("存在子菜单，无法删除")
	}
	roleIDs, err := s.roleRepo.GetRoleIDsByMenuID(ctx, id)
	if err != nil {
		return err
	}
	if err := s.menuRepo.Delete(ctx, id); err != nil {
		return err
	}
	if err := s.syncRoles(ctx, roleIDs); err != nil {
		return err
	}
	cache.ClearAllPermissionCache()
	return nil
}

func (s *MenuService) GetTree(ctx context.Context) ([]*repository.MenuRecord, error) {
	menus, err := s.menuRepo.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	return repository.BuildMenuTree(menus, 0), nil
}

func (s *MenuService) GetUserMenuTree(ctx context.Context, userID uint64) ([]*repository.MenuRecord, error) {
	user, err := repository.NewAdminRepo().GetByID(ctx, userID)
	if err != nil {
		return nil, notFoundOr(err, "用户不存在")
	}
	allMenus, err := s.menuRepo.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	authorized := make(map[uint64]struct{})
	superAdmin := false
	for _, role := range user.Roles {
		if role.Status != 1 {
			continue
		}
		if role.Code == domain.SuperAdminRoleCode {
			superAdmin = true
			break
		}
		menus, err := s.roleRepo.GetMenusByRoleID(ctx, role.ID)
		if err != nil {
			return nil, err
		}
		for _, menu := range menus {
			authorized[menu.ID] = struct{}{}
		}
	}
	if superAdmin {
		for _, menu := range allMenus {
			authorized[menu.ID] = struct{}{}
		}
	}
	if len(authorized) == 0 {
		return []*repository.MenuRecord{}, nil
	}

	// 自动补齐已授权菜单的父级目录，避免历史数据只授权叶子菜单时侧栏不可达。
	menuByID := make(map[uint64]repository.MenuRecord, len(allMenus))
	for _, menu := range allMenus {
		menuByID[menu.ID] = menu
	}
	for id := range authorized {
		current := menuByID[id]
		for current.ParentID != 0 {
			parent, ok := menuByID[current.ParentID]
			if !ok {
				break
			}
			authorized[parent.ID] = struct{}{}
			current = parent
		}
	}
	visible := make([]repository.MenuRecord, 0, len(authorized))
	for _, menu := range allMenus {
		if _, ok := authorized[menu.ID]; !ok {
			continue
		}
		if menu.Type <= 1 && menu.Visible == 1 && menu.Status == 1 {
			visible = append(visible, menu)
		}
	}
	return repository.BuildMenuTree(visible, 0), nil
}

// GetUserPermissions 返回用户拥有的菜单及按钮权限标识。
func (s *MenuService) GetUserPermissions(ctx context.Context, userID uint64) ([]string, error) {
	user, err := repository.NewAdminRepo().GetByID(ctx, userID)
	if err != nil {
		return nil, notFoundOr(err, "用户不存在")
	}
	for _, role := range user.Roles {
		if role.Status == 1 && role.Code == domain.SuperAdminRoleCode {
			return []string{"*"}, nil
		}
	}
	permissions := make([]string, 0)
	seen := make(map[string]struct{})
	for _, role := range user.Roles {
		if role.Status != 1 {
			continue
		}
		menus, err := s.roleRepo.GetMenusByRoleID(ctx, role.ID)
		if err != nil {
			return nil, err
		}
		for _, menu := range menus {
			if menu.Status != 1 || menu.Permission == "" {
				continue
			}
			if _, ok := seen[menu.Permission]; ok {
				continue
			}
			seen[menu.Permission] = struct{}{}
			permissions = append(permissions, menu.Permission)
		}
	}
	return permissions, nil
}

func (s *MenuService) validateParent(ctx context.Context, menuID, parentID uint64) error {
	if parentID == 0 {
		return nil
	}
	cycle, err := s.menuRepo.WouldCreateCycle(ctx, menuID, parentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("上级菜单不存在")
		}
		return err
	}
	if cycle {
		return errors.New("上级菜单不能是当前菜单或其子菜单")
	}
	return nil
}

func (s *MenuService) syncRoles(ctx context.Context, roleIDs []uint64) error {
	roleService := NewRoleService()
	for _, roleID := range roleIDs {
		if err := roleService.syncMenuPolicies(ctx, roleID); err != nil {
			return err
		}
	}
	return nil
}
