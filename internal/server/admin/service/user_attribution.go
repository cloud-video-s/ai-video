package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"ai-video/internal/domain"
	"ai-video/internal/gen/model"
	"ai-video/internal/repository"

	"gorm.io/gorm"
)

type UserAttributionService struct {
	repo        *repository.UserAttributionRepo
	channelRepo *repository.ChannelRepo
}

func NewUserAttributionService() *UserAttributionService {
	return &UserAttributionService{
		repo: repository.NewUserAttributionRepo(), channelRepo: repository.NewChannelRepo(),
	}
}

type ListUserAttributionRequest struct {
	Keyword     string `form:"keyword" binding:"max=128"`
	ChannelCode string `form:"channel_code" binding:"max=64"`
	Event       string `form:"event" binding:"omitempty,oneof=activation key_behavior payment first_payment registration"`
	Reached     *bool  `form:"reached"`
	StartedAt   string `form:"started_at" binding:"omitempty,datetime=2006-01-02"`
	EndedAt     string `form:"ended_at" binding:"omitempty,datetime=2006-01-02"`
}

type UpdateUserAttributionRequest struct {
	ChannelCode  string     `json:"channel_code" binding:"max=64"`
	OAID         string     `json:"oaid" binding:"max=128"`
	IMEI         string     `json:"imei" binding:"max=128"`
	AndroidID    string     `json:"android_id" binding:"max=128"`
	IP           string     `json:"ip" binding:"max=64"`
	UserAgent    string     `json:"user_agent" binding:"max=1024"`
	AttributedAt *time.Time `json:"attributed_at"`
	Remark       string     `json:"remark" binding:"max=255"`
}

type RecordAttributionEventRequest struct {
	Event  string `json:"event" binding:"required,oneof=activation key_behavior payment first_payment registration"`
	Action string `json:"action" binding:"required,oneof=callback deduct"`
}

func (s *UserAttributionService) List(
	ctx context.Context, page, pageSize int, req *ListUserAttributionRequest,
) ([]model.VideoUserAttribution, int64, error) {
	startedAt, err := parseAttributionDate(req.StartedAt, false)
	if err != nil {
		return nil, 0, err
	}
	endedAt, err := parseAttributionDate(req.EndedAt, true)
	if err != nil {
		return nil, 0, err
	}
	if req.Reached != nil && strings.TrimSpace(req.Event) == "" {
		return nil, 0, errors.New("筛选达标状态时必须选择事件")
	}
	list, total, err := s.repo.PageList(ctx, page, pageSize, &repository.UserAttributionListFilter{
		Keyword: strings.TrimSpace(req.Keyword), ChannelCode: strings.TrimSpace(req.ChannelCode),
		Event: strings.TrimSpace(req.Event), Reached: req.Reached, StartedAt: startedAt, EndedAt: endedAt,
	})
	if err != nil {
		return nil, 0, err
	}
	for i := range list {
		s.enrichChannel(ctx, &list[i])
	}
	return list, total, nil
}

func (s *UserAttributionService) GetByID(ctx context.Context, id uint64) (*model.VideoUserAttribution, error) {
	item, err := s.repo.GetByID(ctx, id, false)
	if err != nil {
		return nil, notFoundOr(err, "归因记录不存在")
	}
	s.enrichChannel(ctx, item)
	return item, nil
}

func (s *UserAttributionService) Update(
	ctx context.Context, id uint64, req *UpdateUserAttributionRequest,
) (*model.VideoUserAttribution, error) {
	item, err := s.repo.GetByID(ctx, id, false)
	if err != nil {
		return nil, notFoundOr(err, "归因记录不存在")
	}
	channelCode := strings.TrimSpace(req.ChannelCode)
	if channelCode != "" {
		if _, err := s.channelRepo.GetByCodeOrID(ctx, channelCode); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.New("渠道不存在")
			}
			return nil, err
		}
	}
	item.ChannelCode = channelCode
	item.OAID = strings.TrimSpace(req.OAID)
	item.IMEI = strings.TrimSpace(req.IMEI)
	item.AndroidID = strings.TrimSpace(req.AndroidID)
	item.IP = strings.TrimSpace(req.IP)
	item.UserAgent = strings.TrimSpace(req.UserAgent)
	item.AttributedAt = req.AttributedAt
	item.Remark = strings.TrimSpace(req.Remark)
	if err := s.repo.Update(ctx, item); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *UserAttributionService) RecordEvent(
	ctx context.Context, id uint64, req *RecordAttributionEventRequest,
) (*model.VideoUserAttribution, error) {
	err := repository.Transaction(ctx, func(ctx context.Context) error {
		item, err := s.repo.GetByID(ctx, id, true)
		if err != nil {
			return notFoundOr(err, "归因记录不存在")
		}
		callbackCount, deductCount, reached, err := attributionEventState(item, req.Event)
		if err != nil {
			return err
		}
		if req.Action == domain.AttributionActionCallback && !reached {
			return errors.New("当前用户尚未达到该事件，不能记录回传")
		}
		if req.Action == domain.AttributionActionDeduct && deductCount >= callbackCount {
			return errors.New("扣除次数不能超过已回传次数")
		}
		column, err := attributionEventColumn(req.Event, req.Action)
		if err != nil {
			return err
		}
		return s.repo.IncrementEvent(ctx, id, column, time.Now())
	})
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *UserAttributionService) SyncUsers(ctx context.Context) (int64, error) {
	return s.repo.SyncUsers(ctx)
}

func (s *UserAttributionService) enrichChannel(ctx context.Context, item *model.VideoUserAttribution) {
	code := strings.TrimSpace(item.ChannelCode)
	if code == "" {
		code = strings.TrimSpace(item.User.ChannelID)
		item.ChannelCode = code
	}
	if code == "" {
		return
	}
	return
}

func parseAttributionDate(value string, endOfDay bool) (*time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parsed, err := time.ParseInLocation("2006-01-02", value, time.Local)
	if err != nil {
		return nil, fmt.Errorf("日期格式错误: %w", err)
	}
	if endOfDay {
		parsed = parsed.Add(24*time.Hour - time.Nanosecond)
	}
	return &parsed, nil
}

func attributionEventState(item *model.VideoUserAttribution, event string) (uint64, uint64, bool, error) {
	switch event {
	case domain.AttributionEventActivation:
		return item.ActivationCallbackCount, item.ActivationDeductCount, item.User.Activated != 0, nil
	case domain.AttributionEventKeyBehavior:
		return item.KeyBehaviorCallbackCount, item.KeyBehaviorDeductCount, item.User.KeyBehaviorMet != 0, nil
	//case domain.AttributionEventPayment:
	//	return item.PaymentCallbackCount, item.PaymentDeductCount, item.User.PaymentMet, nil
	//case domain.AttributionEventFirstPayment:
	//	return item.FirstPaymentCallbackCount, item.FirstPaymentDeductCount, item.User.FirstPaymentMet, nil
	//case domain.AttributionEventRegistration:
	//	return item.RegistrationCallbackCount, item.RegistrationDeductCount, item.User.Registered, nil
	default:
		return 0, 0, false, errors.New("不支持的归因事件")
	}
}

func attributionEventColumn(event, action string) (string, error) {
	columns := map[string][2]string{
		domain.AttributionEventActivation:   {"activation_callback_count", "activation_deduct_count"},
		domain.AttributionEventKeyBehavior:  {"key_behavior_callback_count", "key_behavior_deduct_count"},
		domain.AttributionEventPayment:      {"payment_callback_count", "payment_deduct_count"},
		domain.AttributionEventFirstPayment: {"first_payment_callback_count", "first_payment_deduct_count"},
		domain.AttributionEventRegistration: {"registration_callback_count", "registration_deduct_count"},
	}
	pair, ok := columns[event]
	if !ok {
		return "", errors.New("不支持的归因事件")
	}
	if action == domain.AttributionActionCallback {
		return pair[0], nil
	}
	if action == domain.AttributionActionDeduct {
		return pair[1], nil
	}
	return "", errors.New("不支持的归因操作")
}
