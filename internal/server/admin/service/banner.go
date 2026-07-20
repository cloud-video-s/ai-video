package service

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"ai-video/internal/model"
	"ai-video/internal/repository"

	"gorm.io/gorm"
)

type BannerService struct {
	repo         *repository.BannerRepo
	templateRepo *repository.TemplateRepo
	countryRepo  *repository.CountryRepo
	channelRepo  *repository.ChannelRepo
	packageRepo  *repository.PackageRepo
	positionRepo *repository.DisplayPositionRepo
}

func NewBannerService() *BannerService {
	return &BannerService{
		repo: repository.NewBannerRepo(), templateRepo: repository.NewTemplateRepo(),
		countryRepo: repository.NewCountryRepo(), channelRepo: repository.NewChannelRepo(),
		packageRepo:  repository.NewPackageRepo(),
		positionRepo: repository.NewDisplayPositionRepo(),
	}
}

type ListBannerRequest struct {
	PositionKey string `form:"position_key" binding:"omitempty,max=100"`
	CountryID   uint64 `form:"country_id"`
	ChannelID   uint64 `form:"channel_id"`
	PackageID   uint64 `form:"package_id"`
	JumpType    uint8  `form:"jump_type" binding:"omitempty,oneof=1 2 3 4"`
	Status      *int8  `form:"status" binding:"omitempty,oneof=0 1"`
	Keyword     string `form:"keyword"`
}

type BannerPayload struct {
	Name                string   `json:"name" binding:"required,max=128"`
	CoverImage          string   `json:"cover_image" binding:"required,max=1024"`
	DisplayPositionKeys []string `json:"display_position_keys" binding:"required,min=1,max=100,dive,required,max=64"`
	CountryIDs          []uint64 `json:"country_ids" binding:"max=100,dive,gt=0"`
	ChannelIDs          []uint64 `json:"channel_ids" binding:"max=100,dive,gt=0"`
	PackageIDs          []uint64 `json:"package_ids" binding:"max=100,dive,gt=0"`
	Remark              string   `json:"remark" binding:"max=500"`
	Sort                uint64   `json:"sort"`
	JumpType            uint8    `json:"jump_type" binding:"required,oneof=1 2 3 4"`
	JumpURL             string   `json:"jump_url" binding:"max=1024"`
	TemplateID          *uint64  `json:"template_id"`
	Status              int8     `json:"status" binding:"oneof=0 1"`
}

func (s *BannerService) List(ctx context.Context, page, pageSize int, req *ListBannerRequest) ([]model.VideoBanner, int64, error) {
	return s.repo.PageList(ctx, page, pageSize, &repository.BannerListFilter{
		CountryID: req.CountryID, ChannelID: req.ChannelID, PackageID: req.PackageID,
		PositionKey: strings.TrimSpace(req.PositionKey),
		JumpType:    req.JumpType, Status: req.Status, Keyword: strings.TrimSpace(req.Keyword),
	})
}

func (s *BannerService) GetByID(ctx context.Context, id uint64) (*model.VideoBanner, error) {
	item, err := s.repo.GetDetail(ctx, id)
	if err != nil {
		return nil, notFoundOr(err, "Banner 不存在")
	}
	return item, nil
}

func (s *BannerService) Create(ctx context.Context, req *BannerPayload) (*model.VideoBanner, error) {
	if err := s.prepareAndValidate(ctx, req); err != nil {
		return nil, err
	}
	item := &model.VideoBanner{}
	applyBannerPayload(item, req)
	if err := repository.Transaction(ctx, func(ctx context.Context) error {
		if err := s.repo.Create(ctx, item); err != nil {
			return err
		}
		return s.repo.ReplaceTargets(ctx, item, bannerTargetIDs(req))
	}); err != nil {
		return nil, err
	}
	return s.repo.GetDetail(ctx, item.ID)
}

func (s *BannerService) Update(ctx context.Context, id uint64, req *BannerPayload) (*model.VideoBanner, error) {
	item, err := s.repo.GetDetail(ctx, id)
	if err != nil {
		return nil, notFoundOr(err, "Banner 不存在")
	}
	if err := s.prepareAndValidate(ctx, req); err != nil {
		return nil, err
	}
	applyBannerPayload(item, req)
	if err := repository.Transaction(ctx, func(ctx context.Context) error {
		if err := s.repo.UpdateFields(ctx, item); err != nil {
			return err
		}
		return s.repo.ReplaceTargets(ctx, item, bannerTargetIDs(req))
	}); err != nil {
		return nil, err
	}
	return s.repo.GetDetail(ctx, item.ID)
}

func (s *BannerService) Delete(ctx context.Context, id uint64) error {
	if _, err := s.repo.GetDetail(ctx, id); err != nil {
		return notFoundOr(err, "Banner 不存在")
	}
	return s.repo.DeleteWithTargets(ctx, id)
}

func (s *BannerService) prepareAndValidate(ctx context.Context, req *BannerPayload) error {
	var err error
	if req.DisplayPositionKeys, err = normalizeBannerPositionKeys(req.DisplayPositionKeys); err != nil {
		return err
	}
	if len(req.DisplayPositionKeys) == 0 {
		return errors.New("请至少选择一个展示位置")
	}
	for _, key := range req.DisplayPositionKeys {
		position, lookupErr := s.positionRepo.GetByKey(ctx, key)
		if lookupErr != nil {
			return notFoundOr(lookupErr, "展示位置不存在")
		}
		if position.Status != 1 {
			return errors.New("所选展示位置中包含已禁用项")
		}
	}
	if req.CountryIDs, err = normalizeTargetIDs(req.CountryIDs, "国家"); err != nil {
		return err
	}
	if req.ChannelIDs, err = normalizeTargetIDs(req.ChannelIDs, "渠道"); err != nil {
		return err
	}
	if req.PackageIDs, err = normalizeTargetIDs(req.PackageIDs, "安装包"); err != nil {
		return err
	}
	for _, id := range req.CountryIDs {
		if _, err := s.countryRepo.GetByID(ctx, uint(id)); err != nil {
			return notFoundOr(err, "国家不存在")
		}
	}
	for _, id := range req.ChannelIDs {
		if _, err := s.channelRepo.GetByID(ctx, uint(id)); err != nil {
			return notFoundOr(err, "渠道不存在")
		}
	}
	for _, id := range req.PackageIDs {
		if _, err := s.packageRepo.GetByID(ctx, uint(id)); err != nil {
			return notFoundOr(err, "安装包不存在")
		}
	}
	return s.validateJump(ctx, req)
}

func normalizeBannerPositionKeys(values []string) ([]string, error) {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if len(value) > 64 || !displayPositionKeyPattern.MatchString(value) {
			return nil, errors.New("展示位置标识格式有误")
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result, nil
}

func (s *BannerService) validateJump(ctx context.Context, req *BannerPayload) error {
	req.JumpURL = strings.TrimSpace(req.JumpURL)
	switch req.JumpType {
	case model.BannerJumpTypeLink:
		if !validBannerLink(req.JumpURL) {
			return errors.New("链接跳转必须提供有效的绝对链接、深链或站内路径")
		}
		req.TemplateID = nil
	case model.BannerJumpTypeTemplate:
		if req.TemplateID == nil || *req.TemplateID == 0 {
			return errors.New("模板跳转必须选择目标模板")
		}
		if _, err := s.templateRepo.GetWithType(ctx, *req.TemplateID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("目标模板不存在")
			}
			return err
		}
		req.JumpURL = ""
	case model.BannerJumpTypeTextToImage, model.BannerJumpTypeTextToVideo:
		req.JumpURL = ""
		req.TemplateID = nil
	default:
		return errors.New("不支持的 Banner 跳转方式")
	}
	return nil
}

func validBannerLink(value string) bool {
	if strings.HasPrefix(value, "/") && !strings.HasPrefix(value, "//") {
		return true
	}
	parsed, err := url.Parse(value)
	return err == nil && parsed.IsAbs() && parsed.Scheme != "" && parsed.Host != ""
}

func applyBannerPayload(item *model.VideoBanner, req *BannerPayload) {
	item.Name = strings.TrimSpace(req.Name)
	item.CoverImage = strings.TrimSpace(req.CoverImage)
	item.Remark = strings.TrimSpace(req.Remark)
	item.Sort = req.Sort
	item.JumpType = req.JumpType
	item.JumpURL = req.JumpURL
	item.TemplateID = req.TemplateID
	item.Status = req.Status
}

func bannerTargetIDs(req *BannerPayload) repository.BannerTargetIDs {
	return repository.BannerTargetIDs{
		DisplayPositionKeys: append([]string(nil), req.DisplayPositionKeys...),
		CountryIDs:          req.CountryIDs, ChannelIDs: req.ChannelIDs, PackageIDs: req.PackageIDs,
	}
}
