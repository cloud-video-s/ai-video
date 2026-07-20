package repository

import (
	"context"
	"time"

	"ai-video/internal/model"

	"gorm.io/gorm"
)

type UserPointsLedgerRepo struct{}

func NewUserPointsLedgerRepo() *UserPointsLedgerRepo { return &UserPointsLedgerRepo{} }

type UserPointsLedgerFilter struct {
	UserID          uint64
	Direction       int8
	SourceType      string
	PointsPackageID uint64
	BusinessID      string
	Keyword         string
	OccurredFrom    *time.Time
	OccurredTo      *time.Time
}

type UserPointsLedgerSummary struct {
	IncomeTotal  int64 `json:"income_total"`
	ExpenseTotal int64 `json:"expense_total"`
}

func (r *UserPointsLedgerRepo) Create(ctx context.Context, item *model.VideoUserPointsLedger) error {
	return dbFrom(ctx).Create(item).Error
}

func (r *UserPointsLedgerRepo) PageList(ctx context.Context, page, pageSize int, filter *UserPointsLedgerFilter) ([]model.VideoUserPointsLedger, int64, UserPointsLedgerSummary, error) {
	buildQuery := func() *gorm.DB {
		db := dbFrom(ctx).Model(&model.VideoUserPointsLedger{})
		if filter == nil {
			return db
		}
		if filter.UserID != 0 {
			db = db.Where("user_id = ?", filter.UserID)
		}
		if filter.Direction != 0 {
			db = db.Where("direction = ?", filter.Direction)
		}
		if filter.SourceType != "" {
			db = db.Where("source_type = ?", filter.SourceType)
		}
		if filter.PointsPackageID != 0 {
			db = db.Where("points_package_id = ?", filter.PointsPackageID)
		}
		if filter.BusinessID != "" {
			db = db.Where("business_id = ?", filter.BusinessID)
		}
		if filter.OccurredFrom != nil {
			db = db.Where("occurred_at >= ?", *filter.OccurredFrom)
		}
		if filter.OccurredTo != nil {
			db = db.Where("occurred_at < ?", *filter.OccurredTo)
		}
		if filter.Keyword != "" {
			keyword := "%" + filter.Keyword + "%"
			db = db.Where(`business_id LIKE ? OR description LIKE ? OR EXISTS (
				SELECT 1 FROM video_user vu WHERE vu.id = video_user_points_ledger.user_id
				AND (vu.username LIKE ? OR vu.imei LIKE ? OR vu.login_account LIKE ? OR vu.google_email LIKE ? OR vu.appid_email LIKE ?)
			)`, keyword, keyword, keyword, keyword, keyword, keyword, keyword)
		}
		return db
	}

	var total int64
	if err := buildQuery().Count(&total).Error; err != nil {
		return nil, 0, UserPointsLedgerSummary{}, err
	}
	var summary UserPointsLedgerSummary
	if err := buildQuery().Select(
		"COALESCE(SUM(CASE WHEN points_change > 0 THEN points_change ELSE 0 END), 0) AS income_total, " +
			"COALESCE(SUM(CASE WHEN points_change < 0 THEN -points_change ELSE 0 END), 0) AS expense_total",
	).Scan(&summary).Error; err != nil {
		return nil, 0, UserPointsLedgerSummary{}, err
	}
	var list []model.VideoUserPointsLedger
	err := buildQuery().Preload("User").Preload("PointsPackage").Order("occurred_at DESC, id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, summary, err
}

func (r *UserPointsLedgerRepo) GetDetail(ctx context.Context, id uint64) (*model.VideoUserPointsLedger, error) {
	var item model.VideoUserPointsLedger
	if err := dbFrom(ctx).Preload("User").Preload("PointsPackage").First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}
