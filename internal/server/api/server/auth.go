package service

import (
	"ai-video/internal/config"
	"ai-video/internal/pkg/utils"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"ai-video/internal/domain"
	"ai-video/internal/gen/model"
	"ai-video/internal/pkg/cache"
	"ai-video/internal/pkg/jwt"
	"ai-video/internal/pkg/oidc"
	"ai-video/internal/pkg/setting"
	"ai-video/internal/repository"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AuthService struct {
	userRepo          *repository.AppUserRepo
	orderRepo         *repository.OrderRepo
	attributionRepo   *repository.UserAttributionRepo
	identityRepo      *repository.UserIdentityRepo
	identityVerifiers map[string]identityTokenVerifier
}

type identityTokenVerifier interface {
	Verify(ctx context.Context, rawToken, expectedNonce string) (*oidc.Identity, error)
}

func NewAuthService() *AuthService {
	authConfig := config.Cfg.ThirdPartyAuth
	timeout := time.Duration(authConfig.HTTPTimeoutMS) * time.Millisecond
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	cacheTTL := time.Duration(authConfig.JWKSCacheSeconds) * time.Second
	return &AuthService{
		userRepo: repository.NewAppUserRepo(), orderRepo: repository.NewOrderRepo(),
		attributionRepo: repository.NewUserAttributionRepo(),
		identityRepo:    repository.NewUserIdentityRepo(),
		identityVerifiers: map[string]identityTokenVerifier{
			domain.IdentityProviderGoogle: oidc.NewVerifier(oidc.Config{Issuers: authConfig.Google.Issuers, Audiences: authConfig.Google.ClientIDs, JWKSURL: authConfig.Google.JWKSURL, HTTPClient: &http.Client{Timeout: timeout}, CacheTTL: cacheTTL}),
			domain.IdentityProviderApple:  oidc.NewVerifier(oidc.Config{Issuers: authConfig.Apple.Issuers, Audiences: authConfig.Apple.ClientIDs, JWKSURL: authConfig.Apple.JWKSURL, HTTPClient: &http.Client{Timeout: timeout}, CacheTTL: cacheTTL}),
		},
	}
}

type LoginRequest struct {
	DeviceCode string `json:"device_code" binding:"required,max=128"`
	ForceNew   bool   `json:"force_new"`
	AccountBaseRequest
}

type AppleOrderLoginRequest struct {
	OrderCode string `json:"order_code" binding:"required,max=191"`
}

var ErrAuthStateInvalid = errors.New("登录状态已失效，请重新登录")
var ErrAppleOrderNotFound = errors.New("Apple 支付订单不存在")
var ErrAppleOrderUserNotFound = errors.New("Apple 支付订单关联用户不存在")
var ErrAppleOrderUserDisabled = errors.New("Apple 支付订单关联用户已禁用")
var ActiveDayKey = "active_day_key_user_id_"

type AuthResponse struct {
	Token        string `json:"token"`
	LoginType    uint32 `json:"login_type"`
	ExpireAt     int64  `json:"expire_at"`
	TokenVersion int64  `json:"token_version"`
}

type ThirdAuthResponse struct {
	AuthResponse
	Status int `json:"status"`
}

type UserResponse struct {
	ID                 uint64 `json:"id"`
	Email              string `json:"email"`
	DeviceCountry      string `json:"device_country"`      // 国家
	ChannelID          string `json:"channel_id"`          // 渠道id
	LoginType          uint32 `json:"login_type"`          // 登录方式 1=未登录 2=google 3=appid
	UserType           uint32 `json:"user_type"`           // 用户类型 1=免费 2=付费
	SubscriptionStatus uint32 `json:"subscription_status"` // 订阅状态 1未订阅 2订阅中 3=已取消 4=已过期
	VipExpiresAt       int64  `json:"vip_expires_at"`      // vip 到期时间
	PointsBalance      uint64 `json:"points_balance"`      // 积分
	Status             int32  `json:"status"`
	LastLoginAt        int64  `json:"last_login_at"`
	LastLoginIP        string `json:"last_login_ip"`
	LoginAccount       string `json:"login_account"`
}

type UpdateCountryRequest struct {
	Country string `json:"country" binding:"omitempty,max=8"`
}

type ActiveReportingRequest struct {
	TimeLong uint64 `json:"time_long" binding:"required,min=1"`
}

func (s *AuthService) Login(ctx *gin.Context, req *LoginRequest, clientIP string) (*AuthResponse, error) {
	req.DeviceCode = strings.TrimSpace(req.DeviceCode)
	if req.DeviceCode == "" {
		return nil, errors.New("设备标识不能为空")
	}
	GetCtxAccountBaseRequest(ctx, &req.AccountBaseRequest)
	now := time.Now()
	var user *model.VideoUser
	country, _ := utils.GetCountryByIP(utils.ClientIP(ctx))
	err := repository.Transaction(ctx, func(ctx context.Context) error {
		firstOpenedAt := req.FirstOpenedAt
		if firstOpenedAt == nil {
			firstOpenedAt = &now
		}
		lastOpenedAt := req.LastOpenedAt
		if lastOpenedAt == nil {
			lastOpenedAt = &now
		}
		var latest *model.VideoUser
		var err error
		if req.ForceNew {
			latest, err = s.userRepo.GetByDeviceCodeSubscription(ctx, req.DeviceCode, true)
		} else {
			latest, err = s.userRepo.GetByDeviceCode(ctx, req.DeviceCode, true)
		}
		isTrue := false
		if errors.Is(err, gorm.ErrRecordNotFound) {
			isTrue = true
		} else {
			if req.ForceNew && latest.Status != 1 {
				return errors.New("当前设备账号已停用")
			}
		}
		if isTrue {
			user = &model.VideoUser{
				DeviceCode: req.DeviceCode,
				Username:   newGuestUsername(), LoginType: uint8(domain.AppUserLoginGuest),
				UserType: uint8(domain.AppUserTypeFree), SubscriptionStatus: domain.AppUserSubscriptionNotSubscribed,
				ClientCountry: req.ClientCountry,
				AppVersion:    req.AppVersion, AppName: req.AppName, PhoneModel: req.PhoneModel,
				FirstOpenedAt: firstOpenedAt, LastOpenedAt: lastOpenedAt,
				AttributionClickedAt: req.AttributionClickedAt, Activated: 1, Registered: 1,
				Status: 1, LastLoginAt: &now, LastLoginIP: clientIP, ServerCountry: country,
				ActiveDays: 1,
			}
			if err = s.userRepo.Create(ctx, user); err != nil {
				return err
			}
			user, err = s.prepareLoginSession(ctx, user.ID)
		} else {
			if err = s.userRepo.Update(ctx, latest.ID, baseTrackingUpdates(ctx, latest, domain.AppUserLoginGuest, &req.AccountBaseRequest, clientIP, now)); err != nil {
				return err
			}
			user, err = s.prepareLoginSession(ctx, latest.ID)
		}
		return err
	})
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			latest, lookupErr := s.userRepo.GetByDeviceCode(ctx, req.DeviceCode, false)
			if lookupErr == nil {
				latest, lookupErr = s.prepareLoginSession(ctx, latest.ID)
				if lookupErr == nil {
					return issueToken(latest, int(latest.LoginType))
				}
			}
		}
		return nil, err
	}
	return issueToken(user, domain.AppUserLoginGuest)
}

// LoginByAppleOrder resolves the user linked to an Apple original transaction
// and issues the same client JWT used by the other login flows.
func (s *AuthService) LoginByAppleOrder(ctx context.Context, req *AppleOrderLoginRequest) (*AuthResponse, error) {
	originalTransactionID := strings.TrimSpace(req.OrderCode)
	if originalTransactionID == "" {
		return nil, ErrAppleOrderNotFound
	}

	order, err := s.orderRepo.GetByAppleOriginalTransactionID(ctx, originalTransactionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAppleOrderNotFound
		}
		return nil, err
	}

	user, err := s.userRepo.GetByID(ctx, order.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAppleOrderUserNotFound
		}
		return nil, err
	}
	if user.Status != 1 {
		return nil, ErrAppleOrderUserDisabled
	}

	user, err = s.prepareLoginSession(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	return issueToken(user, int(user.LoginType))
}

func (s *AuthService) prepareLoginSession(ctx context.Context, userID uint64) (*model.VideoUser, error) {
	if setting.GetBool(setting.UserSingleDeviceLoginKey) {
		if err := s.userRepo.IncrementTokenVersion(ctx, userID); err != nil {
			return nil, err
		}
	}
	return s.userRepo.GetByID(ctx, userID)
}

func (s *AuthService) Logout(token string) error {
	claims, err := jwt.ParseApiToken(token)
	if err != nil || claims.ExpiresAt == nil {
		return nil
	}
	return blacklistAPIToken(token, claims.ExpiresAt.Time)
}

func (s *AuthService) Refresh(ctx context.Context, userID uint64, tokenVersion int64, currentToken string) (*AuthResponse, error) {
	claims, err := jwt.ParseApiToken(currentToken)
	if err != nil || claims.ExpiresAt == nil || claims.UserID != userID || claims.TokenVersion != tokenVersion {
		return nil, ErrAuthStateInvalid
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAuthStateInvalid
		}
		return nil, err
	}
	if user.Status != 1 || user.TokenVersion != tokenVersion || user.DeviceCode != claims.DeviceCode {
		return nil, ErrAuthStateInvalid
	}

	result, err := issueToken(user, int(user.LoginType))
	if err != nil {
		return nil, err
	}
	if err := blacklistAPIToken(currentToken, claims.ExpiresAt.Time); err != nil {
		return nil, fmt.Errorf("刷新客户端 Token 失败: %w", err)
	}
	return result, nil
}

func (s *AuthService) GetProfile(ctx context.Context, userID uint64) (*UserResponse, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户不存在")
		}
		return nil, err
	}
	data := &UserResponse{
		ID:                 user.ID,
		Email:              user.Email,
		DeviceCountry:      user.ClientCountry,
		ChannelID:          user.ChannelID,
		LoginType:          uint32(user.LoginType),
		UserType:           uint32(user.UserType),
		PointsBalance:      uint64(user.PointsBalance + user.VipPoints),
		SubscriptionStatus: uint32(user.SubscriptionStatus),
		Status:             int32(user.Status),
		LastLoginIP:        user.LastLoginIP,
		LoginAccount:       user.LoginAccount,
	}
	if user.VipExpiresAt != nil {
		data.VipExpiresAt = user.VipExpiresAt.Unix()
	}
	if user.LastLoginAt != nil {
		data.LastLoginAt = user.LastLoginAt.Unix()
	}
	return data, nil
}

func (s *AuthService) UpdateCountry(ctx context.Context, userID uint64, req *UpdateCountryRequest, clientIP, countryHeader string) (*UserResponse, error) {
	deviceCountry := normalizeCountry(req.Country)
	ipCountry, lookupErr := ResolveCountry(ctx, clientIP, countryHeader)
	if deviceCountry == "" && ipCountry == "" {
		if lookupErr != nil {
			return nil, lookupErr
		}
		return nil, errors.New("客户端未提供国家，且无法根据 IP 获取国家")
	}
	if deviceCountry == "" {
		deviceCountry = ipCountry
	}
	updates := map[string]interface{}{"device_country": deviceCountry, "last_login_ip": clientIP}
	if err := s.userRepo.Update(ctx, userID, updates); err != nil {
		return nil, err
	}
	if err := s.attributionRepo.UpsertDevice(ctx, userID, map[string]interface{}{"ip": clientIP}); err != nil {
		return nil, err
	}
	return s.GetProfile(ctx, userID)
}

func (s *AuthService) SaveUserActiveLong(ctx context.Context, userID uint64, req *ActiveReportingRequest) (interface{}, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户不存在")
		}
		return nil, err
	}
	err = s.userRepo.SaveUserActiveLong(ctx, user.ID, req.TimeLong)
	return nil, err
}

func issueToken(user *model.VideoUser, loginType int) (*AuthResponse, error) {
	token, err := jwt.GenerateApiToken(user.ID, user.DeviceCode, user.TokenVersion, uint32(loginType))
	if err != nil {
		return nil, fmt.Errorf("生成客户端 Token 失败: %w", err)
	}
	cfg := config.Cfg.ApiJwt
	return &AuthResponse{
		Token: token, LoginType: uint32(loginType), ExpireAt: time.Now().Add(time.Duration(cfg.Expire) * time.Second).Unix(), TokenVersion: user.TokenVersion,
	}, nil
}

func blacklistAPIToken(token string, expiresAt time.Time) error {
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return nil
	}
	return cache.BlacklistToken(token, ttl)
}

func baseTrackingUpdates(ctx context.Context, user *model.VideoUser, loginType int, req *AccountBaseRequest, clientIP string, now time.Time) map[string]any {
	updates := map[string]any{"last_opened_at": now, "last_login_at": now, "last_login_ip": clientIP,
		"client_country": req.ClientCountry,
		"app_name":       req.AppName,
		"phone_model":    req.PhoneModel,
		"login_type":     loginType,
	}
	if req.LastOpenedAt != nil {
		updates["last_opened_at"] = *req.LastOpenedAt
	}
	if config.Redis.Get(ctx, fmt.Sprintf("%s%d", ActiveDayKey, user.ID)).Val() == "" {
		updates["active_days"] = user.ActiveDays + 1
	}
	return updates
}

func newGuestUsername() string {
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err == nil {
		return "guest_" + hex.EncodeToString(randomBytes)
	}
	return fmt.Sprintf("guest_%d", time.Now().UnixNano())
}

func thirdPartyLoginBinding(provider string, clientIP, serverCountry string, now time.Time) map[string]any {
	updates := map[string]any{
		"login_type":     providerLoginType(provider),
		"registered":     true,
		"last_login_ip":  clientIP,
		"last_login_at":  now,
		"server_country": serverCountry,
	}
	return updates
}

func (s *AuthService) CheckUserVIP(ctx *gin.Context, userID uint64) bool {
	user, err := repository.NewAppUserRepo().GetByID(ctx, userID)
	if err != nil {
		return false
	}
	if user.SubscriptionStatus != 2 {
		return false
	}
	if user.VipExpiresAt == nil || user.VipExpiresAt.Unix() <= time.Now().Unix() {
		return false
	}
	return true
}
