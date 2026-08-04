package service

import (
	"testing"
	"time"
)

func TestCurrentMonthRangeUsesConfiguredLocalTimezone(t *testing.T) {
	original := time.Local
	loc := time.FixedZone("CST", 8*60*60)
	time.Local = loc
	t.Cleanup(func() { time.Local = original })

	start, end := currentMonthRange(time.Date(2026, 7, 31, 17, 30, 0, 0, time.UTC))
	wantStart := time.Date(2026, 8, 1, 0, 0, 0, 0, loc)
	wantEnd := time.Date(2026, 9, 1, 0, 0, 0, 0, loc)
	if !start.Equal(wantStart) || !end.Equal(wantEnd) {
		t.Fatalf("currentMonthRange() = %s - %s, want %s - %s", start, end, wantStart, wantEnd)
	}
}
