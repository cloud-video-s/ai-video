package apidoc

import (
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"

	apiservice "ai-video/internal/server/api/server"

	"ai-video/internal/pkg/upload"

	"github.com/gin-gonic/gin"
)

type Document struct {
	OpenAPI    string                    `json:"openapi"`
	Info       map[string]any            `json:"info"`
	Servers    []map[string]any          `json:"servers"`
	Tags       []map[string]any          `json:"tags"`
	Paths      map[string]map[string]any `json:"paths"`
	Components map[string]any            `json:"components"`
}

type endpointType struct {
	body  reflect.Type
	query reflect.Type
}

var endpointTypes = map[string]endpointType{
	"POST /api/auth/login":                 {body: typeOf[apiservice.LoginRequest]()},
	"POST /api/auth/re-register":           {body: typeOf[apiservice.LoginRequest]()},
	"POST /api/auth/google":                {body: typeOf[apiservice.ThirdPartyLoginRequest]()},
	"POST /api/auth/apple":                 {body: typeOf[apiservice.ThirdPartyLoginRequest]()},
	"PUT /api/users/me/country":            {body: typeOf[apiservice.UpdateCountryRequest]()},
	"POST /api/users/me/identities/google": {body: typeOf[apiservice.BindIdentityRequest]()},
	"POST /api/users/me/identities/apple":  {body: typeOf[apiservice.BindIdentityRequest]()},
	"GET /api/banners/list":                {query: typeOf[apiservice.ClientBannerRequest]()},
	"GET /api/templates/by-position":       {query: typeOf[apiservice.ClientTemplateDisplayRequest]()},
}

var resourceNames = map[string]string{
	"health": "健康检查", "configs": "系统配置", "auth": "认证", "users": "用户",
	"banners": "Banner", "templates": "视频模板", "uploads": "文件上传", "profile": "个人资料", "identities": "第三方账号",
}

var publicRoutes = map[string]bool{
	"GET /api/health": true, "GET /api/configs/public": true,
	"POST /api/auth/login": true, "POST /api/auth/re-register": true,
	"POST /api/auth/google": true, "POST /api/auth/apple": true,
}

var paginatedRoutes = map[string]bool{
	"/api/uploads": true,
}

var operationIDSanitizer = regexp.MustCompile(`[^A-Za-z0-9]+`)

func typeOf[T any]() reflect.Type { return reflect.TypeOf((*T)(nil)).Elem() }

// Build generates OpenAPI from the routes actually registered in Gin. Route
// coverage therefore stays current even when a handler has no explicit schema mapping.
func Build(routes []gin.RouteInfo) Document {
	document := Document{
		OpenAPI: "3.0.3",
		Info: map[string]any{
			"title": "AI Video API", "version": "1.0.0",
			"description": "根据当前 Gin 路由自动生成，仅包含 /api 客户端接口。接口统一返回 {code, message, data}。",
		},
		Servers: []map[string]any{{"url": "/", "description": "当前服务"}},
		Paths:   make(map[string]map[string]any),
		Components: map[string]any{
			"securitySchemes": map[string]any{
				"bearerAuth": map[string]any{"type": "http", "scheme": "bearer", "bearerFormat": "JWT"},
			},
			"schemas": map[string]any{
				"APIResponse": map[string]any{
					"type": "object", "required": []string{"code", "message"},
					"properties": map[string]any{
						"code":    map[string]any{"type": "integer", "example": 0},
						"message": map[string]any{"type": "string", "example": "success"},
						"data":    map[string]any{"nullable": true},
					},
				},
				"UploadBatchRequest": map[string]any{
					"type": "object", "required": []string{"files"},
					"properties": map[string]any{"files": map[string]any{
						"type": "array", "minItems": 1, "items": schemaForType(typeOf[upload.FileSpec]()),
					}},
				},
			},
		},
	}

	tags := make(map[string]struct{})
	for _, route := range routes {
		if route.Method == http.MethodHead || route.Method == http.MethodOptions || !strings.HasPrefix(route.Path, "/api/") {
			continue
		}
		path, pathParams := normalizePath(route.Path)
		tag, resource := routeTag(route.Path)
		tags[tag] = struct{}{}
		operation := buildOperation(route, pathParams, tag, resource)
		if document.Paths[path] == nil {
			document.Paths[path] = make(map[string]any)
		}
		document.Paths[path][strings.ToLower(route.Method)] = operation
	}
	tagNames := make([]string, 0, len(tags))
	for tag := range tags {
		tagNames = append(tagNames, tag)
	}
	sort.Strings(tagNames)
	for _, tag := range tagNames {
		document.Tags = append(document.Tags, map[string]any{"name": tag})
	}
	return document
}

func buildOperation(route gin.RouteInfo, pathParams []string, tag, resource string) map[string]any {
	key := route.Method + " " + route.Path
	operation := map[string]any{
		"tags": []string{tag}, "summary": operationSummary(route.Method, route.Path, resource),
		"operationId": operationID(route),
		"description": fmt.Sprintf("Gin handler: `%s`", route.Handler),
		"responses": map[string]any{
			"200": map[string]any{"description": "成功", "content": jsonContent(refSchema("APIResponse"))},
			"400": errorResponse("请求参数错误"), "401": errorResponse("未登录或令牌失效"),
			"403": errorResponse("无权限"), "500": errorResponse("服务器错误"),
		},
	}
	if !publicRoutes[key] {
		operation["security"] = []map[string][]string{{"bearerAuth": {}}}
	}
	parameters := make([]any, 0, len(pathParams)+8)
	for _, name := range pathParams {
		parameters = append(parameters, map[string]any{
			"name": name, "in": "path", "required": true,
			"schema": map[string]any{"type": pathParameterType(name)},
		})
	}
	metadata := endpointTypes[key]
	if metadata.query != nil {
		parameters = append(parameters, queryParameters(metadata.query)...)
	}
	if route.Method == http.MethodGet && paginatedRoutes[route.Path] {
		parameters = appendPagination(parameters)
	}
	if strings.Contains(route.Path, "/chunks/:index") {
		parameters = append(parameters, map[string]any{
			"name": "X-Chunk-SHA256", "in": "header", "required": false,
			"description": "当前分片的 SHA-256，可选", "schema": map[string]any{"type": "string", "pattern": "^[a-fA-F0-9]{64}$"},
		})
	}
	if len(parameters) > 0 {
		operation["parameters"] = parameters
	}
	if route.Method == http.MethodPost || route.Method == http.MethodPut || route.Method == http.MethodPatch {
		operation["requestBody"] = requestBody(route, metadata.body)
	}
	return operation
}

func requestBody(route gin.RouteInfo, bodyType reflect.Type) map[string]any {
	if strings.Contains(route.Path, "/chunks/:index") {
		return map[string]any{"required": true, "content": map[string]any{
			"application/octet-stream": map[string]any{"schema": map[string]any{"type": "string", "format": "binary"}},
		}}
	}
	var schema map[string]any
	if strings.HasSuffix(route.Path, "/batches") {
		schema = refSchema("UploadBatchRequest")
	} else if bodyType != nil {
		schema = schemaForType(bodyType)
	} else {
		schema = map[string]any{"type": "object", "additionalProperties": true}
	}
	return map[string]any{"required": true, "content": jsonContent(schema)}
}

func schemaForType(valueType reflect.Type) map[string]any {
	nullable := false
	for valueType.Kind() == reflect.Pointer {
		nullable = true
		valueType = valueType.Elem()
	}
	if valueType == reflect.TypeOf(time.Time{}) {
		return map[string]any{"type": "string", "format": "date-time", "nullable": nullable}
	}
	var schema map[string]any
	switch valueType.Kind() {
	case reflect.Struct:
		properties := make(map[string]any)
		required := make([]string, 0)
		for i := 0; i < valueType.NumField(); i++ {
			field := valueType.Field(i)
			if field.PkgPath != "" {
				continue
			}
			name, tagged := fieldName(field)
			if name == "-" {
				continue
			}
			fieldSchema := schemaForType(field.Type)
			if field.Anonymous && !tagged {
				if nested, ok := fieldSchema["properties"].(map[string]any); ok {
					for nestedName, nestedSchema := range nested {
						properties[nestedName] = nestedSchema
					}
				}
				if nestedRequired, ok := fieldSchema["required"].([]string); ok {
					required = append(required, nestedRequired...)
				}
				continue
			}
			applyBindingConstraints(fieldSchema, field.Tag.Get("binding"))
			properties[name] = fieldSchema
			if hasBinding(field.Tag.Get("binding"), "required") {
				required = append(required, name)
			}
		}
		schema = map[string]any{"type": "object", "properties": properties}
		if len(required) > 0 {
			schema["required"] = uniqueStrings(required)
		}
	case reflect.Slice, reflect.Array:
		schema = map[string]any{"type": "array", "items": schemaForType(valueType.Elem())}
	case reflect.Bool:
		schema = map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		schema = map[string]any{"type": "integer", "format": "int64"}
	case reflect.Float32, reflect.Float64:
		schema = map[string]any{"type": "number", "format": "double"}
	case reflect.Map, reflect.Interface:
		schema = map[string]any{"type": "object", "additionalProperties": true}
	default:
		schema = map[string]any{"type": "string"}
	}
	if nullable {
		schema["nullable"] = true
	}
	return schema
}

func queryParameters(valueType reflect.Type) []any {
	valueType = indirectType(valueType)
	parameters := make([]any, 0)
	if valueType.Kind() != reflect.Struct {
		return parameters
	}
	for i := 0; i < valueType.NumField(); i++ {
		field := valueType.Field(i)
		if field.PkgPath != "" {
			continue
		}
		name, tagged := formFieldName(field)
		if field.Anonymous && !tagged {
			parameters = append(parameters, queryParameters(field.Type)...)
			continue
		}
		if name == "" || name == "-" {
			continue
		}
		schema := schemaForType(field.Type)
		applyBindingConstraints(schema, field.Tag.Get("binding"))
		parameter := map[string]any{
			"name": name, "in": "query", "required": hasBinding(field.Tag.Get("binding"), "required"), "schema": schema,
		}
		if schema["type"] == "array" {
			parameter["style"], parameter["explode"] = "form", true
		}
		parameters = append(parameters, parameter)
	}
	return parameters
}

func fieldName(field reflect.StructField) (string, bool) {
	if value, ok := field.Tag.Lookup("json"); ok {
		return strings.Split(value, ",")[0], true
	}
	if value, ok := field.Tag.Lookup("form"); ok {
		return strings.Split(value, ",")[0], true
	}
	return lowerFirst(field.Name), false
}

func formFieldName(field reflect.StructField) (string, bool) {
	if value, ok := field.Tag.Lookup("form"); ok {
		return strings.Split(value, ",")[0], true
	}
	return "", false
}

func applyBindingConstraints(schema map[string]any, binding string) {
	for _, rule := range strings.Split(binding, ",") {
		if strings.HasPrefix(rule, "oneof=") {
			values := strings.Fields(strings.TrimPrefix(rule, "oneof="))
			enum := make([]any, len(values))
			for i := range values {
				enum[i] = values[i]
			}
			schema["enum"] = enum
		}
	}
}

func hasBinding(binding, wanted string) bool {
	for _, rule := range strings.Split(binding, ",") {
		if rule == wanted {
			return true
		}
	}
	return false
}

func normalizePath(path string) (string, []string) {
	params := make([]string, 0)
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		if strings.HasPrefix(segment, ":") || strings.HasPrefix(segment, "*") {
			name := strings.TrimLeft(segment, ":*")
			params = append(params, name)
			segments[i] = "{" + name + "}"
		}
	}
	return strings.Join(segments, "/"), params
}

func routeTag(path string) (string, string) {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	scope := "公共"
	if len(segments) > 0 {
		if segments[0] == "admin" {
			scope = "后台"
		} else if segments[0] == "api" {
			scope = "客户端"
		}
	}
	resource := "其它"
	for _, segment := range segments[1:] {
		if name, ok := resourceNames[segment]; ok {
			resource = name
			break
		}
	}
	return scope + " / " + resource, resource
}

func operationSummary(method, path, resource string) string {
	last := path[strings.LastIndex(path, "/")+1:]
	special := map[string]string{
		"login": "登录", "logout": "退出登录", "refresh": "刷新配置缓存", "sync": "同步数据",
		"clone": "克隆" + resource, "default": "设置默认" + resource, "status": "更新" + resource + "状态",
		"display": "更新" + resource + "展示方式", "options": "查询" + resource + "选项",
		"permissions": "查询权限", "profile": "查询个人资料", "health": "健康检查",
		"complete": "完成分片上传", "batches": "初始化分片上传",
	}
	if summary, ok := special[last]; ok {
		return summary
	}
	switch method {
	case http.MethodGet:
		if strings.Contains(path, ":") {
			return "获取" + resource + "详情"
		}
		return "查询" + resource + "列表"
	case http.MethodPost:
		return "新增" + resource
	case http.MethodPut, http.MethodPatch:
		return "更新" + resource
	case http.MethodDelete:
		return "删除" + resource
	default:
		return method + " " + resource
	}
}

func operationID(route gin.RouteInfo) string {
	value := strings.Trim(operationIDSanitizer.ReplaceAllString(route.Method+"_"+route.Path, "_"), "_")
	return strings.ToLower(value)
}

func appendPagination(parameters []any) []any {
	names := make(map[string]bool)
	for _, item := range parameters {
		if parameter, ok := item.(map[string]any); ok {
			if name, ok := parameter["name"].(string); ok {
				names[name] = true
			}
		}
	}
	if !names["page"] {
		parameters = append(parameters, map[string]any{"name": "page", "in": "query", "schema": map[string]any{"type": "integer", "minimum": 1, "default": 1}})
	}
	if !names["page_size"] {
		parameters = append(parameters, map[string]any{"name": "page_size", "in": "query", "schema": map[string]any{"type": "integer", "minimum": 1, "maximum": 100, "default": 20}})
	}
	return parameters
}

func pathParameterType(name string) string {
	if name == "id" || strings.HasSuffix(name, "_id") || name == "index" {
		return "integer"
	}
	return "string"
}

func refSchema(name string) map[string]any {
	return map[string]any{"$ref": "#/components/schemas/" + name}
}

func jsonContent(schema map[string]any) map[string]any {
	return map[string]any{"application/json": map[string]any{"schema": schema}}
}

func errorResponse(description string) map[string]any {
	return map[string]any{"description": description, "content": jsonContent(refSchema("APIResponse"))}
}

func indirectType(valueType reflect.Type) reflect.Type {
	for valueType.Kind() == reflect.Pointer {
		valueType = valueType.Elem()
	}
	return valueType
}

func lowerFirst(value string) string {
	if value == "" {
		return value
	}
	return strings.ToLower(value[:1]) + value[1:]
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
