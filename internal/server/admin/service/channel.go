package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"ai-video/internal/gen/model"
	"ai-video/internal/repository"

	"gorm.io/gorm"
)

const (
	callbackActivation   = "activation"
	callbackLogin        = "login"
	callbackOrderCreated = "order_created"
	callbackPayment      = "payment"
	callbackSubscription = "subscription"
)

var (
	channelCodePattern  = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	uploadMethodPattern = regexp.MustCompile(`^[A-Z][A-Z0-9_-]{0,31}$`)
	channelPlatforms    = stringSet("Adjust", "热力引擎", "AppsFlyer")
	channelSystems      = stringSet("iOS", "Android", "web")
	channelUploadModes  = stringSet("API", "SDK")
	channelLandingPages = stringSet("API")
	callbackEvents      = stringSet(
		callbackActivation, callbackLogin, callbackOrderCreated, callbackPayment, callbackSubscription,
	)
)

type ChannelService struct {
	repo        *repository.ChannelRepo
	packageRepo *repository.PackageRepo
	adminRepo   *repository.AdminRepo
	mediaRepo   *repository.MediaRepo
}

func NewChannelService() *ChannelService {
	return &ChannelService{
		repo: repository.NewChannelRepo(), packageRepo: repository.NewPackageRepo(), adminRepo: repository.NewAdminRepo(),
		mediaRepo: repository.NewMediaRepo(),
	}
}

type ListChannelRequest struct {
	ListSortRequest
	AgencyCompany   string `form:"agency_company" binding:"max=128"`
	DeliveryPackage string `form:"delivery_package" binding:"max=128"`
	AdPlatform      string `form:"ad_platform" binding:"max=64"`
	UploadMethod    string `form:"upload_method" binding:"max=32"`
	Status          *int8  `form:"status" binding:"omitempty,oneof=0 1"`
	Keyword         string `form:"keyword" binding:"max=128"`
}

type ChannelCallbackConfig struct {
	Rules []ChannelCallbackRule `json:"rules"`
}

type ChannelCallbackRule struct {
	TriggerEvent             string   `json:"trigger_event"`
	CallbackEvents           []string `json:"callback_events"`
	OrderCountThreshold      uint64   `json:"order_count_threshold"`
	PaymentMinimumAmount     float64  `json:"payment_minimum_amount"`
	SubscriptionDelayMinutes uint32   `json:"subscription_delay_minutes"`
	AmountDeductionEnabled   bool     `json:"amount_deduction_enabled"`
	AmountDeductionPercent   float64  `json:"amount_deduction_percent"`
}

type ChannelPayload struct {
	ChannelCode     string                `json:"channel_code" binding:"omitempty,max=64"`
	ChannelName     string                `json:"channel_name" binding:"required,max=128"`
	AccountChannel  string                `json:"account_channel" binding:"max=128"`
	AgencyCompany   string                `json:"agency_company" binding:"max=128"`
	AdPlatform      string                `json:"ad_platform" binding:"required,max=64"`
	AdMedia         string                `json:"ad_media" binding:"required,max=64"`
	MediaID         uint64                `json:"-"`
	DeliveryPackage string                `json:"delivery_package" binding:"required,max=128"`
	SystemType      string                `json:"system_type" binding:"required,max=16"`
	OwnerAdminID    uint64                `json:"owner_admin_id"`
	AdAccount       string                `json:"ad_account" binding:"required,max=128"`
	TrackingURL     string                `json:"tracking_url" binding:"max=1024"`
	LandingPage     string                `json:"landing_page" binding:"required,max=1024"`
	PortRebate      float64               `json:"port_rebate"`
	ServiceOrderFee float64               `json:"service_order_fee"`
	UploadMethod    string                `json:"upload_method" binding:"required,max=32"`
	CallbackConfig  ChannelCallbackConfig `json:"callback_config"`
	Status          *int8                 `json:"status" binding:"omitempty,oneof=0 1"`
}

type ChannelStatusPayload struct {
	Status *int8 `json:"status" binding:"required,oneof=0 1"`
}

type Channel struct {
	ID                  uint64                `json:"channel_id"`
	ChannelCode         string                `json:"channel_code"`
	ChannelName         string                `json:"channel_name"`
	AccountChannel      string                `json:"account_channel"`
	AgencyCompany       string                `json:"agency_company"`
	AdPlatform          string                `json:"ad_platform"`
	AdMedia             string                `json:"ad_media"`
	DeliveryPackage     string                `json:"delivery_package"`
	DeliveryPackageName string                `json:"delivery_package_name"`
	SystemType          string                `json:"system_type"`
	OwnerAdminID        uint64                `json:"owner_admin_id"`
	OwnerUsername       string                `json:"owner_username"`
	OwnerNickname       string                `json:"owner_nickname"`
	AdAccount           string                `json:"ad_account"`
	TrackingURL         string                `json:"tracking_url"`
	LandingPage         string                `json:"landing_page"`
	PortRebate          float64               `json:"port_rebate"`
	ServiceOrderFee     float64               `json:"service_order_fee"`
	UploadMethod        string                `json:"upload_method"`
	CallbackConfig      ChannelCallbackConfig `json:"callback_config"`
	Status              int8                  `json:"status"`
	CreatedAt           time.Time             `json:"created_at"`
	UpdatedAt           time.Time             `json:"updated_at"`
}

func (s *ChannelService) List(ctx context.Context, page, pageSize int, req *ListChannelRequest) ([]Channel, int64, error) {
	rows, total, err := s.repo.PageList(ctx, page, pageSize, &repository.ChannelListFilter{
		ListSort:      req.listSort(),
		AgencyCompany: strings.TrimSpace(req.AgencyCompany), DeliveryPackage: strings.TrimSpace(req.DeliveryPackage),
		AdPlatform: strings.TrimSpace(req.AdPlatform), UploadMethod: strings.ToUpper(strings.TrimSpace(req.UploadMethod)),
		Status: req.Status, Keyword: strings.TrimSpace(req.Keyword),
	})
	if err != nil {
		return nil, 0, err
	}
	items := make([]Channel, 0, len(rows))
	for i := range rows {
		item, convertErr := channelFromRecord(&rows[i])
		if convertErr != nil {
			return nil, 0, convertErr
		}
		items = append(items, *item)
	}
	return items, total, nil
}

func (s *ChannelService) ListOptions(ctx context.Context) ([]model.VideoChannel, error) {
	return s.repo.ListOptions(ctx)
}

func (s *ChannelService) ListMediaOptions(ctx context.Context) ([]*model.VideoMedium, error) {
	return s.mediaRepo.ListOptions(ctx)
}

func (s *ChannelService) GetByID(ctx context.Context, id uint64) (*Channel, error) {
	item, err := s.repo.GetRecordByID(ctx, id)
	if err != nil {
		return nil, notFoundOr(err, "渠道不存在")
	}
	return channelFromRecord(item)
}

func (s *ChannelService) Create(ctx context.Context, req *ChannelPayload, currentAdminID uint64) (*Channel, error) {
	normalizeChannelPayload(req)
	if req.ChannelCode == "" {
		code, err := generateChannelCode()
		if err != nil {
			return nil, err
		}
		req.ChannelCode = code
	}
	if req.OwnerAdminID == 0 {
		req.OwnerAdminID = currentAdminID
	}
	if err := s.validatePayload(ctx, req, 0); err != nil {
		return nil, err
	}
	status := int8(1)
	if req.Status != nil {
		status = *req.Status
	}
	item, err := channelRecordFromPayload(req, status)
	if err != nil {
		return nil, err
	}
	if err := s.repo.CreateRecord(ctx, item); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, errors.New("渠道唯一码已存在，请重试")
		}
		return nil, err
	}
	return s.GetByID(ctx, item.ID)
}

func (s *ChannelService) Update(ctx context.Context, id uint64, req *ChannelPayload) (*Channel, error) {
	existing, err := s.repo.GetRecordByID(ctx, id)
	if err != nil {
		return nil, notFoundOr(err, "渠道不存在")
	}
	normalizeChannelPayload(req)
	if req.ChannelCode != "" && req.ChannelCode != existing.ChannelCode {
		return nil, errors.New("渠道唯一码由系统生成，创建后不可修改")
	}
	req.ChannelCode = existing.ChannelCode
	if req.OwnerAdminID == 0 && existing.OwnerAdminID != nil {
		req.OwnerAdminID = *existing.OwnerAdminID
	}
	if err := s.validatePayload(ctx, req, id); err != nil {
		return nil, err
	}
	status := existing.Status
	if req.Status != nil {
		status = *req.Status
	}
	updated, err := channelRecordFromPayload(req, status)
	if err != nil {
		return nil, err
	}
	updated.ID = id
	if err := s.repo.UpdateRecord(ctx, updated); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *ChannelService) UpdateStatus(ctx context.Context, id uint64, status int8) error {
	if _, err := s.repo.GetRecordByID(ctx, id); err != nil {
		return notFoundOr(err, "渠道不存在")
	}
	return s.repo.UpdateStatus(ctx, id, status)
}

func (s *ChannelService) Delete(ctx context.Context, id uint64) error {
	if _, err := s.repo.GetRecordByID(ctx, id); err != nil {
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
	item, err := s.repo.GetByCode(ctx, req.ChannelCode)
	if err == nil && item.ID != currentID {
		return errors.New("渠道唯一码已存在")
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	media, err := s.mediaRepo.GetByName(ctx, req.AdMedia)
	if err != nil {
		return notFoundOr(err, "投放媒体不存在")
	}
	req.MediaID = media.ID
	appPackage, err := s.packageRepo.GetByCode(ctx, req.DeliveryPackage)
	if err != nil {
		return notFoundOr(err, "投放包不存在")
	}
	if appPackage.Status != 1 {
		return errors.New("投放包已禁用")
	}
	owner, err := s.adminRepo.GetByID(ctx, req.OwnerAdminID)
	if err != nil {
		return notFoundOr(err, "所属用户不存在")
	}
	if owner.Status != 1 {
		return errors.New("所属用户已禁用")
	}
	return nil
}

func validateChannelPayloadFields(req *ChannelPayload) error {
	if req.ChannelCode != "" && !channelCodePattern.MatchString(req.ChannelCode) {
		return errors.New("渠道唯一码只能包含字母、数字、点、下划线和中划线")
	}
	if strings.TrimSpace(req.ChannelName) == "" {
		return errors.New("渠道名称不能为空")
	}
	if _, ok := channelPlatforms[req.AdPlatform]; !ok {
		return errors.New("归因平台必须是 Adjust、热力引擎或 AppsFlyer")
	}
	if strings.TrimSpace(req.AdMedia) == "" {
		return errors.New("投放媒体不能为空")
	}
	if strings.TrimSpace(req.DeliveryPackage) == "" {
		return errors.New("投放包不能为空")
	}
	if _, ok := channelSystems[req.SystemType]; !ok {
		return errors.New("系统必须是 iOS、Android 或 web")
	}
	if req.OwnerAdminID == 0 {
		return errors.New("所属用户不能为空")
	}
	if strings.TrimSpace(req.AdAccount) == "" {
		return errors.New("投放账户不能为空")
	}
	if _, ok := channelUploadModes[req.UploadMethod]; !ok || !uploadMethodPattern.MatchString(req.UploadMethod) {
		return errors.New("上传方式必须是 API 或 SDK")
	}
	if _, ok := channelLandingPages[req.LandingPage]; !ok {
		return errors.New("落地页必须是 API")
	}
	if req.PortRebate < 0 || req.PortRebate > 100 {
		return errors.New("开户返点必须在 0 到 100 之间")
	}
	if req.ServiceOrderFee < 0 || req.ServiceOrderFee > 9999999999.99 {
		return errors.New("服务单费必须在 0 到 9999999999.99 之间")
	}
	if trackingURL := strings.TrimSpace(req.TrackingURL); trackingURL != "" {
		parsed, err := url.Parse(trackingURL)
		if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return errors.New("监测链接必须是有效的 HTTP(S) 地址")
		}
	}
	return validateChannelCallbackConfig(&req.CallbackConfig)
}

func validateChannelCallbackConfig(config *ChannelCallbackConfig) error {
	triggerEvents := make(map[string]struct{}, len(config.Rules))
	for i := range config.Rules {
		rule := &config.Rules[i]
		if _, ok := callbackEvents[rule.TriggerEvent]; !ok {
			return fmt.Errorf("不支持的触发事件: %s", rule.TriggerEvent)
		}
		if _, duplicate := triggerEvents[rule.TriggerEvent]; duplicate {
			return fmt.Errorf("触发事件重复: %s", rule.TriggerEvent)
		}
		triggerEvents[rule.TriggerEvent] = struct{}{}

		selected := make(map[string]struct{}, len(rule.CallbackEvents))
		for _, event := range rule.CallbackEvents {
			if _, ok := callbackEvents[event]; !ok {
				return fmt.Errorf("%s配置包含不支持的回传事件: %s", callbackEventName(rule.TriggerEvent), event)
			}
			if _, duplicate := selected[event]; duplicate {
				return fmt.Errorf("%s配置的回传事件重复: %s", callbackEventName(rule.TriggerEvent), event)
			}
			selected[event] = struct{}{}
		}
		prefix := callbackEventName(rule.TriggerEvent) + "配置："
		if _, ok := selected[callbackOrderCreated]; ok && rule.OrderCountThreshold == 0 {
			return errors.New(prefix + "创建订单回传的次数必须大于 0")
		}
		if rule.OrderCountThreshold > 999999 {
			return errors.New(prefix + "创建订单回传的次数不能超过 999999")
		}
		if rule.PaymentMinimumAmount < 0 || rule.PaymentMinimumAmount > 9999999999.99 {
			return errors.New(prefix + "付费回传的最低金额必须在 0 到 9999999999.99 之间")
		}
		if rule.SubscriptionDelayMinutes > 525600 {
			return errors.New(prefix + "订阅回传延时不能超过 525600 分钟")
		}
		if rule.AmountDeductionPercent < 0 || rule.AmountDeductionPercent > 100 {
			return errors.New(prefix + "金额扣量比例必须在 0 到 100 之间")
		}
		if rule.AmountDeductionEnabled {
			_, payment := selected[callbackPayment]
			_, subscription := selected[callbackSubscription]
			if !payment && !subscription {
				return errors.New(prefix + "金额扣量仅适用于付费或订阅回传")
			}
			if rule.AmountDeductionPercent <= 0 || rule.AmountDeductionPercent > 100 {
				return errors.New(prefix + "金额扣量比例必须大于 0 且不超过 100")
			}
		}
	}
	return nil
}

func normalizeChannelPayload(req *ChannelPayload) {
	req.ChannelCode = strings.TrimSpace(req.ChannelCode)
	req.ChannelName = strings.TrimSpace(req.ChannelName)
	req.AccountChannel = strings.TrimSpace(req.AccountChannel)
	req.AgencyCompany = strings.TrimSpace(req.AgencyCompany)
	req.AdPlatform = strings.TrimSpace(req.AdPlatform)
	req.AdMedia = strings.TrimSpace(req.AdMedia)
	req.DeliveryPackage = strings.TrimSpace(req.DeliveryPackage)
	req.SystemType = strings.TrimSpace(req.SystemType)
	req.AdAccount = strings.TrimSpace(req.AdAccount)
	req.TrackingURL = strings.TrimSpace(req.TrackingURL)
	req.LandingPage = strings.TrimSpace(req.LandingPage)
	req.UploadMethod = strings.ToUpper(strings.TrimSpace(req.UploadMethod))
	for i := range req.CallbackConfig.Rules {
		rule := &req.CallbackConfig.Rules[i]
		rule.TriggerEvent = strings.ToLower(strings.TrimSpace(rule.TriggerEvent))
		rule.CallbackEvents = normalizeCallbackEvents(rule.CallbackEvents)
	}
}

func channelRecordFromPayload(req *ChannelPayload, status int8) (*repository.ChannelRecord, error) {
	callbackJSON, err := json.Marshal(req.CallbackConfig)
	if err != nil {
		return nil, fmt.Errorf("序列化回传配置失败: %w", err)
	}
	ownerID := req.OwnerAdminID
	return &repository.ChannelRecord{
		VideoChannel: model.VideoChannel{
			ChannelCode: req.ChannelCode, ChannelName: req.ChannelName, AccountChannel: req.AccountChannel,
			AgencyCompany: req.AgencyCompany, MediaID: req.MediaID, AdPlatform: req.AdPlatform,
			DeliveryPackage: req.DeliveryPackage, SystemType: req.SystemType, OwnerAdminID: &ownerID,
			AdAccount: req.AdAccount, TrackingURL: req.TrackingURL, LandingPage: req.LandingPage,
			PortRebate: req.PortRebate, ServiceOrderFee: req.ServiceOrderFee, UploadMethod: req.UploadMethod,
			CallbackConfig: string(callbackJSON), Status: status,
		},
		AdMedia: req.AdMedia,
	}, nil
}

func channelFromRecord(item *repository.ChannelRecord) (*Channel, error) {
	config := ChannelCallbackConfig{Rules: []ChannelCallbackRule{}}
	if value := strings.TrimSpace(item.CallbackConfig); value != "" && value != "null" {
		decoded, err := decodeChannelCallbackConfig(value)
		if err != nil {
			return nil, fmt.Errorf("渠道 %d 的回传配置格式无效: %w", item.ID, err)
		}
		config = decoded
	}
	ownerID := uint64(0)
	if item.OwnerAdminID != nil {
		ownerID = *item.OwnerAdminID
	}
	return &Channel{
		ID: item.ID, ChannelCode: item.ChannelCode, ChannelName: item.ChannelName,
		AccountChannel: item.AccountChannel, AgencyCompany: item.AgencyCompany,
		AdPlatform: item.AdPlatform, AdMedia: item.AdMedia, DeliveryPackage: item.DeliveryPackage,
		DeliveryPackageName: item.DeliveryPackageName, SystemType: item.SystemType,
		OwnerAdminID: ownerID, OwnerUsername: item.OwnerUsername, OwnerNickname: item.OwnerNickname,
		AdAccount: item.AdAccount, TrackingURL: item.TrackingURL, LandingPage: item.LandingPage,
		PortRebate: item.PortRebate, ServiceOrderFee: item.ServiceOrderFee, UploadMethod: item.UploadMethod,
		CallbackConfig: config, Status: item.Status, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}, nil
}

type storedChannelCallbackConfig struct {
	Rules                    []ChannelCallbackRule `json:"rules"`
	Events                   []string              `json:"events"`
	OrderCountThreshold      uint64                `json:"order_count_threshold"`
	PaymentMinimumAmount     float64               `json:"payment_minimum_amount"`
	SubscriptionDelayMinutes uint32                `json:"subscription_delay_minutes"`
	AmountDeductionEnabled   bool                  `json:"amount_deduction_enabled"`
	AmountDeductionPercent   float64               `json:"amount_deduction_percent"`
}

func decodeChannelCallbackConfig(value string) (ChannelCallbackConfig, error) {
	stored := storedChannelCallbackConfig{}
	if err := json.Unmarshal([]byte(value), &stored); err != nil {
		return ChannelCallbackConfig{}, err
	}
	if stored.Rules != nil {
		return ChannelCallbackConfig{Rules: stored.Rules}, nil
	}
	rules := make([]ChannelCallbackRule, 0, len(stored.Events))
	for _, event := range stored.Events {
		rule := ChannelCallbackRule{
			TriggerEvent: event, CallbackEvents: []string{event},
			OrderCountThreshold:      stored.OrderCountThreshold,
			PaymentMinimumAmount:     stored.PaymentMinimumAmount,
			SubscriptionDelayMinutes: stored.SubscriptionDelayMinutes,
		}
		if event == callbackPayment || event == callbackSubscription {
			rule.AmountDeductionEnabled = stored.AmountDeductionEnabled
			rule.AmountDeductionPercent = stored.AmountDeductionPercent
		}
		rules = append(rules, rule)
	}
	return ChannelCallbackConfig{Rules: rules}, nil
}

func normalizeCallbackEvents(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	events := make([]string, 0, len(values))
	for _, event := range values {
		event = strings.ToLower(strings.TrimSpace(event))
		if event == "" {
			continue
		}
		if _, exists := seen[event]; exists {
			continue
		}
		seen[event] = struct{}{}
		events = append(events, event)
	}
	return events
}

func callbackEventName(event string) string {
	switch event {
	case callbackActivation:
		return "激活"
	case callbackLogin:
		return "登陆"
	case callbackOrderCreated:
		return "创建订单"
	case callbackPayment:
		return "付费"
	case callbackSubscription:
		return "订阅"
	default:
		return event
	}
}

func generateChannelCode() (string, error) {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("生成渠道唯一码失败: %w", err)
	}
	return "CH_" + strings.ToUpper(hex.EncodeToString(bytes)), nil
}

func stringSet(values ...string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}
