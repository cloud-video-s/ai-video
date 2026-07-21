package service

import (
	"testing"

	"ai-video/internal/domain"
	"ai-video/internal/gen/model"
)

func TestAttributionEventStateUsesCurrentUserFlags(t *testing.T) {
	item := &model.VideoUserAttribution{
		ActivationCallbackCount: 2,
		ActivationDeductCount:   1,
		User: model.VideoUser{
			Activated: 1, KeyBehaviorMet: 0, PaymentMet: true,
		},
	}
	callback, deduct, reached, err := attributionEventState(item, domain.AttributionEventActivation)
	if err != nil || callback != 2 || deduct != 1 || !reached {
		t.Fatalf("activation state = %d, %d, %v, %v", callback, deduct, reached, err)
	}
	_, _, reached, err = attributionEventState(item, domain.AttributionEventKeyBehavior)
	if err != nil || reached {
		t.Fatalf("key behavior reached = %v, err = %v", reached, err)
	}
}

func TestAttributionEventColumnIsWhitelisted(t *testing.T) {
	column, err := attributionEventColumn(domain.AttributionEventPayment, domain.AttributionActionDeduct)
	if err != nil || column != "payment_deduct_count" {
		t.Fatalf("column = %q, err = %v", column, err)
	}
	if _, err := attributionEventColumn("unknown", domain.AttributionActionCallback); err == nil {
		t.Fatal("unknown event must fail")
	}
}
