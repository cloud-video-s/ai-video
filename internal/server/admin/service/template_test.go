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
