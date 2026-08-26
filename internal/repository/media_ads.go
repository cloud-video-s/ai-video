package repository

import (
	"ai-video/internal/gen/model"
	"context"
)

type MediaAdsRepo struct{}

func NewMediaAdsRepo() *MediaAdsRepo {
	return &MediaAdsRepo{}
}

type MediaAdsOption struct {
	MediaId     uint64 `gorm:"column:media_id" json:"media_id"`
	AdAccountId uint64 `gorm:"column:ad_account_id" json:"ad_account_id"`
	CampaignId  uint64 `gorm:"column:campaign_id" json:"campaign_id"`
	AdgroupId   uint64 `gorm:"column:ad_group_id" json:"ad_group_id"`
	CreativeId  uint64 `gorm:"column:creative_id" json:"creative_id"`
}

func (r *MediaAdsRepo) GetByMediaAds(ctx context.Context, req MediaAdsOption) (*model.VideoMediaAd, error) {
	q := qFrom(ctx).VideoMediaAd
	sql := q.WithContext(ctx)
	if req.MediaId > 0 {
		sql = sql.Where(q.MediaID.Eq(req.MediaId))
	}
	if req.AdAccountId > 0 {
		sql = sql.Where(q.AdAccountID.Eq(req.AdAccountId))
	}
	if req.AdgroupId > 0 {
		sql = sql.Where(q.AdgroupID.Eq(req.AdgroupId))
	}
	if req.CampaignId > 0 {
		sql = sql.Where(q.CampaignID.Eq(int64(req.CampaignId)))
	}
	if req.CreativeId > 0 {
		sql = sql.Where(q.CreativeID.Eq(int64(req.CreativeId)))
	}
	return sql.First()
}

func (r *MediaAdsRepo) Update(ctx context.Context, id uint64, updates map[string]any) error {
	q := qFrom(ctx).VideoMediaAd
	_, err := q.WithContext(ctx).Where(q.ID.Eq(id)).Updates(updates)
	if err != nil {
		return err
	}
	return nil
}

func (r *MediaAdsRepo) GetMediaAdsFind(ctx context.Context, req MediaAdsOption) ([]*model.VideoMediaAd, error) {
	q := qFrom(ctx).VideoMediaAd
	sql := q.WithContext(ctx)
	if req.MediaId > 0 {
		sql = sql.Where(q.MediaID.Eq(req.MediaId))
	}
	if req.AdAccountId > 0 {
		sql = sql.Where(q.AdAccountID.Eq(req.AdAccountId))
	}
	if req.AdgroupId > 0 {
		sql = sql.Where(q.AdgroupID.Eq(req.AdgroupId))
	}
	if req.CampaignId > 0 {
		sql = sql.Where(q.CampaignID.Eq(int64(req.CampaignId)))
	}
	if req.CreativeId > 0 {
		sql = sql.Where(q.CreativeID.Eq(int64(req.CreativeId)))
	}
	return sql.Find()
}

func (r *MediaAdsRepo) Create(ctx context.Context, info *model.VideoMediaAd) error {
	return qFrom(ctx).VideoMediaAd.WithContext(ctx).UnderlyingDB().Create(info).Error
}

func (r *MediaAdsRepo) Delete(ctx context.Context, id uint64) error {
	q := qFrom(ctx).VideoMediaAd
	_, err := q.WithContext(ctx).Where(q.ID.Eq(id)).Delete()
	if err != nil {
		return err
	}
	return nil
}
