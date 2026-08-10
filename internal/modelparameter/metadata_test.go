package modelparameter

import (
	"reflect"
	"testing"
)

func TestValueOptionsRoundTrip(t *testing.T) {
	options, err := ValueOptions(`[{"value":"auto","alias":"自动"},{"value":"disabled","alias":"单图"}]`, []interface{}{"auto", "disabled"})
	if err != nil {
		t.Fatalf("ValueOptions() error = %v", err)
	}
	want := []ValueOption{{Value: "auto", Alias: "自动"}, {Value: "disabled", Alias: "单图"}}
	if !reflect.DeepEqual(options, want) {
		t.Fatalf("ValueOptions() = %#v", options)
	}
}

func TestValueOptionsConvertsLegacyAliasArray(t *testing.T) {
	options, err := ValueOptions(`["自动","二档","开启"]`, []interface{}{"auto", float64(2), true})
	if err != nil {
		t.Fatalf("ValueOptions() error = %v", err)
	}
	want := []ValueOption{{Value: "auto", Alias: "自动"}, {Value: float64(2), Alias: "二档"}, {Value: true, Alias: "开启"}}
	if !reflect.DeepEqual(options, want) {
		t.Fatalf("ValueOptions() = %#v, want %#v", options, want)
	}
}

func TestValueOptionsRejectsMismatchedValue(t *testing.T) {
	if _, err := ValueOptions(`[{"value":"disabled","alias":"单图"}]`, []interface{}{"auto"}); err == nil {
		t.Fatal("ValueOptions() expected an error for mismatched values")
	}
}
