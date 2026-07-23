package commerce

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"
	"time"

	"ai-video/internal/config"
	"ai-video/internal/domain"
	"ai-video/internal/gen/model"

	"gorm.io/gorm"
)

var (
	ErrAppleEvidenceInvalid    = errors.New("invalid Apple transaction evidence")
	ErrAppleSignatureInvalid   = errors.New("Apple transaction signature verification failed")
	ErrAppleUnsignedProduction = errors.New("unsigned Apple transaction is not allowed in production")
	ErrAppleBundleMismatch     = errors.New("Apple transaction bundle does not match request package")
	ErrAppleProductNotFound    = errors.New("Apple product is not configured for this package")
	ErrAppleProductAmbiguous   = errors.New("Apple product is configured as more than one product type")
	ErrApplePurchaseInactive   = errors.New("Apple subscription is inactive or expired")
	ErrApplePurchaseRevoked    = errors.New("Apple transaction has been revoked")
)

// ApplePurchaseRequest mirrors the StoreKit result returned by the app. The
// signedTransactionInfo field may be a compact JWS in production. A decoded
// JSON value is accepted only for Sandbox while the server is not in release
// mode, matching the development payload supplied by the client.
type ApplePurchaseRequest struct {
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

type appleSignedTransaction struct {
	TransactionID         string `json:"transactionId"`
	OriginalTransactionID string `json:"originalTransactionId"`
	BundleID              string `json:"bundleId"`
	ProductID             string `json:"productId"`
	PurchaseDate          int64  `json:"purchaseDate"`
	OriginalPurchaseDate  int64  `json:"originalPurchaseDate"`
	ExpiresDate           int64  `json:"expiresDate"`
	RevocationDate        int64  `json:"revocationDate"`
	Quantity              int64  `json:"quantity"`
	Type                  string `json:"type"`
	SignedDate            int64  `json:"signedDate"`
	Environment           string `json:"environment"`
	TransactionReason     string `json:"transactionReason"`
	Price                 int64  `json:"price"`
	Currency              string `json:"currency"`
}

type verifiedAppleTransaction struct {
	appleSignedTransaction
	PurchaseAt   time.Time
	ExpiresAt    *time.Time
	RevokedAt    *time.Time
	PaidAmount   float64
	EvidenceMode string
}

type ApplePurchaseResponse struct {
	OrderNo               string     `json:"order_no"`
	Status                string     `json:"status"`
	ProductType           string     `json:"product_type"`
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
		if !req.IsActive || verified.ExpiresAt == nil || !verified.ExpiresAt.After(time.Now()) {
			return nil, ErrApplePurchaseInactive
		}
	}

	productType, productID, err := s.resolveAppleProduct(ctx, verified.ProductID, expectedBundle)
	if err != nil {
		return nil, err
	}
	if productType == domain.OrderProductVIPSubscription && !isSubscriptionType(verified.Type) {
		return nil, ErrPaymentMismatch
	}
	if productType == domain.OrderProductPointsPackage && isSubscriptionType(verified.Type) {
		return nil, ErrPaymentMismatch
	}

	order, err := s.CreateOrder(ctx, CreateOrderRequest{
		UserID: userID, ProductType: productType, ProductID: productID,
		PaymentMethod:   domain.PaymentMethodAppleIAP,
		ClientRequestID: appleClientRequestID(verified.TransactionID),
		Renewal: strings.EqualFold(verified.TransactionReason, "RENEWAL") ||
			verified.TransactionID != verified.OriginalTransactionID,
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

func (s *Service) resolveAppleProduct(ctx context.Context, productCode, packageCode string) (string, uint64, error) {
	vip, vipErr := s.vipProducts.GetAppleProduct(ctx, productCode, packageCode)
	points, pointsErr := s.pointProducts.GetAppleProduct(ctx, productCode, packageCode)
	vipFound := vipErr == nil
	pointsFound := pointsErr == nil
	if vipErr != nil && !errors.Is(vipErr, gorm.ErrRecordNotFound) {
		return "", 0, vipErr
	}
	if pointsErr != nil && !errors.Is(pointsErr, gorm.ErrRecordNotFound) {
		return "", 0, pointsErr
	}
	if vipFound && pointsFound {
		return "", 0, ErrAppleProductAmbiguous
	}
	if vipFound {
		return domain.OrderProductVIPSubscription, vip.ID, nil
	}
	if pointsFound {
		return domain.OrderProductPointsPackage, points.ID, nil
	}
	return "", 0, ErrAppleProductNotFound
}

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

func verifyAppleJWS(compact string, target *appleSignedTransaction) error {
	parts := strings.Split(compact, ".")
	if len(parts) != 3 {
		return ErrAppleSignatureInvalid
	}
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return ErrAppleSignatureInvalid
	}
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ErrAppleSignatureInvalid
	}
	var header struct {
		Alg string   `json:"alg"`
		X5C []string `json:"x5c"`
	}
	if json.Unmarshal(headerJSON, &header) != nil || header.Alg != "ES256" || len(header.X5C) == 0 {
		return ErrAppleSignatureInvalid
	}
	if err := json.Unmarshal(payloadJSON, target); err != nil {
		return ErrAppleSignatureInvalid
	}
	certificates := make([]*x509.Certificate, 0, len(header.X5C))
	for _, encoded := range header.X5C {
		der, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return ErrAppleSignatureInvalid
		}
		certificate, err := x509.ParseCertificate(der)
		if err != nil {
			return ErrAppleSignatureInvalid
		}
		certificates = append(certificates, certificate)
	}
	intermediates := x509.NewCertPool()
	for _, certificate := range certificates[1:] {
		intermediates.AddCert(certificate)
	}
	verifyTime := time.Now()
	if target.SignedDate > 0 {
		verifyTime = time.UnixMilli(target.SignedDate)
	}
	if _, err := certificates[0].Verify(x509.VerifyOptions{
		Intermediates: intermediates, CurrentTime: verifyTime,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	}); err != nil {
		return fmt.Errorf("%w: %v", ErrAppleSignatureInvalid, err)
	}
	publicKey, ok := certificates[0].PublicKey.(*ecdsa.PublicKey)
	if !ok {
		return ErrAppleSignatureInvalid
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || len(signature) != 64 {
		return ErrAppleSignatureInvalid
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	r := new(big.Int).SetBytes(signature[:32])
	s := new(big.Int).SetBytes(signature[32:])
	if !ecdsa.Verify(publicKey, digest[:], r, s) {
		return ErrAppleSignatureInvalid
	}
	return nil
}

func appleClientRequestID(transactionID string) string {
	digest := sha256.Sum256([]byte(strings.TrimSpace(transactionID)))
	return "apple:" + base64.RawURLEncoding.EncodeToString(digest[:])
}

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

func isSubscriptionType(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return strings.Contains(value, "subscription") || strings.Contains(value, "auto-renewable")
}

func durationAbs(value time.Duration) time.Duration {
	if value < 0 {
		return -value
	}
	return value
}
