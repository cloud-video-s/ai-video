package commerce

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"ai-video/internal/domain"
	"ai-video/internal/repository"

	"gorm.io/gorm"
)

var (
	ErrPackageNotAvailable    = errors.New("application package is not available")
	ErrProductNotAvailable    = errors.New("product is not available for this application package")
	ErrPaymentPackageMismatch = errors.New("payment method does not match application package")
)

// CreatePaymentOrderRequest contains the three business identifiers required
// to start a store purchase. ClientRequestID is optional but recommended for
// safe retrying; the server generates and returns one when it is omitted.
type CreatePaymentOrderRequest struct {
	ShopType        uint32 `json:"shop_type" binding:"required,oneof=1 2"`
	ProductID       uint64 `json:"product_id" binding:"required"`
	PayType         uint32 `json:"pay_type" binding:"required,oneof=1 2"`
	ClientRequestID string `json:"client_request_id" binding:"omitempty,max=64"`
}

// PaymentClientContext carries the authenticated delivery dimensions that
// must be rechecked before a client is allowed to start a store purchase.
type PaymentClientContext struct {
	AppCode              string
	PackageCode          string
	VersionCode          string
	CountryCode          string
	ChannelCode          string
	SystemType           int
	CheckDeliveryTargets bool
}

// StorePaymentInfo tells the app which native store product to purchase.
// Apple uses BundleID while Google Play uses PackageName.
type StorePaymentInfo struct {
	PayType     uint32 `json:"pay_type"`
	ProductID   string `json:"product_id"`
	ProductType string `json:"product_type"`
	BundleID    string `json:"bundle_id,omitempty"`
	PackageName string `json:"package_name,omitempty"`
	Quantity    uint32 `json:"quantity"`
	ConfirmPath string `json:"confirm_path,omitempty"`
}

// CreatePaymentOrderResponse combines the pending local order with the native
// payment information required by StoreKit or Google Play Billing.
type CreatePaymentOrderResponse struct {
	OrderNo         string           `json:"order_no"`
	ClientRequestID string           `json:"client_request_id"`
	ShopType        uint32           `json:"shop_type"`
	ProductID       uint64           `json:"product_id"`
	ProductCode     string           `json:"product_code"`
	ProductName     string           `json:"product_name"`
	PayType         uint32           `json:"pay_type"`
	Status          uint32           `json:"status"`
	Currency        string           `json:"currency"`
	PayableAmount   float64          `json:"payable_amount"`
	ExpiresAt       time.Time        `json:"expires_at"`
	PaymentInfo     StorePaymentInfo `json:"payment_info"`
}

// CreatePaymentOrder validates the requested product against the authenticated
// package, snapshots a pending order, and returns platform payment parameters.
func (s *Service) CreatePaymentOrder(ctx context.Context, userID uint64, packageCode string, req CreatePaymentOrderRequest) (*CreatePaymentOrderResponse, error) {
	return s.createPaymentOrder(ctx, userID, PaymentClientContext{PackageCode: packageCode}, req)
}

// CreatePaymentOrderForClient is the API-facing variant. In addition to the
// package/payment checks retained by CreatePaymentOrder, points products must
// match the current user, app, version, country, system, and channel targets.
func (s *Service) CreatePaymentOrderForClient(ctx context.Context, userID uint64, client PaymentClientContext, req CreatePaymentOrderRequest) (*CreatePaymentOrderResponse, error) {
	client.CheckDeliveryTargets = true
	return s.createPaymentOrder(ctx, userID, client, req)
}

func (s *Service) createPaymentOrder(ctx context.Context, userID uint64, client PaymentClientContext, req CreatePaymentOrderRequest) (*CreatePaymentOrderResponse, error) {
	packageCode := strings.TrimSpace(client.PackageCode)
	if userID == 0 || packageCode == "" {
		return nil, ErrPackageNotAvailable
	}

	payType, expectedSystem, err := paymentMethodForPayType(req.PayType)
	if err != nil {
		return nil, err
	}
	appPackage, err := s.packages.GetByCode(ctx, packageCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPackageNotAvailable
		}
		return nil, err
	}
	if appPackage.Status != 1 {
		return nil, ErrPackageNotAvailable
	}
	if client.CheckDeliveryTargets && strings.TrimSpace(client.AppCode) != "" &&
		appPackage.AppCode != strings.TrimSpace(client.AppCode) {
		return nil, ErrPackageNotAvailable
	}
	if int(appPackage.SystemType) != expectedSystem {
		return nil, ErrPaymentPackageMismatch
	}
	if client.CheckDeliveryTargets && client.SystemType != 0 && client.SystemType != expectedSystem {
		return nil, ErrPaymentPackageMismatch
	}

	if err := s.validateOrderProductForClient(ctx, userID, req.ShopType, req.ProductID, client, int(appPackage.SystemType)); err != nil {
		return nil, err
	}
	clientRequestID := strings.TrimSpace(req.ClientRequestID)
	if clientRequestID == "" {
		clientRequestID = "order:" + newOrderNo()
	}
	order, err := s.CreateOrder(ctx, CreateOrderRequest{
		UserID: userID, ProductType: req.ShopType, ProductID: req.ProductID,
		PayType: payType, ClientRequestID: clientRequestID,
	})
	if err != nil {
		return nil, err
	}

	storeProductType := "subscription"
	if req.ShopType == domain.OrderProductPointsPackage {
		storeProductType = "inapp"
	}
	paymentInfo := StorePaymentInfo{
		PayType: payType, ProductID: order.ProductCode,
		ProductType: storeProductType, Quantity: 1,
	}
	if req.PayType == domain.OrderPayTypeApple {
		paymentInfo.BundleID = packageCode
		paymentInfo.ConfirmPath = "/api/payments/apple/pay"
	} else {
		paymentInfo.PackageName = packageCode
	}
	return &CreatePaymentOrderResponse{
		OrderNo: order.OrderNo, ClientRequestID: order.ClientRequestID,
		ShopType: order.ProductType, ProductID: order.ProductID,
		ProductCode: order.ProductCode, ProductName: order.ProductName,
		PayType: req.PayType, Status: order.Status, Currency: order.Currency,
		PayableAmount: order.PayableAmount, ExpiresAt: order.ExpiresAt,
		PaymentInfo: paymentInfo,
	}, nil
}

func (s *Service) validateOrderProductForClient(
	ctx context.Context,
	userID uint64,
	shopType uint32,
	productID uint64,
	client PaymentClientContext,
	packageSystem int,
) error {
	packageCode := strings.TrimSpace(client.PackageCode)
	if shopType != domain.OrderProductPointsPackage || !client.CheckDeliveryTargets {
		return s.validateOrderProduct(ctx, shopType, productID, packageCode)
	}

	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	countryCode := strings.ToUpper(strings.TrimSpace(client.CountryCode))
	if countryCode == "" {
		countryCode = strings.ToUpper(strings.TrimSpace(user.ClientCountry))
	}
	systemType := client.SystemType
	if systemType == 0 {
		systemType = packageSystem
	}
	items, err := s.points.ListForClient(ctx, repository.ClientPointsTargets{
		ProductID: productID, AppCode: strings.TrimSpace(client.AppCode),
		PackageCode: packageCode, VersionCode: strings.TrimSpace(client.VersionCode),
		CountryCode: countryCode, ChannelCode: strings.TrimSpace(client.ChannelCode),
		System: paymentSystemName(systemType), UserType: int(user.UserType),
	})
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return ErrProductNotAvailable
	}
	return nil
}

func (s *Service) validateOrderProduct(ctx context.Context, shopType uint32, productID uint64, packageCode string) error {
	var err error
	switch shopType {
	case domain.OrderProductVIPSubscription:
		_, err = s.vipProducts.GetEnabledForPackage(ctx, productID, packageCode)
	case domain.OrderProductPointsPackage:
		_, err = s.points.GetEnabledForPackage(ctx, productID, packageCode)
	default:
		return ErrUnsupportedProduct
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrProductNotAvailable
	}
	return err
}

func paymentMethodForPayType(payType uint32) (pay uint32, expectedSystem int, err error) {
	switch payType {
	case domain.OrderPayTypeApple:
		return domain.PaymentMethodAppleIAP, domain.SystemTypeIos, nil
	case domain.OrderPayTypeGoogle:
		return domain.PaymentMethodGooglePlay, domain.SystemTypeA, nil
	default:
		return 0, 0, fmt.Errorf("%w: pay_type=%d", ErrUnsupportedPaymentMethod, payType)
	}
}

func paymentSystemName(systemType int) string {
	switch systemType {
	case domain.SystemTypeIos:
		return "ios"
	case domain.SystemTypeA:
		return "android"
	default:
		return ""
	}
}
