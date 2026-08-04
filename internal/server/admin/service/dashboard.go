package service

import (
	"context"
	"strings"
	"time"

	"ai-video/internal/domain"
	"ai-video/internal/generation"
	"ai-video/internal/repository"
)

type DashboardService struct {
	dashboardRepo *repository.DashboardRepo
	orderRepo     *repository.OrderRepo
}

func NewDashboardService() *DashboardService {
	return &DashboardService{
		dashboardRepo: repository.NewDashboardRepo(),
		orderRepo:     repository.NewOrderRepo(),
	}
}

// DashboardRequest contains filters for the subscription-order table. The
// monthly statistic cards always retain the current natural-month scope.
type DashboardRequest struct {
	UserID        uint64 `form:"user_id"`
	ProductCode   string `form:"product_code" binding:"max=191"`
	Status        uint32 `form:"status" binding:"max=20"`
	PaymentMethod string `form:"payment_method" binding:"max=32"`
	Keyword       string `form:"keyword" binding:"max=255"`
	DateFrom      string `form:"date_from" binding:"omitempty,datetime=2006-01-02"`
	DateTo        string `form:"date_to" binding:"omitempty,datetime=2006-01-02"`
}

type DashboardStatisticsView struct {
	Month                         string                               `json:"month"`
	PeriodStart                   time.Time                            `json:"period_start"`
	PeriodEnd                     time.Time                            `json:"period_end"`
	Timezone                      string                               `json:"timezone"`
	RegisteredUserCount           int64                                `json:"registered_user_count"`
	SuccessfulSubscriptionCount   int64                                `json:"successful_subscription_count"`
	TransactionAmounts            []repository.DashboardCurrencyAmount `json:"transaction_amounts"`
	SuccessfulGenerationTaskCount int64                                `json:"successful_generation_task_count"`
}

type DashboardSubscriptionOrdersView struct {
	List  []OrderAdminView `json:"list"`
	Total int64            `json:"total"`
	Page  int              `json:"page"`
	Size  int              `json:"size"`
}

type DashboardView struct {
	Statistics         DashboardStatisticsView         `json:"statistics"`
	SubscriptionOrders DashboardSubscriptionOrdersView `json:"subscription_orders"`
}

func (s *DashboardService) Get(ctx context.Context, page, pageSize int, req *DashboardRequest) (*DashboardView, error) {
	start, end := currentMonthRange(time.Now())
	stats, err := s.dashboardRepo.MonthStats(ctx, start, end, generation.TaskStatusSuccess)
	if err != nil {
		return nil, err
	}

	createdFrom, createdTo, err := parseOrderDateRange(req.DateFrom, req.DateTo)
	if err != nil {
		return nil, err
	}
	records, total, err := s.orderRepo.PageAdminList(ctx, page, pageSize, &repository.OrderAdminFilter{
		UserID: req.UserID, ProductType: domain.OrderProductVIPSubscription,
		ProductCode: strings.TrimSpace(req.ProductCode), Status: req.Status,
		PaymentMethod: strings.TrimSpace(req.PaymentMethod), Keyword: strings.TrimSpace(req.Keyword),
		CreatedFrom: createdFrom, CreatedTo: createdTo,
	})
	if err != nil {
		return nil, err
	}
	orders := make([]OrderAdminView, 0, len(records))
	for i := range records {
		orders = append(orders, orderAdminView(&records[i], false))
	}

	return &DashboardView{
		Statistics: DashboardStatisticsView{
			Month: start.Format("2006-01"), PeriodStart: start, PeriodEnd: end,
			Timezone: time.Local.String(), RegisteredUserCount: stats.RegisteredUserCount,
			SuccessfulSubscriptionCount:   stats.SuccessfulSubscriptionCount,
			TransactionAmounts:            stats.TransactionAmounts,
			SuccessfulGenerationTaskCount: stats.SuccessfulGenerationTaskCount,
		},
		SubscriptionOrders: DashboardSubscriptionOrdersView{
			List: orders, Total: total, Page: page, Size: pageSize,
		},
	}, nil
}

func currentMonthRange(now time.Time) (time.Time, time.Time) {
	localNow := now.In(time.Local)
	start := time.Date(localNow.Year(), localNow.Month(), 1, 0, 0, 0, 0, time.Local)
	return start, start.AddDate(0, 1, 0)
}
