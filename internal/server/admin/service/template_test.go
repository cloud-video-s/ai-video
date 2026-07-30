package service

import "testing"

func TestValidateTemplatePayloadAcceptsIntegerKinds(t *testing.T) {
	base := TemplatePayload{
		TemplateTypeID: 2,
		ModelID:        3,
		Name:           "demo",
		CoverImageURL:  "https://example.com/cover.jpg",
		OriginalURL:    "https://example.com/source.mp4",
	}
	for _, kind := range []int64{1, 2} {
		payload := base
		payload.TemplateType = kind
		if err := validateTemplatePayload(&payload); err != nil {
			t.Fatalf("template_type=%d rejected: %v", kind, err)
		}
	}
	for _, kind := range []int64{-1, 0, 3} {
		payload := base
		payload.TemplateType = kind
		if err := validateTemplatePayload(&payload); err == nil {
			t.Fatalf("template_type=%d must be rejected", kind)
		}
	}
}

func TestNormalizeTemplatePayloadSupportsLegacyCategoryField(t *testing.T) {
	payload := TemplatePayload{
		LegacyTemplateTypeID: 9,
		Name:                 " demo ",
		CoverImageURL:        " cover ",
		OriginalURL:          " source ",
	}
	normalizeTemplatePayload(&payload)
	if payload.TemplateTypeID != 9 || payload.Name != "demo" || payload.CoverImageURL != "cover" || payload.OriginalURL != "source" {
		t.Fatalf("normalized payload = %#v", payload)
	}
}

func TestValidateTemplateModelTypeRequiresExactMatch(t *testing.T) {
	for _, kind := range []int64{1, 2} {
		if err := validateTemplateModelType(kind, uint32(kind)); err != nil {
			t.Fatalf("matching kind %d rejected: %v", kind, err)
		}
	}
	if err := validateTemplateModelType(1, 2); err == nil {
		t.Fatal("mismatched template_type and model_type must be rejected")
	}
}
