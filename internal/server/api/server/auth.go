package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"ai-video/internal/model"
	"ai-video/internal/pkg/cache"
	"ai-video/internal/pkg/jwt"
	"ai-video/internal/repository"

	"gorm.io/gorm"
)

type AuthService struct {
	userRepo *repository.AppUserRepo
}

func NewAuthService() *AuthService {
	return &AuthService{userRepo: repository.NewAppUserRepo()}
}

type AccountBaseRequest struct {
	DeviceCountry        string     `json:"device_country" binding:"omitempty,max=64"`
	ChannelID            string     `json:"channel_id" binding:"omitempty,max=64"`
	AppVersion           string     `json:"app_version" binding:"omitempty,max=32"`
	AIVersion            string     `json:"ai_version" binding:"omitempty,max=32"`
	PhoneModel           string     `json:"phone_model" binding:"omitempty,max=128"`
	FirstOpenedAt        *time.Time `json:"first_opened_at"`
	LastOpenedAt         *time.Time `json:"last_opened_at"`
	AttributionClickedAt *time.Time `json:"attribution_clicked_at"`
}

type DeviceRegisterRequest struct {
	PhoneCode string `json:"phone_code" binding:"required,max=128"`
	ForceNew  bool   `json:"force_new"`
	AccountBaseRequest
}

type AuthResponse struct {
	Token string           `json:"token"`
	User  *model.VideoUser `json:"user"`
}

type UpdateCountryRequest struct {
	Country string `json:"country" binding:"omitempty,max=8"`
}

func (s *AuthService) RegisterDevice(ctx context.Context, req *DeviceRegisterRequest, clientIP, countryHeader string) (*AuthResponse, error) {
	req.PhoneCode = strings.TrimSpace(req.PhoneCode)
	if req.PhoneCode == "" {
		return nil, errors.New("设备标识不能为空")
	}

	ipCountry, _ := ResolveCountry(ctx, clientIP, countryHeader)
	now := time.Now()
	var user *model.VideoUser
	err := repository.Transaction(ctx, func(ctx context.Context) error {
		latest, err := s.userRepo.GetLatestByPhoneCode(ctx, req.PhoneCode, true)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		createNew := errors.Is(err, gorm.ErrRecordNotFound) || req.ForceNew
		if latest != nil {
			if latest.Status != 1 {
				return errors.New("当前设备账号已停用")
			}
			createNew = createNew || latest.Email != nil || latest.Registered
		}

		if !createNew {
			updates := baseTrackingUpdates(&req.AccountBaseRequest, clientIP, ipCountry, now)
			if err := s.userRepo.Update(ctx, latest.ID, updates); err != nil {
				return err
			}
			user, err = s.userRepo.GetByID(ctx, latest.ID)
			return err
		}

		registrationNo := uint32(1)
		var previousID *uint64
		if latest != nil {
			registrationNo = latest.RegistrationNo + 1
			id := latest.ID
			previousID = &id
		}

		firstOpenedAt := req.FirstOpenedAt
		if firstOpenedAt == nil {
			firstOpenedAt = &now
		}
		lastOpenedAt := req.LastOpenedAt
		if lastOpenedAt == nil {
			lastOpenedAt = &now
		}
		deviceCountry := normalizeCountry(req.DeviceCountry)
		if deviceCountry == "" {
			deviceCountry = ipCountry
		}

		user = &model.VideoUser{
			PhoneCode: req.PhoneCode, RegistrationNo: registrationNo, ReRegisteredFromID: previousID,
			Username: newGuestUsername(), LoginType: model.AppUserLoginGuest,
			UserType: model.AppUserTypeFree, SubscriptionStatus: model.AppUserSubscriptionNotSubscribed,
			DeviceCountry: deviceCountry, IPCountry: ipCountry, ChannelID: req.ChannelID,
			AppVersion: req.AppVersion, AIVersion: req.AIVersion, PhoneModel: req.PhoneModel,
			FirstOpenedAt: firstOpenedAt, LastOpenedAt: lastOpenedAt,
			AttributionClickedAt: req.AttributionClickedAt, Activated: true, Registered: false,
			Status: 1, LastLoginAt: &now, LastLoginIP: clientIP,
		}
		return s.userRepo.Create(ctx, user)
	})
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			latest, lookupErr := s.userRepo.GetLatestByPhoneCode(ctx, req.PhoneCode, false)
			if lookupErr == nil {
				return issueToken(latest)
			}
		}
		return nil, err
	}
	return issueToken(user)
}

func (s *AuthService) ReRegister(ctx context.Context, req *DeviceRegisterRequest, clientIP, countryHeader string) (*AuthResponse, error) {
	req.ForceNew = true
	return s.RegisterDevice(ctx, req, clientIP, countryHeader)
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

func (s *AuthService) GetProfile(ctx context.Context, userID uint64) (*model.VideoUser, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户不存在")
		}
		return nil, err
	}
	return user, nil
}

func (s *AuthService) UpdateCountry(ctx context.Context, userID uint64, req *UpdateCountryRequest, clientIP, countryHeader string) (*model.VideoUser, error) {
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
	if ipCountry != "" {
		updates["ip_country"] = ipCountry
	}
	if err := s.userRepo.Update(ctx, userID, updates); err != nil {
		return nil, err
	}
	return s.GetProfile(ctx, userID)
}

func issueToken(user *model.VideoUser) (*AuthResponse, error) {
	token, err := jwt.GenerateApiToken(user.ID, user.PhoneCode, user.TokenVersion)
	if err != nil {
		return nil, fmt.Errorf("生成客户端 Token 失败: %w", err)
	}
	return &AuthResponse{Token: token, User: user}, nil
}

func baseTrackingUpdates(req *AccountBaseRequest, clientIP, ipCountry string, now time.Time) map[string]interface{} {
	updates := map[string]interface{}{"last_opened_at": now, "last_login_at": now, "last_login_ip": clientIP, "activated": true}
	if country := normalizeCountry(req.DeviceCountry); country != "" {
		updates["device_country"] = country
	} else if ipCountry != "" {
		updates["device_country"] = ipCountry
	}
	if ipCountry != "" {
		updates["ip_country"] = ipCountry
	}
	if req.ChannelID != "" {
		updates["channel_id"] = req.ChannelID
	}
	if req.AppVersion != "" {
		updates["app_version"] = req.AppVersion
	}
	if req.AIVersion != "" {
		updates["ai_version"] = req.AIVersion
	}
	if req.PhoneModel != "" {
		updates["phone_model"] = req.PhoneModel
	}
	if req.FirstOpenedAt != nil {
		updates["first_opened_at"] = *req.FirstOpenedAt
	}
	if req.LastOpenedAt != nil {
		updates["last_opened_at"] = *req.LastOpenedAt
	}
	if req.AttributionClickedAt != nil {
		updates["attribution_clicked_at"] = *req.AttributionClickedAt
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
