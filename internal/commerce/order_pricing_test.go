package commerce

import (
	"testing"

	"ai-video/internal/gen/model"
)

func TestSubscriptionOrderTerms(t *testing.T) {
	product := &model.VideoVipSubscription{
		FirstSubscriptionPrice: 3.99, FirstSubscriptionRevenue: 2.80, FirstBonusPoints: 120,
		SubscriptionPrice: 9.99, SubscriptionRevenue: 7.00, SubscriptionPoints: 60,
	}
	tests := []struct {
		name                string
		hasPaidSubscription bool
		wantPrice           float64
		wantRevenue         float64
		wantBonus           uint64
	}{
		{
			name:      "first subscription uses introductory terms",
			wantPrice: 3.99, wantRevenue: 2.80, wantBonus: 120,
		},
		{
			name: "existing subscription uses regular terms", hasPaidSubscription: true,
			wantPrice: 9.99, wantRevenue: 7.00, wantBonus: 60,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			price, revenue, bonus := subscriptionOrderTerms(product, tt.hasPaidSubscription)
			if price != tt.wantPrice || revenue != tt.wantRevenue || bonus != tt.wantBonus {
				t.Fatalf("unexpected terms: price=%v revenue=%v bonus=%d", price, revenue, bonus)
			}
		})
	}
}
