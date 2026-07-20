package service

import (
	"context"
	"errors"
	"net/url"
	"regexp"
	"strings"

	"ai-video/internal/model"
	"ai-video/internal/repository"

	"gorm.io/gorm"
)

var (
	channelCodePattern  = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	uploadMethodPattern = regexp.MustCompile(`^[A-Z][A-Z0-9_-]{0,31}$`)
)

type ChannelService struct {
	repo *repository.ChannelRepo
}

func NewChannelService() *ChannelService {
	return &ChannelService{repo: repository.NewChannelRepo()}
}

type ListChannelRequest struct {
	AdPlatform   string `form:"ad_platform"`
	UploadMethod string `form:"upload_method"`
	Status       *int8  `form:"status" binding:"omitempty,oneof=0 1"`
	Keyword      string `form:"keyword"`
}

type ChannelPayload struct {
	ChannelCode     string  `json:"channel_code" binding:"required,max=64"`
	ChannelName     string  `json:"channel_name" binding:"required,max=128"`
	AgencyCompany   string  `json:"agency_company" binding:"max=128"`
	AdPlatform      string  `json:"ad_platform" binding:"required,max=64"`
	DeliveryPackage string  `json:"delivery_package" binding:"required,max=255"`
	TrackingURL     string  `json:"tracking_url" binding:"max=1024"`
	PortRebate      float64 `json:"port_rebate"`
	ServiceOrderFee float64 `json:"service_order_fee"`
	UploadMethod    string  `json:"upload_method" binding:"required,max=32"`
	Status          int8    `json:"status" binding:"oneof=0 1"`
}

type ChannelStatusPayload struct {
	Status *int8 `json:"status" binding:"required,oneof=0 1"`
}

func (s *ChannelService) List(ctx context.Context, page, pageSize int, req *ListChannelRequest) ([]model.VideoChannel, int64, error) {
	return s.repo.PageList(ctx, page, pageSize, &repository.ChannelListFilter{
		AdPlatform: strings.TrimSpace(req.AdPlatform), UploadMethod: strings.ToUpper(strings.TrimSpace(req.UploadMethod)),
		Status: req.Status, Keyword: strings.TrimSpace(req.Keyword),
	})
}

func (s *ChannelService) ListOptions(ctx context.Context) ([]model.VideoChannel, error) {
	return s.repo.ListOptions(ctx)
}

func (s *ChannelService) GetByID(ctx context.Context, id uint64) (*model.VideoChannel, error) {
	item, err := s.repo.GetByID(ctx, uint(id))
	if err != nil {
		return nil, notFoundOr(err, "渠道不存在")
	}
	return item, nil
}

func (s *ChannelService) Create(ctx context.Context, req *ChannelPayload) (*model.VideoChannel, error) {
	if err := s.validatePayload(ctx, req, 0); err != nil {
		return nil, err
	}
	item := &model.VideoChannel{}
	applyChannelPayload(item, req)
	if err := s.repo.Create(ctx, item); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, errors.New("渠道唯一识别码已存在")
		}
		return nil, err
	}
	return item, nil
}

func (s *ChannelService) Update(ctx context.Context, id uint64, req *ChannelPayload) (*model.VideoChannel, error) {
	item, err := s.repo.GetByID(ctx, uint(id))
	if err != nil {
		return nil, notFoundOr(err, "渠道不存在")
	}
	if err := s.validatePayload(ctx, req, id); err != nil {
		return nil, err
	}
	applyChannelPayload(item, req)
	if err := s.repo.UpdateFields(ctx, item); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, errors.New("渠道唯一识别码已存在")
		}
		return nil, err
	}
	return item, nil
}

func (s *ChannelService) UpdateStatus(ctx context.Context, id uint64, status int8) error {
	item, err := s.repo.GetByID(ctx, uint(id))
	if err != nil {
		return notFoundOr(err, "渠道不存在")
	}
	item.Status = status
	return s.repo.Update(ctx, item, "Status")
}

func (s *ChannelService) Delete(ctx context.Context, id uint64) error {
	if _, err := s.repo.GetByID(ctx, uint(id)); err != nil {
		return notFoundOr(err, "渠道不存在")
	}
	count, err := s.repo.TemplateCount(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("该渠道仍被视频模板使用，无法删除")
	}
	return s.repo.Delete(ctx, uint(id))
}

func (s *ChannelService) validatePayload(ctx context.Context, req *ChannelPayload, currentID uint64) error {
	if err := validateChannelPayloadFields(req); err != nil {
		return err
	}
	code := strings.TrimSpace(req.ChannelCode)
	item, err := s.repo.GetByCode(ctx, code)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if item.ChannelID != currentID {
		return errors.New("渠道唯一识别码已存在")
	}
	return nil
}

func validateChannelPayloadFields(req *ChannelPayload) error {
	code := strings.TrimSpace(req.ChannelCode)
	if !channelCodePattern.MatchString(code) {
		return errors.New("渠道唯一识别码只能包含字母、数字、点、下划线和中划线")
	}
	if strings.TrimSpace(req.ChannelName) == "" || strings.TrimSpace(req.AdPlatform) == "" || strings.TrimSpace(req.DeliveryPackage) == "" {
		return errors.New("渠道名称、投放平台和投放包不能为空")
	}
	if req.PortRebate < 0 || req.PortRebate > 100 {
		return errors.New("端口返点必须在 0 到 100 之间")
	}
	if req.ServiceOrderFee < 0 {
		return errors.New("服务单费不能小于 0")
	}
	if req.ServiceOrderFee > 9999999999.99 {
		return errors.New("服务单费不能超过 9999999999.99")
	}
	method := strings.ToUpper(strings.TrimSpace(req.UploadMethod))
	if !uploadMethodPattern.MatchString(method) {
		return errors.New("上传方式只能包含大写字母、数字、下划线和中划线")
	}
	if trackingURL := strings.TrimSpace(req.TrackingURL); trackingURL != "" {
		parsed, err := url.Parse(trackingURL)
		if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return errors.New("监测链接必须是有效的 HTTP(S) 地址")
		}
	}
	return nil
}

func applyChannelPayload(item *model.VideoChannel, req *ChannelPayload) {
	item.ChannelCode = strings.TrimSpace(req.ChannelCode)
	item.ChannelName = strings.TrimSpace(req.ChannelName)
	item.AgencyCompany = strings.TrimSpace(req.AgencyCompany)
	item.AdPlatform = strings.TrimSpace(req.AdPlatform)
	item.DeliveryPackage = strings.TrimSpace(req.DeliveryPackage)
	item.TrackingURL = strings.TrimSpace(req.TrackingURL)
	item.PortRebate = req.PortRebate
	item.ServiceOrderFee = req.ServiceOrderFee
	item.UploadMethod = strings.ToUpper(strings.TrimSpace(req.UploadMethod))
	item.Status = req.Status
}
