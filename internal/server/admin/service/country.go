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

var countryCodePattern = regexp.MustCompile(`^[A-Z]{2}$`)

type CountryService struct {
	repo *repository.CountryRepo
}

func NewCountryService() *CountryService {
	return &CountryService{repo: repository.NewCountryRepo()}
}

type ListCountryRequest struct {
	Keyword string `form:"keyword"`
	Status  *int8  `form:"status" binding:"omitempty,oneof=0 1"`
}

type CountryPayload struct {
	Code     string `json:"code" binding:"required,len=2"`
	NameZh   string `json:"name_zh" binding:"required,max=100"`
	Language string `json:"language" binding:"omitempty,oneof=zh-CN en-US ja-JP ko-KR es-ES"`
	Status   int8   `json:"status" binding:"oneof=0 1"`
}

type CountryStatusPayload struct {
	Status *int8 `json:"status" binding:"required,oneof=0 1"`
}

func (s *CountryService) List(ctx context.Context, page, pageSize int, req *ListCountryRequest) ([]model.VideoCountry, int64, error) {
	return s.repo.PageList(ctx, page, pageSize, &repository.CountryListFilter{
		Keyword: strings.TrimSpace(req.Keyword),
		Status:  req.Status,
	})
}

func (s *CountryService) ListOptions(ctx context.Context) ([]model.VideoCountry, error) {
	return s.repo.ListEnabled(ctx)
}

func (s *CountryService) GetByID(ctx context.Context, id uint64) (*model.VideoCountry, error) {
	item, err := s.repo.GetByID(ctx, uint(id))
	if err != nil {
		return nil, notFoundOr(err, "国家不存在")
	}
	return item, nil
}

func (s *CountryService) Create(ctx context.Context, req *CountryPayload) (*model.VideoCountry, error) {
	item := &model.VideoCountry{}
	if err := applyCountryPayload(item, req); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, item); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, errors.New("国家代码已存在")
		}
		return nil, err
	}
	return item, nil
}

func (s *CountryService) Update(ctx context.Context, id uint64, req *CountryPayload) (*model.VideoCountry, error) {
	item, err := s.repo.GetByID(ctx, uint(id))
	if err != nil {
		return nil, notFoundOr(err, "国家不存在")
	}
	if err := applyCountryPayload(item, req); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateFields(ctx, item); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, errors.New("国家代码已存在")
		}
		return nil, err
	}
	return item, nil
}

func (s *CountryService) UpdateStatus(ctx context.Context, id uint64, status int8) error {
	item, err := s.repo.GetByID(ctx, uint(id))
	if err != nil {
		return notFoundOr(err, "国家不存在")
	}
	item.Status = status
	return s.repo.Update(ctx, item, "Status")
}

func (s *CountryService) Delete(ctx context.Context, id uint64) error {
	if _, err := s.repo.GetByID(ctx, uint(id)); err != nil {
		return notFoundOr(err, "国家不存在")
	}
	count, err := s.repo.TemplateCount(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("该国家仍被视频模板使用，无法删除")
	}
	return s.repo.HardDelete(ctx, uint(id))
}

func applyCountryPayload(item *model.VideoCountry, req *CountryPayload) error {
	code := strings.ToUpper(strings.TrimSpace(req.Code))
	if !countryCodePattern.MatchString(code) {
		return errors.New("国家代码必须是 2 位英文字母")
	}
	nameZh := strings.TrimSpace(req.NameZh)
	if nameZh == "" {
		return errors.New("中文名称不能为空")
	}
	item.Code = code
	item.NameZh = nameZh
	item.Language = strings.TrimSpace(req.Language)
	if item.Language == "" {
		item.Language = i18n.LocaleForCountry(code)
	} else {
		item.Language = i18n.NormalizeLocale(item.Language)
	}
	item.Status = req.Status
	return nil
}
