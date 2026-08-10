package service

import (
	"reflect"
	"testing"

	"ai-video/internal/modelparameter"
)

func TestBuildAndViewModelParameterKeepsValueAliasPairs(t *testing.T) {
	isDisplay := uint32(1)
	req := &ModelParameterPayload{
		ParamKey: "mode", ValueType: "string", ParameterType: ParameterTypeOption,
		DefaultValue: "auto", AllowedValues: []interface{}{"auto", "disabled"},
		AllowedValueOptions: []modelparameter.ValueOption{{Value: "auto", Alias: "自动"}, {Value: "disabled", Alias: "单图"}}, Alias: "生成模式",
		DisplayType: "select", IsDisplay: &isDisplay,
		Constraints: map[string]interface{}{},
	}
	item, err := buildModelParameter(7, req)
	if err != nil {
		t.Fatalf("buildModelParameter() error = %v", err)
	}
	view, err := modelParameterView(item)
	if err != nil {
		t.Fatalf("modelParameterView() error = %v", err)
	}
	if !reflect.DeepEqual(view.AllowedValues, req.AllowedValues) {
		t.Fatalf("AllowedValues = %#v", view.AllowedValues)
	}
	if !reflect.DeepEqual(view.AllowedValueOptions, req.AllowedValueOptions) {
		t.Fatalf("AllowedValueOptions = %#v", view.AllowedValueOptions)
	}
	if view.IsDisplay != 1 {
		t.Fatalf("IsDisplay = %d", view.IsDisplay)
	}
	if len(view.Constraints) != 0 {
		t.Fatalf("Constraints leaked alias metadata: %#v", view.Constraints)
	}
}

func TestSetDefaultIsDisplayPreservesCompatibility(t *testing.T) {
	req := &ModelParameterPayload{}
	setDefaultIsDisplay(req, 1)
	if req.IsDisplay == nil || *req.IsDisplay != 1 {
		t.Fatalf("IsDisplay = %#v", req.IsDisplay)
	}
	hidden := uint32(0)
	req.IsDisplay = &hidden
	setDefaultIsDisplay(req, 1)
	if *req.IsDisplay != 0 {
		t.Fatalf("explicit IsDisplay changed to %d", *req.IsDisplay)
	}
}

func TestNormalizeAllowedValueOptionsKeepsValueAndAliasTogether(t *testing.T) {
	options := []modelparameter.ValueOption{{Value: "auto", Alias: " 自动 "}, {Value: "disabled", Alias: "单图"}}
	if err := normalizeAllowedValueOptions(options); err != nil {
		t.Fatalf("normalizeAllowedValueOptions() error = %v", err)
	}
	want := []modelparameter.ValueOption{{Value: "auto", Alias: "自动"}, {Value: "disabled", Alias: "单图"}}
	if !reflect.DeepEqual(options, want) {
		t.Fatalf("options = %#v", options)
	}
	if err := normalizeAllowedValueOptions([]modelparameter.ValueOption{{Value: "auto", Alias: " "}}); err == nil {
		t.Fatal("expected blank alias to fail")
	}
}

func TestNormalizeSelectionOptionsAcceptsLegacyAllowedValues(t *testing.T) {
	req := &ModelParameterPayload{AllowedValues: []interface{}{"auto", "disabled"}}
	if err := normalizeSelectionOptions(req); err != nil {
		t.Fatalf("normalizeSelectionOptions() error = %v", err)
	}
	want := []modelparameter.ValueOption{{Value: "auto", Alias: "auto"}, {Value: "disabled", Alias: "disabled"}}
	if !reflect.DeepEqual(req.AllowedValueOptions, want) {
		t.Fatalf("AllowedValueOptions = %#v", req.AllowedValueOptions)
	}
}

func TestNormalizeSelectionOptionsRejectsMismatchedCompatibilityFields(t *testing.T) {
	req := &ModelParameterPayload{
		AllowedValues:       []interface{}{"disabled"},
		AllowedValueOptions: []modelparameter.ValueOption{{Value: "auto", Alias: "自动"}},
	}
	if err := normalizeSelectionOptions(req); err == nil {
		t.Fatal("expected mismatched compatibility fields to fail")
	}
}
