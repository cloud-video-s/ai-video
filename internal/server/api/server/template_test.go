package service

import (
	"reflect"
	"testing"

	"ai-video/internal/gen/model"
)

func TestBuildClientTemplateGroupsSortsAndDropsEmptyCategories(t *testing.T) {
	types := []model.VideoTemplateType{
		{ID: 1, CategoryName: "low", Sort: 1},
		{ID: 3, CategoryName: "empty", Sort: 99},
		{ID: 2, CategoryName: "high", Sort: 10},
	}
	rows := []model.VideoTemplate{
		{ID: 11, TemplateTypeID: 1, Name: "more-views", Sort: 5, UsageCount: 1, ViewCount: 9},
		{ID: 21, TemplateTypeID: 2, Name: "primary-sort", Sort: 9},
		{ID: 12, TemplateTypeID: 1, Name: "fewer-views", Sort: 5, UsageCount: 1, ViewCount: 2},
		{ID: 13, TemplateTypeID: 1, Name: "usage", Sort: 5, UsageCount: 2},
	}

	got := buildClientTemplateGroups(types, rows)
	if len(got) != 2 {
		t.Fatalf("category count = %d, want 2", len(got))
	}
	if got[0].ID != 2 || got[1].ID != 1 {
		t.Fatalf("category order = [%d %d], want [2 1]", got[0].ID, got[1].ID)
	}
	lowTemplates := got[1].Templates
	if len(lowTemplates) != 3 {
		t.Fatalf("low category template count = %d, want 3", len(lowTemplates))
	}
	if lowTemplates[0].Name != "usage" || lowTemplates[1].Name != "more-views" || lowTemplates[2].Name != "fewer-views" {
		t.Fatalf("template order = [%s %s %s]", lowTemplates[0].Name, lowTemplates[1].Name, lowTemplates[2].Name)
	}
}

func TestClientTemplateModelParameterView(t *testing.T) {
	item := model.VideoTemplateModelParameter{
		TemplateID: 7, ParamKey: "duration", ParamType: "integer", ParameterType: 1,
		DefaultValue: "5", AllowedValues: "[3,5]", Constraints: "{}", SortOrder: 2,
	}
	got, err := clientTemplateModelParameterView(&item)
	if err != nil {
		t.Fatal(err)
	}
	if got.ParamKey != "duration" || got.ValueType != "integer" || got.DefaultValue != float64(5) {
		t.Fatalf("clientTemplateModelParameterView() = %#v", got)
	}
	wantAllowed := []interface{}{float64(3), float64(5)}
	if !reflect.DeepEqual(got.AllowedValues, wantAllowed) {
		t.Fatalf("allowed_values = %#v, want %#v", got.AllowedValues, wantAllowed)
	}

	template := model.VideoTemplate{ID: 7, TemplateTypeID: 1, ModelID: 9, Name: "demo"}
	view := mapClientTemplate(&template, clientTemplateModelConfiguration{
		ModelID: 9, ModelCode: "kling-v3", ModelName: "Kling V3",
		Parameters: []ClientTemplateModelParameter{got},
	})
	if view.ModelID != 9 || view.ModelCode != "kling-v3" || len(view.ModelParameters) != 1 {
		t.Fatalf("mapClientTemplate() model configuration = %#v", view)
	}
}
