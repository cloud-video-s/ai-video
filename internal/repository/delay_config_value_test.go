package repository

import "testing"

func TestParseDelayConfigNumber(t *testing.T) {
	tests := []struct {
		value   string
		want    int64
		wantErr bool
	}{
		{value: "0", want: 0},
		{value: "5", want: 5},
		{value: "-1", want: -1},
		{value: "true", want: 1},
		{value: "false", want: 0},
		{value: "invalid", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			got, err := parseDelayConfigNumber(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseDelayConfigNumber(%q) unexpectedly succeeded", tt.value)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("parseDelayConfigNumber(%q) = %d, %v; want %d", tt.value, got, err, tt.want)
			}
		})
	}
}
