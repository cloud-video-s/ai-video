package apidoc

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBuildGeneratesPathsSchemasAndSecurity(t *testing.T) {
	routes := []gin.RouteInfo{
		{Method: http.MethodPost, Path: "/admin/login", Handler: "admin.Login"},
		{Method: http.MethodPut, Path: "/admin/banners/:id", Handler: "admin.Banner.Update"},
		{Method: http.MethodPost, Path: "/api/auth/login", Handler: "api.Auth.Login"},
		{Method: http.MethodGet, Path: "/api/banners/list", Handler: "api.Banner.List"},
		{Method: http.MethodPut, Path: "/api/uploads/images/:upload_id/chunks/:index", Handler: "upload.PutChunk"},
	}
	document := Build(routes)
	if document.OpenAPI != "3.0.3" {
		t.Fatalf("OpenAPI = %q", document.OpenAPI)
	}
	if document.Paths["/admin/login"] != nil || document.Paths["/admin/banners/{id}"] != nil {
		t.Fatal("admin routes must not be included in the client API document")
	}
	login := document.Paths["/api/auth/login"]["post"].(map[string]any)
	if _, secured := login["security"]; secured {
		t.Fatal("public login route unexpectedly requires bearer auth")
	}
	requestBody := login["requestBody"].(map[string]any)
	content := requestBody["content"].(map[string]any)
	schema := content["application/json"].(map[string]any)["schema"].(map[string]any)
	properties := schema["properties"].(map[string]any)
	if properties["login_type"] == nil || properties["app_package"] == nil {
		t.Fatalf("login DTO schema is incomplete: %#v", properties)
	}
	banner := document.Paths["/api/banners/list"]["get"].(map[string]any)
	if _, secured := banner["security"]; !secured {
		t.Fatal("protected API route is missing bearer auth")
	}
	if document.Paths["/api/uploads/images/{upload_id}/chunks/{index}"] == nil {
		t.Fatal("Gin path parameters were not converted to OpenAPI syntax")
	}
	if _, err := json.Marshal(document); err != nil {
		t.Fatalf("marshal document: %v", err)
	}
}

func TestRegisterServesOpenAPIAndSwaggerUI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.GET("/api/health", func(c *gin.Context) { c.Status(http.StatusOK) })
	engine.GET("/admin/users", func(c *gin.Context) { c.Status(http.StatusOK) })
	Register(engine)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/docs/openapi.json", nil)
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("OpenAPI status = %d", response.Code)
	}
	var document Document
	if err := json.Unmarshal(response.Body.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	if document.Paths["/api/health"] == nil {
		t.Fatal("registered route is missing from served document")
	}
	if document.Paths["/admin/users"] != nil {
		t.Fatal("admin route leaked into served document")
	}

	response = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "/docs/ui", nil)
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("Swagger UI status = %d", response.Code)
	}
}
