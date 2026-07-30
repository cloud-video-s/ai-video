package service

import (
	"reflect"
	"testing"

	"ai-video/internal/gen/model"
	"ai-video/internal/repository"
)

func TestParseGenerationTaskURLs(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{name: "JSON array", raw: `["https://cdn.example/a.jpg","https://cdn.example/b.jpg"]`, want: []string{"https://cdn.example/a.jpg", "https://cdn.example/b.jpg"}},
		{name: "separators and duplicates", raw: "https://cdn.example/a.mp4; https://cdn.example/b.mp4|https://cdn.example/a.mp4", want: []string{"https://cdn.example/a.mp4", "https://cdn.example/b.mp4"}},
		{name: "empty", raw: "  ", want: []string{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := parseGenerationTaskURLs(test.raw); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("parseGenerationTaskURLs() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestGenerationTaskViewPrefersLocalResults(t *testing.T) {
	record := &repository.UserGenerationTaskAdminRecord{
		Task: model.VideoUserGenerationTask{
			ID: 1, UserID: 2, ModelID: 3, TaskCode: "task-1", Status: 6, Progress: 100,
			RequestPayload: `{"input":{"prompt":"hello"}}`, ProviderResponse: `not-json`,
			RemoteUrls: `["https://remote.example/result.mp4"]`,
			LocalUrls:  `["/storage/result.mp4"]`,
		},
		User:  &model.VideoUser{ID: 2, Username: "Alice"},
		Model: &model.VideoModel{ID: 3, Name: "Video Model", Code: "video-model", ModelType: 2},
	}

	listView := generationTaskView(record, false)
	if !reflect.DeepEqual(listView.PreviewURLs, []string{"/storage/result.mp4"}) {
		t.Fatalf("PreviewURLs = %#v", listView.PreviewURLs)
	}
	if listView.MediaType != "video" || listView.ResultCount != 1 {
		t.Fatalf("media type/count = %q/%d", listView.MediaType, listView.ResultCount)
	}
	if listView.RequestPayload != nil || listView.ProviderResponse != nil {
		t.Fatal("list view must omit verbose provider payloads")
	}

	detailView := generationTaskView(record, true)
	payload, ok := detailView.RequestPayload.(map[string]interface{})
	if !ok || payload["input"] == nil {
		t.Fatalf("RequestPayload = %#v", detailView.RequestPayload)
	}
	if detailView.ProviderResponse != "not-json" {
		t.Fatalf("ProviderResponse = %#v", detailView.ProviderResponse)
	}
}

func TestParseGenerationTaskDateRange(t *testing.T) {
	from, to, err := parseGenerationTaskDateRange("2026-07-01", "2026-07-02")
	if err != nil {
		t.Fatal(err)
	}
	if from == nil || to == nil || to.Sub(*from).Hours() != 48 {
		t.Fatalf("date range = %v to %v", from, to)
	}
	if _, _, err := parseGenerationTaskDateRange("2026-07-03", "2026-07-02"); err == nil {
		t.Fatal("expected reversed date range to fail")
	}
}
