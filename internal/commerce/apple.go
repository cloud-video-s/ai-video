package commerce

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"ai-video/internal/config"
	"ai-video/internal/domain"
	"ai-video/internal/gen/model"

	"gorm.io/gorm"
)

var (
	ErrAppleEvidenceInvalid     = errors.New("invalid Apple transaction evidence")
	ErrAppleSignatureInvalid    = errors.New("Apple transaction signature verification failed")
	ErrAppleUnsignedProduction  = errors.New("unsigned Apple transaction is not allowed in production")
	ErrAppleBundleMismatch      = errors.New("Apple transaction bundle does not match request package")
	ErrAppleEnvironmentMismatch = errors.New("Apple transaction environment does not match notification")
	ErrAppleProductNotFound     = errors.New("Apple product is not configured for this package")
	ErrAppleProductAmbiguous    = errors.New("Apple product is configured as more than one product type")
	ErrApplePurchaseInactive    = errors.New("Apple subscription is inactive or expired")
	ErrApplePurchaseRevoked     = errors.New("Apple transaction has been revoked")
)

// ApplePurchaseRequest mirrors the StoreKit result returned by the app. The
// signedTransactionInfo field may be a compact JWS in production. A decoded
// JSON value is accepted only for Sandbox while the server is not in release
// mode, matching the development payload supplied by the client.
type ApplePurchaseRequest struct {
	ShopType              int        `json:"shop_type" binding:"required,max=2"`
	BundleID              string     `json:"bundleID" binding:"required,max=191"`
	ExpirationDate        *time.Time `json:"expirationDate"`
	IsActive              bool       `json:"isActive"`
	OriginalTransactionID string     `json:"originalTransactionID" binding:"required,max=191"`
	ProductID             string     `json:"productID" binding:"required,max=191"`
	PurchaseDate          time.Time  `json:"purchaseDate" binding:"required"`
	RevocationDate        *time.Time `json:"revocationDate"`
	SignedTransactionInfo string     `json:"signedTransactionInfo" binding:"required"`
	Source                string     `json:"source" binding:"omitempty,max=64"`
	TransactionID         string     `json:"transactionID" binding:"required,max=191"`
}

// appleSignedTransaction is shared by StoreKit purchase confirmation and
// nested App Store Server Notifications V2 transaction data.
type appleSignedTransaction struct {
	TransactionID         string `json:"transactionId"`
	OriginalTransactionID string `json:"originalTransactionId"`
	BundleID              string `json:"bundleId"`
	ProductID             string `json:"productId"`
	PurchaseDate          int64  `json:"purchaseDate"`
	OriginalPurchaseDate  int64  `json:"originalPurchaseDate"`
	ExpiresDate           int64  `json:"expiresDate"`
	RevocationDate        int64  `json:"revocationDate"`
	RevocationReason      string `json:"revocationReason"`
	Quantity              int64  `json:"quantity"`
	Type                  string `json:"type"`
	SignedDate            int64  `json:"signedDate"`
	Environment           string `json:"environment"`
	TransactionReason     string `json:"transactionReason"`
	Price                 int64  `json:"price"`
	Currency              string `json:"currency"`
	OfferType             int64  `json:"offerType"`
	OfferIdentifier       string `json:"offerIdentifier"`
}

// verifiedAppleTransaction combines Apple's signed fields with normalized
// timestamps and currency values used by the order service.
type verifiedAppleTransaction struct {
	appleSignedTransaction
	PurchaseAt   time.Time
	ExpiresAt    *time.Time
	RevokedAt    *time.Time
	PaidAmount   float64
	EvidenceMode string
}

// ApplePurchaseResponse reports the fulfilled local order together with the
// verified StoreKit transaction state.
type ApplePurchaseResponse struct {
	OrderNo               string     `json:"order_no"`
	Status                string     `json:"status"`
	ProductType           uint32     `json:"product_type"`
	ProductID             uint64     `json:"product_id"`
	ProductCode           string     `json:"product_code"`
	TransactionID         string     `json:"transaction_id"`
	OriginalTransactionID string     `json:"original_transaction_id"`
	Currency              string     `json:"currency"`
	PaidAmount            float64    `json:"paid_amount"`
	PurchaseDate          time.Time  `json:"purchase_date"`
	ExpirationDate        *time.Time `json:"expiration_date,omitempty"`
	IsActive              bool       `json:"is_active"`
	Environment           string     `json:"environment"`
	EvidenceMode          string     `json:"evidence_mode"`
}

// ConfirmApplePurchase verifies the StoreKit result, resolves the SKU under
// the authenticated app package, creates an order and fulfills it exactly once.
func (s *Service) ConfirmApplePurchase(ctx context.Context, userID uint64, expectedBundle string, req ApplePurchaseRequest) (*ApplePurchaseResponse, error) {
	if userID == 0 {
		return nil, errors.New("authenticated user is required")
	}
	expectedBundle = strings.TrimSpace(expectedBundle)
	verified, err := verifyApplePurchase(req, expectedBundle, config.Cfg.Server.Mode != "release")
	if err != nil {
		return nil, err
	}

	if existing, lookupErr := s.orders.GetByPaymentTransaction(ctx, domain.PaymentMethodAppleIAP, verified.TransactionID); lookupErr == nil {
		if existing.UserID != userID || existing.ProductCode != verified.ProductID {
			return nil, ErrPaymentTransactionUsed
		}
		return applePurchaseResponse(existing, verified), nil
	} else if !errors.Is(lookupErr, gorm.ErrRecordNotFound) {
		return nil, lookupErr
	}
	if verified.RevokedAt != nil {
		return nil, ErrApplePurchaseRevoked
	}
	if isSubscriptionType(verified.Type) {
		//if !req.IsActive || verified.ExpiresAt == nil || !verified.ExpiresAt.After(time.Now()) {
		//	return nil, ErrApplePurchaseInactive
		//}
	}
	if !isSubscriptionType(verified.Type) {
		return nil, ErrPaymentMismatch
	}
	shopID, err := s.resolveAppleProduct(ctx, req.ShopType, verified.ProductID, expectedBundle)
	if err != nil {
		return nil, err
	}
	productType := domain.OrderProductVIPSubscription
	if req.ShopType == 2 {
		productType = domain.OrderProductPointsPackage
	}
	order, err := s.CreateOrder(ctx, CreateOrderRequest{
		UserID: userID, ProductType: uint32(productType), ProductID: shopID,
		PaymentMethod:   domain.PaymentMethodAppleIAP,
		ClientRequestID: appleClientRequestID(verified.TransactionID),
		Renewal: strings.EqualFold(verified.TransactionReason, "RENEWAL") ||
			verified.TransactionID != verified.OriginalTransactionID,
		PaidAmount: verified.PaidAmount,
	})
	if err != nil {
		return nil, err
	}
	paid, err := s.ConfirmApplePayment(ctx, order.OrderNo, ApplePaymentResult{
		TransactionID: verified.TransactionID, OriginalTransactionID: verified.OriginalTransactionID,
		ProductCode: verified.ProductID, Currency: verified.Currency, PaidAmount: verified.PaidAmount,
		SignedTransaction: req.SignedTransactionInfo, PurchaseDate: verified.PurchaseAt,
		SubscriptionExpiresAt: verified.ExpiresAt,
	})
	if err != nil {
		return nil, err
	}
	return applePurchaseResponse(paid, verified), nil
}

// resolveAppleProduct maps an Apple product identifier to the configured VIP
// or points product under the authenticated application package.
func (s *Service) resolveAppleProduct(ctx context.Context, shopType int, sukCode, packageCode string) (shopID uint64, err error) {
	if shopType == 1 {
		vip, vipErr := s.vipProducts.GetAppleProduct(ctx, sukCode, packageCode)
		if vipErr != nil && !errors.Is(vipErr, gorm.ErrRecordNotFound) {
			return 0, vipErr
		}
		return vip.ID, nil
	}
	if shopType == 2 {
		points, pointsErr := s.pointProducts.GetAppleProduct(ctx, sukCode, packageCode)
		if pointsErr != nil && !errors.Is(pointsErr, gorm.ErrRecordNotFound) {
			return 0, pointsErr
		}
		return points.ID, nil
	}
	return 0, ErrAppleProductNotFound
}

// verifyApplePurchase verifies signed StoreKit evidence and checks every
// client-supplied identifier and timestamp against the signed transaction.
// Unsigned JSON is accepted only for explicitly enabled Sandbox development.
func verifyApplePurchase(req ApplePurchaseRequest, expectedBundle string, allowUnsignedSandbox bool) (*verifiedAppleTransaction, error) {
	req.BundleID = strings.TrimSpace(req.BundleID)
	req.TransactionID = strings.TrimSpace(req.TransactionID)
	req.OriginalTransactionID = strings.TrimSpace(req.OriginalTransactionID)
	req.ProductID = strings.TrimSpace(req.ProductID)
	evidence := strings.TrimSpace(req.SignedTransactionInfo)
	if expectedBundle == "" || req.BundleID == "" || req.TransactionID == "" || req.ProductID == "" || evidence == "" || req.PurchaseDate.IsZero() {
		return nil, ErrAppleEvidenceInvalid
	}

	var signed appleSignedTransaction
	mode := "jws"
	if strings.Count(evidence, ".") == 2 && !strings.HasPrefix(evidence, "{") {
		if err := verifyAppleJWS(evidence, &signed); err != nil {
			return nil, err
		}
	} else {
		mode = "sandbox_json"
		if err := json.Unmarshal([]byte(evidence), &signed); err != nil {
			return nil, fmt.Errorf("%w: signedTransactionInfo is neither JWS nor JSON", ErrAppleEvidenceInvalid)
		}
		if !allowUnsignedSandbox || !strings.EqualFold(signed.Environment, "Sandbox") {
			return nil, ErrAppleUnsignedProduction
		}
	}

	if req.BundleID != expectedBundle || signed.BundleID != expectedBundle {
		return nil, ErrAppleBundleMismatch
	}
	if signed.TransactionID != req.TransactionID || signed.OriginalTransactionID != req.OriginalTransactionID || signed.ProductID != req.ProductID {
		return nil, ErrAppleEvidenceInvalid
	}
	if signed.Quantity <= 0 || signed.Price < 0 || strings.TrimSpace(signed.Currency) == "" || signed.PurchaseDate <= 0 {
		return nil, ErrAppleEvidenceInvalid
	}
	purchaseAt := time.UnixMilli(signed.PurchaseDate)
	if durationAbs(purchaseAt.Sub(req.PurchaseDate)) > 2*time.Second || purchaseAt.After(time.Now().Add(5*time.Minute)) {
		return nil, ErrAppleEvidenceInvalid
	}
	result := &verifiedAppleTransaction{
		appleSignedTransaction: signed,
		PurchaseAt:             purchaseAt, PaidAmount: math.Round((float64(signed.Price)/1000)*100) / 100,
		EvidenceMode: mode,
	}
	result.Currency = strings.ToUpper(strings.TrimSpace(result.Currency))
	if signed.ExpiresDate > 0 {
		expiresAt := time.UnixMilli(signed.ExpiresDate)
		result.ExpiresAt = &expiresAt
		if req.ExpirationDate != nil && durationAbs(expiresAt.Sub(*req.ExpirationDate)) > 2*time.Second {
			return nil, ErrAppleEvidenceInvalid
		}
	} else if req.ExpirationDate != nil {
		return nil, ErrAppleEvidenceInvalid
	}
	if signed.RevocationDate > 0 {
		revokedAt := time.UnixMilli(signed.RevocationDate)
		result.RevokedAt = &revokedAt
		if req.RevocationDate == nil || durationAbs(revokedAt.Sub(*req.RevocationDate)) > 2*time.Second {
			return nil, ErrAppleEvidenceInvalid
		}
	} else if req.RevocationDate != nil {
		return nil, ErrAppleEvidenceInvalid
	}
	return result, nil
}

// appleClientRequestID derives the stable idempotency key used to find an order
// again when Apple retries or reorders notifications.
func appleClientRequestID(transactionID string) string {
	digest := sha256.Sum256([]byte(strings.TrimSpace(transactionID)))
	return "apple:" + base64.RawURLEncoding.EncodeToString(digest[:])
}

// applePurchaseResponse builds the public response and calculates current
// activity from verified revocation and expiration fields.
func applePurchaseResponse(order *model.VideoOrder, transaction *verifiedAppleTransaction) *ApplePurchaseResponse {
	active := transaction.RevokedAt == nil
	if isSubscriptionType(transaction.Type) {
		active = active && transaction.ExpiresAt != nil && transaction.ExpiresAt.After(time.Now())
	}
	return &ApplePurchaseResponse{
		OrderNo: order.OrderNo, Status: order.Status, ProductType: order.ProductType,
		ProductID: order.ProductID, ProductCode: order.ProductCode,
		TransactionID: transaction.TransactionID, OriginalTransactionID: transaction.OriginalTransactionID,
		Currency: transaction.Currency, PaidAmount: transaction.PaidAmount,
		PurchaseDate: transaction.PurchaseAt, ExpirationDate: transaction.ExpiresAt,
		IsActive: active, Environment: transaction.Environment, EvidenceMode: transaction.EvidenceMode,
	}
}

// isSubscriptionType recognizes StoreKit subscription type descriptions while
// tolerating the casing used by Sandbox fixtures.
func isSubscriptionType(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return strings.Contains(value, "subscription") || strings.Contains(value, "auto-renewable")
}

// durationAbs returns the non-negative magnitude used by timestamp tolerance
// checks.
func durationAbs(value time.Duration) time.Duration {
	if value < 0 {
		return -value
	}
	return value
}
