package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSetAPIRequestMetadataPropagatesToRequestContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/api/test", nil)

	want := APIRequestMetadata{
		UserID:         42,
		TokenVersion:   3,
		LoginType:      2,
		AppCode:        "video",
		AppPackageCode: "video-android",
		AppVersion:     "1.2.3",
		ChannelCode:    "google-play",
		DeviceCountry:  "CN",
		PhoneModel:     "Pixel 9",
		SystemType:     2,
	}

	setAPIRequestMetadata(c, want)

	got, ok := APIRequestMetadataFromContext(c.Request.Context())
	if !ok {
		t.Fatal("API request metadata was not propagated to request context")
	}
	if got != want {
		t.Fatalf("metadata mismatch: got %+v, want %+v", got, want)
	}
	if got := c.Request.Context().Value(CtxDeviceCountryKey); got != want.DeviceCountry {
		t.Fatalf("legacy country context value = %v, want %q", got, want.DeviceCountry)
	}

	// Simulate a new Gin context that only retains the standard request
	// context; getters should still resolve the propagated values.
	downstream, _ := gin.CreateTestContext(httptest.NewRecorder())
	downstream.Request = c.Request
	if got := GetAPIUserID(downstream); got != want.UserID {
		t.Fatalf("GetAPIUserID() = %d, want %d", got, want.UserID)
	}
	if got := GetAPIAPPCode(downstream); got != want.AppCode {
		t.Fatalf("GetAPIAPPCode() = %q, want %q", got, want.AppCode)
	}
	if got := GetAPISystemType(downstream); got != want.SystemType {
		t.Fatalf("GetAPISystemType() = %d, want %d", got, want.SystemType)
	}
}

func TestGetClientSystemType(t *testing.T) {
	tests := []struct {
		name       string
		systemType string
		userAgent  string
		want       int
	}{
		{
			name:       "explicit numeric Android header",
			systemType: "2",
			userAgent:  "unknown",
			want:       2,
		},
		{
			name:       "explicit iOS name",
			systemType: "iOS",
			userAgent:  "unknown",
			want:       1,
		},
		{
			name:      "Android before Linux",
			userAgent: "Mozilla/5.0 (Linux; Android 15; Pixel 9)",
			want:      2,
		},
		{
			name:      "iPhone before Mac OS",
			userAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X)",
			want:      1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("GET", "/api/test", nil)
			c.Request.Header.Set(HeaderSystemType, tt.systemType)
			c.Request.Header.Set("User-Agent", tt.userAgent)

			if got := getClientSystemType(c); got != tt.want {
				t.Fatalf("getClientSystemType() = %d, want %d", got, tt.want)
			}
		})
	}
}
