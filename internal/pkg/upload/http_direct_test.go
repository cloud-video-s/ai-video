package upload

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type stubDirectUploadSigner struct {
	credential *DirectUploadCredential
	err        error
	request    DirectUploadRequest
}

func (s *stubDirectUploadSigner) Sign(_ context.Context, request DirectUploadRequest) (*DirectUploadCredential, error) {
	s.request = request
	return s.credential, s.err
}

func TestDirectUploadSignatureRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	expiresAt := time.Now().Add(10 * time.Minute).Truncate(time.Second)
	signer := &stubDirectUploadSigner{credential: &DirectUploadCredential{
		Provider: StorageAliyunOSS, Method: "PUT", UploadURL: "https://example.com/signed",
		Headers: map[string]string{"Content-Type": "image/png"}, ObjectKey: "uploads/images/a.png",
		FileURL: "https://cdn.example.com/uploads/images/a.png", ExpiresAt: expiresAt,
	}}
	engine := gin.New()
	handler := NewHTTPHandler(nil, WithDirectUploadSigner(signer))
	handler.RegisterDirectRoute(engine.Group("/api/uploads"))

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/uploads/oss/signature", strings.NewReader(
		`{"media_type":"image","file_name":"a.png","size":123,"content_type":"image/png"}`,
	))
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	if signer.request.FileName != "a.png" || signer.request.Size != 123 {
		t.Fatalf("signer request = %+v", signer.request)
	}
	var envelope struct {
		Code int                    `json:"code"`
		Data DirectUploadCredential `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Code != 0 || envelope.Data.UploadURL != signer.credential.UploadURL {
		t.Fatalf("response = %+v", envelope)
	}
}

func TestDirectUploadSignatureRouteUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	handler := NewHTTPHandler(nil, WithDirectUploadSigner(&stubDirectUploadSigner{
		err: fmt.Errorf("%w: provider is local", ErrDirectUploadUnavailable),
	}))
	handler.RegisterDirectRoute(engine.Group("/api/uploads"))

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/uploads/oss/signature", strings.NewReader(
		`{"media_type":"image","file_name":"a.png","size":123,"content_type":"image/png"}`,
	))
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
}
