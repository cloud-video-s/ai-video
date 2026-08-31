package adjustevent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"
	"time"

	"ai-video/internal/config"
	"ai-video/internal/gen/model"
	"ai-video/internal/pkg/adjust"
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

type channelCallbackConfig struct {
	Rules []channelCallbackRule `json:"rules"`
}

type channelCallbackRule struct {
	TriggerEvent             string   `json:"trigger_event"`
	CallbackEvents           []string `json:"callback_events"`
	OrderCountThreshold      uint64   `json:"order_count_threshold"`
	PaymentMinimumAmount     float64  `json:"payment_minimum_amount"`
	SubscriptionDelayMinutes uint32   `json:"subscription_delay_minutes"`
	AmountDeductionEnabled   bool     `json:"amount_deduction_enabled"`
	AmountDeductionPercent   float64  `json:"amount_deduction_percent"`
}

type storedChannelCallbackConfig struct {
	Rules                    []channelCallbackRule `json:"rules"`
	Events                   []string              `json:"events"`
	OrderCountThreshold      uint64                `json:"order_count_threshold"`
	PaymentMinimumAmount     float64               `json:"payment_minimum_amount"`
	SubscriptionDelayMinutes uint32                `json:"subscription_delay_minutes"`
	AmountDeductionEnabled   bool                  `json:"amount_deduction_enabled"`
	AmountDeductionPercent   float64               `json:"amount_deduction_percent"`
}

type eventDataStore interface {
	GetUser(context.Context, uint64) (*model.VideoUser, error)
	GetUserAttribution(context.Context, uint64) (*model.VideoUserAttribution, error)
	GetAdjustAttribution(context.Context, string) (*model.VideoAdjustAttribution, error)
	GetChannel(context.Context, uint64) (*model.VideoChannel, error)
	GetOrder(context.Context, string) (*model.VideoOrder, error)
	ResolveChannel(context.Context, uint64) (uint64, error)
	SavePending(context.Context, Message) error
	DeletePending(context.Context, string) error
}

type gormEventStore struct {
	users        *repository.AppUserRepo
	attributions *repository.UserAttributionRepo
	adjust       *repository.AdjustAttributionRepo
	channels     *repository.ChannelRepo
	orders       *repository.OrderRepo
	pending      pendingRepository
}

func newGORMEventStore() *gormEventStore {
	return &gormEventStore{
		users: repository.NewAppUserRepo(), attributions: repository.NewUserAttributionRepo(),
		adjust: repository.NewAdjustAttributionRepo(), channels: repository.NewChannelRepo(),
		orders: repository.NewOrderRepo(), pending: pendingRepository{},
	}
}

func (store *gormEventStore) GetUser(ctx context.Context, userID uint64) (*model.VideoUser, error) {
	return store.users.GetByID(ctx, userID)
}

func (store *gormEventStore) GetUserAttribution(ctx context.Context, userID uint64) (*model.VideoUserAttribution, error) {
	return store.attributions.GetByUserID(ctx, userID)
}

func (store *gormEventStore) GetAdjustAttribution(ctx context.Context, adid string) (*model.VideoAdjustAttribution, error) {
	return store.adjust.GetByADID(ctx, adid, false)
}

func (store *gormEventStore) GetChannel(ctx context.Context, channelID uint64) (*model.VideoChannel, error) {
	return store.channels.GetEnabledByID(ctx, channelID)
}

func (store *gormEventStore) GetOrder(ctx context.Context, orderNo string) (*model.VideoOrder, error) {
	return store.orders.GetByOrderNo(ctx, orderNo, false)
}

func (store *gormEventStore) ResolveChannel(ctx context.Context, channel uint64) (uint64, error) {
	if channel == 0 {
		return 0, nil
	}
	item, err := store.channels.GetByCodeOrID(ctx, channel)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return item.ID, nil
}

func (store *gormEventStore) SavePending(ctx context.Context, message Message) error {
	return store.pending.Save(ctx, message)
}

func (store *gormEventStore) DeletePending(ctx context.Context, eventID string) error {
	return store.pending.Delete(ctx, eventID)
}

type messagePublisher interface {
	PublishMessage(context.Context, Message) error
}

type eventReporter interface {
	Report(context.Context, string, adjust.EventToken, adjust.Event) error
}

type adjustReporter struct {
	authToken   string
	baseURL     string
	environment adjust.Environment
}

func (reporter adjustReporter) Report(ctx context.Context, appToken string, token adjust.EventToken, event adjust.Event) error {
	client, err := adjust.NewClient(adjust.ClientConfig{
		AppToken: appToken, AuthToken: reporter.authToken, BaseURL: reporter.baseURL,
	})
	if err != nil {
		return err
	}
	event.Environment = reporter.environment
	return client.ReportEvent(ctx, token, event)
}

type processor struct {
	store     eventDataStore
	publisher messagePublisher
	reporter  eventReporter
}

func (processor *processor) Process(ctx context.Context, message Message) error {
	message.normalize()
	if err := message.validate(); err != nil {
		return err
	}
	if message.Kind == messageKindReport {
		return processor.processReport(ctx, message)
	}
	return processor.processTrigger(ctx, message)
}

func (processor *processor) processTrigger(ctx context.Context, message Message) error {
	attribution, err := processor.store.GetUserAttribution(ctx, message.UserID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if err = processor.store.SavePending(ctx, message); err != nil {
			return err
		}
		config.Logger(ctx).Infow("Adjust event is waiting for attribution", "event_id", message.EventID,
			"user_id", message.UserID, "action", message.Action)
		return nil
	}
	if err != nil {
		return err
	}

	channelID := message.ChannelID
	if channelID == 0 {
		channelID = attribution.ChannelID
	}
	if channelID == 0 {
		if err = processor.store.SavePending(ctx, message); err != nil {
			return err
		}
		config.Logger(ctx).Infow("Adjust event is waiting for attributed channel", "event_id", message.EventID,
			"user_id", message.UserID, "action", message.Action)
		return nil
	}
	channel, err := processor.store.GetChannel(ctx, channelID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		config.Logger(ctx).Infow("skip Adjust event for missing or disabled channel", "event_id", message.EventID,
			"user_id", message.UserID, "channel_id", channelID)
		return processor.store.DeletePending(ctx, message.EventID)
	}
	if err != nil {
		return err
	}
	if !strings.EqualFold(strings.TrimSpace(channel.AdPlatform), "Adjust") {
		config.Logger(ctx).Infow("skip non-Adjust channel event", "event_id", message.EventID,
			"user_id", message.UserID, "channel_id", channelID, "platform", channel.AdPlatform)
		return processor.store.DeletePending(ctx, message.EventID)
	}
	message.ChannelID = channelID

	callbackConfig, err := decodeCallbackConfig(channel.CallbackConfig)
	if err != nil {
		return fmt.Errorf("decode channel %d callback config: %w", channel.ID, err)
	}
	trigger, _ := triggerName(message.Action)
	rule := callbackConfig.ruleFor(trigger)
	if rule == nil || len(rule.CallbackEvents) == 0 {
		config.Logger(ctx).Infow("skip Adjust event without channel callback rule", "event_id", message.EventID,
			"user_id", message.UserID, "channel_id", channelID, "trigger", trigger)
		return processor.store.DeletePending(ctx, message.EventID)
	}

	var order *model.VideoOrder
	if message.OrderNo != "" {
		order, err = processor.store.GetOrder(ctx, message.OrderNo)
		if err != nil {
			return err
		}
	}
	reports := buildReportMessages(message, *rule, order)
	for _, report := range reports {
		if err := processor.publisher.PublishMessage(ctx, report); err != nil {
			return err
		}
	}
	config.Logger(ctx).Infow("Adjust event rule evaluated", "event_id", message.EventID,
		"user_id", message.UserID, "channel_id", channelID, "trigger", trigger, "report_count", len(reports))
	return processor.store.DeletePending(ctx, message.EventID)
}

func (processor *processor) processReport(ctx context.Context, message Message) error {
	attribution, err := processor.store.GetUserAttribution(ctx, message.UserID)
	if err != nil {
		return err
	}
	appToken := config.Cfg.Adjust.CampaignAppToken
	deviceIDs := map[adjust.DeviceIDType]string{}
	if attribution.Idfa == "" {
		if attribution.AndroidID != "" {
			attribution.Idfa = attribution.AndroidID
		}
		if attribution.GpsAdid != "" && attribution.Idfa == "" {
			attribution.Idfa = attribution.GpsAdid
		}
	}
	//addDeviceID(deviceIDs, adjust.DeviceIDTypeADID, attribution.AdjustAdid)
	//addDeviceID(deviceIDs, adjust.DeviceIDTypeGPSADID, attribution.GoogleAdID)
	//addDeviceID(deviceIDs, adjust.DeviceIDTypeAndroidID, attribution.AndroidID)
	//addDeviceID(deviceIDs, adjust.DeviceIDTypeIMEI, attribution.IMEI)
	addDeviceID(deviceIDs, adjust.DeviceIDTypeIDFA, attribution.Idfa)
	//addDeviceID(deviceIDs, adjust.DeviceIDTypeIDFV, attribution.Idfv)
	if len(deviceIDs) == 0 {
		return errors.New("Adjust event device identifier is unavailable for attributed user")
	}

	ipAddress := strings.TrimSpace(attribution.DeviceIP)
	if parsed := net.ParseIP(ipAddress); parsed == nil || parsed.To4() == nil {
		ipAddress = ""
	}
	callbackParams := map[string]string{
		"event_id": message.EventID, "source_event_id": message.ParentID,
		"user_id":       strconv.FormatUint(message.UserID, 10),
		"source_action": string(message.Action), "channel_id": strconv.FormatUint(message.ChannelID, 10),
	}
	if message.OrderNo != "" {
		callbackParams["order_no"] = message.OrderNo
	}
	event := adjust.Event{
		DeviceIDs: deviceIDs,
		IPAddress: ipAddress, UserAgent: strings.TrimSpace(attribution.UserAgent),
		CreatedAt:      message.OccurredAt,
		CallbackParams: callbackParams, Revenue: message.Revenue, Currency: message.Currency,
	}
	if err = processor.reporter.Report(ctx, appToken, message.ReportAction, event); err != nil {
		return err
	}
	config.Logger(ctx).Infow("Adjust event reported", "event_id", message.EventID,
		"user_id", message.UserID, "channel_id", message.ChannelID, "action", message.ReportAction)
	return nil
}

func addDeviceID(values map[adjust.DeviceIDType]string, kind adjust.DeviceIDType, value string) {
	if value = strings.TrimSpace(value); value != "" {
		values[kind] = value
	}
}

func firstValue(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func decodeCallbackConfig(value string) (channelCallbackConfig, error) {
	value = strings.TrimSpace(value)
	if value == "" || value == "null" {
		return channelCallbackConfig{Rules: []channelCallbackRule{}}, nil
	}
	var stored storedChannelCallbackConfig
	if err := json.Unmarshal([]byte(value), &stored); err != nil {
		return channelCallbackConfig{}, err
	}
	if stored.Rules != nil {
		return channelCallbackConfig{Rules: stored.Rules}, nil
	}
	rules := make([]channelCallbackRule, 0, len(stored.Events))
	for _, event := range stored.Events {
		rule := channelCallbackRule{
			TriggerEvent: event, CallbackEvents: []string{event},
			OrderCountThreshold: stored.OrderCountThreshold, PaymentMinimumAmount: stored.PaymentMinimumAmount,
			SubscriptionDelayMinutes: stored.SubscriptionDelayMinutes,
		}
		if event == callbackPayment || event == callbackSubscription {
			rule.AmountDeductionEnabled = stored.AmountDeductionEnabled
			rule.AmountDeductionPercent = stored.AmountDeductionPercent
		}
		rules = append(rules, rule)
	}
	return channelCallbackConfig{Rules: rules}, nil
}

func (config channelCallbackConfig) ruleFor(trigger string) *channelCallbackRule {
	for i := range config.Rules {
		if strings.EqualFold(strings.TrimSpace(config.Rules[i].TriggerEvent), trigger) {
			return &config.Rules[i]
		}
	}
	return nil
}

func buildReportMessages(trigger Message, rule channelCallbackRule, order *model.VideoOrder) []Message {
	result := make([]Message, 0, len(rule.CallbackEvents))
	amount, currency := orderAmount(order)
	for _, callbackEvent := range rule.CallbackEvents {
		token, ok := callbackToken(callbackEvent)
		if !ok {
			continue
		}
		if token == adjust.EventTokenOrderCreated &&
			(rule.OrderCountThreshold == 0 || trigger.OrderCount != rule.OrderCountThreshold) {
			continue
		}
		if token == adjust.EventTokenPayment && amount < rule.PaymentMinimumAmount {
			continue
		}

		report := Message{
			Kind: messageKindReport, EventID: reportEventID(trigger.EventID, token), ParentID: trigger.EventID,
			UserID: trigger.UserID, Action: trigger.Action, ChannelID: trigger.ChannelID,
			ReportAction: token, OrderNo: trigger.OrderNo, OrderCount: trigger.OrderCount,
			OccurredAt: trigger.OccurredAt,
		}
		if token == adjust.EventTokenPayment || token == adjust.EventTokenSubscription {
			revenue := amount
			if rule.AmountDeductionEnabled {
				revenue *= (100 - rule.AmountDeductionPercent) / 100
			}
			revenue = math.Round(revenue*10000) / 10000
			if revenue >= 0.001 && currency != "" {
				report.Revenue = &revenue
				report.Currency = currency
			}
		}
		if token == adjust.EventTokenSubscription && rule.SubscriptionDelayMinutes > 0 {
			report.NotBefore = trigger.OccurredAt.Add(time.Duration(rule.SubscriptionDelayMinutes) * time.Minute)
		}
		result = append(result, report)
	}
	return result
}

func orderAmount(order *model.VideoOrder) (float64, string) {
	if order == nil {
		return 0, ""
	}
	amount := order.PaidAmount
	if amount <= 0 {
		amount = order.PayableAmount
	}
	return amount, strings.ToUpper(strings.TrimSpace(order.Currency))
}
