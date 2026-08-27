package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"ai-video/internal/config"
	"ai-video/internal/domain"
	"ai-video/internal/gen/model"
	"ai-video/internal/middleware"
	"ai-video/internal/pkg/adjust"
	"ai-video/internal/pkg/monitor"
	"ai-video/internal/pkg/oidc"
	"ai-video/internal/pkg/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ThirdPartyLoginRequest struct {
	ThirdType     string `json:"third_type" binding:"required,max=50"`
	ThirdCode     string `json:"third_code" binding:"omitempty,max=100"`
	Email         string `json:"email" binding:"omitempty,max=50"`
	IDToken       string `json:"id_token" binding:"omitempty,max=16384"`
	IdentityToken string `json:"identity_token" binding:"omitempty,max=16384"`
	Nonce         string `json:"nonce" binding:"omitempty,max=255"`
	ForceNew      bool   `json:"force_new"`
	AccountBaseRequest
}

type BindIdentityRequest struct {
	IDToken       string `json:"id_token" binding:"omitempty,max=16384"`
	IdentityToken string `json:"identity_token" binding:"omitempty,max=16384"`
	Nonce         string `json:"nonce" binding:"omitempty,max=255"`
	DisplayName   string `json:"display_name" binding:"omitempty,max=128"`
	GivenName     string `json:"given_name" binding:"omitempty,max=128"`
	FamilyName    string `json:"family_name" binding:"omitempty,max=128"`
}

var (
	ErrIdentityProviderNotConfigured = errors.New("third-party identity provider is not configured")
	ErrDeviceCodeNotConfigured       = errors.New("当前设备已绑定另一个同类型第三方账号，是否确认登录？")
)

func (s *AuthService) ThirdPartyLogin(ctx *gin.Context, req *ThirdPartyLoginRequest, clientIP string) (*AuthResponse, error) {
	provider, err := normalizeIdentityProvider(req.ThirdType)
	if err != nil {
		return nil, err
	}
	GetCtxAccountBaseRequest(ctx, &req.AccountBaseRequest)
	if req.ThirdCode == "" {
		identity, err := s.verifyIdentity(ctx, provider, firstToken(req.IDToken, req.IdentityToken), req.Nonce)
		if err != nil {
			return nil, err
		}
		if err := validateIdentityUserColumns(identity); err != nil {
			return nil, err
		}
		req.ThirdCode = identity.Subject
		req.Email = identity.Email
	} else {
		if req.ThirdCode == "" {
			return nil, ErrIdentityProviderNotConfigured
		}
	}
	return s.loginVerifiedIdentity(ctx, req, clientIP)
}

func (s *AuthService) loginVerifiedIdentity(ctx *gin.Context, req *ThirdPartyLoginRequest, clientIP string) (*AuthResponse, error) {
	now := time.Now()
	var user *model.VideoUser
	boundIdentity := false
	apiUserID := middleware.GetAPIUserID(ctx)
	serverCountry := utils.ClientIP(ctx)
	var err error
	user, err = s.userRepo.GetByThirdCode(ctx, req.ThirdCode, true)
	if errors.Is(err, gorm.ErrRecordNotFound) || user == nil {
		if req.Email == "" {
			return nil, ErrIdentityProviderNotConfigured
		}
		user, err = s.userRepo.GetByID(ctx, apiUserID)
		if err != nil {
			return nil, errors.New("user not found")
		}

		if user.ThirdCode != "" && user.ThirdCode != req.ThirdCode {
			return nil, errors.New("当前账号已绑定邮箱")
		}
		firstOpenedAt := req.FirstOpenedAt
		if firstOpenedAt == nil {
			firstOpenedAt = &now
		}
		lastOpenedAt := req.LastOpenedAt
		if lastOpenedAt == nil {
			lastOpenedAt = &now
		}
		user.Email = req.Email
		user.ThirdCode = req.ThirdCode
		user.LoginType = providerLoginType(req.ThirdType)
		user.LastLoginIP = clientIP
		user.LastLoginAt = &now
		user.ServerCountry = serverCountry
		updates := thirdPartyLoginBinding(req.ThirdType, clientIP, serverCountry, now)
		updates["account_binding_time"] = now
		updates["active_days"] = 1
		updates["third_code"] = req.ThirdCode
		updates["email"] = req.Email
		if err := s.userRepo.Update(ctx, user.ID, updates); err != nil {
			monitor.Report(config.Logger(ctx.Request.Context()), monitor.KindError, "third_party_auth", err,
				"user_id", user.ID,
			)
			return nil, errors.New("failed to update third party login info")
		}
		boundIdentity = true
	}

	if user.ID != apiUserID {
		if user.Status != 1 {
			return nil, errors.New("当前邮箱绑定账号已停用，暂时无法使用")
		}
		if err := s.userRepo.Update(ctx, user.ID, baseTrackingUpdates(ctx, user, int(providerLoginType(req.ThirdType)), &req.AccountBaseRequest, clientIP, now)); err != nil {
			return nil, errors.New("failed to update third party login info")
		}
	}
	user, err = s.prepareLoginSession(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	token, err := issueToken(user, int(providerLoginType(req.ThirdType)))
	if err != nil {
		return nil, err
	}
	if user.ID != apiUserID {
		token.DeviceCode = user.DeviceCode
	}
	if boundIdentity {
		enqueueAuthAdjustEvent(ctx.Request.Context(), user.ID, adjust.EventTokenLogin)
	}
	return token, nil
}

func (s *AuthService) verifyIdentity(ctx context.Context, provider, rawToken, nonce string) (*oidc.Identity, error) {
	verifier := s.identityVerifiers[provider]
	if verifier == nil {
		return nil, ErrIdentityProviderNotConfigured
	}
	identity, err := verifier.Verify(ctx, strings.TrimSpace(rawToken), strings.TrimSpace(nonce))
	if err != nil {
		if errors.Is(err, oidc.ErrInvalidToken) {
			return nil, err
		}
		if strings.Contains(err.Error(), "not configured") {
			return nil, fmt.Errorf("%w: %s", ErrIdentityProviderNotConfigured, provider)
		}
		return nil, err
	}
	return identity, nil
}

func normalizeIdentityProvider(provider string) (string, error) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider != domain.IdentityProviderGoogle && provider != domain.IdentityProviderApple {
		return "", errors.New("不支持的第三方登录类型")
	}
	return provider, nil
}

func firstToken(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func providerLoginType(provider string) uint8 {
	if provider == domain.IdentityProviderGoogle {
		return uint8(domain.AppUserLoginGoogle)
	}
	return uint8(domain.AppUserLoginAppID)
}

func validateIdentityUserColumns(identity *oidc.Identity) error {
	if identity == nil || strings.TrimSpace(identity.Subject) == "" {
		return errors.New("第三方账号唯一编码为空")
	}
	if len([]rune(strings.TrimSpace(identity.Subject))) > 50 {
		return errors.New("第三方账号唯一编码长度超过 50 个字符")
	}
	if len([]rune(strings.TrimSpace(identity.Email))) > 50 {
		return errors.New("第三方账号邮箱长度超过 50 个字符")
	}
	return nil
}
