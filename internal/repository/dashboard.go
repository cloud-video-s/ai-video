package repository

import (
	"context"
	"time"

	"ai-video/internal/domain"
	"ai-video/internal/gen/model"
)

// DashboardCurrencyAmount keeps transaction totals separated by currency.
// Adding amounts across currencies would make the dashboard value misleading.
type DashboardCurrencyAmount struct {
	Currency string  `gorm:"column:currency" json:"currency"`
	Amount   float64 `gorm:"column:amount" json:"amount"`
}

// DashboardMonthStats contains the event counts for one natural month.
type DashboardMonthStats struct {
	RegisteredUserCount           int64                     `json:"registered_user_count"`
	SuccessfulSubscriptionCount   int64                     `json:"successful_subscription_count"`
	TransactionAmounts            []DashboardCurrencyAmount `json:"transaction_amounts"`
	SuccessfulGenerationTaskCount int64                     `json:"successful_generation_task_count"`
}

type DashboardRepo struct{}

func NewDashboardRepo() *DashboardRepo { return &DashboardRepo{} }

// MonthStats calculates historical event metrics using business-event times:
// user creation, payment completion, and generation completion respectively.
// Unscoped queries keep past events in the totals even if a row is later soft
// deleted.
func (r *DashboardRepo) MonthStats(ctx context.Context, start, end time.Time, successfulTaskStatus int) (*DashboardMonthStats, error) {
	db := qFrom(ctx).UnderlyingDB()
	stats := &DashboardMonthStats{TransactionAmounts: make([]DashboardCurrencyAmount, 0)}

	if err := db.Unscoped().Model(&model.VideoUser{}).
		Where("registered = ? AND created_at >= ? AND created_at < ?", 1, start, end).
		Count(&stats.RegisteredUserCount).Error; err != nil {
		return nil, err
	}

	if err := db.Unscoped().Model(&model.VideoOrder{}).
		Where("status IN ? AND product_type = ?  AND pay_channel = ? AND pay_time >= ? AND pay_time < ?",
			[]int{domain.OrderStatusPaid, domain.OrderStatusEnd}, domain.OrderProductVIPSubscription, 2, start, end).
		Count(&stats.SuccessfulSubscriptionCount).Error; err != nil {
		return nil, err
	}
	if err := db.Unscoped().Model(&model.VideoOrder{}).
		Where("pay_channel = ? AND status IN ? AND pay_time >= ? AND pay_time < ?", 2, []int{domain.OrderStatusPaid, domain.OrderStatusEnd}, start, end).
		Select("currency, COALESCE(SUM(paid_amount), 0) AS amount").
		Group("currency").Order("currency ASC").
		Scan(&stats.TransactionAmounts).Error; err != nil {
		return nil, err
	}

	if err := db.Unscoped().Model(&model.VideoUserGenerationTask{}).
		Where("status = ? AND finished_at >= ? AND finished_at < ?", successfulTaskStatus, start, end).
		Count(&stats.SuccessfulGenerationTaskCount).Error; err != nil {
		return nil, err
	}
	return stats, nil
}
