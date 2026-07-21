// Package domain contains business vocabulary that is independent from the
// database-generated model and query packages.
package domain

const (
	SuperAdminRoleCode = "admin"

	AppUserLoginGuest  uint32 = 1
	AppUserLoginGoogle uint32 = 2
	AppUserLoginAppID  uint32 = 3
	AppUserTypeFree    uint32 = 1
	AppUserTypePaid    uint32 = 2

	AppUserSubscriptionNotSubscribed uint32 = 1
	AppUserSubscriptionSubscribed    uint32 = 2
	AppUserSubscriptionCancelled     uint32 = 3

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

	VideoTemplateKindAction   = "action"
	VideoTemplateKindFaceSwap = "face_swap"

	PointsDirectionIncome  int32 = 1
	PointsDirectionExpense int32 = 2

	VIPPlanTypeNormal          = "normal"
	VIPPlanTypeTrial           = "trial"
	VIPPlanTypePaywall         = "paywall"
	VIPDisplayModeHidden int32 = 0
	VIPDisplayModeNormal int32 = 1
)
