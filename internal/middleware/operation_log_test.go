package middleware

import (
	"strings"
	"testing"
)

func TestRedactJSONMasksConfigEntrySecrets(t *testing.T) {
	result := redactJSON([]byte(`{"items":[{"key":"upload.oss.access_key_id","value":"id-value"},{"key":"upload.oss.access_key_secret","value":"secret-value"}]}`))
	if strings.Contains(result, "id-value") || strings.Contains(result, "secret-value") {
		t.Fatalf("sensitive config value was not redacted: %s", result)
	}
	if count := strings.Count(result, `"value":"***"`); count != 2 {
		t.Fatalf("redacted value count = %d, want 2: %s", count, result)
	}
}
