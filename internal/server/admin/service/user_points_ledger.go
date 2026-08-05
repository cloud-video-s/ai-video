package service

import (
	"ai-video/internal/repository"
	"context"
	"errors"
	"strings"
	"time"
)

type UserPointsLedgerService struct {
	repo *repository.UserPointsLedgerRepo
}

func NewUserPointsLedgerService() *UserPointsLedgerService {
	return &UserPointsLedgerService{repo: repository.NewUserPointsLedgerRepo()}
}

type ListUserPointsLedgerRequest struct {
	UserID          uint64 `form:"user_id"`
	Direction       int8   `form:"direction" binding:"omitempty,oneof=1 2"`
	SourceType      uint32 `form:"source_type" binding:"omitempty,oneof=1 2 3 4 5 6 7 8"`
	PointsID        uint64 `form:"points_id"`
	PointsPackageID uint64 `form:"points_package_id"`
	OrderCode       string `form:"order_code" binding:"max=191"`
	BusinessID      string `form:"business_id" binding:"max=191"`
	Keyword         string `form:"keyword" binding:"max=255"`
	DateFrom        string `form:"date_from" binding:"omitempty,datetime=2006-01-02"`
	DateTo          string `form:"date_to" binding:"omitempty,datetime=2006-01-02"`
}

func (s *UserPointsLedgerService) List(ctx context.Context, page, pageSize int, req *ListUserPointsLedgerRequest) ([]repository.UserPointsLedgerRecord, int64, repository.UserPointsLedgerSummary, error) {
	from, to, err := parsePointsLedgerDateRange(req.DateFrom, req.DateTo)
	if err != nil {
		return nil, 0, repository.UserPointsLedgerSummary{}, err
	}
	pointsID := req.PointsID
	if pointsID == 0 {
		pointsID = req.PointsPackageID
	}
	orderCode := strings.TrimSpace(req.OrderCode)
	if orderCode == "" {
		orderCode = strings.TrimSpace(req.BusinessID)
	}
	return s.repo.PageList(ctx, page, pageSize, &repository.UserPointsLedgerFilter{
		UserID:       req.UserID,
		Direction:    req.Direction,
		SourceType:   req.SourceType,
		PointsID:     pointsID,
		OrderCode:    orderCode,
		Keyword:      strings.TrimSpace(req.Keyword),
		OccurredFrom: from,
		OccurredTo:   to,
	})
}

func (s *UserPointsLedgerService) GetByID(ctx context.Context, id uint64) (*repository.UserPointsLedgerRecord, error) {
	item, err := s.repo.GetDetail(ctx, id)
	if err != nil {
		return nil, notFoundOr(err, "积分明细不存在")
	}
	return item, nil
}

func parsePointsLedgerDateRange(fromValue, toValue string) (*time.Time, *time.Time, error) {
	var from, to *time.Time
	if fromValue != "" {
		value, err := time.ParseInLocation("2006-01-02", fromValue, time.Local)
		if err != nil {
			return nil, nil, errors.New("开始日期格式错误")
		}
		from = &value
	}
	if toValue != "" {
		value, err := time.ParseInLocation("2006-01-02", toValue, time.Local)
		if err != nil {
			return nil, nil, errors.New("结束日期格式错误")
		}
		value = value.AddDate(0, 0, 1)
		to = &value
	}
	if from != nil && to != nil && !from.Before(*to) {
		return nil, nil, errors.New("开始日期不能晚于结束日期")
	}
	return from, to, nil
}
