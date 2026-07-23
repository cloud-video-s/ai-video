package repository

import (
	"context"
	"strconv"
	"strings"

	"ai-video/internal/gen/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gen/field"
)

type ChannelRepo struct {
	BaseRepo[model.VideoChannel]
}

func (r *ChannelRepo) ResolveEnabledTargets(ctx *gin.Context, codeOrID, deliveryPackage string) ([]model.VideoChannel, error) {
	q := qFrom(ctx).VideoChannel
	dao := q.WithContext(ctx).Where(q.Status.Eq(1))
	if value := strings.TrimSpace(codeOrID); value != "" {
		if id, err := strconv.ParseUint(value, 10, 64); err == nil && id > 0 {
			dao = dao.Where(q.ChannelID.Eq(id))
		} else {
			dao = dao.Where(q.ChannelCode.Eq(value))
		}
	}
	if value := strings.TrimSpace(deliveryPackage); value != "" {
		dao = dao.Where(q.DeliveryPackage.Eq(value))
	}
	if strings.TrimSpace(codeOrID) == "" && strings.TrimSpace(deliveryPackage) == "" {
		return []model.VideoChannel{}, nil
	}
	rows, err := dao.Order(q.ChannelID.Asc()).Find()
	if err != nil {
		return nil, err
	}
	return valuesOf(rows), nil
}

func NewChannelRepo() *ChannelRepo {
	return &ChannelRepo{}
}

type ChannelListFilter struct {
	AdPlatform   string
	UploadMethod string
	Status       *int8
	Keyword      string
}

func (r *ChannelRepo) PageList(ctx context.Context, page, pageSize int, filter *ChannelListFilter) ([]model.VideoChannel, int64, error) {
	q := qFrom(ctx).VideoChannel
	dao := q.WithContext(ctx)
	if filter != nil {
		if filter.AdPlatform != "" {
			dao = dao.Where(q.AdPlatform.Eq(filter.AdPlatform))
		}
		if filter.UploadMethod != "" {
			dao = dao.Where(q.UploadMethod.Eq(filter.UploadMethod))
		}
		if filter.Status != nil {
			dao = dao.Where(q.Status.Eq(*filter.Status))
		}
		if filter.Keyword != "" {
			keyword := "%" + filter.Keyword + "%"
			dao = dao.Where(field.Or(
				q.ChannelCode.Like(keyword), q.ChannelName.Like(keyword),
				q.AgencyCompany.Like(keyword), q.DeliveryPackage.Like(keyword),
			))
		}
	}
	total, err := dao.Count()
	if err != nil {
		return nil, 0, err
	}
	rows, err := dao.Order(q.ChannelID.Desc()).Offset((page - 1) * pageSize).Limit(pageSize).Find()
	return valuesOf(rows), total, err
}

func (r *ChannelRepo) ListOptions(ctx context.Context) ([]model.VideoChannel, error) {
	q := qFrom(ctx).VideoChannel
	rows, err := q.WithContext(ctx).Where(q.Status.Eq(1)).Order(q.ChannelName.Asc(), q.ChannelID.Asc()).Find()
	return valuesOf(rows), err
}

func (r *ChannelRepo) GetByCode(ctx context.Context, code string) (*model.VideoChannel, error) {
	q := qFrom(ctx).VideoChannel
	return q.WithContext(ctx).Where(q.ChannelCode.Eq(code)).First()
}

func (r *ChannelRepo) GetByCodeOrID(ctx context.Context, value string) (*model.VideoChannel, error) {
	if id, err := strconv.ParseUint(value, 10, 64); err == nil && id > 0 {
		q := qFrom(ctx).VideoChannel
		if item, getErr := q.WithContext(ctx).Where(q.ChannelID.Eq(id)).First(); getErr == nil {
			return item, nil
		}
	}
	return r.GetByCode(ctx, value)
}

func (r *ChannelRepo) UpdateFields(ctx context.Context, item *model.VideoChannel) error {
	q := qFrom(ctx).VideoChannel
	_, err := q.WithContext(ctx).Where(q.ChannelID.Eq(item.ChannelID)).Select(
		q.ChannelCode, q.ChannelName, q.AgencyCompany, q.AdPlatform, q.DeliveryPackage,
		q.TrackingURL, q.PortRebate, q.ServiceOrderFee, q.UploadMethod, q.Status,
	).Updates(item)
	return err
}

func (r *ChannelRepo) TemplateCount(ctx context.Context, channelID uint64) (int64, error) {
	// 最新模板模型不再关联渠道。
	return 0, nil
}
