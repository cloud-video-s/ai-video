package service

import (
	"ai-video/internal/gen/model"
	"ai-video/internal/repository"
	"context"
	"fmt"
	"strings"
)

type APIService struct {
	apiRepo  *repository.ApiRepo
	menuRepo *repository.MenuRepo
	roleRepo *repository.RoleRepo
}

func NewAPIService() *APIService {
	return &APIService{
		apiRepo: repository.NewApiRepo(), menuRepo: repository.NewMenuRepo(),
		roleRepo: repository.NewRoleRepo(),
	}
}

type CreateAPIRequest struct {
	Path        string `json:"path" binding:"required,max=255"`
	Method      string `json:"method" binding:"required,max=16"`
	Group       string `json:"group" binding:"max=64"`
	Description string `json:"description" binding:"max=255"`
}

type UpdateAPIRequest struct {
	Path        string `json:"path" binding:"required,max=255"`
	Method      string `json:"method" binding:"required,max=16"`
	Group       string `json:"group" binding:"max=64"`
	Description string `json:"description" binding:"max=255"`
}

type ListAPIRequest struct {
	Group   string `form:"group" binding:"omitempty,max=64"`
	Method  string `form:"method" binding:"omitempty,oneof=GET POST PUT PATCH DELETE OPTIONS HEAD"`
	Keyword string `form:"keyword" binding:"omitempty,max=255"`
}

func (s *APIService) Create(ctx context.Context, req *CreateAPIRequest) error {
	path, method, err := normalizeAPIIdentity(req.Path, req.Method)
	if err != nil {
		return err
	}
	exists, err := s.apiRepo.Exists(ctx, path, method, 0)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("相同路径和请求方法的 API 已存在")
	}
	api := &model.VideoAPI{
		Path: path, Method: method,
		Group: strings.TrimSpace(req.Group), Description: strings.TrimSpace(req.Description),
	}
	return s.apiRepo.Create(ctx, api)
}

func (s *APIService) GetByID(ctx context.Context, id uint) (*model.VideoAPI, error) {
	return s.apiRepo.GetByID(ctx, id)
}

func (s *APIService) Update(ctx context.Context, id uint, req *UpdateAPIRequest) error {
	api, err := s.apiRepo.GetByID(ctx, id)
	if err != nil {
		return notFoundOr(err, "API 不存在")
	}
	path, method, err := normalizeAPIIdentity(req.Path, req.Method)
	if err != nil {
		return err
	}
	api.Path = path
	api.Method = method
	api.Group = strings.TrimSpace(req.Group)
	api.Description = strings.TrimSpace(req.Description)
	exists, err := s.apiRepo.Exists(ctx, api.Path, api.Method, uint64(id))
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("相同路径和请求方法的 API 已存在")
	}
	roleIDs, err := s.affectedRoleIDs(ctx, uint64(id))
	if err != nil {
		return err
	}
	if err := s.apiRepo.Update(ctx, api); err != nil {
		return err
	}
	return s.syncRoles(ctx, roleIDs)
}

func (s *APIService) Delete(ctx context.Context, id uint) error {
	if _, err := s.apiRepo.GetByID(ctx, id); err != nil {
		return notFoundOr(err, "API 不存在")
	}
	roleIDs, err := s.affectedRoleIDs(ctx, uint64(id))
	if err != nil {
		return err
	}
	if err := s.apiRepo.Delete(ctx, id); err != nil {
		return err
	}
	return s.syncRoles(ctx, roleIDs)
}

func (s *APIService) List(ctx context.Context, page, pageSize int, req *ListAPIRequest) ([]model.VideoAPI, int64, error) {
	if req == nil {
		req = &ListAPIRequest{}
	}
	return s.apiRepo.PageList(ctx, page, pageSize, &repository.APIListFilter{
		Group: strings.TrimSpace(req.Group), Method: strings.ToUpper(strings.TrimSpace(req.Method)),
		Keyword: strings.TrimSpace(req.Keyword),
	})
}

func (s *APIService) ListAll(ctx context.Context) ([]model.VideoAPI, error) {
	return s.apiRepo.ListAll(ctx)
}

func (s *APIService) affectedRoleIDs(ctx context.Context, apiID uint64) ([]uint64, error) {
	menuIDs, err := s.menuRepo.GetMenuIDsByAPIID(ctx, apiID)
	if err != nil {
		return nil, err
	}
	seen := make(map[uint64]struct{})
	result := make([]uint64, 0)
	for _, menuID := range menuIDs {
		roleIDs, err := s.roleRepo.GetRoleIDsByMenuID(ctx, menuID)
		if err != nil {
			return nil, err
		}
		for _, roleID := range roleIDs {
			if _, ok := seen[roleID]; ok {
				continue
			}
			seen[roleID] = struct{}{}
			result = append(result, roleID)
		}
	}
	return result, nil
}

func (s *APIService) syncRoles(ctx context.Context, roleIDs []uint64) error {
	roleService := NewRoleService()
	for _, roleID := range roleIDs {
		if err := roleService.syncMenuPolicies(ctx, roleID); err != nil {
			return err
		}
	}
	return nil
}

func normalizeAPIIdentity(path, method string) (string, string, error) {
	path = strings.TrimSpace(path)
	if path == "" || !strings.HasPrefix(path, "/") {
		return "", "", fmt.Errorf("API 路径不能为空且必须以 / 开头")
	}
	method = strings.ToUpper(strings.TrimSpace(method))
	switch method {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD":
		return path, method, nil
	default:
		return "", "", fmt.Errorf("不支持的请求方法: %s", method)
	}
}
