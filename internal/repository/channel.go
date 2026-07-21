package repository

import (
	"context"
	"strconv"
	"strings"

	"ai-video/internal/gen/model"

	"github.com/gin-gonic/gin"
)

type ChannelRepo struct {
	BaseRepo[model.VideoChannel]
}

func (r *ChannelRepo) ResolveEnabledTargets(ctx *gin.Context, codeOrID, deliveryPackage string) ([]model.VideoChannel, error) {
	db := dbFrom(ctx).Model(&model.VideoChannel{}).Where("status = ?", 1)
	if value := strings.TrimSpace(codeOrID); value != "" {
		if id, err := strconv.ParseUint(value, 10, 64); err == nil && id > 0 {
			db = db.Where("channel_id = ?", id)
		} else {
			db = db.Where("channel_code = ?", value)
		}
	}
	if value := strings.TrimSpace(deliveryPackage); value != "" {
		db = db.Where("delivery_package = ?", value)
	}
	if strings.TrimSpace(codeOrID) == "" && strings.TrimSpace(deliveryPackage) == "" {
		return []model.VideoChannel{}, nil
	}
	var list []model.VideoChannel
	if err := db.Order("channel_id ASC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
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
	q := &QueryOptions{Where: map[string]interface{}{}, Order: []string{"channel_id DESC"}}
	if filter != nil {
		if filter.AdPlatform != "" {
			q.Where["ad_platform"] = filter.AdPlatform
		}
		if filter.UploadMethod != "" {
			q.Where["upload_method"] = filter.UploadMethod
		}
		if filter.Status != nil {
			q.Where["status"] = *filter.Status
		}
		if filter.Keyword != "" {
			keyword := "%" + filter.Keyword + "%"
			q.Conds = append(q.Conds, Cond{
				Query: "channel_code LIKE ? OR channel_name LIKE ? OR agency_company LIKE ? OR delivery_package LIKE ?",
				Args:  []interface{}{keyword, keyword, keyword, keyword},
			})
		}
	}
	return r.BaseRepo.PageList(ctx, page, pageSize, q)
}

func (r *ChannelRepo) ListOptions(ctx context.Context) ([]model.VideoChannel, error) {
	return r.BaseRepo.List(ctx, &QueryOptions{
		Where: map[string]interface{}{"status": int8(1)},
		Order: []string{"channel_name ASC", "channel_id ASC"},
	})
}

func (r *ChannelRepo) GetByCode(ctx context.Context, code string) (*model.VideoChannel, error) {
	return r.BaseRepo.GetOne(ctx, &QueryOptions{Where: map[string]interface{}{"channel_code": code}})
}

func (r *ChannelRepo) GetByCodeOrID(ctx context.Context, value string) (*model.VideoChannel, error) {
	if id, err := strconv.ParseUint(value, 10, 64); err == nil && id > 0 {
		if item, getErr := r.GetByID(ctx, uint(id)); getErr == nil {
			return item, nil
		}
	}
	return r.GetByCode(ctx, value)
}

func (r *ChannelRepo) UpdateFields(ctx context.Context, item *model.VideoChannel) error {
	return r.BaseRepo.Update(ctx, item,
		"ChannelCode", "ChannelName", "AgencyCompany", "AdPlatform", "DeliveryPackage",
		"TrackingURL", "PortRebate", "ServiceOrderFee", "UploadMethod", "Status",
	)
}

func (r *ChannelRepo) TemplateCount(ctx context.Context, channelID uint64) (int64, error) {
	var templateCount, typeCount int64
	if err := dbFrom(ctx).Table("video_template_channel").Where("channel_id = ?", channelID).Count(&templateCount).Error; err != nil {
		return 0, err
	}
	if err := dbFrom(ctx).Table("video_template_type_channel").Where("channel_id = ?", channelID).Count(&typeCount).Error; err != nil {
		return 0, err
	}
	return templateCount + typeCount, nil
}
