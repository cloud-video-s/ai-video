package apidoc

import (
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"

	"ai-video/internal/commerce"
	"ai-video/internal/generation"
	apiservice "ai-video/internal/server/api/server"

	"ai-video/internal/gen/model"
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
	body     reflect.Type
	query    reflect.Type
	response reflect.Type
}

type uploadBatchResponse struct {
	Uploads []upload.Session `json:"uploads"`
}

type uploadBatchRequest struct {
	Files []upload.FileSpec `json:"files" binding:"required,min=1"`
}

type generationModelResponse struct {
	Code              string         `json:"code"`
	Name              string         `json:"name"`
	Provider          string         `json:"provider"`
	ModelName         string         `json:"model_name"`
	Description       string         `json:"description"`
	DefaultParameters map[string]any `json:"default_parameters"`
}

type generationTaskListRequest struct {
	Page     int    `form:"page" binding:"omitempty,min=1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	Status   string `form:"status" binding:"omitempty,max=32"`
}

type generationTaskListResponse struct {
	List  []generation.TaskView `json:"list"`
	Total int64                 `json:"total"`
	Page  int                   `json:"page"`
	Size  int                   `json:"size"`
}

type responseExampleEnvelope struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

var delayConfigResponseExample = map[string]int64{
	"OBPaymentCloseDely":         5,
	"OBPaymentRetain":            0,
	"HomePaymentBannerShow":      0,
	"LaunchPaymentCloseDelay":    5,
	"LaunchPaymentRetain":        0,
	"BannerPaumentCloseDelay":    5,
	"BannerPaymentCloseRetain":   0,
	"PaymenCloseDelay":           5,
	"PaymenCloseRetain":          0,
	"FunctionPaymentCloseDelay":  5,
	"FunctionPaymentCloseRetain": 0,
	"FunctionUseLoging":          0,
}

var bannerResponseExampleTemplateID = uint64(42)

var responseDataExamples = map[string]any{
	"GET /api/ob_delay": delayConfigResponseExample,
	"GET /api/banners/list": []apiservice.ClientBanner{
		{
			ID: 12, Name: "首页夏日活动", PositionKey: "home_banner", Status: 1,
			JumpType: 2, CoverImage: "https://cdn.example.com/banners/summer.jpg",
			Route: "/templates/42", TemplateID: &bannerResponseExampleTemplateID, Sort: 10,
			TargetTemplate: &apiservice.ClientBannerTemplate{
				ID: 42, Name: "夏日视频模板", TemplateType: "action",
				CoverImage:     "https://cdn.example.com/templates/42.jpg",
				TemplateVideo:  "https://cdn.example.com/templates/42.mp4",
				ThumbnailVideo: "https://cdn.example.com/templates/42-thumb.mp4", Status: 1,
			},
		},
	},
	"POST /api/templates/:id/favorite": apiservice.TemplateFavoriteResponse{
		TemplateID: 1, Favorited: true, FavoriteCount: 1,
	},
	"DELETE /api/templates/:id/favorite": apiservice.TemplateFavoriteResponse{
		TemplateID: 1, Favorited: false, FavoriteCount: 0,
	},
	"GET /api/vip/recommend": apiservice.VIPRecommendResponse{
		ID: 1, VipType: 1, SukCode: "111111", Name: "首页ob套餐", LevelName: "", Currency: "USD",
		VIPDurationDays: 1, TrialDays: 0, BadgeText: "", AgreementDefaultChecked: 0,
		DisplayMode: 1, Status: 1, FreeTrial: 0, IsSubscription: 1, IsDefault: 0,
		SubscriptionDescription: "0", SubscriptionPrice: 0, OriginalPrice: 0,
		SubscriptionPoints: 0, SubscriptionPeriod: 1, Sort: 0, Description: "", Remark: "",
		CreatedAt: 1784859297, UpdatedAt: 1784835434,
	},
	"GET /api/vip/list": []apiservice.VIPRecommendResponse{
		{
			ID: 2, VipType: 2, SukCode: "222222", Name: "首页OB拦截套餐", LevelName: "普通套餐", Currency: "USD",
			VIPDurationDays: 1, TrialDays: 0, BadgeText: "", AgreementDefaultChecked: 0,
			DisplayMode: 1, Status: 1, FreeTrial: 0, IsSubscription: 1, IsDefault: 0,
			SubscriptionDescription: "", SubscriptionPrice: 0, OriginalPrice: 0,
			SubscriptionPoints: 0, SubscriptionPeriod: 1, Sort: 0, Description: "", Remark: "",
			CreatedAt: 1784859371, UpdatedAt: 1784835434,
		},
	},
}

var endpointTypes = map[string]endpointType{
	"GET /api/health":                                  {response: typeOf[map[string]string]()},
	"GET /api/configs/list":                            {response: typeOf[map[string]string]()},
	"POST /api/auth/login":                             {body: typeOf[apiservice.LoginRequest](), response: typeOf[apiservice.AuthResponse]()},
	"POST /api/third_binding":                          {body: typeOf[apiservice.ThirdPartyLoginRequest](), response: typeOf[apiservice.ThirdAuthResponse]()},
	"POST /api/auth/logout":                            {},
	"GET /api/users/me":                                {response: typeOf[apiservice.UserResponse]()},
	"PUT /api/users/me/country":                        {body: typeOf[apiservice.UpdateCountryRequest](), response: typeOf[apiservice.UserResponse]()},
	"GET /api/users/me/identities":                     {response: typeOf[[]model.VideoUserIdentity]()},
	"GET /api/ob_delay":                                {response: typeOf[map[string]int64]()},
	"GET /api/banners/list":                            {query: typeOf[apiservice.ClientBannerRequest](), response: typeOf[[]apiservice.ClientBanner]()},
	"GET /api/templates/recommend":                     {query: typeOf[apiservice.ClientTemplateRecommendRequest](), response: typeOf[[]apiservice.ClientTemplate]()},
	"GET /api/templates/list":                          {query: typeOf[apiservice.ClientTemplateRequest](), response: typeOf[[]apiservice.ClientTemplateType]()},
	"GET /api/templates/categories":                    {query: typeOf[apiservice.ClientTemplateRequest](), response: typeOf[[]apiservice.ClientTemplateType]()},
	"GET /api/templates/template_list":                 {query: typeOf[apiservice.TemplateListRequest](), response: typeOf[[]apiservice.ClientTemplate]()},
	"GET /api/templates/template_info":                 {query: typeOf[apiservice.TemplateInfoRequest](), response: typeOf[apiservice.ClientTemplate]()},
	"POST /api/templates/:id/favorite":                 {response: typeOf[apiservice.TemplateFavoriteResponse]()},
	"DELETE /api/templates/:id/favorite":               {response: typeOf[apiservice.TemplateFavoriteResponse]()},
	"GET /api/generation/models":                       {response: typeOf[[]generationModelResponse]()},
	"POST /api/generation/tasks":                       {body: typeOf[generation.CreateTaskRequest](), response: typeOf[generation.TaskView]()},
	"GET /api/generation/tasks":                        {query: typeOf[generationTaskListRequest](), response: typeOf[generationTaskListResponse]()},
	"GET /api/generation/tasks/:id":                    {response: typeOf[generation.TaskView]()},
	"GET /api/generation/tasks/:id/events":             {response: typeOf[generation.TaskView]()},
	"DELETE /api/generation/tasks/:id":                 {},
	"GET /api/vip/recommend":                           {query: typeOf[apiservice.VipRecommendRequest](), response: typeOf[apiservice.VIPRecommendResponse]()},
	"GET /api/vip/list":                                {query: typeOf[apiservice.VipVipListRequest](), response: typeOf[[]apiservice.VIPRecommendResponse]()},
	"POST /api/payments/apple/confirm":                 {body: typeOf[commerce.ApplePurchaseRequest](), response: typeOf[commerce.ApplePurchaseResponse]()},
	"POST /api/uploads/images/batches":                 {response: typeOf[uploadBatchResponse]()},
	"POST /api/uploads/videos/batches":                 {response: typeOf[uploadBatchResponse]()},
	"GET /api/uploads/images/:upload_id":               {response: typeOf[upload.Session]()},
	"GET /api/uploads/videos/:upload_id":               {response: typeOf[upload.Session]()},
	"PUT /api/uploads/images/:upload_id/chunks/:index": {response: typeOf[upload.Session]()},
	"PUT /api/uploads/videos/:upload_id/chunks/:index": {response: typeOf[upload.Session]()},
	"POST /api/uploads/images/:upload_id/complete":     {response: typeOf[upload.Session]()},
	"POST /api/uploads/videos/:upload_id/complete":     {response: typeOf[upload.Session]()},
}

var operationDescriptions = map[string]string{
	"GET /api/health":       "检查 API 服务是否正常运行。",
	"GET /api/configs/list": "获取客户端可见的公开应用配置。", "POST /api/auth/login": "使用设备标识登录或创建游客账号。",
	"POST /api/third_binding": "为当前用户绑定或切换 Google、Apple 等第三方身份。", "POST /api/auth/logout": "注销当前 Bearer Token。",
	"GET /api/users/me": "获取当前登录用户资料。", "PUT /api/users/me/country": "更新当前用户的设备国家或地区。",
	"GET /api/users/me/identities": "查询当前用户已绑定的第三方身份。", "DELETE /api/users/me/identities/:provider": "解绑指定第三方身份。",
	"GET /api/ob_delay":            "获取客户端延迟配置。",
	"GET /api/banners/list":        "按必填的 position_key 查询当前客户端可见的 Banner。服务端同时使用公共请求头中的 Video_App_Code、Video_App_Package_Code、Video_App_Version、Video_Device_Country，以及登录用户的会员状态进行投放匹配。某个维度没有关联记录时表示该维度支持全部；存在关联记录时必须命中。展示位置、国家、应用、应用包、版本和会员类型之间按 AND 关系组合。",
	"GET /api/templates/recommend": "查询指定展示位置已配置的推荐模板，仅返回模板数据。",
	"GET /api/templates/list":      "查询首页分类及其模板。", "GET /api/templates/categories": "查询模板分类及其模板。",
	"GET /api/templates/template_list": "分页查询指定模板分类和展示位置下的模板，仅返回模板数据。",
	"GET /api/templates/template_info": "根据模板 ID 查询当前用户可见的模板详情。",
	"POST /api/templates/:id/favorite": "收藏指定模板；重复收藏保持幂等。", "DELETE /api/templates/:id/favorite": "取消收藏指定模板；重复取消保持幂等。",
	"GET /api/generation/models": "查询客户端可用的生成模型和默认参数。", "POST /api/generation/tasks": "创建并提交视频生成任务。",
	"GET /api/generation/tasks": "分页查询当前用户的生成任务。", "GET /api/generation/tasks/:id": "查询指定生成任务详情。",
	"GET /api/generation/tasks/:id/events": "通过 SSE 实时订阅生成任务状态，任务结束后连接关闭。", "DELETE /api/generation/tasks/:id": "删除指定生成任务。",
	"GET /api/vip/recommend":           "查询当前用户适用的推荐 VIP 套餐。",
	"GET /api/vip/list":                "按必填的 vip_types 查询当前应用、包、版本及登录用户状态下可展示的 VIP 套餐列表，仅返回 status=1、display_mode=1 的套餐。",
	"POST /api/payments/apple/confirm": "校验 StoreKit 交易、创建订单并发放对应商品。",
}

var operationSummaries = map[string]string{
	"GET /api/health": "健康检查", "GET /api/configs/list": "获取客户端配置",
	"POST /api/auth/login": "游客登录", "POST /api/third_binding": "绑定第三方身份",
	"POST /api/auth/logout": "退出登录", "GET /api/users/me": "获取当前用户",
	"PUT /api/users/me/country": "更新用户国家", "GET /api/users/me/identities": "查询绑定身份",
	"DELETE /api/users/me/identities/:provider": "解绑第三方身份", "GET /api/ob_delay": "获取延迟配置",
	"GET /api/banners/list": "查询 Banner", "GET /api/templates/recommend": "查询推荐模板",
	"GET /api/templates/list": "查询模板列表", "GET /api/templates/categories": "查询模板分类",
	"GET /api/templates/template_list": "查询分类模板", "GET /api/templates/template_info": "查询模板详情",
	"POST /api/templates/:id/favorite":   "收藏模板",
	"DELETE /api/templates/:id/favorite": "取消收藏模板", "GET /api/vip/recommend": "查询推荐 VIP 套餐",
	"GET /api/vip/list":          "查询 VIP 套餐列表",
	"GET /api/generation/models": "查询生成模型", "POST /api/generation/tasks": "创建生成任务",
	"GET /api/generation/tasks": "查询生成任务", "GET /api/generation/tasks/:id": "获取生成任务",
	"GET /api/generation/tasks/:id/events": "订阅生成任务事件", "DELETE /api/generation/tasks/:id": "删除生成任务",
	"POST /api/payments/apple/confirm": "确认 Apple 支付",
	"POST /api/uploads/images/batches": "初始化图片上传", "POST /api/uploads/videos/batches": "初始化视频上传",
	"GET /api/uploads/images/:upload_id": "查询图片上传进度", "GET /api/uploads/videos/:upload_id": "查询视频上传进度",
	"PUT /api/uploads/images/:upload_id/chunks/:index": "上传图片分片", "PUT /api/uploads/videos/:upload_id/chunks/:index": "上传视频分片",
	"POST /api/uploads/images/:upload_id/complete": "完成图片上传", "POST /api/uploads/videos/:upload_id/complete": "完成视频上传",
}

var fieldDescriptions = map[string]string{
	"imei": "设备唯一标识", "device_code": "设备唯一标识", "force_new": "是否强制创建新账号", "id_token": "Google 等提供方签发的 ID Token",
	"identity_token": "Apple 签发的 Identity Token", "nonce": "用于防重放校验的随机值", "display_name": "显示名称",
	"given_name": "名", "family_name": "姓", "device_country": "设备国家或地区代码", "client_country": "客户端国家或地区代码", "channel_id": "渠道标识",
	"app_version": "应用版本号", "app_name": "应用名称", "phone_model": "设备型号", "channel_package": "渠道包标识",
	"app_package": "应用包名", "login_type": "登录类型：1 游客，2 Google，3 Apple", "first_opened_at": "首次打开时间",
	"last_opened_at": "最近打开时间", "attribution_clicked_at": "归因点击时间", "country": "国家或地区代码",
	"position_key": "展示位置唯一标识；Banner 查询时为必填 Query 参数", "package": "应用包名", "package_code": "应用包名", "package_version": "应用版本号",
	"channel": "渠道标识", "user_type": "用户类型：1 免费，2 付费", "subscription_status": "订阅状态：1 未订阅，2 已订阅，3 已取消",
	"token": "Bearer JWT", "expire_at": "Token 过期时间（Unix 秒）", "token_version": "Token 版本号",
	"id": "记录 ID", "email": "邮箱", "vip_expires_at": "VIP 到期时间（Unix 秒）", "points_balance": "积分余额",
	"status": "状态", "last_login_at": "最近登录时间", "last_login_ip": "最近登录 IP", "login_account": "登录账号",
	"appid_binding": "是否已绑定 Apple", "google_binding": "是否已绑定 Google", "provider": "身份提供方",
	"provider_subject": "身份提供方用户唯一标识", "issuer": "Token 签发方", "audience": "Token 受众",
	"email_verified": "邮箱是否已验证", "is_private_email": "是否为隐私邮箱", "avatar_url": "头像地址",
	"key": "配置键", "value": "配置值", "name": "名称", "template_type": "模板类型", "cover_image": "封面图片地址",
	"template_video": "模板视频地址", "thumbnail_video": "缩略视频地址", "jump_type": "跳转类型：1 链接，2 模板，3 文生图，4 文生视频", "route": "客户端最终跳转路由、深链或外部链接",
	"target_template": "模板跳转时返回的目标模板摘要；其他跳转类型不返回", "template_id": "目标模板 ID；仅模板跳转时返回", "sort": "排序值", "category_name": "分类名称",
	"description": "说明", "position_keys": "支持的展示位置", "user_types": "适用用户类型", "subscription_statuses": "适用订阅状态",
	"templates": "模板列表", "video_template_type_id": "模板分类 ID", "prompt": "模板提示词", "usage_count": "使用次数",
	"favorite_count": "收藏次数", "favorited": "当前用户是否已收藏", "view_count": "浏览次数", "display_config_id": "展示配置 ID", "display_sort": "展示排序",
	"page": "页码，从 1 开始", "pageSize": "每页数量", "template_type_id": "模板分类 ID", "third_type": "第三方身份类型：google 或 apple",
	"third_code": "第三方平台用户标识",
	"model_code": "生成模型代码", "client_request_id": "客户端幂等请求 ID", "input": "模型输入参数", "parameters": "模型扩展参数",
	"model_config_id": "生成模型配置 ID", "external_task_id": "第三方任务 ID", "progress": "任务进度，范围 0-100",
	"local_urls": "生成结果的本地访问地址", "error_message": "任务失败原因", "usage_duration": "任务耗时",
	"default_parameters": "模型默认参数", "model_name": "提供方模型名称",
	"bundleID": "Apple Bundle ID", "productID": "Apple 商品 ID", "transactionID": "Apple 交易 ID",
	"originalTransactionID": "Apple 原始交易 ID", "signedTransactionInfo": "Apple 签名交易 JWS",
	"purchaseDate": "购买时间", "expirationDate": "订阅到期时间", "revocationDate": "撤销时间", "isActive": "订阅是否有效",
	"vip_type": "VIP 套餐类型", "vip_types": "VIP 套餐类型数组；使用重复 Query 参数传递，例如 vip_types=1&vip_types=2",
	"suk_code": "商店产品 SKU", "level_name": "会员等级名称", "currency": "ISO 货币代码",
	"vip_duration_days": "VIP 权益持续天数", "trial_days": "免费试用天数", "badge_text": "徽章文案",
	"agreement_default_checked": "订阅协议是否默认勾选", "display_mode": "展示模式：0 隐藏，1 正常",
	"free_trial": "是否启用免费试用", "is_subscription": "是否循环订阅", "is_default": "是否为默认套餐",
	"subscription_description": "订阅说明", "subscription_price": "当前用户适用的订阅价格",
	"original_price": "原价", "subscription_points": "订阅赠送积分", "subscription_period": "订阅周期：1 周，2 月，3 季，4 年",
	"files": "待上传文件列表", "file_name": "文件名", "size": "文件字节数", "content_type": "MIME 类型", "sha256": "文件 SHA-256",
	"uploads": "上传会话列表", "upload_id": "上传会话 ID", "kind": "媒体类型：image 或 video", "original_name": "原始文件名",
	"extension": "文件扩展名", "total_size": "文件总字节数", "chunk_size": "分片字节数", "total_chunks": "分片总数",
	"uploaded_chunks": "已上传分片序号", "expected_sha256": "预期文件 SHA-256", "uploader_type": "上传者类型",
	"uploader_id": "上传者 ID", "storage_provider": "存储提供方", "completed": "是否上传完成", "file_path": "存储路径",
	"file_url": "文件访问地址", "created_at": "创建时间", "updated_at": "更新时间", "expires_at": "过期时间",
}

var resourceNames = map[string]string{
	"health": "健康检查", "configs": "系统配置", "auth": "认证", "users": "用户",
	"banners": "Banner", "templates": "视频模板", "generation": "内容生成", "payments": "支付", "vip": "VIP",
	"uploads": "文件上传", "profile": "个人资料", "identities": "第三方账号",
}

var publicRoutes = map[string]bool{
	"GET /api/health": true, "POST /api/auth/login": true,
}

var paginatedRoutes = map[string]bool{
	"/api/generation/tasks": true,
}

var sseRoutes = map[string]bool{
	"GET /api/generation/tasks/:id/events": true,
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
				"bearerAuth": map[string]any{
					"type": "http", "scheme": "bearer", "bearerFormat": "JWT",
					"description": "鉴权接口必须在请求 Header 中携带 Authorization: Bearer <JWT>；JWT 由登录接口返回。公开接口无需携带。",
				},
			},
			"parameters":                  clientHeaderParameterComponents(),
			"x-common-request-parameters": clientHeaderParameters(),
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
	metadata := endpointTypes[key]
	operation := map[string]any{
		"tags": []string{tag}, "summary": operationTitle(key, route.Method, route.Path, resource),
		"operationId": operationID(route),
		"description": operationDescription(key, route.Handler),
		"responses": map[string]any{
			"200": map[string]any{"description": "成功", "content": jsonContent(refSchema("APIResponse"))},
			"400": errorResponse("请求参数错误"), "401": errorResponse("未登录或令牌失效"),
			"403": errorResponse("无权限"), "500": errorResponse("服务器错误"),
		},
	}
	if sseRoutes[key] {
		eventSchema := responseSchemaForType(metadata.response)
		operation["responses"].(map[string]any)["200"] = map[string]any{
			"description": "SSE 任务状态事件流；事件名为 task，心跳事件名为 heartbeat",
			"content": map[string]any{"text/event-stream": map[string]any{
				"schema": map[string]any{"type": "string", "example": "event: task\ndata: {...}\n\n"},
			}},
		}
		operation["x-response-parameters"] = flattenResponseSchema(eventSchema, "event.data", true)
		operation["x-response-example"] = map[string]any{
			"event": "task", "data": exampleForSchema(eventSchema, "data"),
		}
	} else {
		operation["responses"].(map[string]any)["200"] = successResponse(metadata.response)
		responseSchema := successResponseSchema(metadata.response)
		operation["x-response-parameters"] = flattenResponseSchema(responseSchema, "", true)
		operation["x-response-example"] = buildResponseExample(key, responseSchema)
	}
	if !publicRoutes[key] {
		operation["security"] = []map[string][]string{{"bearerAuth": {}}}
	}
	parameters := make([]any, 0, len(pathParams)+12)
	for _, name := range pathParams {
		parameters = append(parameters, map[string]any{
			"name": name, "in": "path", "required": true,
			"description": pathParameterDescription(name), "schema": map[string]any{"type": pathParameterType(name)},
		})
	}
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
	displayParameters := make([]any, 0, len(parameters)+8)
	displayParameters = append(displayParameters, parameters...)
	displayParameters = append(displayParameters, bodyParameters(route, metadata.body)...)
	if len(displayParameters) > 0 {
		operation["x-request-parameters"] = displayParameters
	}
	if metadata.body != nil || strings.HasSuffix(route.Path, "/batches") || strings.Contains(route.Path, "/chunks/:index") {
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
		schema = requestSchemaForType(bodyType)
	} else {
		schema = map[string]any{"type": "object", "additionalProperties": true}
	}
	contentType := "application/json"
	if bodyParameterLocation(bodyType) == "form" {
		contentType = "application/x-www-form-urlencoded"
	}
	return map[string]any{"required": true, "content": map[string]any{
		contentType: map[string]any{"schema": schema},
	}}
}

func bodyParameters(route gin.RouteInfo, bodyType reflect.Type) []any {
	if strings.Contains(route.Path, "/chunks/:index") {
		return []any{map[string]any{
			"name": "body", "in": "body", "required": true,
			"description": "当前分片的二进制内容", "schema": map[string]any{"type": "string", "format": "binary"},
		}}
	}
	if strings.HasSuffix(route.Path, "/batches") {
		bodyType = typeOf[uploadBatchRequest]()
	}
	if bodyType == nil {
		return nil
	}
	return flattenBodySchema(requestSchemaForType(bodyType), bodyParameterLocation(bodyType), "", true)
}

func requestSchemaForType(valueType reflect.Type) map[string]any {
	schema := schemaForType(valueType)
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		return schema
	}
	for name := range commonContextFieldNames {
		delete(properties, name)
	}
	if required, ok := schema["required"].([]string); ok {
		filtered := required[:0]
		for _, name := range required {
			if !commonContextFieldNames[name] {
				filtered = append(filtered, name)
			}
		}
		if len(filtered) == 0 {
			delete(schema, "required")
		} else {
			schema["required"] = filtered
		}
	}
	return schema
}

var commonContextFieldNames = map[string]bool{
	"device_country": true, "client_country": true, "app_code": true, "app_name": true,
	"app_package_code": true, "app_package": true, "app_version": true,
	"channel_code": true, "phone_model": true, "login_type": true,
}

func flattenBodySchema(schema map[string]any, location, prefix string, parentRequired bool) []any {
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		return nil
	}
	requiredNames := make(map[string]bool)
	if required, ok := schema["required"].([]string); ok {
		for _, name := range required {
			requiredNames[name] = true
		}
	}
	names := make([]string, 0, len(properties))
	for name := range properties {
		names = append(names, name)
	}
	sort.Strings(names)
	parameters := make([]any, 0, len(names))
	for _, name := range names {
		fieldSchema, ok := properties[name].(map[string]any)
		if !ok {
			continue
		}
		fullName := name
		if prefix != "" {
			fullName = prefix + "." + name
		}
		required := parentRequired && requiredNames[name]
		parameters = append(parameters, map[string]any{
			"name": fullName, "in": location, "required": required,
			"description": fieldSchema["description"], "schema": fieldSchema,
		})
		nestedSchema := fieldSchema
		nestedPrefix := fullName
		if fieldSchema["type"] == "array" {
			if items, ok := fieldSchema["items"].(map[string]any); ok {
				nestedSchema = items
				nestedPrefix += "[]"
			}
		}
		parameters = append(parameters, flattenBodySchema(nestedSchema, location, nestedPrefix, required)...)
	}
	return parameters
}

func bodyParameterLocation(valueType reflect.Type) string {
	if valueType == nil {
		return "json"
	}
	valueType = indirectType(valueType)
	if valueType.Kind() != reflect.Struct {
		return "json"
	}
	hasJSON := false
	hasForm := false
	for i := 0; i < valueType.NumField(); i++ {
		field := valueType.Field(i)
		_, jsonTagged := field.Tag.Lookup("json")
		_, formTagged := field.Tag.Lookup("form")
		hasJSON = hasJSON || jsonTagged
		hasForm = hasForm || formTagged
		if field.Anonymous && !jsonTagged && !formTagged {
			location := bodyParameterLocation(field.Type)
			hasJSON = hasJSON || location == "json"
			hasForm = hasForm || location == "form"
		}
	}
	if hasForm && !hasJSON {
		return "form"
	}
	return "json"
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
			applyFieldDescription(fieldSchema, name)
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
	case reflect.Map:
		schema = map[string]any{"type": "object", "additionalProperties": schemaForType(valueType.Elem())}
	case reflect.Interface:
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
		applyFieldDescription(schema, name)
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
		switch {
		case strings.HasPrefix(rule, "oneof="):
			values := strings.Fields(strings.TrimPrefix(rule, "oneof="))
			enum := make([]any, len(values))
			for i := range values {
				enum[i] = values[i]
			}
			schema["enum"] = enum
		case strings.HasPrefix(rule, "max="):
			schema[maximumKey(schema)] = numericConstraint(strings.TrimPrefix(rule, "max="))
		case strings.HasPrefix(rule, "min="):
			schema[minimumKey(schema)] = numericConstraint(strings.TrimPrefix(rule, "min="))
		case strings.HasPrefix(rule, "gt="):
			schema["minimum"] = numericConstraint(strings.TrimPrefix(rule, "gt="))
			schema["exclusiveMinimum"] = true
		}
	}
}

func applyFieldDescription(schema map[string]any, name string) {
	if description := fieldDescriptions[name]; description != "" {
		schema["description"] = description
	}
}

func numericConstraint(value string) any {
	var number int64
	if _, err := fmt.Sscan(value, &number); err == nil {
		return number
	}
	return value
}

func maximumKey(schema map[string]any) string {
	switch schema["type"] {
	case "string":
		return "maxLength"
	case "array":
		return "maxItems"
	}
	return "maximum"
}

func minimumKey(schema map[string]any) string {
	switch schema["type"] {
	case "string":
		return "minLength"
	case "array":
		return "minItems"
	}
	return "minimum"
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

func operationTitle(key, method, path, resource string) string {
	if summary := operationSummaries[key]; summary != "" {
		return summary
	}
	return operationSummary(method, path, resource)
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

func pathParameterDescription(name string) string {
	descriptions := map[string]string{
		"id": "记录 ID", "upload_id": "上传会话 ID", "index": "分片序号，从 0 开始", "provider": "身份提供方：google 或 apple",
	}
	if description := descriptions[name]; description != "" {
		return description
	}
	return "路径参数 " + name
}

func clientHeaderParameters() []any {
	headers := []struct {
		name, description string
		required          bool
	}{
		{"Video_App_Code", "应用代码，对应应用配置中的 app_code；用于 Banner 等内容的应用范围匹配", true},
		{"Video_App_Package_Code", "应用包代码，对应安装包配置中的 package_code；用于 Banner 等内容的包范围匹配", true},
		{"Video_App_Version", "应用版本号；用于 Banner 等内容的版本范围匹配", true},
		{"Video_Phone_Model", "设备型号", true},
		{"Video_Channel_Code", "渠道代码，对应渠道配置中的 channel_code", true},
		{"Video_Device_Country", "ISO 3166-1 alpha-2 国家或地区代码；用于 Banner 等内容的国家范围匹配和语言配置，未传时根据客户端 IP 或用户资料推断", false},
		{"Accept-Language", "国家未配置语言时的响应语言回退值，例如 zh-CN、en-US；最终语言见响应头 Content-Language", false},
	}
	parameters := make([]any, 0, len(headers))
	for _, header := range headers {
		parameters = append(parameters, map[string]any{
			"name": header.name, "in": "header", "required": header.required,
			"description": header.description, "schema": map[string]any{"type": "string"},
		})
	}
	return parameters
}

func clientHeaderParameterComponents() map[string]any {
	components := make(map[string]any)
	for _, raw := range clientHeaderParameters() {
		parameter := raw.(map[string]any)
		components[parameter["name"].(string)] = parameter
	}
	return components
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

func successResponse(responseType reflect.Type) map[string]any {
	return map[string]any{
		"description": "请求成功",
		"content":     jsonContent(successResponseSchema(responseType)),
	}
}

func successResponseSchema(responseType reflect.Type) map[string]any {
	dataSchema := map[string]any{"nullable": true, "description": "响应数据"}
	if responseType != nil {
		dataSchema = responseSchemaForType(responseType)
		dataSchema["description"] = "响应数据"
	}
	return map[string]any{
		"type": "object", "required": []string{"code", "message", "data"},
		"properties": map[string]any{
			"code":    map[string]any{"type": "integer", "description": "业务状态码，0 表示成功", "example": 0},
			"message": map[string]any{"type": "string", "description": "结果说明", "example": "success"},
			"data":    dataSchema,
		},
	}
}

func responseSchemaForType(valueType reflect.Type) map[string]any {
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
		required := make([]string, 0, valueType.NumField())
		for i := 0; i < valueType.NumField(); i++ {
			field := valueType.Field(i)
			if field.PkgPath != "" {
				continue
			}
			name, tagged := fieldName(field)
			if name == "-" {
				continue
			}
			fieldSchema := responseSchemaForType(field.Type)
			applyBindingConstraints(fieldSchema, field.Tag.Get("binding"))
			applyFieldDescription(fieldSchema, name)
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
			properties[name] = fieldSchema
			if !jsonFieldOmitEmpty(field) {
				required = append(required, name)
			}
		}
		schema = map[string]any{"type": "object", "properties": properties}
		if len(required) > 0 {
			schema["required"] = uniqueStrings(required)
		}
	case reflect.Slice, reflect.Array:
		schema = map[string]any{"type": "array", "items": responseSchemaForType(valueType.Elem())}
	default:
		schema = schemaForType(valueType)
	}
	if nullable {
		schema["nullable"] = true
	}
	return schema
}

func jsonFieldOmitEmpty(field reflect.StructField) bool {
	tag, ok := field.Tag.Lookup("json")
	if !ok {
		return false
	}
	for _, option := range strings.Split(tag, ",")[1:] {
		if option == "omitempty" {
			return true
		}
	}
	return false
}

func flattenResponseSchema(schema map[string]any, prefix string, parentRequired bool) []any {
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		return nil
	}
	requiredNames := make(map[string]bool)
	if required, ok := schema["required"].([]string); ok {
		for _, name := range required {
			requiredNames[name] = true
		}
	}
	names := make([]string, 0, len(properties))
	for name := range properties {
		names = append(names, name)
	}
	sort.Strings(names)
	parameters := make([]any, 0, len(names))
	for _, name := range names {
		fieldSchema, ok := properties[name].(map[string]any)
		if !ok {
			continue
		}
		fullName := name
		if prefix != "" {
			fullName = prefix + "." + name
		}
		required := parentRequired && requiredNames[name]
		parameters = append(parameters, map[string]any{
			"name": fullName, "required": required,
			"description": fieldSchema["description"], "schema": fieldSchema,
		})
		nestedSchema := fieldSchema
		nestedPrefix := fullName
		if fieldSchema["type"] == "array" {
			if items, ok := fieldSchema["items"].(map[string]any); ok {
				nestedSchema = items
				nestedPrefix += "[]"
			}
		}
		parameters = append(parameters, flattenResponseSchema(nestedSchema, nestedPrefix, required)...)
	}
	return parameters
}

func buildResponseExample(key string, schema map[string]any) responseExampleEnvelope {
	data := responseDataExamples[key]
	if data == nil {
		properties := schema["properties"].(map[string]any)
		data = exampleForSchema(properties["data"].(map[string]any), "data")
	}
	return responseExampleEnvelope{Code: 0, Message: "success", Data: data}
}

func exampleForSchema(schema map[string]any, name string) any {
	if example, exists := schema["example"]; exists {
		return example
	}
	if enum, ok := schema["enum"].([]any); ok && len(enum) > 0 {
		return enum[0]
	}
	switch schema["type"] {
	case "object":
		properties, ok := schema["properties"].(map[string]any)
		if !ok {
			return map[string]any{"key": "value"}
		}
		names := make([]string, 0, len(properties))
		for propertyName := range properties {
			names = append(names, propertyName)
		}
		sort.Strings(names)
		value := make(map[string]any, len(names))
		for _, propertyName := range names {
			if propertySchema, ok := properties[propertyName].(map[string]any); ok {
				value[propertyName] = exampleForSchema(propertySchema, propertyName)
			}
		}
		return value
	case "array":
		if items, ok := schema["items"].(map[string]any); ok {
			return []any{exampleForSchema(items, name+"[]")}
		}
		return []any{}
	case "integer", "number":
		if name == "id" || strings.HasSuffix(name, "_id") {
			return 1
		}
		return 0
	case "boolean":
		return false
	case "string":
		if schema["format"] == "date-time" {
			return "2026-07-21T12:00:00+08:00"
		}
		if example := stringFieldExamples[name]; example != "" {
			return example
		}
		return "string"
	default:
		return nil
	}
}

var stringFieldExamples = map[string]string{
	"key": "OBPaymentCloseDely", "value": "5", "token": "eyJhbGciOi...",
	"country": "CN", "position_key": "home", "file_url": "/uploads/example.jpg",
	"name": "示例名称", "description": "示例说明", "status": "success",
}

func operationDescription(key, handler string) string {
	if description := operationDescriptions[key]; description != "" {
		return description
	}
	return fmt.Sprintf("客户端接口。内部处理方法：`%s`。", handler)
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
