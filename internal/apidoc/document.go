package apidoc

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"

	"ai-video/internal/commerce"
	"ai-video/internal/generation"
	"ai-video/internal/modelparameter"
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

type adjustAttributionCallbackRequest struct {
	CallbackToken        string `json:"callback_token" form:"callback_token" binding:"required"`
	ADID                 string `json:"adid" form:"adid" binding:"required,max=64"`
	AppToken             string `json:"app_token" form:"app_token" binding:"omitempty,max=64"`
	TrackerToken         string `json:"tracker_token" form:"tracker_token" binding:"omitempty,max=64"`
	TrackerName          string `json:"tracker_name" form:"tracker_name" binding:"omitempty,max=255"`
	OutdatedTracker      string `json:"outdated_tracker" form:"outdated_tracker" binding:"omitempty,max=64"`
	OutdatedTrackerName  string `json:"outdated_tracker_name" form:"outdated_tracker_name" binding:"omitempty,max=255"`
	Network              string `json:"network" form:"network" binding:"omitempty,max=255"`
	Campaign             string `json:"campaign" form:"campaign" binding:"omitempty,max=255"`
	Adgroup              string `json:"adgroup" form:"adgroup" binding:"omitempty,max=255"`
	Creative             string `json:"creative" form:"creative" binding:"omitempty,max=255"`
	ActivityKind         string `json:"activity_kind" form:"activity_kind" binding:"omitempty,max=64"`
	AttributionType      string `json:"attribution_type" form:"attribution_type" binding:"omitempty,max=32"`
	Reattributed         bool   `json:"reattributed" form:"reattributed"`
	IsRedownload         bool   `json:"is_redownload" form:"is_redownload"`
	ClickTime            string `json:"click_time" form:"click_time"`
	InstalledAt          string `json:"installed_at" form:"installed_at"`
	ReattributedAt       string `json:"reattributed_at" form:"reattributed_at"`
	AttributionUpdatedAt string `json:"attribution_updated_at" form:"attribution_updated_at"`
	CreatedAt            string `json:"created_at" form:"created_at"`
}

type templateCategoriesQuery struct {
	apiservice.BasePage
}

type uploadBatchResponse struct {
	Uploads []upload.Session `json:"uploads"`
}

type uploadBatchRequest struct {
	Files []upload.FileSpec `json:"files" binding:"required,min=1"`
}

type uploadListQuery struct {
	apiservice.BasePage
	Status          int8   `form:"status" binding:"omitempty,oneof=1 2"`
	MediaType       string `form:"media_type" binding:"omitempty,oneof=image video"`
	FileType        string `form:"file_type" binding:"max=32"`
	StorageProvider string `form:"storage_provider" binding:"omitempty,oneof=aliyun_oss"`
	Keyword         string `form:"keyword" binding:"max=255"`
}

type uploadListResponse struct {
	List  []model.VideoUpload `json:"list"`
	Total int64               `json:"total"`
	Page  int                 `json:"page"`
	Size  int                 `json:"size"`
}

type trackingEventDocRequest struct {
	TrackingType  string `json:"tracking_type" binding:"required,oneof=OB_Payment_show OB_Payment_back_show Home_Show Launc_Payment_Show Launc_Payment_back_Show Payment_Show Payment_Create Payment_Suc Case_create"`
	ExtensionType string `json:"extension_type"`
	ModelID       uint64 `json:"model_id"`
}

type trackingEventDocResponse struct {
	ID            uint64    `json:"id"`
	TrackingType  string    `json:"tracking_type"`
	ExtensionType string    `json:"extension_type"`
	ModelID       uint64    `json:"model_id"`
	CreatedAt     time.Time `json:"created_at"`
}

type generationTaskListResponse struct {
	Page       int64                 `json:"page"`
	PageSize   int64                 `json:"pageSize"`
	Total      int64                 `json:"total"`
	TotalPages int64                 `json:"totalPages"`
	List       []generation.TaskView `json:"list"`
}

type clientPointsListResponse struct {
	Page       int64                             `json:"page"`
	PageSize   int64                             `json:"pageSize"`
	Total      int64                             `json:"total"`
	TotalPages int64                             `json:"totalPages"`
	List       []apiservice.ClientPointsResponse `json:"list"`
}

type clientTemplateTypePageResponse struct {
	Page       int64                           `json:"page"`
	PageSize   int64                           `json:"pageSize"`
	Total      int64                           `json:"total"`
	TotalPages int64                           `json:"totalPages"`
	List       []apiservice.ClientTemplateType `json:"list"`
}

type clientTemplatePageResponse struct {
	Page       int64                       `json:"page"`
	PageSize   int64                       `json:"pageSize"`
	Total      int64                       `json:"total"`
	TotalPages int64                       `json:"totalPages"`
	List       []apiservice.ClientTemplate `json:"list"`
}

type templateTaskInput struct {
	Images []string `json:"images" binding:"required,min=1"`
}

type templateTaskRequest struct {
	TemplateID uint64            `json:"template_id" binding:"required"`
	Input      templateTaskInput `json:"input" binding:"required"`
}

type loginDocRequest struct {
	DeviceCode           string     `json:"device_code" binding:"required,max=128"`
	ForceNew             bool       `json:"force_new"`
	FirstOpenedAt        *time.Time `json:"first_opened_at"`
	LastOpenedAt         *time.Time `json:"last_opened_at"`
	AttributionClickedAt *time.Time `json:"attribution_clicked_at"`
}

type thirdPartyLoginDocRequest struct {
	ThirdType            string     `json:"third_type" binding:"required,max=50"`
	ThirdCode            string     `json:"third_code" binding:"omitempty,max=100"`
	Email                string     `json:"email" binding:"omitempty,max=50"`
	IDToken              string     `json:"id_token" binding:"omitempty,max=16384"`
	IdentityToken        string     `json:"identity_token" binding:"omitempty,max=16384"`
	Nonce                string     `json:"nonce" binding:"omitempty,max=255"`
	ForceNew             bool       `json:"force_new"`
	FirstOpenedAt        *time.Time `json:"first_opened_at"`
	LastOpenedAt         *time.Time `json:"last_opened_at"`
	AttributionClickedAt *time.Time `json:"attribution_clicked_at"`
}

var requestBodyExamples = map[string]any{
	"POST /api/attributions/adjust/report": map[string]any{
		"trackerToken": "22hydf4k", "trackerName": "TikTok SAN tracker",
		"campaign": "TT campaign", "network": "TikTok SAN",
		"creative": "instruction.mp4", "adgroup": "TT adgroup",
		"clickLabel": "", "costType": "", "costAmount": nil, "costCurrency": "",
		"fbInstallReferrer": "", "googleAdId": "928cdf5a-d453-45a6-8016-115481cbeaa5",
		"adid": "0a09e2a1de95add39162efdf3adff446", "idfa": "", "idfv": "",
	},
	"POST /api/attributions/adjust/callback": map[string]any{
		"callback_token": "configured-callback-token", "adid": "adjust-device-id",
		"app_token": "app-token", "tracker_token": "abc123",
		"tracker_name": "Paid Tracker", "network": "Example Network", "campaign": "Launch",
		"activity_kind": "install", "click_time": "2026-08-17T10:00:00Z",
		"installed_at": "2026-08-17T10:05:00Z", "created_at": "2026-08-17T10:06:00Z",
	},
	"POST /api/auth/login": map[string]any{
		"device_code": "device-0123456789abcdef", "force_new": false,
	},
	"POST /api/auth/apple_order_login": map[string]any{
		"order_code": []string{"2000001209105682", "2000001209105683"}, "force_new": false,
	},
	"POST /api/third_binding": map[string]any{
		"third_type": "apple", "identity_token": "provider-signed-jwt", "nonce": "optional-request-nonce",
	},
	"POST /api/users/active_reporting": map[string]any{
		"time_long": uint64(300),
	},
	"POST /api/tracking/events": map[string]any{
		"tracking_type": "Payment_Create", "extension_type": "OB_back", "model_id": uint64(123),
	},
	"POST /api/orders": map[string]any{
		"shop_type": uint32(2), "product_id": uint64(8), "pay_type": uint32(1),
		"client_request_id": "points-order-20260820-0001",
	},
	"POST /api/templates/complaint": map[string]any{
		"template_id": uint64(3), "complaint_type": "Hate speech or discrimination", "content": "测试",
	},
	"POST /api/generation/template-tasks": map[string]any{
		"template_id": uint64(9),
		"input": map[string]any{"images": []string{
			"https://balaaitest.oss-ap-southeast-1.aliyuncs.com/uploads/images/2026/07/30/12f35f980837d2e557f1e07a3078d1dc.png",
		}},
	},
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

var applePurchaseResponseExampleExpiration = time.Date(2026, 7, 22, 16, 47, 39, 0, time.FixedZone("UTC+8", 8*60*60))

var generationTaskResponseExample = generation.TaskView{
	ID: 13, TaskCode: "eafe15f0-780f-4a7f-9c62-7e99484be521", TaskType: generation.TaskTypeVideo,
	Status: generation.TaskStatusFailure, Progress: 100,
	Input: map[string]any{
		"end_frame":   "https://cdn.example.com/uploads/images/end-frame.png",
		"first_frame": "https://cdn.example.com/uploads/images/first-frame.png",
		"prompt":      "生成一个可爱风格的动漫视频",
		"video":       "https://cdn.example.com/uploads/videos/reference.mp4",
	},
	Parameters: map[string]any{
		"aspect_ratio": "1:1", "duration": float64(10),
		"external_task_id": "eafe15f0-780f-4a7f-9c62-7e99484be521", "mode": "std",
	},
	LocalURLs:     []string{},
	ErrorMessage:  generation.TaskFailureMessage,
	UsageDuration: 10,
	SubmittedAt:   generationTaskExampleTime(2026, 7, 30, 14, 44, 41, 602),
	StartedAt:     generationTaskExampleTime(2026, 7, 30, 14, 44, 43, 804),
	FinishedAt:    generationTaskExampleTime(2026, 7, 30, 14, 48, 33, 154),
	CreatedAt:     *generationTaskExampleTime(2026, 7, 30, 14, 44, 28, 235),
	UpdatedAt:     *generationTaskExampleTime(2026, 7, 30, 14, 48, 33, 155),
}

var generationTaskListResponseExample = generationTaskListResponse{
	Page: 1, PageSize: 10, Total: 3, TotalPages: 1,
	List: []generation.TaskView{
		generationTaskResponseExample,
		{
			ID: 12, TaskCode: "c55e71fc-f2c1-4f8e-aae7-9f380fc0b0ea", TaskType: generation.TaskTypeVideo,
			Status: generation.TaskStatusFailure, Progress: 100,
			Input: map[string]any{
				"end_frame":   "https://cdn.example.com/uploads/images/end-frame.png",
				"first_frame": "https://cdn.example.com/uploads/images/first-frame.png",
				"prompt":      "生成一个可爱风格的动漫视频",
				"video":       "https://cdn.example.com/uploads/videos/reference.mp4",
			},
			Parameters: map[string]any{
				"aspect_ratio": "1:1", "duration": float64(10),
				"external_task_id": "c55e71fc-f2c1-4f8e-aae7-9f380fc0b0ea", "mode": "std",
			},
			LocalURLs:     []string{},
			ErrorMessage:  generation.TaskFailureMessage,
			UsageDuration: 10,
			SubmittedAt:   generationTaskExampleTime(2026, 7, 30, 11, 57, 59, 450),
			StartedAt:     generationTaskExampleTime(2026, 7, 30, 11, 58, 2, 749),
			FinishedAt:    generationTaskExampleTime(2026, 7, 30, 14, 48, 31, 158),
			CreatedAt:     *generationTaskExampleTime(2026, 7, 30, 11, 56, 50, 131),
			UpdatedAt:     *generationTaskExampleTime(2026, 7, 30, 14, 48, 31, 158),
		},
		{
			ID: 11, TaskCode: "d9dd18d6-6cd4-4df3-8dff-f7c3622990b7", TaskType: generation.TaskTypeVideo,
			Status: generation.TaskStatusSuccess, Progress: 100,
			Input: map[string]any{
				"end_frame": "", "first_frame": "",
				"images": []string{
					"https://cdn.example.com/uploads/images/reference-1.png",
					"https://cdn.example.com/uploads/images/reference-2.png",
				},
				"prompt": "生成一个可爱风格的动漫视频", "video": "",
			},
			Parameters: map[string]any{
				"aspect_ratio": "1:1", "duration": float64(10),
				"external_task_id": "d9dd18d6-6cd4-4df3-8dff-f7c3622990b7", "mode": "std",
			},
			LocalURLs:     []string{"https://cdn.example.com/uploads/generated/1/task-11-1.mp4"},
			CoverImageURL: "https://cdn.example.com/uploads/generated/1/task-11-cover.jpg",
			UsageDuration: 10,
			SubmittedAt:   generationTaskExampleTime(2026, 7, 30, 9, 26, 25, 938),
			FinishedAt:    generationTaskExampleTime(2026, 7, 30, 11, 24, 22, 563),
			CreatedAt:     *generationTaskExampleTime(2026, 7, 30, 9, 21, 32, 387),
			UpdatedAt:     *generationTaskExampleTime(2026, 7, 30, 11, 24, 39, 676),
		},
	},
}

var clientPointsListResponseExample = clientPointsListResponse{
	Page: 1, PageSize: 10, Total: 1, TotalPages: 1,
	List: []apiservice.ClientPointsResponse{{
		ID: 1, UserID: 8, Direction: 1, PointsChange: 100,
		BalanceBefore: 20, BalanceAfter: 120, Description: "购买 VIP 月卡赠送",
		CreatedAt: 1785816000, UpdatedAt: 1785816000,
	}},
}

func generationTaskExampleTime(year, month, day, hour, minute, second, millisecond int) *time.Time {
	value := time.Date(year, time.Month(month), day, hour, minute, second, millisecond*int(time.Millisecond), time.FixedZone("UTC+8", 8*60*60))
	return &value
}

func exampleValueOptions(values []interface{}, aliases []string) []modelparameter.ValueOption {
	options := make([]modelparameter.ValueOption, len(values))
	for i := range values {
		options[i] = modelparameter.ValueOption{Value: values[i], Alias: aliases[i]}
	}
	return options
}

var generationModelResponseExample = []apiservice.GenerationModelView{
	{
		Name: "Kling v3 视频生成", ModelCode: "kling-v3", Score: 95,
		Icon: "https://cdn.example.com/models/kling-v3.png", Description: "支持多种画面比例和生成模式的视频模型",
		Parameters: []apiservice.GenerationModelParameter{
			{
				ParamKey: "aspect_ratio", DefaultValue: "16:9",
				AllowedValues: exampleValueOptions([]interface{}{"16:9", "9:16", "1:1"}, []string{"横屏", "竖屏", "方形"}),
				Description:   "生成视频的宽高比", ParameterType: 1, Constraints: "{}",
				Alias: "画面比例", DisplayType: "select", IsDisplay: 1,
			},
			{
				ParamKey: "character_orientation", DefaultValue: "video",
				AllowedValues: exampleValueOptions([]interface{}{"video", "image"}, []string{"视频", "图片"}),
				Description:   "生成类型", ParameterType: 1, Constraints: "{}",
				Alias: "角色参考类型", DisplayType: "select", IsDisplay: 1,
			},
			{
				ParamKey: "duration", DefaultValue: 15,
				AllowedValues: exampleValueOptions([]interface{}{3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}, []string{"3 秒", "4 秒", "5 秒", "6 秒", "7 秒", "8 秒", "9 秒", "10 秒", "11 秒", "12 秒", "13 秒", "14 秒", "15 秒"}),
				Description:   "生成视频的时长（秒）", ParameterType: 1, Constraints: "{}",
				Alias: "视频时长", DisplayType: "select", IsDisplay: 1,
			},
			{
				ParamKey: "keep_original_sound", DefaultValue: "no",
				AllowedValues: exampleValueOptions([]interface{}{"yes", "no"}, []string{"保留", "不保留"}),
				Description:   "是否保留参考视频中的原始音频", ParameterType: 1, Constraints: "{}",
				Alias: "保留原声", DisplayType: "radio", IsDisplay: 1,
			},
			{
				ParamKey: "mode", DefaultValue: "std",
				AllowedValues: exampleValueOptions([]interface{}{"std", "pro"}, []string{"标准", "专业"}),
				Description:   "生成质量模式。std 为 720P，pro 为 1080P", ParameterType: 1, Constraints: "{}",
				Alias: "生成质量", DisplayType: "select", IsDisplay: 1,
			},
			{
				ParamKey: "prompt", DefaultValue: nil, AllowedValues: []modelparameter.ValueOption{},
				Description: "生成提示词", ParameterType: 2, Constraints: `{"max_length": 2500}`,
				Alias: "提示词", DisplayType: "textarea", IsDisplay: 1,
			},
			{
				ParamKey: "first_frame_url", DefaultValue: nil, AllowedValues: []modelparameter.ValueOption{},
				Description: "首帧图像 URL", ParameterType: 2, Constraints: `{"max_length": 1}`,
				Alias: "首帧图片", DisplayType: "image", IsDisplay: 1,
			},
			{
				ParamKey: "images", DefaultValue: nil, AllowedValues: []modelparameter.ValueOption{},
				Description: "首帧图像输入", ParameterType: 2, Constraints: `{"max_length": 10}`,
				Alias: "参考图片", DisplayType: "images", IsDisplay: 1,
			},
			{
				ParamKey: "img_url", DefaultValue: nil, AllowedValues: []modelparameter.ValueOption{},
				Description: "参考图像 URL", ParameterType: 2, Constraints: `{"max_length": 1}`,
				Alias: "参考图", DisplayType: "image", IsDisplay: 1,
			},
			{
				ParamKey: "negative_prompt", DefaultValue: nil, AllowedValues: []modelparameter.ValueOption{},
				Description: "限制不期望内容的负向提示词", ParameterType: 2, Constraints: `{"max_length": 2500}`,
				Alias: "负向提示词", DisplayType: "textarea", IsDisplay: 1,
			},
			{
				ParamKey: "video_url", DefaultValue: nil, AllowedValues: []modelparameter.ValueOption{},
				Description: "motion_control 模式的参考视频 URL", ParameterType: 2, Constraints: `{"max_length": 1}`,
				Alias: "参考视频", DisplayType: "video", IsDisplay: 1,
			},
		},
	},
}

var clientTemplateResponseExample = apiservice.ClientTemplate{
	ID: 101, TemplateTypeID: 1, Name: "动漫视频", TemplateType: 2,
	Icon:          "https://cdn.example.com/templates/101-icon.png",
	CoverImageURL: "https://cdn.example.com/templates/101-cover.jpg",
	OriginalURL:   "https://cdn.example.com/templates/101.mp4",
	ThumbnailURL:  "https://cdn.example.com/templates/101-thumbnail.mp4",
	Prompt:        "生成动漫风格视频", Description: "动漫风格模板", Sort: 100,
	UsageCount: 120, FavoriteCount: 18, ViewCount: 360, IsFavorite: 1, ModelScore: 95,
}

var clientTemplateListResponseExample = []apiservice.ClientTemplate{clientTemplateResponseExample}

var templateCategoryListResponseExample = []apiservice.ClientTemplateType{
	{
		ID: 1, CategoryName: "热门模板", Icon: "https://cdn.example.com/template-types/hot.png", Description: "当前客户端可用的热门模板", Sort: 100,
		Templates: clientTemplateListResponseExample,
	},
}

var templateCategoriesResponseExample = clientTemplateTypePageResponse{
	Page: 1, PageSize: 5, Total: 1, TotalPages: 1, List: templateCategoryListResponseExample,
}

var templateListPageResponseExample = clientTemplatePageResponse{
	Page: 1, PageSize: 10, Total: 1, TotalPages: 1, List: clientTemplateListResponseExample,
}

var responseDataExamples = map[string]any{
	"GET /api/ob_delay":                delayConfigResponseExample,
	"GET /api/users/points":            clientPointsListResponseExample,
	"GET /api/templates/categories":    templateCategoriesResponseExample,
	"GET /api/templates/list":          templateCategoriesResponseExample,
	"GET /api/templates/recommend":     clientTemplateListResponseExample,
	"GET /api/templates/template_list": templateListPageResponseExample,
	"GET /api/templates/template_info": clientTemplateResponseExample,
	"GET /api/generation/models":       generationModelResponseExample,
	"GET /api/generation/tasks":        generationTaskListResponseExample,
	"GET /api/generation/tasks/:id":    generationTaskResponseExample,
	"POST /api/tracking/events": trackingEventDocResponse{
		ID: 12345, TrackingType: "Payment_Create", ExtensionType: "OB_back", ModelID: 123,
		CreatedAt: time.Date(2026, 8, 20, 10, 11, 12, 0, time.FixedZone("UTC+8", 8*60*60)),
	},
	"POST /api/orders": commerce.CreatePaymentOrderResponse{
		OrderNo: "20260820143000a1b2c3d4e5f6", ClientRequestID: "points-order-20260820-0001",
		ShopType: 2, ProductID: 8, ProductCode: "credits_100", ProductName: "100 Credits",
		PayType: 1, Status: 1, Currency: "USD", PayableAmount: 1.99,
		ExpiresAt: time.Date(2026, 8, 20, 14, 37, 0, 0, time.FixedZone("UTC+8", 8*60*60)),
		PaymentInfo: commerce.StorePaymentInfo{
			PayType: 1, ProductID: "credits_100", ProductType: "inapp",
			BundleID: "com.example.ios", Quantity: 1, ConfirmPath: "/api/payments/apple/pay",
		},
	},
	"POST /api/payments/apple/pay": commerce.ApplePurchaseResponse{
		OrderNo: "20260728090907cc7d7c1ffd15", Status: 4, ProductType: 1,
		ProductID: 1, ProductCode: "dolaai18", TransactionID: "2000001209105682",
		OriginalTransactionID: "2000001209105682", Currency: "USD", PaidAmount: 19.99,
		PurchaseDate:   time.Date(2026, 7, 22, 16, 42, 39, 0, time.FixedZone("UTC+8", 8*60*60)),
		ExpirationDate: &applePurchaseResponseExampleExpiration, IsActive: false,
		Environment: "Sandbox", EvidenceMode: "jws",
	},
	"POST /api/uploads/oss/signature": upload.DirectUploadCredential{
		UploadID: "0123456789abcdef0123456789abcdef", Provider: upload.StorageAliyunOSS, Method: "PUT",
		UploadURL: "https://example-bucket.oss-cn-hangzhou.aliyuncs.com/uploads/images/2026/07/28/example.png?x-oss-signature-version=OSS4-HMAC-SHA256&x-oss-signature=...",
		Headers: map[string]string{
			"Content-Length": "12345", "Content-Type": "image/png", "X-Oss-Forbid-Overwrite": "true",
		},
		ObjectKey:  "uploads/images/2026/07/28/example.png",
		FileURL:    "/uploads/images/2026/07/28/example.png",
		PreviewURL: "https://test-cdn.zdrawai.com/uploads/images/2026/07/28/example.png",
		ExpiresAt:  time.Date(2026, 7, 28, 12, 10, 0, 0, time.FixedZone("UTC+8", 8*60*60)),
	},
	"GET /api/uploads": uploadListResponse{
		List: []model.VideoUpload{{
			UploadID: "0123456789abcdef0123456789abcdef", MediaType: "image", FileType: "png",
			StorageProvider: upload.StorageAliyunOSS, FilePath: "uploads/images/2026/08/05/example.png",
			FileURL: "/uploads/images/2026/08/05/example.png", Status: 1,
		}},
		Total: 1, Page: 1, Size: 10,
	},
	"GET /api/banners/list": []apiservice.ClientBanner{
		{
			ID: 12, Name: "首页夏日活动", PositionKey: "home_banner", Status: 1,
			JumpType: 2, CoverImage: "https://cdn.example.com/banners/summer.jpg",
			Route: "/templates/42", TemplateID: &bannerResponseExampleTemplateID, Sort: 10,
			TargetTemplate: &apiservice.ClientBannerTemplate{
				ID: 42, Name: "夏日视频模板", TemplateType: 1,
				CoverImageURL: "https://cdn.example.com/templates/42.jpg",
				OriginalURL:   "https://cdn.example.com/templates/42.mp4",
				ThumbnailURL:  "https://cdn.example.com/templates/42-thumb.mp4", Status: 1,
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
	"GET /api/points/list": []apiservice.ClientPointProductResponse{
		{
			ID: 8, ProductCode: "credits_100", Name: "100 Credits",
			ResourceType: "credits",
			Points:       100, Currency: "USD", SalePrice: 1.99, OriginalPrice: 2.99,
			Icon:        "限时优惠",
			Description: "Add 100 credits", ButtonText: "Buy", IsDefault: true,
			Status: 1, Sort: 10, CreatedAt: 1784859371, UpdatedAt: 1784859371,
		},
	},
	"GET /api/tools/list": []apiservice.ClientTool{
		{
			ID: 16, Name: "照片转视频", Icon: "/uploads/images/tools/photo-video-icon.png",
			BackgroundImage: "/uploads/images/tools/photo-video-background.png",
			ToolType:        2, ModelID: 9, ConfigType: 3,
			ConfigData: json.RawMessage(`{"ratio_options":[{"name":"16:9","value":"16:9","sort":1}]}`),
			BadgeImage: "/uploads/images/tools/new.png", Sort: 10,
			Prompt: "让照片自然动起来", Status: 1,
		},
	},
}

var endpointTypes = map[string]endpointType{
	"GET /api/health":                                  {response: typeOf[map[string]string]()},
	"GET /api/configs/list":                            {response: typeOf[map[string]string]()},
	"GET /api/attributions/adjust/callback":            {query: typeOf[adjustAttributionCallbackRequest](), response: typeOf[apiservice.AdjustCallbackResult]()},
	"POST /api/attributions/adjust/callback":           {body: typeOf[adjustAttributionCallbackRequest](), response: typeOf[apiservice.AdjustCallbackResult]()},
	"POST /api/attributions/adjust/report":             {body: typeOf[apiservice.AdjustAppReportRequest](), response: typeOf[apiservice.AdjustAppReportResult]()},
	"POST /api/auth/login":                             {body: typeOf[loginDocRequest](), response: typeOf[apiservice.AuthResponse]()},
	"POST /api/auth/apple_order_login":                 {body: typeOf[apiservice.AppleOrderLoginRequest](), response: typeOf[apiservice.AuthResponse]()},
	"POST /api/auth/refresh":                           {response: typeOf[apiservice.AuthResponse]()},
	"POST /api/third_binding":                          {body: typeOf[thirdPartyLoginDocRequest](), response: typeOf[apiservice.AuthResponse]()},
	"POST /api/auth/logout":                            {},
	"GET /api/users/me":                                {response: typeOf[apiservice.UserResponse]()},
	"PUT /api/users/me/country":                        {body: typeOf[apiservice.UpdateCountryRequest](), response: typeOf[apiservice.UserResponse]()},
	"GET /api/users/points":                            {query: typeOf[apiservice.ClientPointsRequest](), response: typeOf[clientPointsListResponse]()},
	"POST /api/users/active_reporting":                 {body: typeOf[apiservice.ActiveReportingRequest]()},
	"GET /api/ob_delay":                                {response: typeOf[map[string]int64]()},
	"GET /api/banners/list":                            {query: typeOf[apiservice.ClientBannerRequest](), response: typeOf[[]apiservice.ClientBanner]()},
	"GET /api/templates/recommend":                     {query: typeOf[apiservice.ClientTemplateRecommendRequest](), response: typeOf[[]apiservice.ClientTemplate]()},
	"GET /api/templates/list":                          {query: typeOf[apiservice.ClientTemplateRequest](), response: typeOf[clientTemplateTypePageResponse]()},
	"GET /api/templates/categories":                    {query: typeOf[templateCategoriesQuery](), response: typeOf[clientTemplateTypePageResponse]()},
	"GET /api/templates/template_list":                 {query: typeOf[apiservice.TemplateListRequest](), response: typeOf[clientTemplatePageResponse]()},
	"GET /api/templates/template_info":                 {query: typeOf[apiservice.TemplateInfoRequest](), response: typeOf[apiservice.ClientTemplate]()},
	"POST /api/templates/:id/favorite":                 {response: typeOf[apiservice.TemplateFavoriteResponse]()},
	"DELETE /api/templates/:id/favorite":               {response: typeOf[apiservice.TemplateFavoriteResponse]()},
	"POST /api/templates/complaint":                    {body: typeOf[apiservice.ClientCategoriesRequest]()},
	"GET /api/generation/models":                       {query: typeOf[apiservice.GenerationModelRequest](), response: typeOf[[]apiservice.GenerationModelView]()},
	"POST /api/generation/tasks":                       {body: typeOf[generation.CreateTaskRequest](), response: typeOf[generation.TaskView]()},
	"POST /api/generation/template-tasks":              {body: typeOf[templateTaskRequest](), response: typeOf[generation.TaskView]()},
	"GET /api/generation/tasks":                        {query: typeOf[apiservice.GenerationListRequest](), response: typeOf[generationTaskListResponse]()},
	"GET /api/generation/tasks/:id":                    {response: typeOf[generation.TaskView]()},
	"DELETE /api/generation/tasks/:id":                 {},
	"GET /api/vip/recommend":                           {query: typeOf[apiservice.VipRecommendRequest](), response: typeOf[apiservice.VIPRecommendResponse]()},
	"GET /api/vip/list":                                {query: typeOf[apiservice.VipVipListRequest](), response: typeOf[[]apiservice.VIPRecommendResponse]()},
	"GET /api/points/list":                             {response: typeOf[[]apiservice.ClientPointProductResponse]()},
	"GET /api/tools/list":                              {response: typeOf[[]apiservice.ClientTool]()},
	"POST /api/tracking/events":                        {body: typeOf[trackingEventDocRequest](), response: typeOf[trackingEventDocResponse]()},
	"POST /api/orders":                                 {body: typeOf[commerce.CreatePaymentOrderRequest](), response: typeOf[commerce.CreatePaymentOrderResponse]()},
	"POST /api/payments/apple/pay":                     {body: typeOf[commerce.ApplePurchaseRequest](), response: typeOf[commerce.ApplePurchaseResponse]()},
	"POST /api/payments/apple/notification":            {body: typeOf[commerce.AppleNotificationV2Request](), response: typeOf[commerce.AppleNotificationV2Summary]()},
	"POST /api/apy":                                    {body: typeOf[commerce.AppleNotificationV2Request](), response: typeOf[commerce.AppleNotificationV2Summary]()},
	"POST /api/uploads/images/batches":                 {response: typeOf[uploadBatchResponse]()},
	"POST /api/uploads/videos/batches":                 {response: typeOf[uploadBatchResponse]()},
	"GET /api/uploads/images/:upload_id":               {response: typeOf[upload.Session]()},
	"GET /api/uploads/videos/:upload_id":               {response: typeOf[upload.Session]()},
	"PUT /api/uploads/images/:upload_id/chunks/:index": {response: typeOf[upload.Session]()},
	"PUT /api/uploads/videos/:upload_id/chunks/:index": {response: typeOf[upload.Session]()},
	"POST /api/uploads/images/:upload_id/complete":     {response: typeOf[upload.Session]()},
	"POST /api/uploads/videos/:upload_id/complete":     {response: typeOf[upload.Session]()},
	"POST /api/uploads/oss/signature":                  {body: typeOf[upload.DirectUploadRequest](), response: typeOf[upload.DirectUploadCredential]()},
	"GET /api/uploads":                                 {query: typeOf[uploadListQuery](), response: typeOf[uploadListResponse]()},
}

var operationDescriptions = map[string]string{
	"GET /api/health":       "检查 API 服务是否正常运行。",
	"GET /api/configs/list": "获取客户端可见的公开应用配置。", "POST /api/auth/login": "使用设备标识登录或创建游客账号。",
	"POST /api/third_binding": "为当前用户绑定或切换 Google、Apple 等第三方身份。", "POST /api/auth/apple_order_login": "按 order_code 数组查询 original_transaction_id 命中的最新 VIP 订阅订单及其关联用户，并为正常状态的关联用户签发客户端 Token；force_new 用于确认切换到其他未绑定第三方身份的订单账号。", "POST /api/auth/refresh": "使用当前未过期的 Bearer Token 签发新 Token，刷新成功后当前 Token 立即失效。", "POST /api/auth/logout": "注销当前 Bearer Token。",
	"GET /api/users/me": "获取当前登录用户资料。", "PUT /api/users/me/country": "更新当前用户的设备国家或地区。",
	"GET /api/users/points":            "分页查询当前用户的积分变动明细。可按收入或支出方向筛选；start_time 和 end_time 必须同时提供才会应用时间范围筛选。",
	"POST /api/users/active_reporting": "上报当前用户本次活跃时长。time_long 必须是大于 0 的整数；成功时响应 data 为 null。",
	"GET /api/ob_delay":                "获取客户端延迟配置。",
	"GET /api/banners/list":            "按必填的 position_key 查询当前客户端可见的 Banner。服务端同时使用公共请求头中的 Video_App_Code、Video_App_Package_Code、Video_App_Version、Video_Device_Country，以及登录用户的会员状态进行投放匹配。某个维度没有关联记录时表示该维度支持全部；存在关联记录时必须命中。展示位置、国家、应用、应用包、版本和会员类型之间按 AND 关系组合。",
	"GET /api/templates/recommend":     "按必填的 position_key 查询当前客户端可见的推荐模板，响应 data 为模板对象数组。每个模板对象返回 id、template_type_id、name、template_type、icon、cover_image_url、original_url、thumbnail_url、prompt、description、sort、usage_count、favorite_count、view_count、is_favorite 和 model_score。",
	"GET /api/templates/list":          "按可选的 position_key 分页查询当前客户端可见的分类及其模板。page 默认 1，page_size 默认 5；响应 data 包含 page、pageSize、total、totalPages 和 list，每个分类的 templates 使用当前统一模板对象结构。",
	"GET /api/templates/categories":    "分页查询 homeCategory 展示位置下当前客户端可见的模板分类及其模板。page 默认 1，page_size 默认 5；响应 data 包含 page、pageSize、total、totalPages 和 list，每个分类最多返回 10 个启用且未删除的模板。分类必须处于启用状态并至少关联一个启用且未删除的模板；没有可用模板、仅有关联禁用模板或已删除模板的分类不会进入分页结果。分类同时按国家、应用、应用包和版本投放范围匹配，未配置某个范围表示该维度支持全部。",
	"GET /api/templates/template_list": "按 page、page_size、template_type_id 和可选 position_key 分页查询模板。page 默认 1，page_size 默认 10；响应 data 包含 page、pageSize、total、totalPages 和 list，list 中每项使用当前统一模板对象结构。",
	"GET /api/templates/template_info": "根据必填的 template_id 查询单个模板对象，并设置当前登录用户的 is_favorite。响应使用当前统一模板对象结构。",
	"POST /api/templates/:id/favorite": "收藏指定模板；重复收藏保持幂等。", "DELETE /api/templates/:id/favorite": "取消收藏指定模板；重复取消保持幂等。",
	"POST /api/templates/complaint": "提交模板投诉。请求体使用 application/json，template_id 和 complaint_type 必填，content 为可选的补充说明。",
	"GET /api/generation/models":    "按必填 model_type 查询平台和模型均启用的模型及其参数；返回全部 parameter_type=2 的请求参数和 is_display=1 的 parameter_type=1 选项参数，并按 parameter_type、sort_order、id 排序。", "POST /api/generation/tasks": "校验请求并创建待异步处理的生成任务，返回任务订单号。input.images、input.video、input.first_frame 和 input.end_frame 中的媒体地址同时支持半链接与 HTTP(S) 全链接。",
	"POST /api/generation/template-tasks": "按模板创建生成任务。请求体仅展示必填的 template_id 和 input.images；图片地址同时支持半链接与 HTTP(S) 全链接，模板提示词和模型设置由服务端补充。",
	"GET /api/generation/tasks":           "分页查询当前用户的生成任务，列表项返回完整任务快照。", "GET /api/generation/tasks/:id": "查询指定生成任务详情，返回结构与列表中的单个任务一致。",
	"DELETE /api/generation/tasks/:id":      "删除指定生成任务。",
	"GET /api/vip/recommend":                "查询当前用户适用的推荐 VIP 套餐。",
	"GET /api/vip/list":                     "按必填的 vip_types 查询当前应用、包、版本及登录用户状态下可展示的 VIP 套餐列表，仅返回 status=1、display_mode=1 的套餐。",
	"GET /api/points/list":                  "查询当前客户端可购买的积分商品。服务端按登录用户类型以及公共请求上下文中的国家、应用、安装包、APP 版本、系统和渠道进行 AND 筛选，仅返回 status=1 的商品。应用、安装包、版本、国家和渠道未配置关联时表示支持全部。系统从 Video_System_Type 读取，未提供时根据 User-Agent 推断 iOS 或 Android。结果按 is_default DESC、sort ASC、id DESC 排序。",
	"GET /api/tools/list":                   "查询全部正常展示的客户端工具。仅返回 status=1 且未删除的工具，不分页；结果按 sort ASC、id ASC 排序。",
	"POST /api/tracking/events":             "上报单个客户端埋点事件。tracking_type 仅支持当前九个事件名且大小写敏感；Payment_Create、Payment_Suc 和 Case_create 必须提供 extension_type。model_id 可选，不适用时可省略。每次成功请求都会新增一条记录，不去重、不覆盖。",
	"POST /api/orders":                      "为当前登录用户创建待支付订单。shop_type=1 表示 VIP 订阅，shop_type=2 表示积分商品；pay_type=1 表示 Apple IAP，pay_type=2 表示 Google Play。创建积分商品订单时，服务端会再次按用户类型、国家、应用、安装包、APP 版本、系统和渠道校验商品投放范围，并返回原生商店支付参数；积分商品的 payment_info.product_type 固定为 inapp。client_request_id 用于幂等重试，未提供时由服务端生成。",
	"POST /api/payments/apple/pay":          "校验 StoreKit 交易、创建订单并发放对应商品。标准三段式 JWS 直接验签，其他客户端凭证通过 transactionID 调用 App Store Server API 获取已签名交易后验签。接口按 Apple 交易 ID 幂等处理。请求中的 isActive 是客户端上报值；响应中的 is_active 由已验签交易的撤销时间和到期时间按服务端当前时间计算。",
	"POST /api/payments/apple/notification": "接收 App Store Server Notifications V2 回调。该公开端点由 Apple 服务器调用，不需要 Bearer Token 或客户端公共请求头；服务端按通知中的 signedPayload 验签并幂等处理退款、续费、订阅过期等事件。",
	"POST /api/uploads/oss/signature":       "校验媒体类型、文件扩展名、MIME 和精确字节数，生成短时效阿里云 OSS V4 预签名 PUT 地址。客户端必须使用响应中的 method、upload_url 和签名 headers 将文件原始字节直接上传到 OSS；该接口需要 Bearer Token，且仅在当前存储方式为 aliyun_oss 时可用。",
	"GET /api/uploads":                      "分页查询当前登录用户的上传文件记录。status=1 表示未完成，status=2 表示已完成；file_url 返回供业务提交和入库使用的半链接。",
}

var operationSummaries = map[string]string{
	"GET /api/health": "健康检查", "GET /api/configs/list": "获取客户端配置",
	"POST /api/auth/login": "游客登录", "POST /api/auth/apple_order_login": "Apple 订单登录", "POST /api/auth/refresh": "刷新 Token", "POST /api/third_binding": "绑定第三方身份",
	"POST /api/auth/logout": "退出登录", "GET /api/users/me": "获取当前用户",
	"PUT /api/users/me/country": "更新用户国家", "GET /api/users/points": "查询积分明细",
	"POST /api/users/active_reporting": "上报活跃时长", "GET /api/ob_delay": "获取延迟配置",
	"GET /api/banners/list": "查询 Banner", "GET /api/templates/recommend": "查询推荐模板",
	"GET /api/templates/list": "查询模板列表", "GET /api/templates/categories": "查询模板分类",
	"GET /api/templates/template_list": "查询分类模板", "GET /api/templates/template_info": "查询模板详情",
	"POST /api/templates/:id/favorite":   "收藏模板",
	"DELETE /api/templates/:id/favorite": "取消收藏模板", "POST /api/templates/complaint": "投诉模板", "GET /api/vip/recommend": "查询推荐 VIP 套餐",
	"GET /api/vip/list":          "查询 VIP 套餐列表",
	"GET /api/points/list":       "查询积分商品列表",
	"GET /api/tools/list":        "查询工具列表",
	"POST /api/tracking/events":  "上报客户端埋点事件",
	"POST /api/orders":           "创建支付订单",
	"GET /api/generation/models": "查询生成模型", "POST /api/generation/tasks": "创建生成任务", "POST /api/generation/template-tasks": "按模板创建生成任务",
	"GET /api/generation/tasks": "查询生成任务", "GET /api/generation/tasks/:id": "获取生成任务", "DELETE /api/generation/tasks/:id": "删除生成任务",
	"POST /api/payments/apple/pay": "确认 Apple 支付", "POST /api/payments/apple/notification": "接收 Apple 支付通知",
	"POST /api/uploads/images/batches": "初始化图片上传", "POST /api/uploads/videos/batches": "初始化视频上传",
	"GET /api/uploads/images/:upload_id": "查询图片上传进度", "GET /api/uploads/videos/:upload_id": "查询视频上传进度",
	"PUT /api/uploads/images/:upload_id/chunks/:index": "上传图片分片", "PUT /api/uploads/videos/:upload_id/chunks/:index": "上传视频分片",
	"POST /api/uploads/images/:upload_id/complete": "完成图片上传", "POST /api/uploads/videos/:upload_id/complete": "完成视频上传",
	"POST /api/uploads/oss/signature": "获取 OSS 直传签名",
	"GET /api/uploads":                "查询文件上传列表",
}

var fieldDescriptions = map[string]string{
	"imei": "设备唯一标识", "device_code": "设备唯一标识", "force_new": "是否确认执行账号选择或切换；具体语义见接口说明", "id_token": "Google 等提供方签发的 ID Token",
	"identity_token": "Apple 签发的 Identity Token", "nonce": "用于防重放校验的随机值", "display_name": "显示名称",
	"given_name": "名", "family_name": "姓", "device_country": "设备国家或地区代码", "client_country": "客户端国家或地区代码", "channel_id": "渠道标识",
	"app_version": "应用版本号", "app_name": "应用名称", "phone_model": "设备型号", "channel_package": "渠道包标识",
	"app_package": "应用包名", "login_type": "登录类型：1 游客，2 Google，3 Apple", "first_opened_at": "首次打开时间",
	"last_opened_at": "最近打开时间", "attribution_clicked_at": "归因点击时间", "country": "国家或地区代码",
	"position_key": "展示位置唯一标识；Banner 查询时为必填 Query 参数", "package": "应用包名", "package_code": "应用包名", "package_version": "应用版本号",
	"channel": "渠道标识", "user_type": "用户类型：1 免费，2 付费", "subscription_status": "订阅状态：1 未订阅，2 已订阅，3 已取消，4 已过期",
	"token": "Bearer JWT", "expire_at": "Token 过期时间（Unix 秒）", "token_version": "Token 版本号",
	"id": "记录 ID", "user_id": "客户端用户 ID", "email": "邮箱", "vip_expires_at": "VIP 到期时间（Unix 秒）", "points_balance": "积分余额",
	"points_type": "积分变动方向：1=收入，2=支出", "start_time": "筛选开始时间（Unix 秒；需与 end_time 同时提供）", "end_time": "筛选结束时间（Unix 秒；需与 start_time 同时提供）",
	"direction": "积分变动方向：1=收入，2=支出", "points_change": "积分变动量；正数表示增加，负数表示减少", "balance_before": "变动前积分余额", "balance_after": "变动后积分余额",
	"time_long":      "本次上报的用户活跃时长，单位秒，必须大于 0",
	"tracking_type":  "埋点事件类型；仅支持文档列出的九个值，且大小写敏感",
	"extension_type": "扩展来源标识；Payment_Create、Payment_Suc 和 Case_create 必填",
	"model_id":       "模板 ID；不适用时可省略或传 0",
	"status":         "状态", "last_login_at": "最近登录时间", "last_login_ip": "最近登录 IP", "login_account": "登录账号",
	"appid_binding": "是否已绑定 Apple", "google_binding": "是否已绑定 Google", "provider": "提供方标识；身份接口表示身份提供方，OSS 直传接口表示存储提供方",
	"provider_subject": "身份提供方用户唯一标识", "issuer": "Token 签发方", "audience": "Token 受众",
	"email_verified": "邮箱是否已验证", "is_private_email": "是否为隐私邮箱", "avatar_url": "头像地址",
	"key": "配置键", "value": "配置值", "name": "名称", "template_type": "模板类型：1=图片模板，2=视频模板", "cover_image": "封面图片地址",
	"cover_image_url": "模板封面图片地址", "original_url": "模板原始媒体地址", "thumbnail_url": "模板缩略媒体地址", "model_score": "生成模型评分",
	"template_video": "模板视频地址", "thumbnail_video": "缩略视频地址", "jump_type": "跳转类型：1 链接，2 模板，3 文生图，4 文生视频", "route": "客户端最终跳转路由、深链或外部链接",
	"target_template": "模板跳转时返回的目标模板摘要；其他跳转类型不返回", "template_id": "模板 ID", "sort": "排序值", "category_name": "分类名称",
	"description": "说明", "position_keys": "支持的展示位置", "user_types": "适用用户类型", "subscription_statuses": "适用订阅状态",
	"templates": "模板列表", "prompt": "模板提示词", "usage_count": "使用次数",
	"favorite_count": "收藏次数", "favorited": "当前用户是否已收藏", "is_favorite": "当前用户是否已收藏：0=否，1=是", "view_count": "浏览次数", "display_config_id": "展示配置 ID", "display_sort": "展示排序",
	"page": "页码，从 1 开始", "page_size": "每页数量", "pageSize": "每页数量", "total": "总记录数", "totalPages": "总页数", "list": "当前页数据列表", "template_type_id": "模板分类 ID", "third_type": "第三方身份类型：google 或 apple",
	"third_code": "第三方平台用户标识",
	"model_code": "生成模型代码", "model_type": "模型类型：1=生成图片，2=生成视频", "client_request_id": "客户端幂等请求 ID", "task_code": "任务唯一编码", "task_type": "任务类型：1=生成图片，2=生成视频；列表查询值 3 表示全部", "input": "生成任务输入参数", "parameters": "生成参数",
	"images": "参考图片 URL 数组", "video": "参考视频 URL", "first_frame": "首帧图片 URL", "end_frame": "尾帧图片 URL",
	"complaint_type": "投诉类型", "content": "投诉补充内容",
	"parameter": "模型参数列表", "param_key": "参数键名", "parameter_type": "参数类型：1=选项参数，2=请求参数",
	"default_value": "参数默认值", "allowed_values": "兼容保留的参数允许值数组", "allowed_value_options": "选择值配置数组，每项包含成对的 value 和 alias", "constraints": "JSON 字符串格式的参数约束；空约束为 {}", "alias": "客户端展示名称", "display_type": "客户端控件类型", "is_display": "是否展示：1=是，0=否",
	"model_config_id": "生成模型配置 ID", "external_task_id": "第三方任务 ID", "progress": "任务进度，范围 0-100",
	"local_urls": "生成结果的持久化访问地址（本地或 OSS）", "error_message": "任务失败原因", "usage_duration": "任务计费用时（秒）",
	"submitted_at": "任务提交到上游的时间", "started_at": "上游开始处理的时间", "finished_at": "任务结束时间",
	"default_parameters": "模型默认参数", "model_name": "提供方模型名称",
	"shop_type": "商品类型：1=VIP 订阅，2=积分商品", "pay_type": "支付类型：1=Apple IAP，2=Google Play", "bundleID": "Apple Bundle ID，必须与 Video_App_Package_Code 及签名交易一致", "productID": "Apple 商品 ID", "transactionID": "Apple 交易 ID",
	"originalTransactionID": "Apple 原始交易 ID", "signedTransactionInfo": "可选的 Apple 三段式签名交易 JWS；非标准格式将改用 transactionID 调用 App Store Server API 查询",
	"signedPayload": "App Store Server Notifications V2 签名载荷 JWS", "notification_type": "Apple 通知类型",
	"subtype": "Apple 通知子类型", "notification_uuid": "Apple 通知唯一标识", "bundle_id": "Apple Bundle ID",
	"environment": "App Store 环境", "original_transaction_id": "Apple 原始交易 ID", "transaction_id": "Apple 交易 ID",
	"product_id": "服务端商品 ID；创建订单时对应 VIP 套餐或积分商品主键", "processed": "是否已完成对应业务处理", "affected_user_id": "受影响的用户 ID",
	"affected_order_no": "受影响的订单号", "action": "本次通知执行的业务动作", "message": "处理结果说明",
	"purchaseDate": "客户端购买时间，RFC 3339 格式，必须与签名交易一致", "expirationDate": "订阅到期时间，RFC 3339 格式；无到期时间时为 null", "revocationDate": "撤销时间，RFC 3339 格式；未撤销时为 null", "isActive": "客户端上报的订阅状态；服务端不以此值作为最终状态",
	"source": "购买入口来源，可选", "order_no": "服务端订单号", "order_code": "Apple 原始交易 ID 数组；按数组中的编号匹配订单", "product_type": "原生商店商品类型；积分商品为 inapp，VIP 商品为 subscription", "payment_info": "客户端发起原生商店支付所需参数", "quantity": "购买数量", "confirm_path": "客户端支付成功后的服务端确认接口", "package_name": "Google Play 包名", "payable_amount": "订单应付金额",
	"product_code": "商店商品 SKU；Apple 支付场景中对应 Apple 商品 ID", "paid_amount": "实付金额，单位由 currency 指定", "purchase_date": "已验签的购买时间",
	"expiration_date": "已验签的订阅到期时间", "is_active": "服务端计算的当前状态：交易未撤销，且订阅到期时间晚于服务端当前时间",
	"evidence_mode": "交易凭证模式，固定为 jws",
	"vip_type":      "VIP 套餐类型", "vip_types": "VIP 套餐类型数组；使用重复 Query 参数传递，例如 vip_types=1&vip_types=2",
	"suk_code": "商店产品 SKU", "level_name": "会员等级名称", "currency": "ISO 货币代码",
	"vip_duration_days": "VIP 权益持续天数", "trial_days": "免费试用天数", "badge_text": "VIP 套餐徽章文案",
	"icon":                      "图标图片地址；半链接由展示端自动拼接 CDN 域名",
	"systems":                   "适用客户端系统数组",
	"resource_type":             "积分商品发放的资源类型",
	"points":                    "购买商品后发放的积分数量",
	"sale_price":                "商品销售价格",
	"button_text":               "购买按钮文案",
	"agreement_default_checked": "订阅协议是否默认勾选", "display_mode": "展示模式：0 隐藏，1 正常",
	"free_trial": "是否启用免费试用", "is_subscription": "是否循环订阅", "is_default": "是否为默认套餐",
	"subscription_description": "订阅说明", "subscription_price": "当前用户适用的订阅价格",
	"original_price": "原价", "subscription_points": "订阅赠送积分", "subscription_period": "订阅周期：1 周，2 月，3 季，4 年",
	"background_image": "工具背景图片地址", "tool_type": "工具类型：1=图片生成，2=视频生成", "config_type": "工具配置类型：0=无，1=参考图，2=年龄，3=比例",
	"config_data": "工具配置内容，结构由 config_type 决定", "badge_image": "工具角标图片地址",
	"files": "待上传文件列表", "file_name": "文件名", "size": "文件字节数", "content_type": "MIME 类型", "sha256": "文件 SHA-256",
	"media_type": "上传媒体类型：image 或 video", "method": "OSS 直传 HTTP 方法，固定为 PUT", "upload_url": "短时效 OSS V4 预签名上传地址",
	"headers": "直传 OSS 时必须携带的签名请求头", "object_key": "服务端生成的 OSS 对象键",
	"uploads": "上传会话列表", "upload_id": "上传会话 ID", "kind": "媒体类型：image 或 video", "original_name": "原始文件名",
	"extension": "文件扩展名", "total_size": "文件总字节数", "chunk_size": "分片字节数", "total_chunks": "分片总数",
	"uploaded_chunks": "已上传分片序号", "expected_sha256": "预期文件 SHA-256", "uploader_type": "上传者类型",
	"uploader_id": "上传者 ID", "storage_provider": "存储提供方", "completed": "是否上传完成", "file_path": "存储路径",
	"file_url": "文件访问地址", "created_at": "创建时间", "updated_at": "更新时间", "expires_at": "过期时间",
}

var resourceNames = map[string]string{
	"health": "健康检查", "configs": "系统配置", "auth": "认证", "users": "用户",
	"banners": "Banner", "templates": "视频模板", "tools": "工具", "generation": "内容生成", "payments": "支付", "vip": "VIP", "points": "积分商品",
	"orders": "支付订单", "uploads": "文件上传", "tracking": "数据埋点", "profile": "个人资料",
}

var publicRoutes = map[string]bool{
	"GET /api/health": true, "POST /api/auth/login": true,
	"POST /api/payments/apple/notification": true,
}

var paginatedRoutes = map[string]bool{
	"/api/generation/tasks": true,
}

var sseRoutes = map[string]bool{}

var operationIDSanitizer = regexp.MustCompile(`[^A-Za-z0-9]+`)

func init() {
	for _, key := range []string{
		"GET /api/attributions/adjust/callback",
		"POST /api/attributions/adjust/callback",
	} {
		publicRoutes[key] = true
		operationSummaries[key] = "接收 Adjust 归因回调"
	}
	operationDescriptions["GET /api/attributions/adjust/callback"] =
		"接收 Adjust 服务端归因回调。使用专用 callback_token 鉴权，按无密钥的标准化载荷幂等落库，并在用户可匹配时更新当前归因汇总。"
	operationDescriptions["POST /api/attributions/adjust/callback"] =
		"接收 Adjust 服务端归因回调。支持 JSON 或表单参数；处理和幂等规则与 GET 回调一致。"
	operationSummaries["POST /api/attributions/adjust/report"] = "上报 APP Adjust 归因数据"
	operationDescriptions["POST /api/attributions/adjust/report"] =
		"接收 Adjust SDK 的客户端归因快照。接口必须使用客户端 Bearer JWT，用户 ID 只从鉴权上下文获取；服务端按 ADID 与 Adjust 回调融合。"
	resourceNames["attributions"] = "归因"
	fieldDescriptions["callback_token"] = "Adjust 回调专用密钥；仅用于鉴权，不会持久化"
	fieldDescriptions["adid"] = "Adjust 设备 ID"
	fieldDescriptions["tracker_token"] = "Adjust tracker token"
	fieldDescriptions["tracker_name"] = "Adjust tracker 名称"
	fieldDescriptions["trackerToken"] = "Adjust SDK tracker token"
	fieldDescriptions["trackerName"] = "Adjust SDK tracker 名称"
	fieldDescriptions["costAmount"] = "Adjust SDK 原始成本值；可能为字符串 NaN"
	fieldDescriptions["googleAdId"] = "Google Advertising ID"
	fieldDescriptions["costAmount"] = "Adjust SDK 成本值；客户端应发送 number 或 null，兼容历史字符串 NaN"
	fieldDescriptions["app_token"] = "Adjust 应用 token"
	fieldDescriptions["outdated_tracker"] = "归因更新前的 tracker token"
	fieldDescriptions["outdated_tracker_name"] = "归因更新前的 tracker 名称"
	fieldDescriptions["installed_at"] = "Adjust 安装时间"
	fieldDescriptions["reattributed_at"] = "Adjust 再归因时间"
	fieldDescriptions["attribution_updated_at"] = "Adjust 归因更新时间"
	fieldDescriptions["is_redownload"] = "是否为重新下载/安装"
	operationDescriptions["POST /api/attributions/adjust/report"] =
		"接收 Adjust SDK 当前归因快照；SDK 归因变化时应重复幂等上报。用户 ID 只从 Bearer JWT 获取，并按 ADID 与服务端 callback 融合。"
	operationDescriptions["GET /api/attributions/adjust/callback"] =
		"接收 Adjust callback，按事件类型区分首次安装、安装更新、再归因、审计忽略和 GDPR 删除；首次获客不会被再归因覆盖。"
	operationDescriptions["POST /api/attributions/adjust/callback"] =
		operationDescriptions["GET /api/attributions/adjust/callback"]
}

func typeOf[T any]() reflect.Type { return reflect.TypeOf((*T)(nil)).Elem() }

// Build generates OpenAPI from the routes actually registered in Gin. Route
// coverage therefore stays current even when a handler has no explicit schema mapping.
func Build(routes []gin.RouteInfo) Document {
	document := Document{
		OpenAPI: "3.0.3",
		Info: map[string]any{
			"title": "AI Video API", "version": "1.0.0",
			"description": "根据当前 Gin 路由自动生成，仅包含 /api 客户端接口。接口统一返回 {code, message, data}；错误 message 为脱敏后的英文提示。",
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
	if key == "POST /api/uploads/oss/signature" {
		operation["responses"].(map[string]any)["413"] = errorResponse("文件超过当前媒体类型的上传大小上限")
		operation["responses"].(map[string]any)["503"] = errorResponse("当前未启用阿里云 OSS，或 OSS 直传配置不可用")
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
	media := map[string]any{"schema": schema}
	if example, exists := requestBodyExamples[route.Method+" "+route.Path]; exists {
		media["example"] = example
	}
	return map[string]any{"required": true, "content": map[string]any{
		contentType: media,
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
	if indirectType(valueType) == typeOf[apiservice.AdjustAppReportRequest]() {
		properties["costAmount"] = map[string]any{
			"nullable": true,
			"oneOf": []any{
				map[string]any{"type": "number", "format": "double"},
				map[string]any{"type": "string", "maxLength": 64},
			},
			"description": fieldDescriptions["costAmount"],
		}
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
	if valueType == reflect.TypeOf(json.RawMessage{}) {
		return map[string]any{"type": "object", "additionalProperties": true, "nullable": nullable}
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
		{"Video_System_Type", "客户端系统类型；支持 1 或 ios、2 或 android，未提供时根据 User-Agent 推断", false},
		{"Video_Device_Country", "ISO 3166-1 alpha-2 国家或地区代码；用于 Banner 等内容的国家范围匹配和语言配置，未传时根据客户端 IP 或用户资料推断", false},
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
	return responseSchemaForTypeWithTrail(valueType, make(map[reflect.Type]bool))
}

func responseSchemaForTypeWithTrail(valueType reflect.Type, trail map[reflect.Type]bool) map[string]any {
	nullable := false
	for valueType.Kind() == reflect.Pointer {
		nullable = true
		valueType = valueType.Elem()
	}
	if valueType == reflect.TypeOf(time.Time{}) {
		return map[string]any{"type": "string", "format": "date-time", "nullable": nullable}
	}
	if valueType.Kind() == reflect.Struct {
		if trail[valueType] {
			return map[string]any{"type": "object", "nullable": nullable}
		}
		trail[valueType] = true
		defer delete(trail, valueType)
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
			fieldSchema := responseSchemaForTypeWithTrail(field.Type, trail)
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
		schema = map[string]any{"type": "array", "items": responseSchemaForTypeWithTrail(valueType.Elem(), trail)}
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
