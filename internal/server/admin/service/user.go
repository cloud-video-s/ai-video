package service

import (
	"context"
	"strings"
	"time"

	"ai-video/internal/model"
	"ai-video/internal/repository"
)

type AppUserService struct{ repo *repository.AppUserRepo }

func NewAppUserService() *AppUserService { return &AppUserService{repo: repository.NewAppUserRepo()} }

type CreateAppUserRequest struct {
	IMEI                     string     `json:"imei" binding:"required,max=128"`
	Username                 string     `json:"username" binding:"required,max=128"`
	DeviceCountry            string     `json:"device_country" binding:"max=64"`
	ChannelID                string     `json:"channel_id" binding:"max=64"`
	AppVersion               string     `json:"app_version" binding:"max=32"`
	AppName                  string     `json:"app_name" binding:"max=255"`
	FirstOpenedAt            *time.Time `json:"first_opened_at"`
	LastOpenedAt             *time.Time `json:"last_opened_at"`
	LoginType                uint32     `json:"login_type" binding:"omitempty,oneof=1 2 3"`
	LoginAccount             string     `json:"login_account" binding:"max=255"`
	AppIDEmail               string     `json:"appid_email" binding:"omitempty,email,max=50"`
	AppIDThirdCode           string     `json:"appid_third_code" binding:"max=50"`
	GoogleEmail              string     `json:"google_email" binding:"omitempty,email,max=50"`
	GoogleThirdCode          string     `json:"google_third_code" binding:"max=50"`
	UserType                 uint32     `json:"user_type" binding:"omitempty,oneof=1 2"`
	ActiveDays               uint32     `json:"active_days"`
	AvgDailyUsageSeconds     uint64     `json:"avg_daily_usage_seconds"`
	VIPExpiresAt             *time.Time `json:"vip_expires_at"`
	PointsBalance            uint64     `json:"points_balance"`
	SubscriptionStatus       uint32     `json:"subscription_status" binding:"omitempty,oneof=1 2 3"`
	FirstOrderCreatedAt      *time.Time `json:"first_order_created_at"`
	FirstPaidAt              *time.Time `json:"first_paid_at"`
	OrderCount               uint64     `json:"order_count"`
	PaymentCount             uint64     `json:"payment_count"`
	SubscriptionPaymentCount uint64     `json:"subscription_payment_count"`
	OneTimePaymentCount      uint64     `json:"one_time_payment_count"`
	OrderAmountMoney         float64    `json:"order_amount_money" binding:"gte=0"`
	ActualAmountMoney        float64    `json:"actual_amount_money" binding:"gte=0"`
	LastPaidAt               *time.Time `json:"last_paid_at"`
	RefundAmountMoney        float64    `json:"refund_amount_money" binding:"gte=0"`
	PointsMoney              float64    `json:"points_money" binding:"gte=0"`
	AiCotsMoney              float64    `json:"ai_cots_money" binding:"gte=0"`
	Activated                uint32     `json:"activated" binding:"oneof=0 1"`
	KeyBehaviorMet           uint32     `json:"key_behavior_met" binding:"oneof=0 1"`
	PaymentMet               bool       `json:"payment_met"`
	FirstPaymentMet          bool       `json:"first_payment_met"`
	Registered               bool       `json:"registered"`
	AttributionClickedAt     *time.Time `json:"attribution_clicked_at"`
	PhoneModel               string     `json:"phone_model" binding:"max=128"`
	ReRegisteredFromID       *uint64    `json:"re_registered_from_id"`
	Status                   *int32     `json:"status" binding:"omitempty,oneof=0 1"`
	LastLoginAt              *time.Time `json:"last_login_at"`
	LastLoginIP              string     `json:"last_login_ip" binding:"max=64"`
}

type UpdateAppUserRequest struct {
	IMEI                     *string    `json:"imei" binding:"omitempty,min=1,max=128"`
	Username                 *string    `json:"username" binding:"omitempty,min=1,max=128"`
	DeviceCountry            *string    `json:"device_country" binding:"omitempty,max=64"`
	ChannelID                *string    `json:"channel_id" binding:"omitempty,max=64"`
	AppVersion               *string    `json:"app_version" binding:"omitempty,max=32"`
	AppName                  *string    `json:"app_name" binding:"omitempty,max=255"`
	FirstOpenedAt            *time.Time `json:"first_opened_at"`
	LastOpenedAt             *time.Time `json:"last_opened_at"`
	LoginType                *uint32    `json:"login_type" binding:"omitempty,oneof=1 2 3"`
	LoginAccount             *string    `json:"login_account" binding:"omitempty,max=255"`
	AppIDEmail               *string    `json:"appid_email" binding:"omitempty,email,max=50"`
	AppIDThirdCode           *string    `json:"appid_third_code" binding:"omitempty,max=50"`
	GoogleEmail              *string    `json:"google_email" binding:"omitempty,email,max=50"`
	GoogleThirdCode          *string    `json:"google_third_code" binding:"omitempty,max=50"`
	UserType                 *uint32    `json:"user_type" binding:"omitempty,oneof=1 2"`
	ActiveDays               *uint32    `json:"active_days"`
	AvgDailyUsageSeconds     *uint64    `json:"avg_daily_usage_seconds"`
	VIPExpiresAt             *time.Time `json:"vip_expires_at"`
	PointsBalance            *uint64    `json:"points_balance"`
	SubscriptionStatus       *uint32    `json:"subscription_status" binding:"omitempty,oneof=1 2 3"`
	FirstOrderCreatedAt      *time.Time `json:"first_order_created_at"`
	FirstPaidAt              *time.Time `json:"first_paid_at"`
	OrderCount               *uint64    `json:"order_count"`
	PaymentCount             *uint64    `json:"payment_count"`
	SubscriptionPaymentCount *uint64    `json:"subscription_payment_count"`
	OneTimePaymentCount      *uint64    `json:"one_time_payment_count"`
	OrderAmountMoney         *float64   `json:"order_amount_money" binding:"omitempty,gte=0"`
	ActualAmountMoney        *float64   `json:"actual_amount_money" binding:"omitempty,gte=0"`
	LastPaidAt               *time.Time `json:"last_paid_at"`
	RefundAmountMoney        *float64   `json:"refund_amount_money" binding:"omitempty,gte=0"`
	PointsMoney              *float64   `json:"points_money" binding:"omitempty,gte=0"`
	AiCotsMoney              *float64   `json:"ai_cots_money" binding:"omitempty,gte=0"`
	Activated                *uint32    `json:"activated" binding:"omitempty,oneof=0 1"`
	KeyBehaviorMet           *uint32    `json:"key_behavior_met" binding:"omitempty,oneof=0 1"`
	PaymentMet               *bool      `json:"payment_met"`
	FirstPaymentMet          *bool      `json:"first_payment_met"`
	Registered               *bool      `json:"registered"`
	AttributionClickedAt     *time.Time `json:"attribution_clicked_at"`
	PhoneModel               *string    `json:"phone_model" binding:"omitempty,max=128"`
	ReRegisteredFromID       *uint64    `json:"re_registered_from_id"`
	Status                   *int32     `json:"status" binding:"omitempty,oneof=0 1"`
	LastLoginAt              *time.Time `json:"last_login_at"`
	LastLoginIP              *string    `json:"last_login_ip" binding:"omitempty,max=64"`
}

type ListAppUserRequest struct {
	Keyword            string  `form:"keyword" binding:"max=255"`
	DeviceCountry      string  `form:"device_country" binding:"max=64"`
	ChannelID          string  `form:"channel_id" binding:"max=64"`
	AppVersion         string  `form:"app_version" binding:"max=32"`
	AppName            string  `form:"app_name" binding:"max=255"`
	LoginType          uint32  `form:"login_type" binding:"omitempty,oneof=1 2 3"`
	UserType           uint32  `form:"user_type" binding:"omitempty,oneof=1 2"`
	SubscriptionStatus uint32  `form:"subscription_status" binding:"omitempty,oneof=1 2 3"`
	Activated          *uint32 `form:"activated" binding:"omitempty,oneof=0 1"`
	Registered         *bool   `form:"registered"`
	PaymentMet         *bool   `form:"payment_met"`
	Status             *int32  `form:"status" binding:"omitempty,oneof=0 1"`
}

func (s *AppUserService) Create(ctx context.Context, req *CreateAppUserRequest) (*model.VideoUser, error) {
	loginType := req.LoginType
	if loginType == 0 {
		loginType = model.AppUserLoginGuest
	}
	userType := req.UserType
	if userType == 0 {
		userType = model.AppUserTypeFree
	}
	subscriptionStatus := req.SubscriptionStatus
	if subscriptionStatus == 0 {
		subscriptionStatus = model.AppUserSubscriptionNotSubscribed
	}
	status := int32(1)
	if req.Status != nil {
		status = *req.Status
	}
	appName := strings.TrimSpace(req.AppName)
	if appName == "" {
		appName = "0"
	}
	user := &model.VideoUser{
		IMEI: strings.TrimSpace(req.IMEI), Username: strings.TrimSpace(req.Username),
		DeviceCountry: strings.TrimSpace(req.DeviceCountry), ChannelID: strings.TrimSpace(req.ChannelID),
		AppVersion: strings.TrimSpace(req.AppVersion), AppName: appName,
		FirstOpenedAt: req.FirstOpenedAt, LastOpenedAt: req.LastOpenedAt,
		LoginType: loginType, LoginAccount: strings.TrimSpace(req.LoginAccount), UserType: userType,
		ActiveDays: req.ActiveDays, AvgDailyUsageSeconds: req.AvgDailyUsageSeconds,
		VipExpiresAt: req.VIPExpiresAt, PointsBalance: req.PointsBalance, SubscriptionStatus: subscriptionStatus,
		FirstOrderCreatedAt: req.FirstOrderCreatedAt, FirstPaidAt: req.FirstPaidAt,
		OrderCount: req.OrderCount, PaymentCount: req.PaymentCount,
		SubscriptionPaymentCount: req.SubscriptionPaymentCount, OneTimePaymentCount: req.OneTimePaymentCount,
		OrderAmountMoney: req.OrderAmountMoney, ActualAmountMoney: req.ActualAmountMoney,
		LastPaidAt: req.LastPaidAt, RefundAmountMoney: req.RefundAmountMoney,
		PointsMoney: req.PointsMoney, AiCotsMoney: req.AiCotsMoney,
		Activated: req.Activated, KeyBehaviorMet: req.KeyBehaviorMet,
		PaymentMet: req.PaymentMet, FirstPaymentMet: req.FirstPaymentMet, Registered: req.Registered,
		AttributionClickedAt: req.AttributionClickedAt, PhoneModel: strings.TrimSpace(req.PhoneModel),
		ReRegisteredFromID: uint64Value(req.ReRegisteredFromID), Status: status,
		LastLoginAt: req.LastLoginAt, LastLoginIP: strings.TrimSpace(req.LastLoginIP),
	}
	user.AppIDEmail = nullableString(req.AppIDEmail)
	user.AppIDThirdCode = nullableString(req.AppIDThirdCode)
	user.GoogleEmail = nullableString(req.GoogleEmail)
	user.GoogleThirdCode = nullableString(req.GoogleThirdCode)
	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *AppUserService) GetByID(ctx context.Context, id uint64) (*model.VideoUser, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, notFoundOr(err, "客户端用户不存在")
	}
	return user, nil
}

func (s *AppUserService) Update(ctx context.Context, id uint64, req *UpdateAppUserRequest) (*model.VideoUser, error) {
	if _, err := s.GetByID(ctx, id); err != nil {
		return nil, err
	}
	updates := make(map[string]interface{})
	setTrimmedAppUserUpdate(updates, "imei", req.IMEI)
	setTrimmedAppUserUpdate(updates, "username", req.Username)
	setTrimmedAppUserUpdate(updates, "device_country", req.DeviceCountry)
	setTrimmedAppUserUpdate(updates, "channel_id", req.ChannelID)
	setTrimmedAppUserUpdate(updates, "app_version", req.AppVersion)
	setTrimmedAppUserUpdate(updates, "app_name", req.AppName)
	setAppUserUpdate(updates, "first_opened_at", req.FirstOpenedAt)
	setAppUserUpdate(updates, "last_opened_at", req.LastOpenedAt)
	setAppUserUpdate(updates, "login_type", req.LoginType)
	setTrimmedAppUserUpdate(updates, "login_account", req.LoginAccount)
	setNullableAppUserUpdate(updates, "appid_email", req.AppIDEmail)
	setNullableAppUserUpdate(updates, "appid_third_code", req.AppIDThirdCode)
	setNullableAppUserUpdate(updates, "google_email", req.GoogleEmail)
	setNullableAppUserUpdate(updates, "google_third_code", req.GoogleThirdCode)
	setAppUserUpdate(updates, "user_type", req.UserType)
	setAppUserUpdate(updates, "active_days", req.ActiveDays)
	setAppUserUpdate(updates, "avg_daily_usage_seconds", req.AvgDailyUsageSeconds)
	setAppUserUpdate(updates, "vip_expires_at", req.VIPExpiresAt)
	setAppUserUpdate(updates, "points_balance", req.PointsBalance)
	setAppUserUpdate(updates, "subscription_status", req.SubscriptionStatus)
	setAppUserUpdate(updates, "first_order_created_at", req.FirstOrderCreatedAt)
	setAppUserUpdate(updates, "first_paid_at", req.FirstPaidAt)
	setAppUserUpdate(updates, "order_count", req.OrderCount)
	setAppUserUpdate(updates, "payment_count", req.PaymentCount)
	setAppUserUpdate(updates, "subscription_payment_count", req.SubscriptionPaymentCount)
	setAppUserUpdate(updates, "one_time_payment_count", req.OneTimePaymentCount)
	setAppUserUpdate(updates, "order_amount_money", req.OrderAmountMoney)
	setAppUserUpdate(updates, "actual_amount_money", req.ActualAmountMoney)
	setAppUserUpdate(updates, "last_paid_at", req.LastPaidAt)
	setAppUserUpdate(updates, "refund_amount_money", req.RefundAmountMoney)
	setAppUserUpdate(updates, "points_money", req.PointsMoney)
	setAppUserUpdate(updates, "ai_cots_money", req.AiCotsMoney)
	setAppUserUpdate(updates, "activated", req.Activated)
	setAppUserUpdate(updates, "key_behavior_met", req.KeyBehaviorMet)
	setAppUserUpdate(updates, "payment_met", req.PaymentMet)
	setAppUserUpdate(updates, "first_payment_met", req.FirstPaymentMet)
	setAppUserUpdate(updates, "registered", req.Registered)
	setAppUserUpdate(updates, "attribution_clicked_at", req.AttributionClickedAt)
	setTrimmedAppUserUpdate(updates, "phone_model", req.PhoneModel)
	setAppUserUpdate(updates, "re_registered_from_id", req.ReRegisteredFromID)
	setAppUserUpdate(updates, "status", req.Status)
	setAppUserUpdate(updates, "last_login_at", req.LastLoginAt)
	setTrimmedAppUserUpdate(updates, "last_login_ip", req.LastLoginIP)
	if err := s.repo.Update(ctx, id, updates); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}

func setAppUserUpdate[T any](updates map[string]interface{}, column string, value *T) {
	if value != nil {
		updates[column] = *value
	}
}

func setTrimmedAppUserUpdate(updates map[string]interface{}, column string, value *string) {
	if value != nil {
		updates[column] = strings.TrimSpace(*value)
	}
}

func setNullableAppUserUpdate(updates map[string]interface{}, column string, value *string) {
	if value != nil {
		updates[column] = nullableString(*value)
	}
}

func nullableString(value string) string {
	value = strings.TrimSpace(value)
	return value
}

func uint64Value(value *uint64) uint64 {
	if value == nil {
		return 0
	}
	return *value
}

func (s *AppUserService) Delete(ctx context.Context, id uint64) error {
	if _, err := s.GetByID(ctx, id); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

func (s *AppUserService) List(ctx context.Context, page, pageSize int, req *ListAppUserRequest) ([]model.VideoUser, int64, error) {
	return s.repo.PageList(ctx, page, pageSize, &repository.AppUserListFilter{
		Keyword: req.Keyword, DeviceCountry: req.DeviceCountry, ChannelID: req.ChannelID,
		AppVersion: req.AppVersion, AppName: req.AppName, LoginType: req.LoginType,
		UserType: req.UserType, SubscriptionStatus: req.SubscriptionStatus,
		Activated: req.Activated, Registered: req.Registered, PaymentMet: req.PaymentMet, Status: req.Status,
	})
}
