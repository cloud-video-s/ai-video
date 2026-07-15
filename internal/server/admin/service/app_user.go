package service

import (
	"context"
	"time"

	"ai-video/internal/model"
	"ai-video/internal/repository"
)

type AppUserService struct {
	repo *repository.AppUserRepo
}

func NewAppUserService() *AppUserService {
	return &AppUserService{repo: repository.NewAppUserRepo()}
}

type CreateAppUserRequest struct {
	Username                 string     `json:"username" binding:"required,max=128"`
	PhoneCode                string     `json:"phone_code" binding:"required,max=128"`
	Email                    string     `json:"email" binding:"omitempty,email,max=255"`
	EmailVerified            bool       `json:"email_verified"`
	DeviceCountry            string     `json:"device_country" binding:"max=64"`
	IPCountry                string     `json:"ip_country" binding:"max=8"`
	ChannelID                string     `json:"channel_id" binding:"max=64"`
	AppVersion               string     `json:"app_version" binding:"max=32"`
	FirstOpenedAt            *time.Time `json:"first_opened_at"`
	LastOpenedAt             *time.Time `json:"last_opened_at"`
	LoginType                string     `json:"login_type" binding:"omitempty,oneof=guest google apple"`
	LoginAccount             string     `json:"login_account" binding:"max=255"`
	UserType                 string     `json:"user_type" binding:"omitempty,oneof=free paid"`
	ActiveDays               uint32     `json:"active_days"`
	AvgDailyUsageSeconds     uint64     `json:"avg_daily_usage_seconds"`
	VIPExpiresAt             *time.Time `json:"vip_expires_at"`
	PointsBalance            int64      `json:"points_balance"`
	SubscriptionStatus       string     `json:"subscription_status" binding:"omitempty,oneof=subscribed not_subscribed cancelled"`
	FirstOrderCreatedAt      *time.Time `json:"first_order_created_at"`
	FirstPaidAt              *time.Time `json:"first_paid_at"`
	OrderCount               uint64     `json:"order_count"`
	PaymentCount             uint64     `json:"payment_count"`
	SubscriptionPaymentCount uint64     `json:"subscription_payment_count"`
	OneTimePaymentCount      uint64     `json:"one_time_payment_count"`
	OrderAmountCents         int64      `json:"order_amount_cents" binding:"gte=0"`
	ActualAmountCents        int64      `json:"actual_amount_cents" binding:"gte=0"`
	LastPaidAt               *time.Time `json:"last_paid_at"`
	RefundAmountCents        int64      `json:"refund_amount_cents" binding:"gte=0"`
	PointsCost               uint64     `json:"points_cost"`
	AIVersion                string     `json:"ai_version" binding:"max=32"`
	Activated                bool       `json:"activated"`
	KeyBehaviorMet           bool       `json:"key_behavior_met"`
	PaymentMet               bool       `json:"payment_met"`
	FirstPaymentMet          bool       `json:"first_payment_met"`
	Registered               bool       `json:"registered"`
	AttributionClickedAt     *time.Time `json:"attribution_clicked_at"`
	PhoneModel               string     `json:"phone_model" binding:"max=128"`
}

type UpdateAppUserRequest struct {
	Username                 *string    `json:"username" binding:"omitempty,min=1,max=128"`
	PhoneCode                *string    `json:"phone_code" binding:"omitempty,min=1,max=128"`
	Email                    *string    `json:"email" binding:"omitempty,email,max=255"`
	EmailVerified            *bool      `json:"email_verified"`
	DeviceCountry            *string    `json:"device_country" binding:"omitempty,max=64"`
	IPCountry                *string    `json:"ip_country" binding:"omitempty,max=8"`
	ChannelID                *string    `json:"channel_id" binding:"omitempty,max=64"`
	AppVersion               *string    `json:"app_version" binding:"omitempty,max=32"`
	FirstOpenedAt            *time.Time `json:"first_opened_at"`
	LastOpenedAt             *time.Time `json:"last_opened_at"`
	LoginType                *string    `json:"login_type" binding:"omitempty,oneof=guest google apple"`
	LoginAccount             *string    `json:"login_account" binding:"omitempty,max=255"`
	UserType                 *string    `json:"user_type" binding:"omitempty,oneof=free paid"`
	ActiveDays               *uint32    `json:"active_days"`
	AvgDailyUsageSeconds     *uint64    `json:"avg_daily_usage_seconds"`
	VIPExpiresAt             *time.Time `json:"vip_expires_at"`
	PointsBalance            *int64     `json:"points_balance"`
	SubscriptionStatus       *string    `json:"subscription_status" binding:"omitempty,oneof=subscribed not_subscribed cancelled"`
	FirstOrderCreatedAt      *time.Time `json:"first_order_created_at"`
	FirstPaidAt              *time.Time `json:"first_paid_at"`
	OrderCount               *uint64    `json:"order_count"`
	PaymentCount             *uint64    `json:"payment_count"`
	SubscriptionPaymentCount *uint64    `json:"subscription_payment_count"`
	OneTimePaymentCount      *uint64    `json:"one_time_payment_count"`
	OrderAmountCents         *int64     `json:"order_amount_cents" binding:"omitempty,gte=0"`
	ActualAmountCents        *int64     `json:"actual_amount_cents" binding:"omitempty,gte=0"`
	LastPaidAt               *time.Time `json:"last_paid_at"`
	RefundAmountCents        *int64     `json:"refund_amount_cents" binding:"omitempty,gte=0"`
	PointsCost               *uint64    `json:"points_cost"`
	AIVersion                *string    `json:"ai_version" binding:"omitempty,max=32"`
	Activated                *bool      `json:"activated"`
	KeyBehaviorMet           *bool      `json:"key_behavior_met"`
	PaymentMet               *bool      `json:"payment_met"`
	FirstPaymentMet          *bool      `json:"first_payment_met"`
	Registered               *bool      `json:"registered"`
	AttributionClickedAt     *time.Time `json:"attribution_clicked_at"`
	PhoneModel               *string    `json:"phone_model" binding:"omitempty,max=128"`
}

type ListAppUserRequest struct {
	Keyword            string `form:"keyword" binding:"max=255"`
	DeviceCountry      string `form:"device_country" binding:"max=64"`
	IPCountry          string `form:"ip_country" binding:"max=8"`
	ChannelID          string `form:"channel_id" binding:"max=64"`
	AppVersion         string `form:"app_version" binding:"max=32"`
	LoginType          string `form:"login_type" binding:"omitempty,oneof=guest google apple"`
	UserType           string `form:"user_type" binding:"omitempty,oneof=free paid"`
	SubscriptionStatus string `form:"subscription_status" binding:"omitempty,oneof=subscribed not_subscribed cancelled"`
	Activated          *bool  `form:"activated"`
	Registered         *bool  `form:"registered"`
	PaymentMet         *bool  `form:"payment_met"`
}

func (s *AppUserService) Create(ctx context.Context, req *CreateAppUserRequest) (*model.VideoUser, error) {
	loginType := req.LoginType
	if loginType == "" {
		loginType = model.AppUserLoginGuest
	}
	userType := req.UserType
	if userType == "" {
		userType = model.AppUserTypeFree
	}
	subscriptionStatus := req.SubscriptionStatus
	if subscriptionStatus == "" {
		subscriptionStatus = model.AppUserSubscriptionNotSubscribed
	}

	user := &model.VideoUser{
		Username: req.Username, PhoneCode: req.PhoneCode, DeviceCountry: req.DeviceCountry, IPCountry: req.IPCountry,
		ChannelID: req.ChannelID, AppVersion: req.AppVersion, FirstOpenedAt: req.FirstOpenedAt,
		LastOpenedAt: req.LastOpenedAt, LoginType: loginType, LoginAccount: req.LoginAccount,
		UserType: userType, ActiveDays: req.ActiveDays, AvgDailyUsageSeconds: req.AvgDailyUsageSeconds,
		VIPExpiresAt: req.VIPExpiresAt, PointsBalance: req.PointsBalance, SubscriptionStatus: subscriptionStatus,
		FirstOrderCreatedAt: req.FirstOrderCreatedAt, FirstPaidAt: req.FirstPaidAt, OrderCount: req.OrderCount,
		PaymentCount: req.PaymentCount, SubscriptionPaymentCount: req.SubscriptionPaymentCount,
		OneTimePaymentCount: req.OneTimePaymentCount, OrderAmountCents: req.OrderAmountCents,
		ActualAmountCents: req.ActualAmountCents, LastPaidAt: req.LastPaidAt,
		RefundAmountCents: req.RefundAmountCents, PointsCost: req.PointsCost, AIVersion: req.AIVersion,
		Activated: req.Activated, KeyBehaviorMet: req.KeyBehaviorMet, PaymentMet: req.PaymentMet,
		FirstPaymentMet: req.FirstPaymentMet, Registered: req.Registered,
		AttributionClickedAt: req.AttributionClickedAt, PhoneModel: req.PhoneModel,
	}
	if req.Email != "" {
		user.Email = &req.Email
		user.EmailVerified = req.EmailVerified
	}
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
	setAppUserUpdate(updates, "username", req.Username)
	setAppUserUpdate(updates, "phone_code", req.PhoneCode)
	setAppUserUpdate(updates, "email", req.Email)
	setAppUserUpdate(updates, "email_verified", req.EmailVerified)
	setAppUserUpdate(updates, "device_country", req.DeviceCountry)
	setAppUserUpdate(updates, "ip_country", req.IPCountry)
	setAppUserUpdate(updates, "channel_id", req.ChannelID)
	setAppUserUpdate(updates, "app_version", req.AppVersion)
	setAppUserUpdate(updates, "first_opened_at", req.FirstOpenedAt)
	setAppUserUpdate(updates, "last_opened_at", req.LastOpenedAt)
	setAppUserUpdate(updates, "login_type", req.LoginType)
	setAppUserUpdate(updates, "login_account", req.LoginAccount)
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
	setAppUserUpdate(updates, "order_amount_cents", req.OrderAmountCents)
	setAppUserUpdate(updates, "actual_amount_cents", req.ActualAmountCents)
	setAppUserUpdate(updates, "last_paid_at", req.LastPaidAt)
	setAppUserUpdate(updates, "refund_amount_cents", req.RefundAmountCents)
	setAppUserUpdate(updates, "points_cost", req.PointsCost)
	setAppUserUpdate(updates, "ai_version", req.AIVersion)
	setAppUserUpdate(updates, "activated", req.Activated)
	setAppUserUpdate(updates, "key_behavior_met", req.KeyBehaviorMet)
	setAppUserUpdate(updates, "payment_met", req.PaymentMet)
	setAppUserUpdate(updates, "first_payment_met", req.FirstPaymentMet)
	setAppUserUpdate(updates, "registered", req.Registered)
	setAppUserUpdate(updates, "attribution_clicked_at", req.AttributionClickedAt)
	setAppUserUpdate(updates, "phone_model", req.PhoneModel)

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

func (s *AppUserService) Delete(ctx context.Context, id uint64) error {
	if _, err := s.GetByID(ctx, id); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

func (s *AppUserService) List(ctx context.Context, page, pageSize int, req *ListAppUserRequest) ([]model.VideoUser, int64, error) {
	return s.repo.PageList(ctx, page, pageSize, &repository.AppUserListFilter{
		Keyword: req.Keyword, DeviceCountry: req.DeviceCountry, IPCountry: req.IPCountry,
		ChannelID: req.ChannelID, AppVersion: req.AppVersion, LoginType: req.LoginType,
		UserType: req.UserType, SubscriptionStatus: req.SubscriptionStatus,
		Activated: req.Activated, Registered: req.Registered, PaymentMet: req.PaymentMet,
	})
}
