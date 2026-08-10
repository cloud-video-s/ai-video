package service

import (
	"context"
	"net/http/httptest"
	"reflect"
	"testing"

	"ai-video/internal/gen/model"

	"github.com/gin-gonic/gin/binding"
)

func TestModelIconRoundTrip(t *testing.T) {
	const icon = "/uploads/images/model-icon.png"
	payload := &ModelPayload{Icon: icon}
	item := &model.VideoModel{}

	applyModelPayload(item, payload)
	if item.Icon != icon {
		t.Fatalf("applyModelPayload() icon = %q, want %q", item.Icon, icon)
	}

	view := modelView(item, false)
	if view.Icon != icon {
		t.Fatalf("modelView() icon = %q, want %q", view.Icon, icon)
	}
}

func TestModelFeaturesRoundTrip(t *testing.T) {
	const modelFeatures = uint32(3)
	payload := &ModelPayload{ModelFeatures: modelFeatures}
	item := &model.VideoModel{}

	applyModelPayload(item, payload)
	if item.ModelFeatures != modelFeatures {
		t.Fatalf("applyModelPayload() model_features = %d, want %d", item.ModelFeatures, modelFeatures)
	}

	view := modelView(item, false)
	if view.ModelFeatures != modelFeatures {
		t.Fatalf("modelView() model_features = %d, want %d", view.ModelFeatures, modelFeatures)
	}
}

func TestValidateModelFeaturesRejectsUnsupportedValue(t *testing.T) {
	req := &ModelPayload{
		Name: "Model", Code: "model", ModelType: 1, ModelFeatures: 5, Version: "v1",
	}

	_, err := NewModelService().validatePayload(context.Background(), req, 0, false)
	if err == nil || err.Error() != "模型类型必须是 1=通用、2=模板、3=生成模型或 4=工具" {
		t.Fatalf("validatePayload() error = %v, want unsupported model_features error", err)
	}
}

func TestListModelRequestBindsModelFeaturesCSV(t *testing.T) {
	httpRequest := httptest.NewRequest("GET", "/admin/models?model_features=1%2C3", nil)
	var request ListModelRequest

	if err := binding.Query.Bind(httpRequest, &request); err != nil {
		t.Fatalf("bind model_features: %v", err)
	}
	if want := []uint32{1, 3}; !reflect.DeepEqual(request.ModelFeatures, want) {
		t.Fatalf("model_features = %v, want %v", request.ModelFeatures, want)
	}
}
