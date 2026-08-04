package commerce

import (
	"crypto/x509"
	"fmt"
	"strings"
)

// App Store Server Notifications V2 notification types.
// https://developer.apple.com/documentation/appstoreservernotifications/notificationtype
const (
	AppleNotificationConsumptionRequest     = "CONSUMPTION_REQUEST"       // 消费请求
	AppleNotificationDidChangeRenewalPref   = "DID_CHANGE_RENEWAL_PREF"   // 续订偏好已更改
	AppleNotificationDidChangeRenewalStatus = "DID_CHANGE_RENEWAL_STATUS" // 续订状态已更改
	AppleNotificationDidFailToRenew         = "DID_FAIL_TO_RENEW"         // 续订失败（扣费失败等）
	AppleNotificationDidRenew               = "DID_RENEW"                 // 已续订成功
	AppleNotificationExpired                = "EXPIRED"                   // 订阅已过期
	AppleNotificationGracePeriodExpired     = "GRACE_PERIOD_EXPIRED"      // 宽限期已过
	AppleNotificationOfferRedeemed          = "OFFER_REDEEMED"            // 优惠已兑换
	AppleNotificationOneTimeCharge          = "ONE_TIME_CHARGE"           // 一次性收费
	AppleNotificationPriceIncrease          = "PRICE_INCREASE"            // 价格上涨（需用户同意）
	AppleNotificationRefund                 = "REFUND"                    // 退款
	AppleNotificationRefundDeclined         = "REFUND_DECLINED"           // 退款被拒
	AppleNotificationRefundReversed         = "REFUND_REVERSED"           // 退款已撤销
	AppleNotificationRenewalExtended        = "RENEWAL_EXTENDED"          // 续订期已延长
	AppleNotificationRenewalExtension       = "RENEWAL_EXTENSION"         // 续订延期通知
	AppleNotificationRevoke                 = "REVOKE"                    // 撤销购买权限
	AppleNotificationSubscribed             = "SUBSCRIBED"                // 已订阅（首次或恢复）
	AppleNotificationTest                   = "TEST"                      // 沙盒测试通知
	AppleNotificationExternalPurchaseToken  = "EXTERNAL_PURCHASE_TOKEN"   // 外部购买令牌
	AppleNotificationRescindConsent         = "RESCIND_CONSENT"           // 撤销同意（如数据共享）
	AppleNotificationMetadataUpdate         = "METADATA_UPDATE"           // 元数据更新
	AppleNotificationMigration              = "MIGRATION"                 // 订阅迁移
	AppleNotificationPriceChange            = "PRICE_CHANGE"              // 价格变更（双向）
)

// App Store Server Notifications V2 subtypes.
// https://developer.apple.com/documentation/appstoreservernotifications/subtype
const (
	AppleSubtypeInitialBuy        = "INITIAL_BUY"          // 首次购买[reference:0][reference:1]
	AppleSubtypeResubscribe       = "RESUBSCRIBE"          // 重新订阅[reference:2]
	AppleSubtypeDowngrade         = "DOWNGRADE"            // 降级（下次续费生效）[reference:3][reference:4]
	AppleSubtypeUpgrade           = "UPGRADE"              // 升级（立即生效）[reference:5]
	AppleSubtypeAutoRenewDisabled = "AUTO_RENEW_DISABLED"  // 自动续费已关闭[reference:6][reference:7]
	AppleSubtypeAutoRenewEnabled  = "AUTO_RENEW_ENABLED"   // 自动续费已开启[reference:8]
	AppleSubtypeBillingRecovery   = "BILLING_RECOVERY"     // 计费恢复（过期订阅续费成功）[reference:9]
	AppleSubtypeBillingRetry      = "BILLING_RETRY"        // 计费重试失败（订阅过期）[reference:10]
	AppleSubtypePriceIncrease     = "PRICE_INCREASE"       // 价格上调[reference:11]
	AppleSubtypeGracePeriod       = "GRACE_PERIOD"         // 宽限期[reference:12]
	AppleSubtypePending           = "PENDING"              // 待处理[reference:13]
	AppleSubtypeAccepted          = "ACCEPTED"             // 用户已同意（价格上调）[reference:14]
	AppleSubtypeSummary           = "SUMMARY"              // 续期延期汇总[reference:15]
	AppleSubtypeFailure           = "FAILURE"              // 续期延期失败[reference:16]
	AppleSubtypeVoluntary         = "VOLUNTARY"            // 用户主动关闭续费后过期[reference:17][reference:18]
	AppleSubtypeProductNotForSale = "PRODUCT_NOT_FOR_SALE" // 产品已下架不可售[reference:19]
	AppleSubtypeUnreported        = "UNREPORTED"           // 外部购买令牌未上报[reference:20]
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
	NotificationType    string `json:"notification_type"`                 // 通知类型（对应 NotificationType 常量）
	Subtype             string `json:"subtype,omitempty"`                 // 子类型（对应 Subtype 常量，如升级、降级等）
	NotificationUUID    string `json:"notification_uuid,omitempty"`       // 通知唯一标识 UUID
	BundleID            string `json:"bundle_id,omitempty"`               // App 的 Bundle ID
	Environment         string `json:"environment,omitempty"`             // 环境（Sandbox / Production）
	OriginalTransaction string `json:"original_transaction_id,omitempty"` // 原始交易 ID（订阅初始交易）
	TransactionID       string `json:"transaction_id,omitempty"`          // 当前交易 ID
	ProductID           string `json:"product_id,omitempty"`              // 产品 ID
	Version             string `json:"version,omitempty"`                 // 通知版本号
	SignedDate          int64  `json:"signed_date,omitempty"`             // JWS 签名时间戳（毫秒）
	AppAppleID          int64  `json:"app_apple_id,omitempty"`            // 应用的 Apple ID（开发者账户中的 App ID）
	SubscriptionStatus  int32  `json:"subscription_status,omitempty"`     // 订阅状态（内部状态值，如活跃/过期等）
	Processed           bool   `json:"processed"`                         // 是否已被本地系统处理（用于幂等）
	AffectedUserID      uint64 `json:"affected_user_id,omitempty"`        // 受影响的内部用户 ID（业务层使用）
	AffectedOrderNo     string `json:"affected_order_no,omitempty"`       // 受影响的内部订单号（业务层使用）
	Action              string `json:"action,omitempty"`                  // 本地执行的操作（如发放权益、记录日志等）
	Message             string `json:"message,omitempty"`                 // 操作结果或错误信息（本地记录）
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

// decodeAppleNotificationPayload preserves the previous internal entry point
// while delegating all work to the V2 decoder.
func decodeAppleNotificationPayload(signedPayload string, roots *x509.CertPool) (*DecodedAppleNotification, error) {
	return decodeAppleNotificationV2Payload(signedPayload, roots)
}

// decodeAppleNotificationV2Payload authenticates the outer notification and
// every nested signed transaction before exposing normalized business fields.
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

// normalizeAppleNotificationV2Payload selects the identity fields from the V2
// data, summary, externalPurchaseToken, or appData payload shape.
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

// decodeAppleNotificationV2Transaction verifies signedTransactionInfo, checks
// its application identity, and copies transaction facts into result.
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

// verifyAppleNotificationV2AppTransaction verifies appData's nested app
// transaction and binds it to the outer Bundle ID and environment.
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
