package scheduledtask

import (
	"testing"
	"time"
)

func TestSubscriptionExpirationTaskIDUsesAbsoluteTime(t *testing.T) {
	utc := time.Date(2026, 8, 14, 1, 30, 0, 0, time.UTC)
	shanghai := time.FixedZone("Asia/Shanghai", 8*60*60)
	local := time.Date(2026, 8, 14, 9, 30, 0, 0, shanghai)

	if got, want := subscriptionExpirationTaskID(local), subscriptionExpirationTaskID(utc); got != want {
		t.Fatalf("equivalent times produced different task IDs: %q != %q", got, want)
	}
}
