package generation

import (
	"testing"

	"ai-video/internal/gen/model"
)

func TestViewOfSanitizesStoredErrorMessage(t *testing.T) {
	tests := []struct {
		name        string
		status      int
		wantMessage string
	}{
		{name: "terminal failure", status: TaskStatusFailure, wantMessage: TaskFailureMessage},
		{name: "retrying task", status: TaskStatusRunning, wantMessage: TaskRetryingMessage},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			view := ViewOf(&model.VideoUserGenerationTask{
				Status:       test.status,
				ErrorMessage: "provider_error=secret upstream failure",
			})
			if view.ErrorMessage != test.wantMessage {
				t.Fatalf("error_message = %q, want %q", view.ErrorMessage, test.wantMessage)
			}
		})
	}
}
