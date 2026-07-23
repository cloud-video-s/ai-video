package service

import (
	"ai-video/internal/config"
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
		userRepo: repository.NewAppUserRepo(), attributionRepo: repository.NewUserAttributionRepo(),
		identityRepo: repository.NewUserIdentityRepo(),
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
	SubscriptionStatus uint32 `json:"subscription_status"` // 订阅状态 1未订阅 2订阅中 3=已取消
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

func (s *AuthService) Login(ctx *gin.Context, req *LoginRequest, clientIP string, userAgent string) (*AuthResponse, error) {
	req.DeviceCode = strings.TrimSpace(req.DeviceCode)
	if req.DeviceCode == "" {
		return nil, errors.New("设备标识不能为空")
	}
	GetCtxAccountBaseRequest(ctx, &req.AccountBaseRequest)
	now := time.Now()
	var user *model.VideoUser
	err := repository.Transaction(ctx, func(ctx context.Context) error {
		firstOpenedAt := req.FirstOpenedAt
		if firstOpenedAt == nil {
			firstOpenedAt = &now
		}
		lastOpenedAt := req.LastOpenedAt
		if lastOpenedAt == nil {
			lastOpenedAt = &now
		}
		if req.ForceNew {
			user = &model.VideoUser{
				DeviceCode: req.DeviceCode,
				Username:   newGuestUsername(), LoginType: uint8(domain.AppUserLoginGuest),
				UserType: uint8(domain.AppUserTypeFree), SubscriptionStatus: domain.AppUserSubscriptionNotSubscribed,
				ClientCountry: req.ClientCountry,
				AppVersion:    req.AppVersion, AppName: req.AppName, PhoneModel: req.PhoneModel,
				FirstOpenedAt: firstOpenedAt, LastOpenedAt: lastOpenedAt,
				AttributionClickedAt: req.AttributionClickedAt, Activated: 1, Registered: 1,
				Status: 1, LastLoginAt: &now, LastLoginIP: clientIP,
			}
			if err := s.userRepo.Create(ctx, user); err != nil {
				return err
			}
			user, _ = s.prepareLoginSession(ctx, user.ID)
			return nil
		}
		latest, err := s.userRepo.GetByDeviceCode(ctx, req.DeviceCode, true)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("账号异常，请稍后重试")
		}
		if latest != nil {
			if latest.Status != 1 || latest.IsFrozen == 1 || latest.IsBlacklisted == 1 {
				return errors.New("当前设备账号已停用")
			}
			updates := baseTrackingUpdates(1, &req.AccountBaseRequest, clientIP, now)
			if err := s.userRepo.Update(ctx, latest.ID, updates); err != nil {
				return err
			}
			if err := s.attributionRepo.UpsertDevice(ctx, latest.ID, attributionTrackingUpdates(&req.AccountBaseRequest, clientIP, userAgent)); err != nil {
				return err
			}
			user, err = s.prepareLoginSession(ctx, latest.ID)
			if err != nil {
				return err
			}
			return nil
		}

		user = &model.VideoUser{
			DeviceCode: req.DeviceCode,
			Username:   newGuestUsername(), LoginType: uint8(domain.AppUserLoginGuest),
			UserType: uint8(domain.AppUserTypeFree), SubscriptionStatus: domain.AppUserSubscriptionNotSubscribed,
			ClientCountry: req.ClientCountry,
			AppVersion:    req.AppVersion, AppName: req.AppPackage, PhoneModel: req.PhoneModel,
			FirstOpenedAt: firstOpenedAt, LastOpenedAt: lastOpenedAt,
			AttributionClickedAt: req.AttributionClickedAt, Activated: 1, Registered: 0,
			Status: 1, LastLoginAt: &now, LastLoginIP: clientIP,
		}
		if err = s.userRepo.Create(ctx, user); err != nil {
			return err
		}
		user, err = s.prepareLoginSession(ctx, user.ID)
		return err
	})
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			latest, lookupErr := s.userRepo.GetByDeviceCode(ctx, req.DeviceCode, false)
			if lookupErr == nil {
				latest, lookupErr = s.prepareLoginSession(ctx, latest.ID)
				if lookupErr == nil {
					return issueToken(latest, uint32(latest.LoginType))
				}
			}
		}
		return nil, err
	}
	return issueToken(user, domain.AppUserLoginGuest)
}

//func (s *AuthService) loginUpdate(ctx *gin.Context, user *model.VideoUser, req *LoginRequest, clientIP string, userAgent string) (*AuthResponse, error) {
//
//}

func (s *AuthService) prepareLoginSession(ctx context.Context, userID uint64) (*model.VideoUser, error) {
	if setting.GetBool(setting.UserSingleDeviceLoginKey) {
		if err := s.userRepo.IncrementTokenVersion(ctx, userID); err != nil {
			return nil, err
		}
	}
	return s.userRepo.GetByID(ctx, userID)
}

func (s *AuthService) ReRegister(ctx *gin.Context, req *LoginRequest, clientIP, userAgent string) (*AuthResponse, error) {
	req.ForceNew = true
	req.DeviceCode = strings.TrimSpace(req.DeviceCode)
	if req.DeviceCode == "" {
		return nil, errors.New("设备标识不能为空")
	}
	GetCtxAccountBaseRequest(ctx, &req.AccountBaseRequest)
	now := time.Now()
	var user *model.VideoUser
	err := repository.Transaction(ctx, func(ctx context.Context) error {
		firstOpenedAt := req.FirstOpenedAt
		if firstOpenedAt == nil {
			firstOpenedAt = &now
		}
		lastOpenedAt := req.LastOpenedAt
		if lastOpenedAt == nil {
			lastOpenedAt = &now
		}

		user = &model.VideoUser{
			DeviceCode: req.DeviceCode,
			Username:   newGuestUsername(), LoginType: uint8(domain.AppUserLoginGuest),
			UserType: uint8(domain.AppUserTypeFree), SubscriptionStatus: domain.AppUserSubscriptionNotSubscribed,
			ClientCountry: req.ClientCountry,
			AppVersion:    req.AppVersion, AppName: req.AppName, PhoneModel: req.PhoneModel,
			FirstOpenedAt: firstOpenedAt, LastOpenedAt: lastOpenedAt,
			AttributionClickedAt: req.AttributionClickedAt, Activated: 1, Registered: 0,
			Status: 1, LastLoginAt: &now, LastLoginIP: clientIP,
		}
		if err := s.userRepo.Create(ctx, user); err != nil {
			return err
		}
		user, _ = s.prepareLoginSession(ctx, user.ID)
		return nil
	})
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			latest, lookupErr := s.userRepo.GetByDeviceCode(ctx, req.DeviceCode, false)
			if lookupErr == nil {
				latest, lookupErr = s.prepareLoginSession(ctx, latest.ID)
				if lookupErr == nil {
					return issueToken(latest, uint32(latest.LoginType))
				}
			}
		}
		return nil, err
	}
	return issueToken(user, domain.AppUserLoginGuest)
}

func (s *AuthService) Logout(token string) error {
	claims, err := jwt.ParseApiToken(token)
	if err != nil || claims.ExpiresAt == nil {
		return nil
	}
	ttl := time.Until(claims.ExpiresAt.Time)
	if ttl <= 0 {
		return nil
	}
	return cache.BlacklistToken(token, ttl)
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
		PointsBalance:      user.PointsBalance,
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

func issueToken(user *model.VideoUser, loginType uint32) (*AuthResponse, error) {
	token, err := jwt.GenerateApiToken(user.ID, user.DeviceCode, user.TokenVersion, loginType)
	if err != nil {
		return nil, fmt.Errorf("生成客户端 Token 失败: %w", err)
	}
	cfg := config.Cfg.ApiJwt
	return &AuthResponse{
		Token: token, LoginType: loginType, ExpireAt: time.Now().Add(time.Duration(cfg.Expire) * time.Second).Unix(), TokenVersion: user.TokenVersion,
	}, nil
}

func baseTrackingUpdates(loginType int, req *AccountBaseRequest, clientIP string, now time.Time) map[string]interface{} {
	updates := map[string]interface{}{"last_opened_at": now, "last_login_at": now, "last_login_ip": clientIP, "activated": uint32(1),
		"client_country": req.ClientCountry,
		"app_name":       req.AppName,
		"phone_model":    req.PhoneModel,
		"login_type":     loginType,
	}
	if req.LastOpenedAt != nil {
		updates["last_opened_at"] = *req.LastOpenedAt
	}
	return updates
}

func attributionTrackingUpdates(req *AccountBaseRequest, clientIP, userAgent string) map[string]interface{} {
	updates := map[string]interface{}{}
	if value := strings.TrimSpace(clientIP); value != "" {
		updates["ip"] = value
	}
	if value := strings.TrimSpace(userAgent); value != "" {
		if len(value) > 1024 {
			value = value[:1024]
		}
		updates["user_agent"] = value
	}
	if req.AttributionClickedAt != nil {
		updates["attributed_at"] = *req.AttributionClickedAt
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

func ThirdPartyLoginBinding(provider string, email, thirdCode, clientIP, serverCountry string, now time.Time) map[string]interface{} {
	updates := map[string]interface{}{
		"login_type":     providerLoginType(provider),
		"login_account":  email,
		"registered":     true,
		"last_login_ip":  clientIP,
		"last_login_at":  now,
		"third_code":     thirdCode,
		"email":          email,
		"server_country": serverCountry,
	}
	return updates
}
