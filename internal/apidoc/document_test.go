package apidoc

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apiservice "ai-video/internal/server/api/server"

	"github.com/gin-gonic/gin"
)

func TestBuildGeneratesPathsSchemasAndSecurity(t *testing.T) {
	routes := []gin.RouteInfo{
		{Method: http.MethodPost, Path: "/admin/login", Handler: "admin.Login"},
		{Method: http.MethodPut, Path: "/admin/banners/:id", Handler: "admin.Banner.Update"},
		{Method: http.MethodPost, Path: "/api/auth/login", Handler: "api.Auth.Login"},
		{Method: http.MethodPost, Path: "/api/third_binding", Handler: "api.Auth.ThirdBinding"},
		{Method: http.MethodPost, Path: "/api/auth/logout", Handler: "api.Auth.Logout"},
		{Method: http.MethodGet, Path: "/api/ob_delay", Handler: "api.DelayConfig.All"},
		{Method: http.MethodGet, Path: "/api/banners/list", Handler: "api.Banner.List"},
		{Method: http.MethodGet, Path: "/api/templates/recommend", Handler: "api.Template.Recommend"},
		{Method: http.MethodGet, Path: "/api/templates/template_list", Handler: "api.Template.TemplateList"},
		{Method: http.MethodGet, Path: "/api/templates/template_info", Handler: "api.Template.TemplateInfo"},
		{Method: http.MethodPost, Path: "/api/templates/:id/favorite", Handler: "api.Template.Favorite"},
		{Method: http.MethodDelete, Path: "/api/templates/:id/favorite", Handler: "api.Template.Unfavorite"},
		{Method: http.MethodGet, Path: "/api/generation/models", Handler: "api.Generation.Models"},
		{Method: http.MethodPost, Path: "/api/generation/tasks", Handler: "api.Generation.Create"},
		{Method: http.MethodGet, Path: "/api/generation/tasks", Handler: "api.Generation.List"},
		{Method: http.MethodGet, Path: "/api/generation/tasks/:id", Handler: "api.Generation.Get"},
		{Method: http.MethodGet, Path: "/api/generation/tasks/:id/events", Handler: "api.Generation.Events"},
		{Method: http.MethodDelete, Path: "/api/generation/tasks/:id", Handler: "api.Generation.Delete"},
		{Method: http.MethodGet, Path: "/api/vip/recommend", Handler: "api.Vip.Recommend"},
		{Method: http.MethodGet, Path: "/api/vip/list", Handler: "api.Vip.List"},
		{Method: http.MethodPost, Path: "/api/payments/apple/confirm", Handler: "api.Payment.ConfirmApple"},
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
	assertParameter(t, login, "device_code", "json", true)
	assertParameterAbsent(t, login, "Authorization")
	requestBody := login["requestBody"].(map[string]any)
	content := requestBody["content"].(map[string]any)
	schema := content["application/json"].(map[string]any)["schema"].(map[string]any)
	properties := schema["properties"].(map[string]any)
	if properties["device_code"] == nil || properties["first_opened_at"] == nil {
		t.Fatalf("login-specific DTO schema is incomplete: %#v", properties)
	}
	if properties["login_type"] != nil || properties["app_package"] != nil || properties["channel_id"] != nil || properties["client_country"] != nil {
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
	assertResponseParameter(t, banner, "data[].target_template", false)
	assertResponseParameter(t, banner, "data[].target_template.id", false)
	assertResponseParameter(t, banner, "data[].target_template.name", false)
	assertResponseParameter(t, banner, "data[].target_template.template_type", false)
	assertResponseParameter(t, banner, "data[].target_template.cover_image", false)
	assertResponseParameter(t, banner, "data[].target_template.template_video", false)
	assertResponseParameter(t, banner, "data[].target_template.thumbnail_video", false)
	assertResponseParameter(t, banner, "data[].id", true)
	assertResponseParameter(t, banner, "data[].name", true)
	assertResponseParameter(t, banner, "data[].position_key", true)
	assertResponseParameter(t, banner, "data[].jump_type", true)
	assertResponseParameter(t, banner, "data[].cover_image", true)
	assertResponseParameter(t, banner, "data[].route", true)
	assertResponseParameter(t, banner, "data[].template_id", false)
	assertResponseParameter(t, banner, "data[].sort", true)
	if description, _ := banner["description"].(string); !strings.Contains(description, "Video_App_Code") ||
		!strings.Contains(description, "没有关联记录") || !strings.Contains(description, "AND") {
		t.Fatalf("banner targeting rules are incomplete: %q", description)
	}
	bannerExample := banner["x-response-example"].(responseExampleEnvelope)
	bannerData := bannerExample.Data.([]apiservice.ClientBanner)
	if len(bannerData) != 1 || bannerData[0].PositionKey != "home_banner" || bannerData[0].TargetTemplate == nil ||
		bannerData[0].TargetTemplate.ID != 42 {
		t.Fatalf("banner response example is incomplete: %#v", bannerExample)
	}
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
	assertResponseParameterAbsent(t, delay, "data[].key")
	assertResponseParameterAbsent(t, delay, "data[].value")
	delayExample := delay["x-response-example"].(responseExampleEnvelope)
	delayData := delayExample.Data.(map[string]int64)
	if len(delayData) != 12 || delayData["OBPaymentCloseDely"] != 5 || delayData["FunctionUseLoging"] != 0 {
		t.Fatalf("delay response example is incomplete: %#v", delayExample)
	}
	recommend := document.Paths["/api/templates/recommend"]["get"].(map[string]any)
	assertResponseParameter(t, recommend, "data[].id", true)
	assertResponseParameter(t, recommend, "data[].name", true)
	assertResponseParameterAbsent(t, recommend, "data[].display_config_id")
	assertResponseParameterAbsent(t, recommend, "data[].position_key")
	assertResponseParameterAbsent(t, recommend, "data[].display_sort")
	if recommend["description"] != operationDescriptions["GET /api/templates/recommend"] {
		t.Fatalf("recommend documentation was not regenerated: %#v", recommend["description"])
	}
	categoryTemplates := document.Paths["/api/templates/template_list"]["get"].(map[string]any)
	assertParameter(t, categoryTemplates, "page", "query", false)
	assertParameter(t, categoryTemplates, "pageSize", "query", false)
	assertParameter(t, categoryTemplates, "position_key", "query", true)
	assertParameter(t, categoryTemplates, "template_type_id", "query", true)
	assertResponseParameter(t, categoryTemplates, "data[].id", true)
	assertResponseParameter(t, categoryTemplates, "data[].name", true)
	assertResponseParameterAbsent(t, categoryTemplates, "data[].display_config_id")
	templateInfo := document.Paths["/api/templates/template_info"]["get"].(map[string]any)
	assertParameter(t, templateInfo, "template_id", "query", true)
	assertResponseParameter(t, templateInfo, "data.id", true)
	createTask := document.Paths["/api/generation/tasks"]["post"].(map[string]any)
	assertParameter(t, createTask, "model_code", "json", true)
	assertParameter(t, createTask, "input", "json", true)
	assertResponseParameter(t, createTask, "data.status", true)
	listTasks := document.Paths["/api/generation/tasks"]["get"].(map[string]any)
	assertParameter(t, listTasks, "page", "query", false)
	assertParameter(t, listTasks, "page_size", "query", false)
	assertParameter(t, listTasks, "status", "query", false)
	assertResponseParameter(t, listTasks, "data.list", true)
	assertResponseParameter(t, listTasks, "data.total", true)
	events := document.Paths["/api/generation/tasks/{id}/events"]["get"].(map[string]any)
	assertParameter(t, events, "id", "path", true)
	eventResponse := events["responses"].(map[string]any)["200"].(map[string]any)
	eventContent := eventResponse["content"].(map[string]any)
	if eventContent["text/event-stream"] == nil {
		t.Fatalf("generation events must document SSE content: %#v", eventResponse)
	}
	assertResponseParameter(t, events, "event.data.id", true)
	vip := document.Paths["/api/vip/recommend"]["get"].(map[string]any)
	assertParameter(t, vip, "vip_type", "query", true)
	assertResponseParameter(t, vip, "data.id", true)
	assertResponseParameter(t, vip, "data.suk_code", true)
	assertResponseParameter(t, vip, "data.level_name", true)
	assertResponseParameter(t, vip, "data.subscription_price", true)
	assertResponseParameterAbsent(t, vip, "data.apps")
	assertResponseParameterAbsent(t, vip, "data.packages")
	assertResponseParameterAbsent(t, vip, "data.package_version")
	assertResponseParameterAbsent(t, vip, "data.country")
	assertResponseParameterAbsent(t, vip, "data.channels")
	assertResponseParameterAbsent(t, vip, "data.deleted_at")
	vipList := document.Paths["/api/vip/list"]["get"].(map[string]any)
	assertParameter(t, vipList, "vip_types", "query", true)
	assertResponseParameter(t, vipList, "data[].id", true)
	assertResponseParameter(t, vipList, "data[].vip_type", true)
	assertResponseParameter(t, vipList, "data[].suk_code", true)
	assertResponseParameter(t, vipList, "data[].level_name", true)
	assertResponseParameter(t, vipList, "data[].subscription_price", true)
	vipListExample := vipList["x-response-example"].(responseExampleEnvelope).Data.([]apiservice.VIPRecommendResponse)
	if len(vipListExample) != 1 || vipListExample[0].ID != 2 || vipListExample[0].SukCode != "222222" || vipListExample[0].CreatedAt != 1784859371 {
		t.Fatalf("vip list response example is incomplete: %#v", vipListExample)
	}
	payment := document.Paths["/api/payments/apple/confirm"]["post"].(map[string]any)
	assertParameter(t, payment, "bundleID", "json", true)
	assertParameter(t, payment, "signedTransactionInfo", "json", true)
	assertResponseParameter(t, payment, "data.transaction_id", true)
	thirdBinding := document.Paths["/api/third_binding"]["post"].(map[string]any)
	assertParameter(t, thirdBinding, "third_type", "json", true)
	assertParameter(t, thirdBinding, "third_code", "json", false)
	assertResponseParameter(t, thirdBinding, "data.token", true)
	if _, secured := thirdBinding["security"]; !secured {
		t.Fatal("third binding route must require bearer auth")
	}
	favoritePath := document.Paths["/api/templates/{id}/favorite"]
	for _, method := range []string{"post", "delete"} {
		operation := favoritePath[method].(map[string]any)
		assertParameter(t, operation, "id", "path", true)
		assertResponseParameter(t, operation, "data.template_id", true)
		assertResponseParameter(t, operation, "data.favorited", true)
		assertResponseParameter(t, operation, "data.favorite_count", true)
		if _, secured := operation["security"]; !secured {
			t.Fatalf("%s favorite operation must require bearer auth", method)
		}
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

func assertResponseParameterAbsent(t *testing.T, operation map[string]any, name string) {
	t.Helper()
	parameters, _ := operation["x-response-parameters"].([]any)
	for _, raw := range parameters {
		if raw.(map[string]any)["name"] == name {
			t.Fatalf("response parameter %q must be absent: %#v", name, parameters)
		}
	}
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
		"Video_App_Code": true, "Video_App_Package_Code": true,
		"Video_App_Version": true, "Video_Phone_Model": true,
		"Video_Channel_Code": true, "Video_Device_Country": false,
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
		if (name == "Video_App_Code" || name == "Video_App_Package_Code" || name == "Video_App_Version") &&
			!strings.Contains(parameter["description"].(string), "Banner") {
			t.Fatalf("banner targeting purpose is missing from %q: %#v", name, parameter)
		}
	}
	if len(found) != len(wantRequired) {
		t.Fatalf("common request headers are incomplete: got %v, want %v", found, wantRequired)
	}
}

func assertCommonHeaderParametersAbsent(t *testing.T, operation map[string]any) {
	t.Helper()
	commonNames := map[string]bool{
		"Video_App_Code": true, "Video_App_Package_Code": true,
		"Video_App_Version": true, "Video_Phone_Model": true,
		"Video_Channel_Code": true, "Video_Device_Country": true,
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
