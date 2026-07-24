package service

import (
	"ai-video/internal/middleware"
	"ai-video/internal/pkg/utils"
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"ai-video/internal/domain"
	"ai-video/internal/gen/model"
	"ai-video/internal/pkg/oidc"
	"ai-video/internal/repository"

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

func (s *AuthService) ThirdPartyLogin(ctx *gin.Context, req *ThirdPartyLoginRequest, clientIP, userAgent string) (*AuthResponse, error) {
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
		if req.ThirdCode == "" || req.Email == "" {
			return nil, ErrIdentityProviderNotConfigured
		}
	}
	return s.loginVerifiedIdentity(ctx, req, clientIP, userAgent)
}

func (s *AuthService) loginVerifiedIdentity(ctx *gin.Context, req *ThirdPartyLoginRequest, clientIP, userAgent string) (*AuthResponse, error) {
	now := time.Now()
	var user *model.VideoUser
	apiUserID := middleware.GetAPIUserID(ctx)
	serverCountry := utils.ClientIP(ctx)
	var err error
	user, err = s.userRepo.GetByThirdCode(ctx, req.ThirdCode, true)
	if errors.Is(err, gorm.ErrRecordNotFound) || user == nil {
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
		if err := s.userRepo.Update(ctx, user.ID, ThirdPartyLoginBinding(req.ThirdType, req.Email, req.ThirdCode, clientIP, serverCountry, now)); err != nil {
			log.Printf("failed to update third party login info: %v", err)
			return nil, errors.New("failed to update third party login info")
		}
	}

	if user.ID != apiUserID {
		if user.Status != 1 || user.IsFrozen != 0 || user.IsBlacklisted != 0 {
			return nil, errors.New("当前邮箱绑定账号已停用，暂时无法使用")
		}
		if req.ForceNew {
			if err := s.userRepo.Update(ctx, user.ID, ThirdPartyLoginBinding(req.ThirdType, req.Email, req.ThirdCode, clientIP, serverCountry, now)); err != nil {
				return nil, errors.New("failed to update third party login info")
			}
		} else {
			return &AuthResponse{}, errors.New("当前设备已绑定另一个同类型第三方账号，是否确认登录？")
		}
	}
	user, err = s.prepareLoginSession(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	return issueToken(user, uint32(providerLoginType(req.ThirdType)))
}

func (s *AuthService) ListIdentities(ctx context.Context, userID uint64) ([]model.VideoUserIdentity, error) {
	return s.identityRepo.ListByUser(ctx, userID)
}

func (s *AuthService) UnbindIdentity(ctx context.Context, userID uint64, provider string) error {
	provider, err := normalizeIdentityProvider(provider)
	if err != nil {
		return err
	}
	return repository.Transaction(ctx, func(ctx context.Context) error {
		user, err := s.userRepo.GetByIDForUpdate(ctx, userID)
		if err != nil {
			return err
		}
		identities, err := s.identityRepo.ListByUser(ctx, userID)
		if err != nil {
			return err
		}
		found := false
		for _, item := range identities {
			if item.Provider == provider {
				found = true
				break
			}
		}
		if !found {
			return errors.New("该第三方账号尚未绑定")
		}
		if len(identities) <= 1 {
			return errors.New("至少保留一个第三方登录方式")
		}
		if err := s.identityRepo.DeleteByUserProvider(ctx, userID, provider); err != nil {
			return err
		}
		updates := make(map[string]interface{})
		if user.LoginType == providerLoginType(provider) {
			for _, item := range identities {
				if item.Provider != provider {
					updates["login_type"] = providerLoginType(item.Provider)
					updates["login_account"] = identityRecordLoginAccount(&item)
					updates["third_code"] = item.ProviderSubject
					updates["email"] = item.Email
					break
				}
			}
		}
		return s.userRepo.Update(ctx, userID, updates)
	})
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

func identityRecordLoginAccount(identity *model.VideoUserIdentity) string {
	if identity.Email != "" {
		return identity.Email
	}
	return identity.Provider + ":" + identity.ProviderSubject
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
