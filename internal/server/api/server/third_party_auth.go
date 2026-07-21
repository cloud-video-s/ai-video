package service

import (
	"ai-video/internal/middleware"
	"context"
	"errors"
	"fmt"
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
	IDToken       string `json:"id_token" binding:"omitempty,max=16384"`
	IdentityToken string `json:"identity_token" binding:"omitempty,max=16384"`
	Nonce         string `json:"nonce" binding:"omitempty,max=255"`
	IMEI          string `json:"imei" binding:"required,max=128"`
	DisplayName   string `json:"display_name" binding:"omitempty,max=128"`
	GivenName     string `json:"given_name" binding:"omitempty,max=128"`
	FamilyName    string `json:"family_name" binding:"omitempty,max=128"`
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

var ErrIdentityProviderNotConfigured = errors.New("third-party identity provider is not configured")

func (s *AuthService) ThirdPartyLogin(
	ctx *gin.Context, provider string, req *ThirdPartyLoginRequest,
	clientIP, userAgent string,
) (*AuthResponse, error) {
	provider, err := normalizeIdentityProvider(provider)
	if err != nil {
		return nil, err
	}
	GetCtxAccountBaseRequest(ctx, &req.AccountBaseRequest)
	req.IMEI = strings.TrimSpace(req.IMEI)
	identity, err := s.verifyIdentity(ctx, provider, firstToken(req.IDToken, req.IdentityToken), req.Nonce)
	if err != nil {
		return nil, err
	}
	if err := validateIdentityUserColumns(identity); err != nil {
		return nil, err
	}
	mergeIdentityNames(identity, req.DisplayName, req.GivenName, req.FamilyName)
	return s.loginVerifiedIdentity(ctx, provider, identity, req, clientIP, userAgent)
}

func (s *AuthService) loginVerifiedIdentity(
	ctx *gin.Context, provider string, identity *oidc.Identity, req *ThirdPartyLoginRequest,
	clientIP, userAgent string,
) (*AuthResponse, error) {
	now := time.Now()
	var user *model.VideoUser
	apiUserID := middleware.GetAPIUserID(ctx)
	err := repository.Transaction(ctx, func(ctx context.Context) error {
		var err error
		user, err = s.userRepo.GetByProviderSubject(ctx, provider, identity.Subject, true)
		identityExists := err == nil && user != nil
		if errors.Is(err, gorm.ErrRecordNotFound) {
			user, err = nil, nil
		}
		if err != nil {
			return err
		}

		if user == nil {
			user, err = s.userRepo.GetByID(ctx, apiUserID)
			if err != nil {
				return err
			}
			firstOpenedAt := req.FirstOpenedAt
			if firstOpenedAt == nil {
				firstOpenedAt = &now
			}
			lastOpenedAt := req.LastOpenedAt
			if lastOpenedAt == nil {
				lastOpenedAt = &now
			}
			user.Email = identity.Email
			user.ThirdCode = identity.Subject
			user.LoginType = providerLoginType(provider)
			user.LastLoginIP = clientIP
			user.LastLoginAt = &now
			if err := s.userRepo.Update(ctx, user.ID, ThirdPartyLoginBinding(provider, identity.Email, identity.Subject, clientIP, now)); err != nil {
				return err
			}
		} else {
			if apiUserID != user.ID {
				return errors.New("当前邮箱已绑定其他用户，是否确认登录？")
			}
			if user.Status != 1 {
				return errors.New("当前账号已停用")
			}
			if subject := directProviderSubject(user, provider); subject != "" && subject != identity.Subject {
				return errors.New("当前设备已绑定另一个同类型第三方账号")
			}
			updates := baseTrackingUpdates(int(providerLoginType(provider)), &req.AccountBaseRequest, clientIP, now)
			updates["login_type"] = providerLoginType(provider)
			updates["login_account"] = identityLoginAccount(provider, identity)
			updates["registered"] = true
			applyIdentityUserProfile(updates, provider, identity, user.Username)
			if err := s.userRepo.Update(ctx, user.ID, updates); err != nil {
				return err
			}
		}

		if !identityExists {
			association := &model.VideoUserIdentity{UserID: user.ID, Provider: provider, ProviderSubject: identity.Subject}
			applyIdentityRecord(association, identity, now)
			if err := s.identityRepo.Create(ctx, association); err != nil {
				return err
			}
		}
		if err := s.attributionRepo.UpsertDevice(ctx, user.ID, attributionTrackingUpdates(&req.AccountBaseRequest, clientIP, userAgent)); err != nil {
			return err
		}
		user, err = s.prepareLoginSession(ctx, user.ID)
		return err
	})
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, errors.New("第三方账号已关联，请重试登录")
		}
		return nil, err
	}
	return issueToken(user, providerLoginType(provider))
}

func (s *AuthService) ListIdentities(ctx context.Context, userID uint64) ([]model.VideoUserIdentity, error) {
	return s.identityRepo.ListByUser(ctx, userID)
}

func (s *AuthService) BindIdentity(ctx context.Context, userID uint64, provider string, req *BindIdentityRequest) (*model.VideoUserIdentity, error) {
	provider, err := normalizeIdentityProvider(provider)
	if err != nil {
		return nil, err
	}
	identity, err := s.verifyIdentity(ctx, provider, firstToken(req.IDToken, req.IdentityToken), req.Nonce)
	if err != nil {
		return nil, err
	}
	if err := validateIdentityUserColumns(identity); err != nil {
		return nil, err
	}
	mergeIdentityNames(identity, req.DisplayName, req.GivenName, req.FamilyName)
	now := time.Now()
	var result *model.VideoUserIdentity
	err = repository.Transaction(ctx, func(ctx context.Context) error {
		user, err := s.userRepo.GetByIDForUpdate(ctx, userID)
		if err != nil {
			return err
		}
		linked, err := s.identityRepo.GetByProviderSubject(ctx, provider, identity.Subject, true)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if linked != nil && linked.UserID != userID {
			return errors.New("该第三方账号已关联其他用户")
		}
		direct, directErr := s.userRepo.GetByProviderSubject(ctx, provider, identity.Subject, true)
		if directErr != nil && !errors.Is(directErr, gorm.ErrRecordNotFound) {
			return directErr
		}
		if direct != nil && direct.ID != userID {
			return errors.New("该第三方账号已关联其他用户")
		}
		if subject := directProviderSubject(user, provider); subject != "" && subject != identity.Subject {
			return errors.New("当前用户已绑定另一个同类型第三方账号")
		}
		current, err := s.identityRepo.GetByUserProvider(ctx, userID, provider, true)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if current != nil && current.ProviderSubject != identity.Subject {
			return errors.New("当前用户已绑定另一个同类型第三方账号")
		}
		if current == nil {
			current = &model.VideoUserIdentity{UserID: userID, Provider: provider, ProviderSubject: identity.Subject}
			applyIdentityRecord(current, identity, now)
			if err := s.identityRepo.Create(ctx, current); err != nil {
				return err
			}
		} else {
			applyIdentityRecord(current, identity, now)
			if err := s.identityRepo.UpdateProfile(ctx, current); err != nil {
				return err
			}
		}
		updates := map[string]interface{}{"registered": true, "login_type": providerLoginType(provider), "login_account": identityLoginAccount(provider, identity)}
		applyIdentityUserProfile(updates, provider, identity, user.Username)
		if err := s.userRepo.Update(ctx, userID, updates); err != nil {
			return err
		}
		result = current
		return nil
	})
	return result, err
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

func providerLoginType(provider string) uint32 {
	if provider == domain.IdentityProviderGoogle {
		return domain.AppUserLoginGoogle
	}
	return domain.AppUserLoginAppID
}

func mergeIdentityNames(identity *oidc.Identity, displayName, givenName, familyName string) {
	if identity.DisplayName == "" {
		identity.DisplayName = strings.TrimSpace(displayName)
	}
	if identity.GivenName == "" {
		identity.GivenName = strings.TrimSpace(givenName)
	}
	if identity.FamilyName == "" {
		identity.FamilyName = strings.TrimSpace(familyName)
	}
	if identity.DisplayName == "" {
		identity.DisplayName = strings.TrimSpace(strings.Join([]string{identity.GivenName, identity.FamilyName}, " "))
	}
}

func identityUsername(identity *oidc.Identity) string {
	if value := strings.TrimSpace(identity.DisplayName); value != "" {
		return value
	}
	if index := strings.Index(identity.Email, "@"); index > 0 {
		return identity.Email[:index]
	}
	return newGuestUsername()
}

func identityLoginAccount(provider string, identity *oidc.Identity) string {
	if identity.Email != "" {
		return identity.Email
	}
	return provider + ":" + identity.Subject
}

func identityRecordLoginAccount(identity *model.VideoUserIdentity) string {
	if identity.Email != "" {
		return identity.Email
	}
	return identity.Provider + ":" + identity.ProviderSubject
}

func applyIdentityRecord(record *model.VideoUserIdentity, identity *oidc.Identity, now time.Time) {
	record.Issuer = identity.Issuer
	record.Audience = identity.Audience
	record.Email = identity.Email
	record.EmailVerified = identity.EmailVerified
	record.IsPrivateEmail = identity.IsPrivateEmail
	record.DisplayName = identity.DisplayName
	record.GivenName = identity.GivenName
	record.FamilyName = identity.FamilyName
	record.AvatarURL = identity.AvatarURL
	record.LastLoginAt = &now
	record.LastTokenIssuedAt = identity.IssuedAt
}

func applyIdentityUserProfile(updates map[string]interface{}, provider string, identity *oidc.Identity, currentUsername string) {
	for column, value := range identityUserColumns(provider, identity) {
		updates[column] = value
	}
	if (strings.HasPrefix(currentUsername, "guest_") || strings.TrimSpace(currentUsername) == "") && identity.DisplayName != "" {
		updates["username"] = identity.DisplayName
	}
}

func identityUserColumns(provider string, identity *oidc.Identity) map[string]interface{} {
	updates := map[string]interface{}{"third_code": strings.TrimSpace(identity.Subject)}
	if identity.EmailVerified && strings.TrimSpace(identity.Email) != "" {
		updates["email"] = strings.ToLower(strings.TrimSpace(identity.Email))
	}
	return updates
}

func directProviderSubject(user *model.VideoUser, provider string) string {
	if user == nil || user.LoginType != providerLoginType(provider) {
		return ""
	}
	return user.ThirdCode
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
