package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-video/internal/pkg/errcode"
	"ai-video/internal/pkg/i18n"

	"github.com/gin-gonic/gin"
)

func TestAPIFailureIsSanitizedAndOriginalErrorIsLogged(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/generation/tasks", nil)
	i18n.MarkAPI(c, i18n.LocaleEnUS)

	original := "query generation tasks: database connection refused"
	Fail(c, errcode.ErrServer, original)

	var result Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Message != "The service is temporarily unavailable. Please try again later." {
		t.Fatalf("message = %q", result.Message)
	}
	if result.Message == original {
		t.Fatal("the original error was returned to the API client")
	}
	privateErrors := c.Errors.ByType(gin.ErrorTypePrivate)
	if len(privateErrors) != 1 {
		t.Fatalf("recorded errors = %d, want 1", len(privateErrors))
	}
	if got := privateErrors[0].Error(); got != original {
		t.Fatalf("recorded original error = %q", got)
	}
}

func TestNonAPIFailureKeepsExistingMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/users", nil)

	Fail(c, errcode.ErrParam, "existing admin message")

	var result Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Message != "existing admin message" {
		t.Fatalf("message = %q", result.Message)
	}
}
