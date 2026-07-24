package service

import (
	"ai-video/internal/domain"
	"ai-video/internal/middleware"
	"ai-video/internal/repository"

	"github.com/gin-gonic/gin"
)

type ClientVipService struct {
	subscriptionRepo *repository.VIPSubscriptionRepo
	userRepo         *repository.AppUserRepo
}

func NewClientVipService() *ClientVipService {
	return &ClientVipService{
		subscriptionRepo: repository.NewVIPSubscriptionRepo(),
		userRepo:         repository.NewAppUserRepo(),
	}
}

func (s *ClientVipService) VipRecommend(ctx *gin.Context, req *VipRecommendRequest) (*VIPRecommendResponse, error) {
	user, err := s.userRepo.GetByID(ctx, middleware.GetAPIUserID(ctx))
	if err != nil {
		return nil, err
	}
	vip, err := s.subscriptionRepo.Recommend(ctx, &repository.VIPSubscriptionListFilter{
		VipType:     req.VipType,
		AppCode:     middleware.GetAPIAPPCode(ctx),
		PackageCode: middleware.GetAPIAppPackageCode(ctx),
		VersionCode: middleware.GetAPIAppVersion(ctx),
	})
	if err != nil {
		return nil, err
	}
	resp := &VIPRecommendResponse{
		ID:                      vip.ID,
		VipType:                 vip.VipType,
		SukCode:                 vip.SukCode,
		Name:                    vip.Name,
		Currency:                vip.Currency,
		VIPDurationDays:         vip.VIPDurationDays,
		TrialDays:               vip.TrialDays,
		BadgeText:               vip.BadgeText,
		AgreementDefaultChecked: vip.AgreementDefaultChecked,
		DisplayMode:             vip.DisplayMode,
		Status:                  vip.Status,
		FreeTrial:               vip.FreeTrial,
		IsSubscription:          vip.IsSubscription,
		IsDefault:               vip.IsDefault,
		SubscriptionDescription: vip.SubscriptionDescription,
		SubscriptionPrice:       vip.SubscriptionPrice,
		OriginalPrice:           vip.OriginalPrice,
		SubscriptionPoints:      vip.SubscriptionPoints,
		SubscriptionPeriod:      vip.SubscriptionPeriod,
		Sort:                    vip.Sort,
		Description:             vip.Description,
		Remark:                  vip.Remark,
		CreatedAt:               vip.CreatedAt.Unix(),
		UpdatedAt:               vip.UpdatedAt.Unix(),
	}

	if user.SubscriptionStatus == domain.SubscriptionStatusUnsubscribed {
		resp.SubscriptionPrice = vip.FirstSubscriptionPrice
		resp.SubscriptionPoints = vip.FirstBonusPoints
	}
	if vip.SubscriptionLevel.ID != 0 {
		resp.LevelName = vip.SubscriptionLevel.Level
	}
	return resp, nil
}

func (s *ClientVipService) VipList(ctx *gin.Context, req *VipVipListRequest) ([]*VIPRecommendResponse, error) {
	user, err := s.userRepo.GetByID(ctx, middleware.GetAPIUserID(ctx))
	if err != nil {
		return nil, err
	}
	vipList, err := s.subscriptionRepo.VipList(ctx, &repository.VIPSubscriptionListFilter{
		VipTypes:           req.VipTypes,
		AppCode:            middleware.GetAPIAPPCode(ctx),
		PackageCode:        middleware.GetAPIAppPackageCode(ctx),
		VersionCode:        middleware.GetAPIAppVersion(ctx),
		UserType:           uint32(user.UserType),
		SubscriptionStatus: uint32(user.SubscriptionStatus),
	})
	if err != nil {
		return nil, err
	}
	resp := make([]*VIPRecommendResponse, 0)
	for _, item := range vipList {
		subscriptionPrice := item.SubscriptionPrice
		subscriptionPoints := item.SubscriptionPoints
		levelName := ""
		if user.SubscriptionStatus == domain.SubscriptionStatusUnsubscribed {
			subscriptionPrice = item.FirstSubscriptionPrice
			subscriptionPoints = item.FirstBonusPoints
		}
		if item.SubscriptionLevel.ID != 0 {
			levelName = item.SubscriptionLevel.Level
		}
		resp = append(resp, &VIPRecommendResponse{
			ID:                      item.ID,
			VipType:                 item.VipType,
			SukCode:                 item.SukCode,
			Name:                    item.Name,
			LevelName:               levelName,
			Currency:                item.Currency,
			VIPDurationDays:         item.VIPDurationDays,
			TrialDays:               item.TrialDays,
			BadgeText:               item.BadgeText,
			AgreementDefaultChecked: item.AgreementDefaultChecked,
			DisplayMode:             item.DisplayMode,
			Status:                  item.Status,
			FreeTrial:               item.FreeTrial,
			IsSubscription:          item.IsSubscription,
			IsDefault:               item.IsDefault,
			SubscriptionPrice:       subscriptionPrice,
			OriginalPrice:           item.OriginalPrice,
			SubscriptionPoints:      subscriptionPoints,
			SubscriptionPeriod:      item.SubscriptionPeriod,
			Sort:                    item.Sort,
			Description:             item.Description,
			Remark:                  item.Remark,
			CreatedAt:               item.CreatedAt.Unix(),
			UpdatedAt:               item.UpdatedAt.Unix(),
		})
	}
	return resp, nil
}

type VipRecommendRequest struct {
	VipType uint64 `form:"vip_type" binding:"required,min=1"`
	AccountBaseRequest
}

type VipVipListRequest struct {
	VipTypes []uint64 `form:"vip_types" binding:"required,min=1"`
	AccountBaseRequest
}

type VIPRecommendResponse struct {
	ID                      uint64  `json:"id"`
	VipType                 uint64  `json:"vip_type"`                  // 套餐类型(展示位置)：1、OB 2、OB拦截 3、app老用户启动 4、app老用户返回拦截 5、默认付费页 6、默认付费页拦截 7、卸载拦截 8、默认订阅套餐界面（预留)
	SukCode                 string  `json:"suk_code"`                  // 商店产品SKU
	Name                    string  `json:"name"`                      // VIP套餐名称
	LevelName               string  `json:"level_name"`                // 会员等级
	Currency                string  `json:"currency"`                  // ISO货币代码
	VIPDurationDays         uint    `json:"vip_duration_days"`         // VIP权益持续时间（天）
	TrialDays               uint    `json:"trial_days"`                // 免费试用天数
	BadgeText               string  `json:"badge_text"`                // 徽章文案
	AgreementDefaultChecked int8    `json:"agreement_default_checked"` // 订阅协议是否默认勾选
	DisplayMode             int8    `json:"display_mode"`              // 展示模式：0隐藏，1正常
	Status                  int8    `json:"status"`                    // 状态：0禁用，1启用
	FreeTrial               int8    `json:"free_trial"`                // 是否启用免费试用
	IsSubscription          int8    `json:"is_subscription"`           // 是否循环订阅
	IsDefault               int8    `json:"is_default"`                // 是否为该平台和套餐组合的默认选项
	SubscriptionDescription string  `json:"subscription_description"`  // 订阅描述
	SubscriptionPrice       float64 `json:"subscription_price"`        // 续订价格
	OriginalPrice           float64 `json:"original_price"`            // 原价（划线价）
	SubscriptionPoints      uint64  `json:"subscription_points"`       // 订阅赠送积分
	SubscriptionPeriod      uint32  `json:"subscription_period"`       // 订阅周期 1=周 2=月 3=季 4=年
	Sort                    int64   `json:"sort"`                      // 排序顺序
	Description             string  `json:"description"`               // 套餐描述
	Remark                  string  `json:"remark"`                    // 内部备注
	CreatedAt               int64   `json:"created_at"`                // 创建时间
	UpdatedAt               int64   `json:"updated_at"`
}
