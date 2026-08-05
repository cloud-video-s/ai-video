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
	User              *model.VideoUser                   `json:"user"`
	IsMember          bool                               `json:"is_member"`
	Identities        []model.VideoUserIdentity          `json:"identities"`
	Attribution       *model.VideoUserAttribution        `json:"attribution"`
	PointsLedgers     []UserCenterPointsLedger           `json:"points_ledgers"`
	PointsLedgerTotal int64                              `json:"points_ledger_total"`
	PointsSummary     repository.UserPointsLedgerSummary `json:"points_summary"`
	Works             []UserCenterWork                   `json:"works"`
	WorkTotal         int64                              `json:"work_total"`
	Orders            []UserCenterOrder                  `json:"orders"`
	OrderTotal        int64                              `json:"order_total"`
}

type UserCenterPointsLedger struct {
	ID            uint64    `json:"id"`
	Direction     int8      `json:"direction"`
	PointsChange  int64     `json:"points_change"`
	BalanceBefore uint64    `json:"balance_before"`
	BalanceAfter  uint64    `json:"balance_after"`
	SourceType    uint32    `json:"source_type"`
	OrderCode     string    `json:"order_code"`
	PointsID      uint64    `json:"points_id"`
	VIPID         uint64    `json:"vip_id"`
	AdminID       uint64    `json:"admin_id"`
	Description   string    `json:"description"`
	OccurredAt    time.Time `json:"occurred_at"`
}

type UserCenterWork struct {
	ID              uint64     `json:"id"`
	ModelID         uint64     `json:"model_id"`
	ClientRequestID string     `json:"client_request_id"`
	TaskCode        string     `json:"task_code"`
	ThirdTaskCode   string     `json:"third_task_code"`
	Status          int        `json:"status"`
	Progress        uint32     `json:"progress"`
	UsageDuration   uint32     `json:"usage_duration"`
	ErrorMessage    string     `json:"error_message,omitempty"`
	SubmittedAt     *time.Time `json:"submitted_at"`
	FinishedAt      *time.Time `json:"finished_at"`
	CreatedAt       time.Time  `json:"created_at"`
}

type UserCenterOrder struct {
	ID              uint64     `json:"id"`
	OrderNo         string     `json:"order_no"`
	ProductType     uint32     `json:"product_type"`
	ProductID       uint64     `json:"product_id"`
	ProductCode     string     `json:"product_code"`
	ProductName     string     `json:"product_name"`
	Currency        string     `json:"currency"`
	PayableAmount   float64    `json:"payable_amount"`
	PaidAmount      float64    `json:"paid_amount"`
	RefundedAmount  float64    `json:"refunded_amount"`
	BonusPoints     int64      `json:"bonus_points"`
	VIPLevel        uint       `json:"vip_level"`
	VIPDurationDays uint       `json:"vip_duration_days"`
	Status          uint32     `json:"status"`
	PayType         uint32     `json:"pay_type"`
	PayTime         *time.Time `json:"pay_time"`
	CreatedAt       time.Time  `json:"created_at"`
}

type UserAccessStateRequest struct {
	Enabled bool `json:"enabled"`
}

type BindUserPhoneRequest struct {
	Phone string `json:"phone" binding:"required,max=32"`
}

type GrantUserVIPRequest struct {
	Level     uint32     `json:"level" binding:"required,min=1,max=999"`
	VIPPoints int64      `json:"vip_points" binding:"min=0,max=999999999"`
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
	const relationPageSize = 20

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
	pointRecords, pointsTotal, summary, err := repository.NewUserPointsLedgerRepo().PageList(ctx, 1, relationPageSize, &repository.UserPointsLedgerFilter{UserID: id})
	if err != nil {
		return nil, err
	}
	works, workTotal, err := repository.NewUserGenerationTaskRepo().PageOwned(ctx, id, 1, relationPageSize, 0)
	if err != nil {
		return nil, err
	}
	orders, orderTotal, err := repository.NewOrderRepo().PageByUser(ctx, id, 1, relationPageSize)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	return &UserCenterDetail{
		User: user, IsMember: user.SubscriptionStatus == domain.AppUserSubscriptionSubscribed && user.VipExpiresAt != nil && user.VipExpiresAt.After(now),
		Identities: identities, Attribution: attribution,
		PointsLedgers: userCenterPointsLedgers(pointRecords), PointsLedgerTotal: pointsTotal, PointsSummary: summary,
		Works: userCenterWorks(works), WorkTotal: workTotal,
		Orders: userCenterOrders(orders), OrderTotal: orderTotal,
	}, nil
}

func userCenterPointsLedgers(records []repository.UserPointsLedgerRecord) []UserCenterPointsLedger {
	items := make([]UserCenterPointsLedger, 0, len(records))
	for _, record := range records {
		ledger := record.VideoUserPointsLedger
		items = append(items, UserCenterPointsLedger{
			ID: ledger.ID, Direction: ledger.Direction, PointsChange: ledger.PointsChange,
			BalanceBefore: ledger.BalanceBefore, BalanceAfter: ledger.BalanceAfter,
			SourceType: ledger.SourceType, OrderCode: ledger.OrderCode,
			PointsID: ledger.PointsID, VIPID: ledger.VipID, AdminID: ledger.AdminID,
			Description: ledger.Description, OccurredAt: ledger.OccurredAt,
		})
	}
	return items
}

func userCenterWorks(records []model.VideoUserGenerationTask) []UserCenterWork {
	items := make([]UserCenterWork, 0, len(records))
	for _, work := range records {
		items = append(items, UserCenterWork{
			ID: work.ID, TaskCode: work.TaskCode, ModelID: work.ModelID,
			ClientRequestID: work.ClientRequestID, ThirdTaskCode: work.ThirdTaskCode,
			Status: work.Status, Progress: work.Progress, UsageDuration: work.UsageDuration,
			ErrorMessage: work.ErrorMessage, SubmittedAt: nonZeroTime(work.SubmittedAt),
			FinishedAt: nonZeroTime(work.FinishedAt), CreatedAt: work.CreatedAt,
		})
	}
	return items
}

func userCenterOrders(records []model.VideoOrder) []UserCenterOrder {
	items := make([]UserCenterOrder, 0, len(records))
	for _, order := range records {
		items = append(items, UserCenterOrder{
			ID: order.ID, OrderNo: order.OrderNo, ProductType: order.ProductType,
			ProductID: order.ProductID, ProductCode: order.ProductCode, ProductName: order.ProductName,
			Currency: order.Currency, PayableAmount: order.PayableAmount, PaidAmount: order.PaidAmount,
			RefundedAmount: order.RefundedAmount, BonusPoints: order.BonusPoints,
			VIPLevel: order.VipLevel, VIPDurationDays: order.VipDurationDays,
			Status: order.Status, PayType: order.PayType,
			PayTime: nonZeroTime(order.PayTime), CreatedAt: order.CreatedAt,
		})
	}
	return items
}

func nonZeroTime(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	return &value
}

func (s *AppUserService) SetFrozen(ctx context.Context, id uint64, frozen bool) error {
	user, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}
	status := appUserStatus(frozen, user.IsBlacklisted != 0)
	return s.repo.Update(ctx, id, map[string]interface{}{
		"is_frozen": boolInt8(frozen), "status": status, "token_version": gorm.Expr("token_version + 1"),
	})
}

func (s *AppUserService) SetBlacklisted(ctx context.Context, id uint64, blacklisted bool) error {
	user, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}
	return s.repo.Update(ctx, id, map[string]interface{}{
		"is_blacklisted": boolInt8(blacklisted),
		"status":         appUserStatus(user.IsFrozen != 0, blacklisted),
		"token_version":  gorm.Expr("token_version + 1"),
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

func (s *AppUserService) GrantVIP(ctx context.Context, id, adminID uint64, req *GrantUserVIPRequest) error {
	startedAt := time.Now()
	if req.StartedAt != nil {
		startedAt = *req.StartedAt
	}
	if !req.ExpiresAt.After(startedAt) {
		return errors.New("VIP 结束时间必须晚于开始时间")
	}
	return repository.Transaction(ctx, func(ctx context.Context) error {
		user, err := s.repo.GetByIDForUpdate(ctx, id)
		if err != nil {
			return notFoundOr(err, "客户端用户不存在")
		}
		beforeBalance := user.VipPoints + user.PointsBalance
		afterVIPPoints := user.VipPoints + req.VIPPoints
		if err := s.repo.Update(ctx, id, map[string]interface{}{
			"vip_started_at": startedAt, "vip_expires_at": req.ExpiresAt,
			"vip_level": uint(req.Level), "vip_points": afterVIPPoints,
			"user_type": domain.AppUserTypePaid, "subscription_status": domain.AppUserSubscriptionSubscribed,
		}); err != nil {
			return err
		}
		if req.VIPPoints == 0 {
			return nil
		}
		now := time.Now()
		return repository.NewUserPointsLedgerRepo().Create(ctx, &model.VideoUserPointsLedger{
			UserID: id, Direction: int8(domain.PointsDirectionIncome), PointsChange: req.VIPPoints,
			BalanceBefore: uint64(beforeBalance), BalanceAfter: uint64(beforeBalance + req.VIPPoints),
			Description: "管理员添加 VIP 赠送积分", SourceType: uint32(domain.PointsSourceAdminOp),
			AdminID: adminID, OccurredAt: now, CreatedAt: now, UpdatedAt: now,
		})
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
	updates := map[string]interface{}{
		"vip_expires_at": base.AddDate(0, 0, int(days)),
		"user_type":      domain.AppUserTypePaid, "subscription_status": domain.AppUserSubscriptionSubscribed,
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
		if source.SubscriptionStatus != domain.AppUserSubscriptionSubscribed || source.VipExpiresAt == nil || !source.VipExpiresAt.After(time.Now()) {
			return errors.New("当前用户没有可转移的有效会员")
		}
		if target.SubscriptionStatus == domain.AppUserSubscriptionSubscribed && target.VipExpiresAt != nil && target.VipExpiresAt.After(time.Now()) {
			return errors.New("目标用户已有有效会员，不能覆盖")
		}
		if err := s.repo.Update(ctx, targetID, map[string]interface{}{
			"vip_started_at": source.VIPStartedAt, "vip_expires_at": source.VipExpiresAt,
			"vip_level": source.VIPLevel,
			"user_type": domain.AppUserTypePaid, "subscription_status": domain.AppUserSubscriptionSubscribed,
		}); err != nil {
			return err
		}
		now := time.Now()
		return s.repo.Update(ctx, id, map[string]interface{}{
			"vip_level": 0, "vip_started_at": nil, "vip_expires_at": now, "user_type": domain.AppUserTypeFree,
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
			"imei": "", "phone_model": "", "client_country": "", "server_country": "", "last_login_ip": "",
			"token_version": gorm.Expr("token_version + 1"),
		}); err != nil {
			return err
		}
		return repository.NewUserAttributionRepo().ClearDevice(ctx, id)
	})
}
