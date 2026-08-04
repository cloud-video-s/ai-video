// Package domain contains business vocabulary that is independent from the
// database-generated model and query packages.
package domain

const (
	SuperAdminRoleCode = "admin"

	AppUserLoginGuest  int    = 1
	AppUserLoginGoogle uint32 = 2
	AppUserLoginAppID  uint32 = 3
	AppUserTypeFree    uint32 = 1
	AppUserTypePaid    uint32 = 2

	AppUserSubscriptionNotSubscribed uint8 = 1
	AppUserSubscriptionSubscribed    uint8 = 2
	AppUserSubscriptionCancelled     uint8 = 3
	AppUserSubscriptionExpired       uint8 = 4

	IdentityProviderGoogle = "google"
	IdentityProviderApple  = "apple"

	AttributionEventActivation   = "activation"
	AttributionEventKeyBehavior  = "key_behavior"
	AttributionEventPayment      = "payment"
	AttributionEventFirstPayment = "first_payment"
	AttributionEventRegistration = "registration"
	AttributionActionCallback    = "callback"
	AttributionActionDeduct      = "deduct"

	UploadUserUnknown int8 = 0
	UploadUserAdmin   int8 = 1
	UploadUserClient  int8 = 2

	BannerJumpTypeLink        uint8 = 1
	BannerJumpTypeTemplate    uint8 = 2
	BannerJumpTypeTextToImage uint8 = 3
	BannerJumpTypeTextToVideo uint8 = 4

	VideoTemplateKindImage int64 = 1
	VideoTemplateKindVideo int64 = 2

	PointsDirectionIncome  int32 = 1
	PointsDirectionExpense int32 = 2

	OrderProductVIPSubscription = 1
	OrderProductPointsPackage   = 2

	OrderStatusPending   = "pending"
	OrderStatusPaid      = "paid"
	OrderStatusCancelled = "cancelled"
	OrderStatusFailed    = "failed"
	OrderStatusRefunded  = "refunded"
	OrderStatusCompleted = "completed"

	PaymentMethodAppleIAP = "apple_iap"

	PointsSourceSubscriptionGift = iota + 1 // 1: 订阅赠送
	PointsSourcePurchase                    // 2: 积分购买（注意与原有 "purchase" 字符串常量重名，若需共存请改用其他前缀如 SourceType）
	PointsSourceModelConsume                // 3: 模型消费
	PointsSourceModelRefund                 // 4: 模型退款
	PointsSourceExpireDeduct                // 5: 订阅过期扣除
	PointsSourceSystemReward                // 6: 系统奖励
	PointsSourceAdminOp                     // 7: 管理员操作
	PointsSourceOther                       // 8: 其他

	VIPPlanTypeNormal          = "normal"
	VIPPlanTypeTrial           = "trial"
	VIPPlanTypePaywall         = "paywall"
	VIPDisplayModeHidden int32 = 0
	VIPDisplayModeNormal int32 = 1

	SystemTypeIos int = 1
	SystemTypeA   int = 2

	SubscriptionStatusUnsubscribed = 1
	SubscriptionStatusSubscribed   = 2
	SubscriptionStatusCanceled     = 3
	SubscriptionStatusExpired      = 4
)
