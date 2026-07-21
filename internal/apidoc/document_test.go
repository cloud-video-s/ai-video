package apidoc

import (
	"ai-video/internal/repository"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBuildGeneratesPathsSchemasAndSecurity(t *testing.T) {
	routes := []gin.RouteInfo{
		{Method: http.MethodPost, Path: "/admin/login", Handler: "admin.Login"},
		{Method: http.MethodPut, Path: "/admin/banners/:id", Handler: "admin.Banner.Update"},
		{Method: http.MethodPost, Path: "/api/auth/login", Handler: "api.Auth.Login"},
		{Method: http.MethodPost, Path: "/api/auth/logout", Handler: "api.Auth.Logout"},
		{Method: http.MethodGet, Path: "/api/ob_delay", Handler: "api.DelayConfig.All"},
		{Method: http.MethodGet, Path: "/api/banners/list", Handler: "api.Banner.List"},
		{Method: http.MethodPost, Path: "/api/uploads/images/batches", Handler: "upload.CreateBatch"},
		{Method: http.MethodPut, Path: "/api/uploads/images/:upload_id/chunks/:index", Handler: "upload.PutChunk"},
	}
	document := Build(routes)
	if document.OpenAPI != "3.0.3" {
		t.Fatalf("OpenAPI = %q", document.OpenAPI)
	}
	securitySchemes := document.Components["securitySchemes"].(map[string]any)
	bearerAuth := securitySchemes["bearerAuth"].(map[string]any)
	if !strings.Contains(bearerAuth["description"].(string), "Authorization: Bearer <JWT>") {
		t.Fatalf("bearer authentication documentation is incomplete: %#v", bearerAuth)
	}
	if document.Paths["/admin/login"] != nil || document.Paths["/admin/banners/{id}"] != nil {
		t.Fatal("admin routes must not be included in the client API document")
	}
	assertCommonHeaderParameters(t, document)
	login := document.Paths["/api/auth/login"]["post"].(map[string]any)
	if _, secured := login["security"]; secured {
		t.Fatal("public login route unexpectedly requires bearer auth")
	}
	assertCommonHeaderParametersAbsent(t, login)
	assertParameter(t, login, "imei", "json", true)
	assertParameterAbsent(t, login, "Authorization")
	requestBody := login["requestBody"].(map[string]any)
	content := requestBody["content"].(map[string]any)
	schema := content["application/json"].(map[string]any)["schema"].(map[string]any)
	properties := schema["properties"].(map[string]any)
	if properties["imei"] == nil || properties["first_opened_at"] == nil {
		t.Fatalf("login-specific DTO schema is incomplete: %#v", properties)
	}
	if properties["login_type"] != nil || properties["app_package"] != nil || properties["channel_id"] != nil {
		t.Fatalf("common context fields must not be repeated in login DTO: %#v", properties)
	}
	banner := document.Paths["/api/banners/list"]["get"].(map[string]any)
	if _, secured := banner["security"]; !secured {
		t.Fatal("protected API route is missing bearer auth")
	}
	parameters := banner["parameters"].([]any)
	position := parameters[len(parameters)-1].(map[string]any)
	positionSchema := position["schema"].(map[string]any)
	if positionSchema["description"] == nil || positionSchema["maxLength"] != int64(100) {
		t.Fatalf("query field documentation is incomplete: %#v", positionSchema)
	}
	assertParameter(t, banner, "position_key", "query", true)
	loginResponse := login["responses"].(map[string]any)["200"].(map[string]any)
	responseContent := loginResponse["content"].(map[string]any)
	responseSchema := responseContent["application/json"].(map[string]any)["schema"].(map[string]any)
	responseProperties := responseSchema["properties"].(map[string]any)
	dataProperties := responseProperties["data"].(map[string]any)["properties"].(map[string]any)
	if dataProperties["token"] == nil || dataProperties["expire_at"] == nil {
		t.Fatalf("login response data schema is incomplete: %#v", dataProperties)
	}
	if login["description"] != operationDescriptions["POST /api/auth/login"] {
		t.Fatalf("login method description is missing: %#v", login["description"])
	}
	logout := document.Paths["/api/auth/logout"]["post"].(map[string]any)
	if _, exists := logout["requestBody"]; exists {
		t.Fatal("body-less logout route must not advertise a request body")
	}
	assertCommonHeaderParametersAbsent(t, logout)
	assertParameterAbsent(t, logout, "Authorization")
	chunkPath := document.Paths["/api/uploads/images/{upload_id}/chunks/{index}"]
	if chunkPath == nil {
		t.Fatal("Gin path parameters were not converted to OpenAPI syntax")
	}
	chunk := chunkPath["put"].(map[string]any)
	assertParameter(t, chunk, "upload_id", "path", true)
	assertParameter(t, chunk, "index", "path", true)
	assertParameter(t, chunk, "body", "body", true)
	batch := document.Paths["/api/uploads/images/batches"]["post"].(map[string]any)
	assertParameter(t, batch, "files", "json", true)
	assertParameter(t, batch, "files[].file_name", "json", true)
	assertParameter(t, batch, "files[].content_type", "json", false)
	delay := document.Paths["/api/ob_delay"]["get"].(map[string]any)
	assertResponseParameter(t, delay, "code", true)
	assertResponseParameter(t, delay, "message", true)
	assertResponseParameter(t, delay, "data", true)
	assertResponseParameter(t, delay, "data[].key", true)
	assertResponseParameter(t, delay, "data[].value", true)
	delayExample := delay["x-response-example"].(responseExampleEnvelope)
	delayData := delayExample.Data.([]repository.DelayConfigValue)
	if len(delayData) != 12 || delayData[0].Key != "OBPaymentCloseDely" || delayData[0].Value != "5" {
		t.Fatalf("delay response example is incomplete: %#v", delayExample)
	}
	if _, err := json.Marshal(document); err != nil {
		t.Fatalf("marshal document: %v", err)
	}
}

func assertResponseParameter(t *testing.T, operation map[string]any, name string, required bool) {
	t.Helper()
	parameters, _ := operation["x-response-parameters"].([]any)
	for _, raw := range parameters {
		parameter := raw.(map[string]any)
		if parameter["name"] != name {
			continue
		}
		if parameter["required"] != required || parameter["schema"] == nil || parameter["description"] == "" {
			t.Fatalf("invalid response parameter %q: %#v", name, parameter)
		}
		return
	}
	t.Fatalf("response parameter %q is missing: %#v", name, parameters)
}

func assertParameter(t *testing.T, operation map[string]any, name, location string, required bool) {
	t.Helper()
	parameters, _ := operation["x-request-parameters"].([]any)
	for _, raw := range parameters {
		parameter := raw.(map[string]any)
		if parameter["name"] != name {
			continue
		}
		if parameter["in"] != location || parameter["required"] != required {
			t.Fatalf("invalid parameter %q: %#v", name, parameter)
		}
		return
	}
	t.Fatalf("parameter %q is missing: %#v", name, parameters)
}

func assertParameterAbsent(t *testing.T, operation map[string]any, name string) {
	t.Helper()
	for _, field := range []string{"parameters", "x-request-parameters"} {
		parameters, _ := operation[field].([]any)
		for _, raw := range parameters {
			if raw.(map[string]any)["name"] == name {
				t.Fatalf("parameter %q must be absent from %s: %#v", name, field, parameters)
			}
		}
	}
}

func assertCommonHeaderParameters(t *testing.T, document Document) {
	t.Helper()
	parameters, ok := document.Components["parameters"].(map[string]any)
	if !ok {
		t.Fatalf("common request headers are missing: %#v", document.Components["parameters"])
	}
	wantRequired := map[string]bool{
		"Video_Channel_ID": true, "Video_App_Version": true,
		"Video_Phone_Model": true, "Video_App_Package": true,
		"Video_Channel_Package": false, "Video_Device_Country": false,
		"Accept-Language": false,
	}
	found := make(map[string]bool, len(wantRequired))
	for _, raw := range parameters {
		parameter := raw.(map[string]any)
		name, _ := parameter["name"].(string)
		required, expected := wantRequired[name]
		if !expected {
			continue
		}
		if parameter["in"] != "header" || parameter["required"] != required || parameter["description"] == "" {
			t.Fatalf("invalid client header parameter %q: %#v", name, parameter)
		}
		found[name] = true
	}
	if len(found) != len(wantRequired) {
		t.Fatalf("common request headers are incomplete: got %v, want %v", found, wantRequired)
	}
}

func assertCommonHeaderParametersAbsent(t *testing.T, operation map[string]any) {
	t.Helper()
	commonNames := map[string]bool{
		"Video_Channel_ID": true, "Video_App_Version": true,
		"Video_Phone_Model": true, "Video_App_Package": true,
		"Video_Channel_Package": true, "Video_Device_Country": true,
		"Accept-Language": true,
	}
	for _, field := range []string{"parameters", "x-request-parameters"} {
		parameters, _ := operation[field].([]any)
		for _, raw := range parameters {
			name, _ := raw.(map[string]any)["name"].(string)
			if commonNames[name] {
				t.Fatalf("common header %q must not be repeated in operation: %#v", name, operation)
			}
		}
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
	if !strings.Contains(response.Body.String(), "API 公共请求信息") || !strings.Contains(response.Body.String(), "公共请求参数") {
		t.Fatal("API document is missing the standalone common request documentation")
	}
	if !strings.Contains(response.Body.String(), "响应参数") || !strings.Contains(response.Body.String(), "响应示例") {
		t.Fatal("API document is missing response parameter documentation or examples")
	}
}
