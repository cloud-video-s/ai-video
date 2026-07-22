package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"ai-video/internal/domain"
	"ai-video/internal/gen/model"
	"ai-video/internal/repository"

	"gorm.io/gorm"
)

type UserCenterDetail struct {
	User          *model.VideoUser                   `json:"user"`
	IsMember      bool                               `json:"is_member"`
	Identities    []model.VideoUserIdentity          `json:"identities"`
	Attribution   *model.VideoUserAttribution        `json:"attribution"`
	PointsLedgers []model.VideoUserPointsLedger      `json:"points_ledgers"`
	PointsSummary repository.UserPointsLedgerSummary `json:"points_summary"`
}

type UserAccessStateRequest struct {
	Enabled bool `json:"enabled"`
}

type BindUserPhoneRequest struct {
	Phone string `json:"phone" binding:"required,max=32"`
}

type GrantUserVIPRequest struct {
	Level     uint32     `json:"level" binding:"required,min=1,max=999"`
	StartedAt *time.Time `json:"started_at"`
	ExpiresAt time.Time  `json:"expires_at" binding:"required"`
}

type ExtendUserVIPRequest struct {
	Days uint32 `json:"days" binding:"required,min=1,max=3650"`
}

type TransferUserVIPRequest struct {
	TargetUserID uint64 `json:"target_user_id" binding:"required"`
}

func (s *AppUserService) Lookup(ctx context.Context, value string) (*model.VideoUser, error) {
	if strings.TrimSpace(value) == "" {
		return nil, errors.New("请输入用户 ID 或邮箱")
	}
	user, err := s.repo.GetByLookup(ctx, value)
	if err != nil {
		return nil, notFoundOr(err, "客户端用户不存在")
	}
	return user, nil
}

func (s *AppUserService) GetCenter(ctx context.Context, id uint64) (*UserCenterDetail, error) {
	user, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	identities, err := repository.NewUserIdentityRepo().ListByUser(ctx, id)
	if err != nil {
		return nil, err
	}
	attribution, err := repository.NewUserAttributionRepo().GetByUserID(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		attribution = nil
	} else if err != nil {
		return nil, err
	}
	ledgers, _, summary, err := repository.NewUserPointsLedgerRepo().PageList(ctx, 1, 20, &repository.UserPointsLedgerFilter{UserID: id})
	if err != nil {
		return nil, err
	}
	now := time.Now()
	return &UserCenterDetail{
		User: user, IsMember: user.VIPLevel > 0 && user.VipExpiresAt != nil && user.VipExpiresAt.After(now),
		Identities: identities, Attribution: attribution, PointsLedgers: ledgers, PointsSummary: summary,
	}, nil
}

func (s *AppUserService) SetFrozen(ctx context.Context, id uint64, frozen bool) error {
	if _, err := s.GetByID(ctx, id); err != nil {
		return err
	}
	status := int32(1)
	if frozen {
		status = 0
	}
	return s.repo.Update(ctx, id, map[string]interface{}{
		"is_frozen": frozen, "status": status, "token_version": gorm.Expr("token_version + 1"),
	})
}

func (s *AppUserService) SetBlacklisted(ctx context.Context, id uint64, blacklisted bool) error {
	if _, err := s.GetByID(ctx, id); err != nil {
		return err
	}
	return s.repo.Update(ctx, id, map[string]interface{}{
		"is_blacklisted": blacklisted, "token_version": gorm.Expr("token_version + 1"),
	})
}

func (s *AppUserService) BindPhone(ctx context.Context, id uint64, phone string) error {
	if _, err := s.GetByID(ctx, id); err != nil {
		return err
	}
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return errors.New("手机号不能为空")
	}
	return s.repo.Update(ctx, id, map[string]interface{}{"phone": phone})
}

func (s *AppUserService) GrantVIP(ctx context.Context, id uint64, req *GrantUserVIPRequest) error {
	if _, err := s.GetByID(ctx, id); err != nil {
		return err
	}
	startedAt := time.Now()
	if req.StartedAt != nil {
		startedAt = *req.StartedAt
	}
	if !req.ExpiresAt.After(startedAt) {
		return errors.New("VIP 结束时间必须晚于开始时间")
	}
	return s.repo.Update(ctx, id, map[string]interface{}{
		"vip_level": req.Level, "vip_started_at": startedAt, "vip_expires_at": req.ExpiresAt,
		"user_type": domain.AppUserTypePaid, "subscription_status": domain.AppUserSubscriptionSubscribed,
	})
}

func (s *AppUserService) ExtendVIP(ctx context.Context, id uint64, days uint32) error {
	user, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}
	now := time.Now()
	base := now
	if user.VipExpiresAt != nil && user.VipExpiresAt.After(now) {
		base = *user.VipExpiresAt
	}
	level := user.VIPLevel
	if level == 0 {
		level = 1
	}
	updates := map[string]interface{}{
		"vip_level": level, "vip_expires_at": base.AddDate(0, 0, int(days)),
		"user_type": domain.AppUserTypePaid, "subscription_status": domain.AppUserSubscriptionSubscribed,
	}
	if user.VIPStartedAt == nil {
		updates["vip_started_at"] = now
	}
	return s.repo.Update(ctx, id, updates)
}

func (s *AppUserService) TerminateVIP(ctx context.Context, id uint64) error {
	if _, err := s.GetByID(ctx, id); err != nil {
		return err
	}
	now := time.Now()
	return s.repo.Update(ctx, id, map[string]interface{}{
		"vip_level": 0, "vip_expires_at": now, "user_type": domain.AppUserTypeFree,
		"subscription_status": domain.AppUserSubscriptionCancelled,
	})
}

func (s *AppUserService) TransferVIP(ctx context.Context, id, targetID uint64) error {
	if id == targetID {
		return errors.New("不能向当前用户转移会员")
	}
	return repository.Transaction(ctx, func(ctx context.Context) error {
		first, second := id, targetID
		if first > second {
			first, second = second, first
		}
		if _, err := s.repo.GetByIDForUpdate(ctx, first); err != nil {
			return notFoundOr(err, "用户不存在")
		}
		if _, err := s.repo.GetByIDForUpdate(ctx, second); err != nil {
			return notFoundOr(err, "目标用户不存在")
		}
		source, err := s.repo.GetByID(ctx, id)
		if err != nil {
			return err
		}
		target, err := s.repo.GetByID(ctx, targetID)
		if err != nil {
			return err
		}
		if source.VIPLevel == 0 || source.VipExpiresAt == nil || !source.VipExpiresAt.After(time.Now()) {
			return errors.New("当前用户没有可转移的有效会员")
		}
		if target.VIPLevel > 0 && target.VipExpiresAt != nil && target.VipExpiresAt.After(time.Now()) {
			return errors.New("目标用户已有有效会员，不能覆盖")
		}
		if err := s.repo.Update(ctx, targetID, map[string]interface{}{
			"vip_level": source.VIPLevel, "vip_started_at": source.VIPStartedAt, "vip_expires_at": source.VipExpiresAt,
			"user_type": domain.AppUserTypePaid, "subscription_status": domain.AppUserSubscriptionSubscribed,
		}); err != nil {
			return err
		}
		now := time.Now()
		return s.repo.Update(ctx, id, map[string]interface{}{
			"vip_level": 0, "vip_expires_at": now, "user_type": domain.AppUserTypeFree,
			"subscription_status": domain.AppUserSubscriptionCancelled,
		})
	})
}

func (s *AppUserService) ClearDevice(ctx context.Context, id uint64) error {
	if _, err := s.GetByID(ctx, id); err != nil {
		return err
	}
	return repository.Transaction(ctx, func(ctx context.Context) error {
		if err := s.repo.Update(ctx, id, map[string]interface{}{
			"imei": "", "phone_model": "", "client_country": "", "last_login_ip": "",
			"token_version": gorm.Expr("token_version + 1"),
		}); err != nil {
			return err
		}
		return repository.NewUserAttributionRepo().ClearDevice(ctx, id)
	})
}
