package service

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"ai-video/internal/gen/model"
	"ai-video/internal/repository"

	"gorm.io/gorm"
)

var packageVersionCodePattern = regexp.MustCompile(`^[A-Za-z0-9._+-]+$`)

type PackageVersionService struct {
	repo        *repository.PackageVersionRepo
	packageRepo *repository.PackageRepo
}

func NewPackageVersionService() *PackageVersionService {
	return &PackageVersionService{repo: repository.NewPackageVersionRepo(), packageRepo: repository.NewPackageRepo()}
}

type ListPackageVersionRequest struct {
	PackageCode string  `form:"package_code" binding:"max=128"`
	VersionCode string  `form:"version_code" binding:"max=50"`
	Status      *uint32 `form:"status" binding:"omitempty,oneof=1 2"`
	Keyword     string  `form:"keyword" binding:"max=255"`
}

type PackageVersionPayload struct {
	PackageCode   string `json:"package_code" binding:"required,max=128"`
	VersionCode   string `json:"version_code" binding:"required,max=50"`
	DownloadURL   string `json:"download_url" binding:"required,max=1024"`
	InstallCount  uint64 `json:"install_count"`
	DownloadCount uint64 `json:"download_count"`
	DeviceCount   uint64 `json:"device_count"`
	Description   string `json:"description" binding:"max=10000"`
	Status        uint8  `json:"status" binding:"required,oneof=1 2"`
}

func (s *PackageVersionService) List(ctx context.Context, page, pageSize int, req *ListPackageVersionRequest) ([]model.VideoPackageVersion, int64, error) {
	return s.repo.PageList(ctx, page, pageSize, &repository.PackageVersionListFilter{
		PackageCode: strings.TrimSpace(req.PackageCode), VersionCode: strings.TrimSpace(req.VersionCode),
		Status: req.Status, Keyword: strings.TrimSpace(req.Keyword),
	})
}

func (s *PackageVersionService) GetByID(ctx context.Context, id uint64) (*model.VideoPackageVersion, error) {
	item, err := s.repo.GetByID(ctx, uint(id))
	if err != nil {
		return nil, notFoundOr(err, "安装包版本不存在")
	}
	return item, nil
}

func (s *PackageVersionService) Create(ctx context.Context, req *PackageVersionPayload) (*model.VideoPackageVersion, error) {
	if err := s.validatePayload(ctx, req, 0); err != nil {
		return nil, err
	}
	item := &model.VideoPackageVersion{}
	applyPackageVersionPayload(item, req)
	if err := s.repo.Create(ctx, item); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, errors.New("该安装包版本已存在")
		}
		return nil, err
	}
	return item, nil
}

func (s *PackageVersionService) Update(ctx context.Context, id uint64, req *PackageVersionPayload) (*model.VideoPackageVersion, error) {
	item, err := s.repo.GetByID(ctx, uint(id))
	if err != nil {
		return nil, notFoundOr(err, "安装包版本不存在")
	}
	if err := s.validatePayload(ctx, req, id); err != nil {
		return nil, err
	}
	applyPackageVersionPayload(item, req)
	if err := s.repo.UpdateFields(ctx, item); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, errors.New("该安装包版本已存在")
		}
		return nil, err
	}
	return item, nil
}

func (s *PackageVersionService) Delete(ctx context.Context, id uint64) error {
	if _, err := s.repo.GetByID(ctx, uint(id)); err != nil {
		return notFoundOr(err, "安装包版本不存在")
	}
	return s.repo.Delete(ctx, uint(id))
}

func (s *PackageVersionService) validatePayload(ctx context.Context, req *PackageVersionPayload, currentID uint64) error {
	packageCode := strings.TrimSpace(req.PackageCode)
	versionCode := strings.TrimSpace(req.VersionCode)
	downloadURL := strings.TrimSpace(req.DownloadURL)
	if packageCode == "" || versionCode == "" || downloadURL == "" {
		return errors.New("安装包、版本号和下载链接不能为空")
	}
	if !packageVersionCodePattern.MatchString(versionCode) {
		return errors.New("版本号只能包含字母、数字、点、下划线、中划线和加号")
	}
	if _, err := s.packageRepo.GetByCode(ctx, packageCode); err != nil {
		return notFoundOr(err, "所属安装包不存在")
	}
	lowerURL := strings.ToLower(downloadURL)
	if !strings.HasPrefix(lowerURL, "http://") && !strings.HasPrefix(lowerURL, "https://") && !strings.HasPrefix(downloadURL, "/") {
		return errors.New("下载链接必须是 HTTP(S) 地址或站内绝对路径")
	}
	existing, err := s.repo.GetByPackageVersion(ctx, packageCode, versionCode)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if existing.ID != currentID {
		return errors.New("该安装包版本已存在")
	}
	return nil
}

func applyPackageVersionPayload(item *model.VideoPackageVersion, req *PackageVersionPayload) {
	item.PackageCode = strings.TrimSpace(req.PackageCode)
	item.VersionCode = strings.TrimSpace(req.VersionCode)
	item.DownloadURL = strings.TrimSpace(req.DownloadURL)
	item.InstallCount = req.InstallCount
	item.DownloadCount = req.DownloadCount
	item.DeviceCount = req.DeviceCount
	item.Description = strings.TrimSpace(req.Description)
	item.Status = req.Status
}
