package domain

import "strings"

// TrackingDataType is the client tracking-event name. These values intentionally
// preserve the spellings from the product specification because they are part
// of the API contract.
type TrackingDataType string

const (
	TrackingOBPaymentShow     TrackingDataType = "OB_Payment_show"
	TrackingOBPaymentBackShow TrackingDataType = "OB_Payment_back_show"
	TrackingHomeShow          TrackingDataType = "Home_Show"
	TrackingLaunchPaymentShow TrackingDataType = "Launc_Payment_Show"
	TrackingLaunchPaymentBack TrackingDataType = "Launc_Payment_back_Show"
	TrackingPaymentShow       TrackingDataType = "Payment_Show"
	TrackingPaymentCreate     TrackingDataType = "Payment_Create"
	TrackingPaymentSuccess    TrackingDataType = "Payment_Suc"
	TrackingCaseCreate        TrackingDataType = "Case_create"
)

// ParseTrackingDataType accepts only the nine event names in the current
// tracking specification. Matching is case-sensitive so reporting cannot split
// one metric across subtly different names.
func ParseTrackingDataType(value string) (TrackingDataType, bool) {
	dataType := TrackingDataType(strings.TrimSpace(value))
	switch dataType {
	case TrackingOBPaymentShow,
		TrackingOBPaymentBackShow,
		TrackingHomeShow,
		TrackingLaunchPaymentShow,
		TrackingLaunchPaymentBack,
		TrackingPaymentShow,
		TrackingPaymentCreate,
		TrackingPaymentSuccess,
		TrackingCaseCreate:
		return dataType, true
	default:
		return "", false
	}
}

// SupportsExtendedFields reports whether the event is one of the final three
// types for which the product specification defines extension identifiers.
func (dataType TrackingDataType) SupportsExtendedFields() bool {
	switch dataType {
	case TrackingPaymentCreate, TrackingPaymentSuccess, TrackingCaseCreate:
		return true
	default:
		return false
	}
}
