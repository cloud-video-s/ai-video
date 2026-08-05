package service

import (
	"context"
	"strings"
	"time"

	"ai-video/internal/domain"
	"ai-video/internal/gen/model"
	"ai-video/internal/repository"

	"gorm.io/gorm"
)

type AppUserService struct{ repo *repository.AppUserRepo }

func NewAppUserService() *AppUserService { return &AppUserService{repo: repository.NewAppUserRepo()} }

type CreateAppUserRequest struct {
	DeviceCode               string     `json:"device_code" binding:"required,max=128"`
	Username                 string     `json:"username" binding:"required,max=128"`
	ClientCountry            string     `json:"client_country" binding:"max=64"`
	ChannelID                string     `json:"channel_id" binding:"max=64"`
	AppVersion               string     `json:"app_version" binding:"max=32"`
	AppName                  string     `json:"app_name" binding:"max=255"`
	FirstOpenedAt            *time.Time `json:"first_opened_at"`
	LastOpenedAt             *time.Time `json:"last_opened_at"`
	LoginType                uint8      `json:"login_type" binding:"omitempty,oneof=1 2 3"`
	LoginAccount             string     `json:"login_account" binding:"max=255"`
	Email                    string     `json:"email" binding:"omitempty,email,max=255"`
	ThirdCode                string     `json:"third_code" binding:"max=50"`
	UserType                 uint8      `json:"user_type" binding:"omitempty,oneof=1 2"`
	ActiveDays               uint       `json:"active_days"`
	ActiveLong               uint64     `json:"active_long"`
	AvgDailyUsageSeconds     uint64     `json:"avg_daily_usage_seconds"`
	VIPStartedAt             *time.Time `json:"vip_started_at"`
	VIPExpiresAt             *time.Time `json:"vip_expires_at"`
	VIPPoints                int64      `json:"vip_points"`
	PointsBalance            int64      `json:"points_balance"`
	SubscriptionStatus       uint8      `json:"subscription_status" binding:"omitempty,oneof=1 2 3 4"`
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
	PointsMoney              uint64     `json:"points_money"`
	AiCotsMoney              float64    `json:"ai_cots_money" binding:"gte=0"`
	Activated                uint       `json:"activated" binding:"oneof=0 1"`
	KeyBehaviorMet           uint       `json:"key_behavior_met" binding:"oneof=0 1"`
	PaymentMet               bool       `json:"payment_met"`
	FirstPaymentMet          bool       `json:"first_payment_met"`
	Registered               bool       `json:"registered"`
	AttributionClickedAt     *time.Time `json:"attribution_clicked_at"`
	PhoneModel               string     `json:"phone_model" binding:"max=128"`
	ReRegisteredFromID       uint64     `json:"re_registered_from_id"`
	PackageCode              string     `json:"package_code" binding:"max=128"`
	IMEI                     string     `json:"imei" binding:"max=50"`
	ServerCountry            string     `json:"server_country" binding:"max=255"`
	Phone                    string     `json:"phone" binding:"max=32"`
	IsFrozen                 bool       `json:"is_frozen"`
	IsBlacklisted            bool       `json:"is_blacklisted"`
	VIPLevel                 uint       `json:"vip_level"`
	Status                   *int8      `json:"status" binding:"omitempty,oneof=0 1 2 3"`
	LastLoginAt              *time.Time `json:"last_login_at"`
	LastLoginIP              string     `json:"last_login_ip" binding:"max=64"`
}

type UpdateAppUserRequest struct {
	DeviceCode               *string    `json:"device_code" binding:"omitempty,min=1,max=128"`
	IMEI                     *string    `json:"imei" binding:"omitempty,max=50"`
	Username                 *string    `json:"username" binding:"omitempty,min=1,max=128"`
	ClientCountry            *string    `json:"client_country" binding:"omitempty,max=64"`
	DeviceCountry            *string    `json:"device_country" binding:"omitempty,max=64"`
	ChannelID                *string    `json:"channel_id" binding:"omitempty,max=64"`
	AppVersion               *string    `json:"app_version" binding:"omitempty,max=32"`
	AppName                  *string    `json:"app_name" binding:"omitempty,max=255"`
	FirstOpenedAt            *time.Time `json:"first_opened_at"`
	LastOpenedAt             *time.Time `json:"last_opened_at"`
	LoginType                *uint8     `json:"login_type" binding:"omitempty,oneof=1 2 3"`
	LoginAccount             *string    `json:"login_account" binding:"omitempty,max=255"`
	Email                    *string    `json:"email" binding:"omitempty,email,max=255"`
	ThirdCode                *string    `json:"third_code" binding:"omitempty,max=50"`
	AppIDEmail               *string    `json:"appid_email" binding:"omitempty,email,max=255"`
	AppIDThirdCode           *string    `json:"appid_third_code" binding:"omitempty,max=50"`
	UserType                 *uint8     `json:"user_type" binding:"omitempty,oneof=1 2"`
	ActiveDays               *uint      `json:"active_days"`
	ActiveLong               *uint64    `json:"active_long"`
	AvgDailyUsageSeconds     *uint64    `json:"avg_daily_usage_seconds"`
	VIPStartedAt             *time.Time `json:"vip_started_at"`
	VIPExpiresAt             *time.Time `json:"vip_expires_at"`
	VIPPoints                *int64     `json:"vip_points"`
	PointsBalance            *int64     `json:"points_balance"`
	SubscriptionStatus       *uint8     `json:"subscription_status" binding:"omitempty,oneof=1 2 3 4"`
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
	PointsMoney              *uint64    `json:"points_money"`
	AiCotsMoney              *float64   `json:"ai_cots_money" binding:"omitempty,gte=0"`
	Activated                *uint      `json:"activated" binding:"omitempty,oneof=0 1"`
	KeyBehaviorMet           *uint      `json:"key_behavior_met" binding:"omitempty,oneof=0 1"`
	PaymentMet               *bool      `json:"payment_met"`
	FirstPaymentMet          *bool      `json:"first_payment_met"`
	Registered               *bool      `json:"registered"`
	AttributionClickedAt     *time.Time `json:"attribution_clicked_at"`
	PhoneModel               *string    `json:"phone_model" binding:"omitempty,max=128"`
	ReRegisteredFromID       *uint64    `json:"re_registered_from_id"`
	PackageCode              *string    `json:"package_code" binding:"omitempty,max=128"`
	ServerCountry            *string    `json:"server_country" binding:"omitempty,max=255"`
	Phone                    *string    `json:"phone" binding:"omitempty,max=32"`
	IsFrozen                 *bool      `json:"is_frozen"`
	IsBlacklisted            *bool      `json:"is_blacklisted"`
	VIPLevel                 *uint      `json:"vip_level"`
	Status                   *int8      `json:"status" binding:"omitempty,oneof=0 1 2 3"`
	LastLoginAt              *time.Time `json:"last_login_at"`
	LastLoginIP              *string    `json:"last_login_ip" binding:"omitempty,max=64"`
}

type ListAppUserRequest struct {
	Keyword            string `form:"keyword" binding:"max=255"`
	ClientCountry      string `form:"client_country" binding:"max=64"`
	DeviceCountry      string `form:"device_country" binding:"max=64"`
	ServerCountry      string `form:"server_country" binding:"max=255"`
	ChannelID          string `form:"channel_id" binding:"max=64"`
	AppVersion         string `form:"app_version" binding:"max=32"`
	AppName            string `form:"app_name" binding:"max=255"`
	PackageCode        string `form:"package_code" binding:"max=128"`
	LoginType          uint8  `form:"login_type" binding:"omitempty,oneof=1 2 3"`
	UserType           uint8  `form:"user_type" binding:"omitempty,oneof=1 2"`
	SubscriptionStatus uint8  `form:"subscription_status" binding:"omitempty,oneof=1 2 3 4"`
	Activated          *uint  `form:"activated" binding:"omitempty,oneof=0 1"`
	Registered         *bool  `form:"registered"`
	PaymentMet         *bool  `form:"payment_met"`
	Status             *int8  `form:"status" binding:"omitempty,oneof=0 1 2 3"`
	IsFrozen           *bool  `form:"is_frozen"`
	IsBlacklisted      *bool  `form:"is_blacklisted"`
}

func (s *AppUserService) Create(ctx context.Context, req *CreateAppUserRequest) (*model.VideoUser, error) {
	user := newAppUser(req)
	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func newAppUser(req *CreateAppUserRequest) *model.VideoUser {
	loginType := req.LoginType
	if loginType == 0 {
		loginType = uint8(domain.AppUserLoginGuest)
	}
	userType := req.UserType
	if userType == 0 {
		userType = uint8(domain.AppUserTypeFree)
	}
	subscriptionStatus := req.SubscriptionStatus
	if subscriptionStatus == 0 {
		subscriptionStatus = domain.AppUserSubscriptionNotSubscribed
	}
	status := domain.AppUserStatusNormal
	frozen := req.IsFrozen
	blacklisted := req.IsBlacklisted
	if req.Status != nil {
		status = *req.Status
		switch status {
		case domain.AppUserStatusFrozen:
			frozen, blacklisted = true, false
		case domain.AppUserStatusBlacklisted:
			frozen, blacklisted = false, true
		default:
			frozen, blacklisted = false, false
		}
	} else {
		status = appUserStatus(frozen, blacklisted)
	}
	appName := strings.TrimSpace(req.AppName)
	if appName == "" {
		appName = "0"
	}
	user := &model.VideoUser{
		DeviceCode: strings.TrimSpace(req.DeviceCode), IMEI: strings.TrimSpace(req.IMEI),
		Username:      strings.TrimSpace(req.Username),
		ClientCountry: strings.TrimSpace(req.ClientCountry), ChannelID: strings.TrimSpace(req.ChannelID),
		ServerCountry: strings.TrimSpace(req.ServerCountry), AppVersion: strings.TrimSpace(req.AppVersion),
		AppName: appName, PackageCode: strings.TrimSpace(req.PackageCode), Phone: strings.TrimSpace(req.Phone),
		FirstOpenedAt: req.FirstOpenedAt, LastOpenedAt: req.LastOpenedAt,
		LoginType: loginType, LoginAccount: strings.TrimSpace(req.LoginAccount), UserType: userType,
		ActiveDays: req.ActiveDays, ActiveLong: req.ActiveLong, AvgDailyUsageSeconds: req.AvgDailyUsageSeconds,
		VIPStartedAt: req.VIPStartedAt, VipExpiresAt: req.VIPExpiresAt,
		VipPoints: req.VIPPoints, PointsBalance: req.PointsBalance, SubscriptionStatus: subscriptionStatus,
		FirstOrderCreatedAt: req.FirstOrderCreatedAt, FirstPaidAt: req.FirstPaidAt,
		OrderCount: req.OrderCount, PaymentCount: req.PaymentCount,
		SubscriptionPaymentCount: req.SubscriptionPaymentCount, OneTimePaymentCount: req.OneTimePaymentCount,
		OrderAmountMoney: req.OrderAmountMoney, ActualAmountMoney: req.ActualAmountMoney,
		LastPaidAt: req.LastPaidAt, RefundAmountMoney: req.RefundAmountMoney,
		PointsMoney: req.PointsMoney, AiCotsMoney: req.AiCotsMoney,
		Activated: req.Activated, KeyBehaviorMet: req.KeyBehaviorMet,
		PaymentMet: boolInt8(req.PaymentMet), FirstPaymentMet: boolInt8(req.FirstPaymentMet),
		Registered: boolInt8(req.Registered), AttributionClickedAt: req.AttributionClickedAt,
		PhoneModel: strings.TrimSpace(req.PhoneModel), ReRegisteredFromID: req.ReRegisteredFromID,
		IsFrozen: boolInt8(frozen), IsBlacklisted: boolInt8(blacklisted),
		VIPLevel: req.VIPLevel, Status: status,
		LastLoginAt: req.LastLoginAt, LastLoginIP: strings.TrimSpace(req.LastLoginIP),
	}
	user.Email = nullableString(req.Email)
	user.ThirdCode = nullableString(req.ThirdCode)
	return user
}

func (s *AppUserService) GetByID(ctx context.Context, id uint64) (*model.VideoUser, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, notFoundOr(err, "客户端用户不存在")
	}
	return user, nil
}

func (s *AppUserService) Update(ctx context.Context, id uint64, req *UpdateAppUserRequest) (*model.VideoUser, error) {
	user, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	updates := make(map[string]interface{})
	setTrimmedAppUserUpdate(updates, "device_code", req.DeviceCode)
	setTrimmedAppUserUpdate(updates, "imei", req.IMEI)
	setTrimmedAppUserUpdate(updates, "username", req.Username)
	clientCountry := req.ClientCountry
	if clientCountry == nil {
		clientCountry = req.DeviceCountry
	}
	setTrimmedAppUserUpdate(updates, "client_country", clientCountry)
	setTrimmedAppUserUpdate(updates, "server_country", req.ServerCountry)
	setTrimmedAppUserUpdate(updates, "channel_id", req.ChannelID)
	setTrimmedAppUserUpdate(updates, "app_version", req.AppVersion)
	setTrimmedAppUserUpdate(updates, "app_name", req.AppName)
	setTrimmedAppUserUpdate(updates, "package_code", req.PackageCode)
	setTrimmedAppUserUpdate(updates, "phone", req.Phone)
	setAppUserUpdate(updates, "first_opened_at", req.FirstOpenedAt)
	setAppUserUpdate(updates, "last_opened_at", req.LastOpenedAt)
	setAppUserUpdate(updates, "login_type", req.LoginType)
	setTrimmedAppUserUpdate(updates, "login_account", req.LoginAccount)
	email := req.Email
	if email == nil {
		email = req.AppIDEmail
	}
	thirdCode := req.ThirdCode
	if thirdCode == nil {
		thirdCode = req.AppIDThirdCode
	}
	setNullableAppUserUpdate(updates, "email", email)
	setNullableAppUserUpdate(updates, "third_code", thirdCode)
	setAppUserUpdate(updates, "user_type", req.UserType)
	setAppUserUpdate(updates, "active_days", req.ActiveDays)
	setAppUserUpdate(updates, "active_long", req.ActiveLong)
	setAppUserUpdate(updates, "avg_daily_usage_seconds", req.AvgDailyUsageSeconds)
	setAppUserUpdate(updates, "vip_started_at", req.VIPStartedAt)
	setAppUserUpdate(updates, "vip_expires_at", req.VIPExpiresAt)
	setAppUserUpdate(updates, "vip_points", req.VIPPoints)
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
	setAppUserUpdate(updates, "attribution_clicked_at", req.AttributionClickedAt)
	setTrimmedAppUserUpdate(updates, "phone_model", req.PhoneModel)
	setAppUserUpdate(updates, "re_registered_from_id", req.ReRegisteredFromID)
	setAppUserUpdate(updates, "vip_level", req.VIPLevel)
	setAppUserUpdate(updates, "last_login_at", req.LastLoginAt)
	setTrimmedAppUserUpdate(updates, "last_login_ip", req.LastLoginIP)
	setBoolAppUserUpdate(updates, "payment_met", req.PaymentMet)
	setBoolAppUserUpdate(updates, "first_payment_met", req.FirstPaymentMet)
	setBoolAppUserUpdate(updates, "registered", req.Registered)
	if req.Status != nil || req.IsFrozen != nil || req.IsBlacklisted != nil {
		status := user.Status
		frozen := user.IsFrozen != 0
		blacklisted := user.IsBlacklisted != 0
		if req.Status != nil {
			status = *req.Status
			switch status {
			case domain.AppUserStatusFrozen:
				frozen, blacklisted = true, false
			case domain.AppUserStatusBlacklisted:
				frozen, blacklisted = false, true
			default:
				frozen, blacklisted = false, false
			}
		} else {
			if req.IsFrozen != nil {
				frozen = *req.IsFrozen
			}
			if req.IsBlacklisted != nil {
				blacklisted = *req.IsBlacklisted
			}
			status = appUserStatus(frozen, blacklisted)
		}
		frozenValue := boolInt8(frozen)
		blacklistedValue := boolInt8(blacklisted)
		updates["status"] = status
		updates["is_frozen"] = frozenValue
		updates["is_blacklisted"] = blacklistedValue
		if status != user.Status || frozenValue != user.IsFrozen || blacklistedValue != user.IsBlacklisted {
			updates["token_version"] = gorm.Expr("token_version + 1")
		}
	}
	if err := s.repo.Update(ctx, id, updates); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}

func setBoolAppUserUpdate(updates map[string]interface{}, column string, value *bool) {
	if value != nil {
		updates[column] = boolInt8(*value)
	}
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

func boolInt8(value bool) int8 {
	if value {
		return 1
	}
	return 0
}

func appUserStatus(frozen, blacklisted bool) int8 {
	if blacklisted {
		return domain.AppUserStatusBlacklisted
	}
	if frozen {
		return domain.AppUserStatusFrozen
	}
	return domain.AppUserStatusNormal
}

func (s *AppUserService) Delete(ctx context.Context, id uint64) error {
	if _, err := s.GetByID(ctx, id); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

func (s *AppUserService) List(ctx context.Context, page, pageSize int, req *ListAppUserRequest) ([]model.VideoUser, int64, error) {
	clientCountry := strings.TrimSpace(req.ClientCountry)
	if clientCountry == "" {
		clientCountry = strings.TrimSpace(req.DeviceCountry)
	}
	return s.repo.PageList(ctx, page, pageSize, &repository.AppUserListFilter{
		Keyword: strings.TrimSpace(req.Keyword), ClientCountry: clientCountry,
		ServerCountry: strings.TrimSpace(req.ServerCountry), ChannelID: strings.TrimSpace(req.ChannelID),
		AppVersion: strings.TrimSpace(req.AppVersion), AppName: strings.TrimSpace(req.AppName),
		PackageCode: strings.TrimSpace(req.PackageCode), LoginType: req.LoginType,
		UserType: req.UserType, SubscriptionStatus: req.SubscriptionStatus,
		Activated: req.Activated, Registered: req.Registered, PaymentMet: req.PaymentMet, Status: req.Status,
		IsFrozen: req.IsFrozen, IsBlacklisted: req.IsBlacklisted,
	})
}
