package service

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"ai-video/internal/gen/model"
	"ai-video/internal/pkg/i18n"
	"ai-video/internal/repository"

	"gorm.io/gorm"
)

var packageCodePattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

type PackageService struct {
	repo *repository.PackageRepo
}

func NewPackageService() *PackageService {
	return &PackageService{repo: repository.NewPackageRepo()}
}

type ListPackageRequest struct {
	PackageCode    string `form:"package_code"`
	PackageVersion string `form:"package_version"`
	SystemType     string `form:"system_type"`
	Status         *int8  `form:"status" binding:"omitempty,oneof=0 1"`
	Keyword        string `form:"keyword"`
}

type PackagePayload struct {
	PackageName    string   `json:"package_name" binding:"required,max=128"`
	PackageCode    string   `json:"package_code" binding:"required,max=128"`
	PackageVersion string   `json:"package_version" binding:"required,max=64"`
	Language       string   `json:"language" binding:"omitempty,oneof=zh-CN en-US ja-JP ko-KR es-ES"`
	SystemTypes    []string `json:"system_types" binding:"required,min=1,max=10,dive,required,max=32"`
	DownloadURL    string   `json:"download_url" binding:"required,max=1024"`
	InstallCount   uint64   `json:"install_count"`
	DownloadCount  uint64   `json:"download_count"`
	DeviceCount    uint64   `json:"device_count"`
	Description    string   `json:"description" binding:"max=2000"`
	Sort           int      `json:"sort"`
	Status         int8     `json:"status" binding:"oneof=0 1"`
}

func (s *PackageService) List(ctx context.Context, page, pageSize int, req *ListPackageRequest) ([]model.VideoPackage, int64, error) {
	return s.repo.PageList(ctx, page, pageSize, &repository.PackageListFilter{
		PackageCode: strings.TrimSpace(req.PackageCode), PackageVersion: strings.TrimSpace(req.PackageVersion),
		SystemType: strings.ToLower(strings.TrimSpace(req.SystemType)), Status: req.Status, Keyword: strings.TrimSpace(req.Keyword),
	})
}

func (s *PackageService) GetByID(ctx context.Context, id uint64) (*model.VideoPackage, error) {
	item, err := s.repo.GetByID(ctx, uint(id))
	if err != nil {
		return nil, notFoundOr(err, "包不存在")
	}
	return item, nil
}

func (s *PackageService) ListOptions(ctx context.Context) ([]model.VideoPackage, error) {
	return s.repo.ListOptions(ctx)
}

func (s *PackageService) Create(ctx context.Context, req *PackagePayload) (*model.VideoPackage, error) {
	if err := s.validatePayload(ctx, req, 0); err != nil {
		return nil, err
	}
	item := &model.VideoPackage{}
	applyPackagePayload(item, req)
	if err := s.repo.Create(ctx, item); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, errors.New("相同标识码和版本的包已存在")
		}
		return nil, err
	}
	return item, nil
}

func (s *PackageService) Update(ctx context.Context, id uint64, req *PackagePayload) (*model.VideoPackage, error) {
	item, err := s.repo.GetByID(ctx, uint(id))
	if err != nil {
		return nil, notFoundOr(err, "包不存在")
	}
	if err := s.validatePayload(ctx, req, id); err != nil {
		return nil, err
	}
	applyPackagePayload(item, req)
	if err := s.repo.UpdateFields(ctx, item); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, errors.New("相同标识码和版本的包已存在")
		}
		return nil, err
	}
	return item, nil
}

func (s *PackageService) Delete(ctx context.Context, id uint64) error {
	if _, err := s.repo.GetByID(ctx, uint(id)); err != nil {
		return notFoundOr(err, "包不存在")
	}
	count, err := s.repo.TemplateCount(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("该安装包仍被视频模板使用，无法删除")
	}
	count, err = s.repo.PointsPackageCount(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("该安装包仍被积分套餐使用，无法删除")
	}
	return s.repo.Delete(ctx, uint(id))
}

func (s *PackageService) validatePayload(ctx context.Context, req *PackagePayload, currentID uint64) error {
	name := strings.TrimSpace(req.PackageName)
	code := strings.TrimSpace(req.PackageCode)
	version := strings.TrimSpace(req.PackageVersion)
	if name == "" || version == "" || strings.TrimSpace(req.DownloadURL) == "" {
		return errors.New("包名称、包版本和下载链接不能为空")
	}
	if !packageCodePattern.MatchString(code) {
		return errors.New("包标识码只能包含字母、数字、点、下划线和中划线")
	}
	systemTypes, err := normalizeSystemTypes(req.SystemTypes)
	if err != nil {
		return err
	}
	req.SystemTypes = systemTypes
	req.Language = normalizePackageLanguage(req.Language)
	downloadURL := strings.ToLower(strings.TrimSpace(req.DownloadURL))
	if !strings.HasPrefix(downloadURL, "http://") && !strings.HasPrefix(downloadURL, "https://") && !strings.HasPrefix(downloadURL, "/") {
		return errors.New("包下载链接必须是 HTTP(S) 地址或站内绝对路径")
	}
	item, err := s.repo.GetByCodeVersion(ctx, code, version)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if item.ID != currentID {
		return errors.New("相同标识码和版本的包已存在")
	}
	return nil
}

func normalizePackageLanguage(value string) string {
	if strings.TrimSpace(value) == "" {
		return i18n.LocaleZhCN
	}
	return i18n.NormalizeLocale(value)
}

func normalizeSystemTypes(values []string) ([]string, error) {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		if len(value) > 32 {
			return nil, errors.New("系统类型长度不能超过 32 个字符")
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	if len(result) == 0 {
		return nil, errors.New("请至少选择一种系统类型")
	}
	if len(result) > 10 {
		return nil, errors.New("系统类型最多选择 10 项")
	}
	return result, nil
}

func applyPackagePayload(item *model.VideoPackage, req *PackagePayload) {
	item.PackageName = strings.TrimSpace(req.PackageName)
	item.PackageCode = strings.TrimSpace(req.PackageCode)
	item.PackageVersion = strings.TrimSpace(req.PackageVersion)
	item.Language = normalizePackageLanguage(req.Language)
	item.SystemTypes = append([]string(nil), req.SystemTypes...)
	item.DownloadURL = strings.TrimSpace(req.DownloadURL)
	item.InstallCount = req.InstallCount
	item.DownloadCount = req.DownloadCount
	item.DeviceCount = req.DeviceCount
	item.Description = strings.TrimSpace(req.Description)
	item.Sort = req.Sort
	item.Status = req.Status
}
