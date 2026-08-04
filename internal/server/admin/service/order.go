package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"ai-video/internal/repository"
)

type OrderAdminService struct {
	repo *repository.OrderRepo
}

func NewOrderAdminService() *OrderAdminService {
	return &OrderAdminService{repo: repository.NewOrderRepo()}
}

type ListOrderRequest struct {
	UserID        uint64 `form:"user_id"`
	ProductType   uint32 `form:"product_type" binding:"omitempty,oneof=1 2"`
	ProductCode   string `form:"product_code" binding:"max=191"`
	Status        string `form:"status" binding:"max=20"`
	PaymentMethod string `form:"payment_method" binding:"max=32"`
	Keyword       string `form:"keyword" binding:"max=255"`
	DateFrom      string `form:"date_from" binding:"omitempty,datetime=2006-01-02"`
	DateTo        string `form:"date_to" binding:"omitempty,datetime=2006-01-02"`
}

type OrderAdminUserView struct {
	ID           uint64 `json:"id"`
	Username     string `json:"username"`
	LoginAccount string `json:"login_account"`
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	IMEI         string `json:"imei"`
	DeviceCode   string `json:"device_code"`
	UserType     uint8  `json:"user_type"`
	Status       int8   `json:"status"`
	Deleted      bool   `json:"deleted"`
}

type OrderAdminView struct {
	ID                    uint64              `json:"id"`
	OrderNo               string              `json:"order_no"`
	ClientRequestID       string              `json:"client_request_id"`
	UserID                uint64              `json:"user_id"`
	ProductType           uint32              `json:"product_type"`
	ProductID             uint64              `json:"product_id"`
	ProductCode           string              `json:"product_code"`
	ProductName           string              `json:"product_name"`
	Currency              string              `json:"currency"`
	ProductAmount         float64             `json:"product_amount"`
	DiscountAmount        float64             `json:"discount_amount"`
	PayableAmount         float64             `json:"payable_amount"`
	PaidAmount            float64             `json:"paid_amount"`
	RefundedAmount        float64             `json:"refunded_amount"`
	BonusPoints           uint64              `json:"bonus_points"`
	VIPLevel              uint                `json:"vip_level"`
	VIPDurationDays       uint                `json:"vip_duration_days"`
	Status                string              `json:"status"`
	PaymentMethod         string              `json:"payment_method"`
	ThirdOrderNo          string              `json:"third_order_no"`
	OriginalTransactionID string              `json:"original_transaction_id"`
	PaymentEvidence       string              `json:"payment_evidence,omitempty"`
	FailureCode           string              `json:"failure_code"`
	FailureMessage        string              `json:"failure_message"`
	CancelReason          string              `json:"cancel_reason"`
	PaidAt                *time.Time          `json:"paid_at"`
	CancelledAt           *time.Time          `json:"cancelled_at"`
	ExpiresAt             *time.Time          `json:"expires_at"`
	CreatedAt             time.Time           `json:"created_at"`
	UpdatedAt             time.Time           `json:"updated_at"`
	DeletedAt             *time.Time          `json:"deleted_at"`
	User                  *OrderAdminUserView `json:"user,omitempty"`
}

func (s *OrderAdminService) List(ctx context.Context, page, pageSize int, req *ListOrderRequest) ([]OrderAdminView, int64, repository.OrderAdminSummary, error) {
	from, to, err := parseOrderDateRange(req.DateFrom, req.DateTo)
	if err != nil {
		return nil, 0, repository.OrderAdminSummary{}, err
	}
	records, total, summary, err := s.repo.PageAdmin(ctx, page, pageSize, &repository.OrderAdminFilter{
		UserID: req.UserID, ProductType: req.ProductType,
		ProductCode: strings.TrimSpace(req.ProductCode), Status: strings.TrimSpace(req.Status),
		PaymentMethod: strings.TrimSpace(req.PaymentMethod), Keyword: strings.TrimSpace(req.Keyword),
		CreatedFrom: from, CreatedTo: to,
	})
	if err != nil {
		return nil, 0, repository.OrderAdminSummary{}, err
	}
	items := make([]OrderAdminView, 0, len(records))
	for i := range records {
		items = append(items, orderAdminView(&records[i], false))
	}
	return items, total, summary, nil
}

func (s *OrderAdminService) GetByID(ctx context.Context, id uint64) (*OrderAdminView, error) {
	record, err := s.repo.GetAdminDetail(ctx, id)
	if err != nil {
		return nil, notFoundOr(err, "订单不存在")
	}
	view := orderAdminView(record, true)
	return &view, nil
}

func orderAdminView(record *repository.OrderAdminRecord, includeEvidence bool) OrderAdminView {
	order := record.Order
	view := OrderAdminView{
		ID: order.ID, OrderNo: order.OrderNo, ClientRequestID: order.ClientRequestID,
		UserID: order.UserID, ProductType: order.ProductType, ProductID: order.ProductID,
		ProductCode: order.ProductCode, ProductName: order.ProductName, Currency: order.Currency,
		ProductAmount: order.ProductAmount, DiscountAmount: order.DiscountAmount,
		PayableAmount: order.PayableAmount, PaidAmount: order.PaidAmount, RefundedAmount: order.RefundedAmount,
		BonusPoints: order.BonusPoints, VIPLevel: order.VipLevel, VIPDurationDays: order.VipDurationDays,
		Status: order.Status, PaymentMethod: order.PaymentMethod, ThirdOrderNo: order.ThirdOrderNo,
		OriginalTransactionID: order.OriginalTransactionID, FailureCode: order.FailureCode,
		FailureMessage: order.FailureMessage, CancelReason: order.CancelReason,
		PaidAt: orderAdminTimePtr(order.PaidAt), CancelledAt: orderAdminTimePtr(order.CancelledAt),
		ExpiresAt: orderAdminTimePtr(order.ExpiresAt), CreatedAt: order.CreatedAt, UpdatedAt: order.UpdatedAt,
	}
	if includeEvidence {
		view.PaymentEvidence = order.PaymentEvidence
	}
	if order.DeletedAt.Valid {
		deletedAt := order.DeletedAt.Time
		view.DeletedAt = &deletedAt
	}
	if record.User != nil {
		view.User = &OrderAdminUserView{
			ID: record.User.ID, Username: record.User.Username, LoginAccount: record.User.LoginAccount,
			Email: record.User.Email, Phone: record.User.Phone, IMEI: record.User.IMEI,
			DeviceCode: record.User.DeviceCode, UserType: record.User.UserType,
			Status: record.User.Status, Deleted: record.User.DeletedAt.Valid,
		}
	}
	return view
}

func parseOrderDateRange(fromValue, toValue string) (*time.Time, *time.Time, error) {
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

func orderAdminTimePtr(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	return &value
}
