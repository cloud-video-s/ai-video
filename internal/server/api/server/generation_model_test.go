package service

import (
	"reflect"
	"testing"

	"ai-video/internal/gen/model"
)

func TestGenerationModelParameterViewIncludesParameterType(t *testing.T) {
	tests := []struct {
		name          string
		item          model.VideoModelParameter
		wantDefault   interface{}
		wantAllowed   []interface{}
		wantParamType uint32
	}{
		{
			name: "option parameter",
			item: model.VideoModelParameter{
				ParamKey: "duration", DefaultValue: "15",
				AllowedValues: "[3,4,5,15]", ParameterType: 1,
			},
			wantDefault: float64(15), wantAllowed: []interface{}{float64(3), float64(4), float64(5), float64(15)},
			wantParamType: 1,
		},
		{
			name: "request parameter",
			item: model.VideoModelParameter{
				ParamKey: "prompt", DefaultValue: "", AllowedValues: "", ParameterType: 2,
			},
			wantDefault: nil, wantAllowed: []interface{}{}, wantParamType: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := generationModelParameterView(&tt.item)
			if err != nil {
				t.Fatal(err)
			}
			if got.DefaultValue != tt.wantDefault || !reflect.DeepEqual(got.AllowedValues, tt.wantAllowed) ||
				got.ParameterType != tt.wantParamType {
				t.Fatalf("generationModelParameterView() = %#v", got)
			}
			if got.AllowedValues == nil {
				t.Fatal("allowed_values must be an empty array instead of null")
			}
		})
	}
}
