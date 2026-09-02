package event

import (
	"ai-video/internal/config"
	"ai-video/internal/domain"
	"ai-video/internal/gen/model"
	"ai-video/internal/repository"
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

func OrderDayCount(ctx context.Context, order *model.VideoOrder) {
	if order == nil {
		return
	}
	now := time.Now()
	unlock, locked := lockDayCount(ctx, "order", now)
	if !locked {
		return
	}
	defer unlock()

	repo := repository.NewOrderRepo()
	noOrder, err := repo.GetByOrderNo(ctx, order.OrderNo, true)
	if err != nil {
		return
	}
	dayResp := repository.NewDayCountRepo()
	dayCount, err := dayResp.GetByDayTime(ctx, now)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		dayCount = &model.VideoDayCount{}
	}
	isTrue := false
	switch noOrder.Status {
	case domain.OrderStatusEnd:
		if order.PayType == domain.PaymentMethodAppleIAP {
			dayCount.AppleMoney = dayCount.AppleMoney + order.ActualAmountMoney
		} else {
			dayCount.GoogleMoney = dayCount.GoogleMoney + order.ActualAmountMoney
		}
		dayCount.TaxCost = dayCount.TaxCost + (order.PaidAmount - order.ActualAmountMoney)
		dayCount.TransactionCount = dayCount.TransactionCount + 1
		dayCount.EstimatedTotalRevenue = dayCount.EstimatedTotalRevenue + order.ActualAmountMoney

		midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		//检查是否为新用户
		userNew := cheUserNew(ctx, order.UserID, midnight)
		if userNew {
			dayCount.NewUserPaymentAmount = dayCount.NewUserPaymentAmount + order.ActualAmountMoney
			count, _ := repo.GetUserOrderCount(ctx, order.UserID, midnight, now)
			if count == 1 {
				dayCount.NewUserPayerCount = dayCount.NewUserPayerCount + 1
			}

		} else {
			dayCount.OldUserPaymentAmount = dayCount.OldUserPaymentAmount + order.ActualAmountMoney
			count, _ := repo.GetUserOrderCount(ctx, order.UserID, midnight, now)
			if count == 1 {
				dayCount.OldUserPayerCount = dayCount.OldUserPayerCount + 1
			}
		}
		isTrue = true
	case domain.OrderStatusRefunded:
		if order.PayType == domain.PaymentMethodAppleIAP {
			dayCount.AppleRefundedMoney = dayCount.AppleRefundedMoney + order.ActualAmountMoney
		} else {
			dayCount.GoogleRefundedMoney = dayCount.GoogleRefundedMoney + order.ActualAmountMoney
		}
		isTrue = true
	}
	//todo 待完善税收成本计算
	if isTrue {
		if dayCount.ID > 0 {
			err = dayResp.Create(ctx, dayCount)
			if err != nil {
				config.Logger(ctx).Error("day_count.Create", "err", err)
			}
		} else {
			err = dayResp.Update(ctx, dayCount)
			if err != nil {
				config.Logger(ctx).Error("day_count.Update", "err", err)
			}
		}
	}
}

// cheUserNew 判断是否新注册
func cheUserNew(c context.Context, userID uint64, dayTime time.Time) bool {
	user, err := repository.NewAppUserRepo().GetByID(c, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false
	}
	if user == nil {
		return false
	}
	if user.CreatedAt.Unix() >= dayTime.Unix() {
		return true
	}
	return false
}
