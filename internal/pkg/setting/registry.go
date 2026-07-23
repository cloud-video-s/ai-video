package setting

import (
	"ai-video/internal/gen/model"
	"ai-video/internal/repository"
	"context"
)

const (
	UserSingleDeviceLoginKey = "user.single_device_login"
	APPNameKey               = "app.name"
	APPAboutKey              = "app.about"
	APPServicePhoneKey       = "app.customer_service_phone"
	APPServiceEmailKey       = "app.customer_service_email"
	APPWebsiteKey            = "app.website"
	APPThemeColorKey         = "app.theme_color"
	APPThemeModeKey          = "app.theme_mode"
	APPLanguageKey           = "app.language"
)

// definition is a known config: its default value, type and metadata. The
// registry is the single source for seeding the DB and for the fallback default
// when both cache and DB miss.
type definition struct {
	Group     string
	Key       string
	Name      string
	Type      string // string | int | bool | float | text | json | select | password
	Value     string
	Options   string
	Remark    string
	IsPublic  int8
	Sensitive int8
	Sort      int
}

// registry lists the built-in configs. Add new tunables here; they are seeded on
// next startup (idempotently) and editable from the admin UI afterwards.
var registry = []definition{
	{Group: "站点", Key: "site.name", Name: "站点名称", Type: "string", Value: "Frame Admin", IsPublic: 1, Remark: "登录页 / 浏览器标题"},
	{Group: "站点", Key: "site.logo", Name: "站点 Logo", Type: "string", Value: "", IsPublic: 1, Remark: "Logo 图片 URL"},
	{Group: "站点", Key: "site.description", Name: "站点描述", Type: "text", Value: "后台管理系统", IsPublic: 1},
	{Group: "站点", Key: "site.copyright", Name: "版权信息", Type: "string", Value: "", IsPublic: 1},
	{Group: "站点", Key: "site.icp", Name: "ICP 备案号", Type: "string", Value: "", IsPublic: 1},

	{Group: "APP 基础信息", Key: APPNameKey, Name: "应用名称", Type: "string", Value: "AI Video", IsPublic: 1, Remark: "APP 对外展示的应用名称", Sort: 10},
	{Group: "APP 基础信息", Key: APPAboutKey, Name: "关于我们", Type: "text", Value: "", IsPublic: 1, Remark: "关于我们页面展示内容", Sort: 20},
	{Group: "APP 基础信息", Key: APPServicePhoneKey, Name: "客服电话", Type: "string", Value: "", IsPublic: 1, Remark: "用户可拨打的客服号码", Sort: 30},
	{Group: "APP 基础信息", Key: APPServiceEmailKey, Name: "客服邮箱", Type: "string", Value: "", IsPublic: 1, Remark: "用户联系邮箱", Sort: 40},
	{Group: "APP 基础信息", Key: APPWebsiteKey, Name: "官方网站", Type: "string", Value: "", IsPublic: 1, Remark: "必须填写 http:// 或 https:// 地址", Sort: 50},
	{Group: "APP 基础信息", Key: APPThemeColorKey, Name: "主题皮肤颜色", Type: "color", Value: "#409EFF", IsPublic: 1, Remark: "APP 主色，格式为 #RRGGBB", Sort: 60},
	{Group: "APP 基础信息", Key: APPThemeModeKey, Name: "皮肤模式", Type: "select", Value: "system", Options: `[{"label":"跟随系统","value":"system"},{"label":"浅色","value":"light"},{"label":"深色","value":"dark"}]`, IsPublic: 1, Sort: 70},
	{Group: "APP 基础信息", Key: APPLanguageKey, Name: "默认语言", Type: "select", Value: "zh-CN", Options: `[{"label":"简体中文","value":"zh-CN"},{"label":"English","value":"en-US"},{"label":"日本語","value":"ja-JP"},{"label":"한국어","value":"ko-KR"}]`, IsPublic: 1, Remark: "APP 首次启动时的默认语言", Sort: 80},

	{Group: "用户", Key: "user.allow_register", Name: "允许注册", Type: "bool", Value: "false", Remark: "是否开放自助注册"},
	{Group: "用户", Key: "user.default_role", Name: "默认角色编码", Type: "string", Value: "", Remark: "注册用户默认角色"},
	{Group: "用户", Key: UserSingleDeviceLoginKey, Name: "单设备登录", Type: "bool", Value: "false", Remark: "开启后，用户每次登录都会使其他设备上的旧 Token 立即失效", Sort: 10},

	{Group: "安全", Key: "security.login_fail_limit", Name: "登录失败锁定次数", Type: "int", Value: "5"},
	{Group: "安全", Key: "security.login_lock_minutes", Name: "锁定时长(分钟)", Type: "int", Value: "15"},
	{Group: "安全", Key: "security.password_min_length", Name: "密码最小长度", Type: "int", Value: "6"},

	{Group: "日志", Key: "log.operation_retain_days", Name: "操作日志保留天数", Type: "int", Value: "30", Remark: "留存清理任务读取此值，0 表示不清理"},

	{Group: "文件上传", Key: "upload.storage_provider", Name: "存储方式", Type: "select", Value: "local", Options: `[{"label":"本地存储","value":"local"},{"label":"阿里云 OSS","value":"aliyun_oss"}]`, Remark: "修改后对新完成的上传立即生效", Sort: 100},
	{Group: "文件上传", Key: "upload.local_base_url", Name: "本地文件访问前缀", Type: "string", Value: "/uploads", Remark: "本地文件的公开 URL 前缀", Sort: 101},
	{Group: "文件上传", Key: "upload.image_extensions", Name: "图片允许格式", Type: "string", Value: ".jpg,.jpeg,.png,.gif,.webp", Options: `[{"label":"JPG","value":".jpg"},{"label":"JPEG","value":".jpeg"},{"label":"PNG","value":".png"},{"label":"GIF","value":".gif"},{"label":"WebP","value":".webp"}]`, Remark: "至少选择一种；保存后新上传立即生效", Sort: 102},
	{Group: "文件上传", Key: "upload.image_max_file_size", Name: "单张图片大小", Type: "int", Value: "20971520", Remark: "单个图片文件上限，界面单位 MB", Sort: 103},
	{Group: "文件上传", Key: "upload.video_extensions", Name: "视频允许格式", Type: "string", Value: ".mp4,.mov,.webm,.mkv", Options: `[{"label":"MP4","value":".mp4"},{"label":"MOV","value":".mov"},{"label":"WebM","value":".webm"},{"label":"MKV","value":".mkv"}]`, Remark: "至少选择一种；保存后新上传立即生效", Sort: 104},
	{Group: "文件上传", Key: "upload.video_max_file_size", Name: "单个视频大小", Type: "int", Value: "2147483648", Remark: "单个视频文件上限，界面单位 MB", Sort: 105},
	{Group: "文件上传", Key: "upload.oss.endpoint", Name: "OSS Endpoint", Type: "string", Value: "", Remark: "例如 oss-cn-hangzhou.aliyuncs.com", Sort: 110},
	{Group: "文件上传", Key: "upload.oss.access_key_id", Name: "OSS AccessKey ID", Type: "password", Value: "", Sensitive: 1, Sort: 111},
	{Group: "文件上传", Key: "upload.oss.access_key_secret", Name: "OSS AccessKey Secret", Type: "password", Value: "", Sensitive: 1, Sort: 112},
	{Group: "文件上传", Key: "upload.oss.bucket", Name: "OSS Bucket", Type: "string", Value: "", Sort: 113},
	{Group: "文件上传", Key: "upload.oss.object_prefix", Name: "OSS 对象前缀", Type: "string", Value: "uploads", Remark: "不需要前后斜杠", Sort: 114},
	{Group: "文件上传", Key: "upload.oss.base_url", Name: "OSS 访问域名", Type: "string", Value: "", Remark: "可选，填写 CDN 或自定义域名", Sort: 115},
}

// defaultValue returns the compiled-in default for key, or "" if unknown.
func defaultValue(key string) string {
	for i := range registry {
		if registry[i].Key == key {
			return registry[i].Value
		}
	}
	return ""
}

func seedDefaults(ctx context.Context) error {
	for _, d := range registry {
		exists, err := repo.Exists(ctx, &repository.QueryOptions{
			Where: map[string]interface{}{"key": d.Key},
		})
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		if err := repo.Create(ctx, &model.VideoConfig{
			Group: d.Group, Key: d.Key, Name: d.Name, Type: d.Type, Value: d.Value,
			Options: d.Options, Remark: d.Remark, IsPublic: d.IsPublic,
			Sensitive: d.Sensitive, Sort: int64(d.Sort), Editable: 1, Builtin: 1,
		}); err != nil {
			return err
		}
	}
	return nil
}
