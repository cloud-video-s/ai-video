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
	AppUserStatusDisabled            int8  = 0
	AppUserStatusNormal              int8  = 1
	AppUserStatusFrozen              int8  = 2
	AppUserStatusBlacklisted         int8  = 3

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
	OrderPayTypeApple           = 1
	OrderPayTypeGoogle          = 2
	OrderTypeNewPurchase        = 1
	OrderTypeRenewal            = 2

	OrderStatusPending   = 1 //待处理
	OrderStatusPaying    = 2 //支付中
	OrderStatusPaid      = 3 //支付完成
	OrderStatusEnd       = 4 //订单完成
	OrderStatusCancelled = 5 //取消
	OrderStatusFailed    = 6 //支付失败
	OrderStatusRefunded  = 7 //已退款

	PaymentMethodAppleIAP   = 1
	PaymentMethodGooglePlay = 2

	PointsSourceSubscriptionGift = 1 // 订阅赠送
	PointsSourcePurchase         = 2 // 积分购买
	PointsSourceModelConsume     = 3 // 模型消费
	PointsSourceModelRefund      = 4 // 模型退款
	PointsSourceExpireDeduct     = 5 // 订阅过期扣除
	PointsSourceSystemReward     = 6 // 系统奖励
	PointsSourceAdminOp          = 7 // 管理员操作
	PointsSourceOther            = 8 // 其他

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
