package repository

import (
	"context"
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

type UserAttributionListFilter struct {
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
				attribution.AndroidID.Like(keyword), attribution.IP.Like(keyword),
				user.Username.Like(keyword), user.IMEI.Like(keyword),
			))
		}
		if filter.ChannelCode != "" {
			dao = dao.Where(field.Or(
				attribution.ChannelCode.Eq(filter.ChannelCode),
				field.And(attribution.ChannelCode.Eq(""), user.ChannelID.Eq(filter.ChannelCode)),
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
	rows, err := dao.Select(attribution.ALL).
		Order(attribution.ID.Desc()).Offset((page - 1) * pageSize).Limit(pageSize).Find()
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
		"oaid": "", "imei": "", "android_id": "", "ip": "", "user_agent": "",
	})
	return err
}

func (r *UserAttributionRepo) Update(ctx context.Context, item *model.VideoUserAttribution) error {
	q := qFrom(ctx).VideoUserAttribution
	_, err := q.WithContext(ctx).Where(q.ID.Eq(item.ID)).Select(
		q.ChannelCode, q.OAID, q.IMEI, q.AndroidID, q.IP, q.UserAgent, q.AttributedAt, q.Remark,
	).Updates(item)
	return err
}

func (r *UserAttributionRepo) UpsertDevice(ctx context.Context, userID uint64, updates map[string]interface{}) error {
	q := qFrom(ctx).VideoUserAttribution
	row := model.VideoUserAttribution{UserID: userID}
	if err := q.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}}, DoNothing: true,
	}).Create(&row); err != nil {
		return err
	}
	if len(updates) == 0 {
		return nil
	}
	_, err := q.WithContext(ctx).Where(q.UserID.Eq(userID)).Updates(updates)
	return err
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
				UserID: item.ID, ChannelCode: item.ChannelID,
				IMEI: item.DeviceCode, IP: item.LastLoginIP, AttributedAt: &attributedAt,
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

func reachedColumn(event string) string {
	switch strings.TrimSpace(event) {
	case domain.AttributionEventActivation:
		return "activated"
	case domain.AttributionEventKeyBehavior:
		return "key_behavior_met"
	case domain.AttributionEventPayment:
		return "payment_met"
	case domain.AttributionEventFirstPayment:
		return "first_payment_met"
	case domain.AttributionEventRegistration:
		return "registered"
	default:
		return ""
	}
}
