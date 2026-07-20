package service

import (
	"testing"
	"time"
)

func TestParsePointsLedgerDateRangeIncludesWholeEndDate(t *testing.T) {
	from, to, err := parsePointsLedgerDateRange("2026-07-01", "2026-07-17")
	if err != nil {
		t.Fatal(err)
	}
	if from == nil || from.Format("2006-01-02 15:04:05") != "2026-07-01 00:00:00" {
		t.Fatalf("unexpected start: %v", from)
	}
	if to == nil || to.Format("2006-01-02 15:04:05") != "2026-07-18 00:00:00" {
		t.Fatalf("unexpected exclusive end: %v", to)
	}
}

func TestParsePointsLedgerDateRangeRejectsReverseRange(t *testing.T) {
	if _, _, err := parsePointsLedgerDateRange("2026-07-18", "2026-07-17"); err == nil {
		t.Fatal("expected reverse date range to be rejected")
	}
}

func TestParsePointsLedgerDateRangeUsesLocalTimezone(t *testing.T) {
	original := time.Local
	time.Local = time.FixedZone("CST", 8*60*60)
	t.Cleanup(func() { time.Local = original })

	from, _, err := parsePointsLedgerDateRange("2026-07-17", "")
	if err != nil {
		t.Fatal(err)
	}
	_, offset := from.Zone()
	if offset != 8*60*60 {
		t.Fatalf("timezone offset=%d, want %d", offset, 8*60*60)
	}
}
