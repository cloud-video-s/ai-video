package repository

import (
	"ai-video/internal/config"
	"ai-video/internal/pkg/utils"
	"context"
	"errors"
	"strings"
	"time"

	"ai-video/internal/domain"
	"ai-video/internal/gen/model"

	"gorm.io/gen/field"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserAttributionRepo struct{}

func NewUserAttributionRepo() *UserAttributionRepo { return &UserAttributionRepo{} }

// FusedUserAttribution is the user-level acquisition snapshot produced after
// an Adjust APP report and callback have been fused successfully.
type FusedUserAttribution struct {
	AppCode              string
	UserID               uint64
	AdjustADID           string
	ChannelID            uint64
	MediaID              uint64
	AttributedAdID       uint64
	AttributedPointID    uint64
	IMEI                 string
	OAID                 string
	AndroidID            string
	GpsAdid              string
	Idfa                 string
	Idfv                 string
	UserAgent            string
	DeviceIP             string
	GoogleAdID           string
	ActivityKind         string
	AttributionType      string
	IsOrganic            uint8
	Reattributed         uint8
	IsRedownload         uint8
	ClickTime            *time.Time
	InstallTime          *time.Time
	AttributedAt         *time.Time
	ReattributedAt       *time.Time
	AttributionUpdatedAt *time.Time
	AdjustCreatedAt      *time.Time
}

type UserAttributionListFilter struct {
	ListSort    ListSort
	Keyword     string
	ChannelCode string
	Event       string
	Reached     *bool
	StartedAt   *time.Time
	EndedAt     *time.Time
}

type UserAttributionRecord struct {
	model.VideoUserAttribution
	User model.VideoUser `json:"user"`
}

func (r *UserAttributionRepo) PageList(ctx context.Context, page, pageSize int, filter *UserAttributionListFilter) ([]UserAttributionRecord, int64, error) {
	q := qFrom(ctx)
	attribution := q.VideoUserAttribution
	user := q.VideoUser
	dao := attribution.WithContext(ctx).Join(user, user.ID.EqCol(attribution.UserID))
	if filter != nil {
		if filter.Keyword != "" {
			keyword := "%" + filter.Keyword + "%"
			dao = dao.Where(field.Or(
				attribution.OAID.Like(keyword), attribution.IMEI.Like(keyword),
				attribution.AndroidID.Like(keyword),
				user.Username.Like(keyword), user.IMEI.Like(keyword),
			))
		}
		if filter.StartedAt != nil {
			dao = dao.Where(attribution.AttributedAt.Gte(*filter.StartedAt))
		}
		if filter.EndedAt != nil {
			dao = dao.Where(attribution.AttributedAt.Lte(*filter.EndedAt))
		}
		if filter.Reached != nil {
			reached := *filter.Reached
			value := uint(0)
			if reached {
				value = 1
			}
			switch strings.TrimSpace(filter.Event) {
			case domain.AttributionEventActivation:
				dao = dao.Where(user.Activated.Eq(value))
			case domain.AttributionEventKeyBehavior:

				dao = dao.Where(user.KeyBehaviorMet.Eq(value))
			case domain.AttributionEventPayment:
				dao = dao.Where(user.PaymentMet.Eq(int8(value)))
			case domain.AttributionEventFirstPayment:
				dao = dao.Where(user.FirstPaymentMet.Eq(int8(value)))
			case domain.AttributionEventRegistration:
				dao = dao.Where(user.Registered.Eq(int8(value)))
			}
		}
	}
	total, err := dao.Count()
	if err != nil {
		return nil, 0, err
	}
	listSort := ListSort{}
	if filter != nil {
		listSort = filter.ListSort
	}
	order := orderForList(listSort, map[string]field.OrderExpr{"id": attribution.ID}, attribution.ID, attribution.ID.Desc())
	rows, err := dao.Select(attribution.ALL).
		Order(order...).Offset((page - 1) * pageSize).Limit(pageSize).Find()
	if err != nil {
		return nil, 0, err
	}
	records, err := r.loadRecords(ctx, valuesOf(rows))
	return records, total, err
}

func (r *UserAttributionRepo) GetByID(ctx context.Context, id uint64, lock bool) (*UserAttributionRecord, error) {
	q := qFrom(ctx).VideoUserAttribution
	dao := q.WithContext(ctx).Where(q.ID.Eq(id))
	if lock {
		dao = dao.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	item, err := dao.First()
	if err != nil {
		return nil, err
	}
	records, err := r.loadRecords(ctx, []model.VideoUserAttribution{*item})
	if err != nil {
		return nil, err
	}
	return &records[0], nil
}

func (r *UserAttributionRepo) loadRecords(ctx context.Context, items []model.VideoUserAttribution) ([]UserAttributionRecord, error) {
	result := make([]UserAttributionRecord, 0, len(items))
	if len(items) == 0 {
		return result, nil
	}
	userIDs := make([]uint64, 0, len(items))
	for i := range items {
		userIDs = append(userIDs, items[i].UserID)
	}
	userQuery := qFrom(ctx).VideoUser
	users, err := userQuery.WithContext(ctx).Where(userQuery.ID.In(userIDs...)).Find()
	if err != nil {
		return nil, err
	}
	userByID := make(map[uint64]model.VideoUser, len(users))
	for _, user := range users {
		if user != nil {
			userByID[user.ID] = *user
		}
	}
	for i := range items {
		result = append(result, UserAttributionRecord{
			VideoUserAttribution: items[i], User: userByID[items[i].UserID],
		})
	}
	return result, nil
}

func (r *UserAttributionRepo) GetByUserID(ctx context.Context, userID uint64) (*model.VideoUserAttribution, error) {
	q := qFrom(ctx).VideoUserAttribution
	return q.WithContext(ctx).Where(q.UserID.Eq(userID)).First()
}

func (r *UserAttributionRepo) ClearDevice(ctx context.Context, userID uint64) error {
	q := qFrom(ctx).VideoUserAttribution
	_, err := q.WithContext(ctx).Where(q.UserID.Eq(userID)).Updates(map[string]interface{}{
		"oaid": "", "imei": "", "android_id": "", "google_ad_id": "",
	})
	return err
}

func (r *UserAttributionRepo) Update(ctx context.Context, item *model.VideoUserAttribution) error {
	q := qFrom(ctx).VideoUserAttribution
	_, err := q.WithContext(ctx).Where(q.ID.Eq(item.ID)).Select(
		q.ChannelID, q.OAID, q.IMEI, q.AndroidID, q.AttributedAt, q.Remark,
	).Updates(item)
	return err
}

// ApplyFusedAttribution locks the user-level acquisition snapshot and applies
func (r *UserAttributionRepo) ApplyFusedAttribution(ctx context.Context, item *model.VideoUserAttribution) (bool, error) {
	if item == nil || item.UserID == 0 {
		return false, gorm.ErrInvalidData
	}
	adjustADID := strings.TrimSpace(item.AdjustAdid)
	if adjustADID == "" {
		return false, gorm.ErrInvalidData
	}

	q := qFrom(ctx).VideoUserAttribution
	existing, err := q.WithContext(ctx).Unscoped().
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where(q.UserID.Eq(item.UserID)).Take()
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, err
	}

	bound, boundErr := q.WithContext(ctx).Unscoped().Select(q.UserID).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where(q.AdjustAdid.Eq(adjustADID)).Take()
	if boundErr == nil && bound.UserID != item.UserID {
		return false, gorm.ErrDuplicatedKey
	}
	if boundErr != nil && !errors.Is(boundErr, gorm.ErrRecordNotFound) {
		return false, boundErr
	}
	now := time.Now()
	updates := map[string]any{
		"app_code": item.AppCode, "user_id": item.UserID, "adjust_adid": adjustADID,
		"channel_id": item.ChannelID, "media_id": item.MediaID,
		"attributed_ad_id": item.AttributedAdID, "attributed_point_id": item.AttributedPointID,
		"oaid": item.Idfa, "imei": item.IMEI, "android_id": "", "google_ad_id": item.GoogleAdID,
		"ad_account_id": item.AdAccountID, "network": item.Network, "campaign": item.Campaign, "adgroup": item.Adgroup, "creative": item.Creative,
		"idfa": item.Idfa, "idfv": item.Idfa, "user_agent": item.UserAgent, "device_ip": item.DeviceIP,
		"activity_kind": item.ActivityKind, "attribution_type": item.AttributionType,
		"is_organic": item.IsOrganic, "reattributed": item.Reattributed, "is_redownload": item.IsRedownload,
		"click_time": item.ClickTime, "install_time": item.InstallTime,
		"attributed_at": item.AttributedAt, "reattributed_at": item.ReattributedAt,
		"attribution_updated_at": item.AttributionUpdatedAt, "adjust_created_at": item.AdjustCreatedAt,
		"updated_at": now, "deleted_at": nil,
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		updates["last_operated_at"] = nil
		updates["created_at"] = now
		updates["activation_callback_count"] = 0
		updates["activation_deduct_count"] = 0
		updates["key_behavior_callback_count"] = 0
		updates["key_behavior_deduct_count"] = 0
		updates["payment_callback_count"] = 0
		updates["payment_deduct_count"] = 0
		updates["first_payment_callback_count"] = 0
		updates["first_payment_deduct_count"] = 0
		updates["registration_callback_count"] = 0
		updates["registration_deduct_count"] = 0
		updates["remark"] = ""
		createErr := q.WithContext(ctx).UnderlyingDB().
			Model(&model.VideoUserAttribution{}).Create(updates).Error
		if createErr != nil {
			return false, createErr
		}
		go r.SaveUserAttribution(ctx, item)
		return true, nil
	}

	existingADID := strings.TrimSpace(existing.AdjustAdid)
	if existingADID != "" && existingADID != adjustADID {
		return false, nil
	}
	_, err = q.WithContext(ctx).Unscoped().Where(q.ID.Eq(existing.ID)).Updates(updates)
	if err != nil {
		return false, err
	}
	go r.SaveUserAttribution(ctx, item)
	return true, nil
}

func (r *UserAttributionRepo) IncrementEvent(ctx context.Context, id uint64, column string, now time.Time) error {
	q := qFrom(ctx).VideoUserAttribution
	dao := q.WithContext(ctx).Where(q.ID.Eq(id))
	var err error
	switch column {
	case "activation_callback_count":
		_, err = dao.UpdateSimple(q.ActivationCallbackCount.Add(1), q.LastOperatedAt.Value(now))
	case "activation_deduct_count":
		_, err = dao.UpdateSimple(q.ActivationDeductCount.Add(1), q.LastOperatedAt.Value(now))
	case "key_behavior_callback_count":
		_, err = dao.UpdateSimple(q.KeyBehaviorCallbackCount.Add(1), q.LastOperatedAt.Value(now))
	case "key_behavior_deduct_count":
		_, err = dao.UpdateSimple(q.KeyBehaviorDeductCount.Add(1), q.LastOperatedAt.Value(now))
	case "payment_callback_count":
		_, err = dao.UpdateSimple(q.PaymentCallbackCount.Add(1), q.LastOperatedAt.Value(now))
	case "payment_deduct_count":
		_, err = dao.UpdateSimple(q.PaymentDeductCount.Add(1), q.LastOperatedAt.Value(now))
	case "first_payment_callback_count":
		_, err = dao.UpdateSimple(q.FirstPaymentCallbackCount.Add(1), q.LastOperatedAt.Value(now))
	case "first_payment_deduct_count":
		_, err = dao.UpdateSimple(q.FirstPaymentDeductCount.Add(1), q.LastOperatedAt.Value(now))
	case "registration_callback_count":
		_, err = dao.UpdateSimple(q.RegistrationCallbackCount.Add(1), q.LastOperatedAt.Value(now))
	case "registration_deduct_count":
		_, err = dao.UpdateSimple(q.RegistrationDeductCount.Add(1), q.LastOperatedAt.Value(now))
	default:
		return gorm.ErrInvalidField
	}
	return err
}

func (r *UserAttributionRepo) SyncUsers(ctx context.Context) (int64, error) {
	var total int64
	var cursor uint64
	for {
		q := qFrom(ctx)
		user := q.VideoUser
		users, err := user.WithContext(ctx).Where(user.ID.Gt(cursor)).Order(user.ID.Asc()).Limit(500).Find()
		if err != nil {
			return total, err
		}
		if len(users) == 0 {
			return total, nil
		}
		userIDs := make([]uint64, 0, len(users))
		for _, item := range users {
			userIDs = append(userIDs, item.ID)
			cursor = item.ID
		}
		attribution := q.VideoUserAttribution
		var existingIDs []uint64
		if err := attribution.WithContext(ctx).Where(attribution.UserID.In(userIDs...)).
			Pluck(attribution.UserID, &existingIDs); err != nil {
			return total, err
		}
		existing := make(map[uint64]struct{}, len(existingIDs))
		for _, id := range existingIDs {
			existing[id] = struct{}{}
		}
		rows := make([]*model.VideoUserAttribution, 0, len(users))
		for _, item := range users {
			if _, ok := existing[item.ID]; ok {
				continue
			}
			var attributedAt time.Time
			if item.AttributionClickedAt != nil {
				attributedAt = *item.AttributionClickedAt
			} else if item.FirstOpenedAt != nil {
				attributedAt = *item.FirstOpenedAt
			}
			rows = append(rows, &model.VideoUserAttribution{
				UserID: item.ID,
				IMEI:   item.DeviceCode, AttributedAt: &attributedAt,
			})
		}
		if len(rows) == 0 {
			continue
		}
		for _, row := range rows {
			if err := attribution.WithContext(ctx).Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "user_id"}}, DoNothing: true,
			}).Create(row); err != nil {
				return total, err
			}
			total++
		}
	}
}

func (r *UserAttributionRepo) SaveUserAttribution(ctx context.Context, item *model.VideoUserAttribution) {
	newTime := time.Now()
	err := qFrom(ctx).VideoUserAttributionHistory.WithContext(ctx).Create(&model.VideoUserAttributionHistory{
		AppCode:                 item.AppCode,
		UserID:                  item.UserID,
		AdjustAdid:              item.AdjustAdid,
		ChannelID:               item.ChannelID,
		MediaID:                 item.MediaID,
		AttributedAdID:          item.AttributedAdID,
		AttributedPointID:       item.AttributedPointID,
		OAID:                    item.OAID,
		IMEI:                    item.IMEI,
		AndroidID:               item.AndroidID,
		GoogleAdID:              item.GoogleAdID,
		ActivityKind:            item.ActivityKind,
		AttributionType:         item.AttributionType,
		IsOrganic:               item.IsOrganic,
		Reattributed:            item.Reattributed,
		IsRedownload:            item.IsRedownload,
		ClickTime:               item.ClickTime,
		InstallTime:             item.InstallTime,
		AttributedAt:            item.AttributedAt,
		ReattributedAt:          item.ReattributedAt,
		AttributionUpdatedAt:    item.AttributionUpdatedAt,
		LastOperatedAt:          &newTime,
		ActivationCallbackCount: 1,
		AdjustCreatedAt:         item.AdjustCreatedAt,
		GpsAdid:                 item.GpsAdid,
		Idfa:                    item.Idfa,
		Idfv:                    item.Idfv,
		UserAgent:               item.UserAgent,
		DeviceIP:                item.DeviceIP,
	})
	if err != nil {
		config.Logger(ctx).Error("UserAttributionHistory Create err:", err)
	}
	campaignStr := utils.ExtractBracketContent(item.Campaign)
	adgroupStr := utils.ExtractBracketContent(item.Adgroup)
	creativeStr := utils.ExtractBracketContent(item.Creative)
	videoMediaAd := model.VideoMediaAd{
		MediaID:      item.MediaID,
		AdAccountID:  item.AdAccountID,
		CampaignID:   utils.StrInt(campaignStr),
		CampaignName: item.Campaign,
		AdgroupID:    uint64(utils.StrInt(adgroupStr)),
		AdgroupName:  item.Adgroup,
		CreativeID:   utils.StrInt(creativeStr),
		CreativeName: item.Creative,
	}
	err = qFrom(ctx).VideoMediaAd.WithContext(ctx).Create(&videoMediaAd)
	if err != nil {
		config.Logger(ctx).Error("VideoMediaAd Create err:", err)
	}
}
