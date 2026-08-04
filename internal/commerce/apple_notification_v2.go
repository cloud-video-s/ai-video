package commerce

import (
	"crypto/x509"
	"fmt"
	"strings"
)

// App Store Server Notifications V2 notification types.
// https://developer.apple.com/documentation/appstoreservernotifications/notificationtype
const (
	AppleNotificationConsumptionRequest     = "CONSUMPTION_REQUEST"
	AppleNotificationDidChangeRenewalPref   = "DID_CHANGE_RENEWAL_PREF"
	AppleNotificationDidChangeRenewalStatus = "DID_CHANGE_RENEWAL_STATUS"
	AppleNotificationDidFailToRenew         = "DID_FAIL_TO_RENEW"
	AppleNotificationDidRenew               = "DID_RENEW"
	AppleNotificationExpired                = "EXPIRED"
	AppleNotificationGracePeriodExpired     = "GRACE_PERIOD_EXPIRED"
	AppleNotificationOfferRedeemed          = "OFFER_REDEEMED"
	AppleNotificationOneTimeCharge          = "ONE_TIME_CHARGE"
	AppleNotificationPriceIncrease          = "PRICE_INCREASE"
	AppleNotificationRefund                 = "REFUND"
	AppleNotificationRefundDeclined         = "REFUND_DECLINED"
	AppleNotificationRefundReversed         = "REFUND_REVERSED"
	AppleNotificationRenewalExtended        = "RENEWAL_EXTENDED"
	AppleNotificationRenewalExtension       = "RENEWAL_EXTENSION"
	AppleNotificationRevoke                 = "REVOKE"
	AppleNotificationSubscribed             = "SUBSCRIBED"
	AppleNotificationTest                   = "TEST"
	AppleNotificationExternalPurchaseToken  = "EXTERNAL_PURCHASE_TOKEN"
	AppleNotificationRescindConsent         = "RESCIND_CONSENT"
	AppleNotificationMetadataUpdate         = "METADATA_UPDATE"
	AppleNotificationMigration              = "MIGRATION"
	AppleNotificationPriceChange            = "PRICE_CHANGE"
)

// App Store Server Notifications V2 subtypes.
// https://developer.apple.com/documentation/appstoreservernotifications/subtype
const (
	AppleSubtypeInitialBuy        = "INITIAL_BUY"
	AppleSubtypeResubscribe       = "RESUBSCRIBE"
	AppleSubtypeDowngrade         = "DOWNGRADE"
	AppleSubtypeUpgrade           = "UPGRADE"
	AppleSubtypeAutoRenewDisabled = "AUTO_RENEW_DISABLED"
	AppleSubtypeAutoRenewEnabled  = "AUTO_RENEW_ENABLED"
	AppleSubtypeBillingRecovery   = "BILLING_RECOVERY"
	AppleSubtypeBillingRetry      = "BILLING_RETRY"
	AppleSubtypePriceIncrease     = "PRICE_INCREASE"
	AppleSubtypeGracePeriod       = "GRACE_PERIOD"
	AppleSubtypePending           = "PENDING"
	AppleSubtypeAccepted          = "ACCEPTED"
	AppleSubtypeSummary           = "SUMMARY"
	AppleSubtypeFailure           = "FAILURE"
	AppleSubtypeVoluntary         = "VOLUNTARY"
	AppleSubtypeProductNotForSale = "PRODUCT_NOT_FOR_SALE"
	AppleSubtypeUnreported        = "UNREPORTED"
)

// AppleNotificationV2Request is the HTTP body sent by App Store Server
// Notifications V2.
type AppleNotificationV2Request struct {
	SignedPayload string `json:"signedPayload" binding:"required"`
}

// AppleNotificationRequest is kept as a source-compatible alias.
type AppleNotificationRequest = AppleNotificationV2Request

// AppleNotificationV2Summary contains verified metadata and the local action
// taken for a V2 notification. It deliberately excludes the compact JWS.
type AppleNotificationV2Summary struct {
	NotificationType    string `json:"notification_type"`
	Subtype             string `json:"subtype,omitempty"`
	NotificationUUID    string `json:"notification_uuid,omitempty"`
	BundleID            string `json:"bundle_id,omitempty"`
	Environment         string `json:"environment,omitempty"`
	OriginalTransaction string `json:"original_transaction_id,omitempty"`
	TransactionID       string `json:"transaction_id,omitempty"`
	ProductID           string `json:"product_id,omitempty"`
	Version             string `json:"version,omitempty"`
	SignedDate          int64  `json:"signed_date,omitempty"`
	AppAppleID          int64  `json:"app_apple_id,omitempty"`
	SubscriptionStatus  int32  `json:"subscription_status,omitempty"`
	Processed           bool   `json:"processed"`
	AffectedUserID      uint64 `json:"affected_user_id,omitempty"`
	AffectedOrderNo     string `json:"affected_order_no,omitempty"`
	Action              string `json:"action,omitempty"`
	Message             string `json:"message,omitempty"`
}

// AppleNotificationSummary is kept as a source-compatible alias.
type AppleNotificationSummary = AppleNotificationV2Summary

type appleSignedRenewalInfo struct {
	Environment string `json:"environment"`
	SignedDate  int64  `json:"signedDate"`
}

type appleSignedAppTransaction struct {
	Environment string `json:"environment"`
	BundleID    string `json:"bundleId"`
	SignedDate  int64  `json:"signedDate"`
}

type appleNotificationV2Data struct {
	Environment        string `json:"environment"`
	AppAppleID         int64  `json:"appAppleId"`
	BundleID           string `json:"bundleId"`
	BundleVersion      string `json:"bundleVersion"`
	SignedTransaction  string `json:"signedTransactionInfo"`
	SignedRenewalInfo  string `json:"signedRenewalInfo"`
	SubscriptionStatus int32  `json:"status"`
	ConsumptionReason  string `json:"consumptionRequestReason"`
}

type appleNotificationV2SummaryPayload struct {
	Environment string `json:"environment"`
	AppAppleID  int64  `json:"appAppleId"`
	BundleID    string `json:"bundleId"`
	ProductID   string `json:"productId"`
}

type appleNotificationV2ExternalPurchaseToken struct {
	ExternalPurchaseID string `json:"externalPurchaseId"`
	TokenCreationDate  int64  `json:"tokenCreationDate"`
	AppAppleID         int64  `json:"appAppleId"`
	BundleID           string `json:"bundleId"`
}

type appleNotificationV2AppData struct {
	Environment              string `json:"environment"`
	AppAppleID               int64  `json:"appAppleId"`
	BundleID                 string `json:"bundleId"`
	SignedAppTransactionInfo string `json:"signedAppTransactionInfo"`
}

// appleNotificationV2Payload mirrors ResponseBodyV2DecodedPayload. Apple uses
// one of Data, Summary, ExternalPurchaseToken, or AppData depending on the
// notification type.
type appleNotificationV2Payload struct {
	NotificationType      string                                    `json:"notificationType"`
	Subtype               string                                    `json:"subtype"`
	NotificationUUID      string                                    `json:"notificationUUID"`
	Data                  *appleNotificationV2Data                  `json:"data"`
	Version               string                                    `json:"version"`
	SignedDate            int64                                     `json:"signedDate"`
	Summary               *appleNotificationV2SummaryPayload        `json:"summary"`
	ExternalPurchaseToken *appleNotificationV2ExternalPurchaseToken `json:"externalPurchaseToken"`
	AppData               *appleNotificationV2AppData               `json:"appData"`
}

// DecodedAppleNotificationV2 is a normalized view of the mutually exclusive
// V2 payload shapes and any verified nested transaction data.
type DecodedAppleNotificationV2 struct {
	NotificationType    string
	Subtype             string
	NotificationUUID    string
	BundleID            string
	Environment         string
	SignedRenewalInfo   string
	SignedTransaction   string
	OriginalTransaction string
	TransactionID       string
	ProductID           string
	PurchaseDate        int64
	ExpiresDate         int64
	RevocationDate      int64
	RevocationReason    string
	TransactionReason   string
	Version             string
	SignedDate          int64
	AppAppleID          int64
	SubscriptionStatus  int32
}

// DecodedAppleNotification is kept as a source-compatible alias.
type DecodedAppleNotification = DecodedAppleNotificationV2

// DecodeAppleNotificationV2Payload verifies the outer signedPayload and all
// nested JWS values before returning normalized business fields.
func DecodeAppleNotificationV2Payload(signedPayload string) (*DecodedAppleNotificationV2, error) {
	return decodeAppleNotificationV2Payload(signedPayload, defaultAppleRootCAs)
}

// DecodeAppleNotificationPayload is kept for callers using the previous name.
func DecodeAppleNotificationPayload(signedPayload string) (*DecodedAppleNotification, error) {
	return DecodeAppleNotificationV2Payload(signedPayload)
}

func decodeAppleNotificationPayload(signedPayload string, roots *x509.CertPool) (*DecodedAppleNotification, error) {
	return decodeAppleNotificationV2Payload(signedPayload, roots)
}

func decodeAppleNotificationV2Payload(signedPayload string, roots *x509.CertPool) (*DecodedAppleNotificationV2, error) {
	if strings.TrimSpace(signedPayload) == "" {
		return nil, ErrAppleEvidenceInvalid
	}

	var payload appleNotificationV2Payload
	if err := verifyAppleJWSWithRoots(signedPayload, &payload, roots); err != nil {
		return nil, err
	}
	payload.NotificationType = strings.TrimSpace(payload.NotificationType)
	payload.NotificationUUID = strings.TrimSpace(payload.NotificationUUID)
	if payload.NotificationType == "" {
		return nil, fmt.Errorf("%w: notificationType is required", ErrAppleEvidenceInvalid)
	}

	result, err := normalizeAppleNotificationV2Payload(&payload)
	if err != nil {
		return nil, err
	}
	if result.SignedTransaction != "" {
		if err := decodeAppleNotificationV2Transaction(result, roots); err != nil {
			return nil, err
		}
	}
	if result.SignedRenewalInfo != "" {
		var renewal appleSignedRenewalInfo
		if err := verifyAppleJWSWithRoots(result.SignedRenewalInfo, &renewal, roots); err != nil {
			return nil, err
		}
		if strings.TrimSpace(renewal.Environment) != result.Environment {
			return nil, ErrAppleEnvironmentMismatch
		}
	}
	if payload.AppData != nil && strings.TrimSpace(payload.AppData.SignedAppTransactionInfo) != "" {
		if err := verifyAppleNotificationV2AppTransaction(result, payload.AppData.SignedAppTransactionInfo, roots); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func normalizeAppleNotificationV2Payload(payload *appleNotificationV2Payload) (*DecodedAppleNotificationV2, error) {
	var bundleID, environment, productID, signedTransaction, signedRenewalInfo string
	var appAppleID int64
	var subscriptionStatus int32
	switch {
	case payload.Data != nil:
		bundleID = payload.Data.BundleID
		environment = payload.Data.Environment
		appAppleID = payload.Data.AppAppleID
		signedTransaction = payload.Data.SignedTransaction
		signedRenewalInfo = payload.Data.SignedRenewalInfo
		subscriptionStatus = payload.Data.SubscriptionStatus
	case payload.Summary != nil:
		bundleID = payload.Summary.BundleID
		environment = payload.Summary.Environment
		appAppleID = payload.Summary.AppAppleID
		productID = payload.Summary.ProductID
	case payload.ExternalPurchaseToken != nil:
		bundleID = payload.ExternalPurchaseToken.BundleID
		appAppleID = payload.ExternalPurchaseToken.AppAppleID
		if strings.HasPrefix(payload.ExternalPurchaseToken.ExternalPurchaseID, "SANDBOX") {
			environment = "Sandbox"
		} else {
			environment = "Production"
		}
	case payload.AppData != nil:
		bundleID = payload.AppData.BundleID
		environment = payload.AppData.Environment
		appAppleID = payload.AppData.AppAppleID
	}

	bundleID = strings.TrimSpace(bundleID)
	environment = strings.TrimSpace(environment)
	if bundleID == "" {
		return nil, fmt.Errorf("%w: notification bundleId is required", ErrAppleBundleMismatch)
	}
	if environment != "Sandbox" && environment != "Production" {
		return nil, ErrAppleEnvironmentMismatch
	}

	return &DecodedAppleNotificationV2{
		NotificationType:   payload.NotificationType,
		Subtype:            strings.TrimSpace(payload.Subtype),
		NotificationUUID:   payload.NotificationUUID,
		BundleID:           bundleID,
		Environment:        environment,
		SignedRenewalInfo:  strings.TrimSpace(signedRenewalInfo),
		SignedTransaction:  strings.TrimSpace(signedTransaction),
		ProductID:          strings.TrimSpace(productID),
		Version:            strings.TrimSpace(payload.Version),
		SignedDate:         payload.SignedDate,
		AppAppleID:         appAppleID,
		SubscriptionStatus: subscriptionStatus,
	}, nil
}

func decodeAppleNotificationV2Transaction(result *DecodedAppleNotificationV2, roots *x509.CertPool) error {
	var transaction appleSignedTransaction
	if err := verifyAppleJWSWithRoots(result.SignedTransaction, &transaction, roots); err != nil {
		return err
	}
	if strings.TrimSpace(transaction.BundleID) != result.BundleID {
		return ErrAppleBundleMismatch
	}
	if strings.TrimSpace(transaction.Environment) != result.Environment {
		return ErrAppleEnvironmentMismatch
	}
	result.OriginalTransaction = strings.TrimSpace(transaction.OriginalTransactionID)
	result.TransactionID = strings.TrimSpace(transaction.TransactionID)
	result.ProductID = strings.TrimSpace(transaction.ProductID)
	result.PurchaseDate = transaction.PurchaseDate
	result.ExpiresDate = transaction.ExpiresDate
	result.RevocationDate = transaction.RevocationDate
	result.RevocationReason = transaction.RevocationReason
	result.TransactionReason = transaction.TransactionReason
	return nil
}

func verifyAppleNotificationV2AppTransaction(result *DecodedAppleNotificationV2, compact string, roots *x509.CertPool) error {
	var transaction appleSignedAppTransaction
	if err := verifyAppleJWSWithRoots(strings.TrimSpace(compact), &transaction, roots); err != nil {
		return err
	}
	if strings.TrimSpace(transaction.BundleID) != result.BundleID {
		return ErrAppleBundleMismatch
	}
	if strings.TrimSpace(transaction.Environment) != result.Environment {
		return ErrAppleEnvironmentMismatch
	}
	return nil
}
