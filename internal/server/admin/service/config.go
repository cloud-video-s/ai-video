package service

import (
	"ai-video/internal/gen/model"
	"ai-video/internal/pkg/setting"
	"ai-video/internal/pkg/upload"
	"ai-video/internal/repository"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

type ConfigService struct {
	repo *repository.ConfigRepo
}

func NewConfigService() *ConfigService {
	return &ConfigService{repo: repository.NewConfigRepo()}
}

var allowedConfigTypes = map[string]bool{
	"string": true, "int": true, "float": true,
	"bool": true, "text": true, "json": true, "select": true, "password": true, "color": true,
}

const sensitiveValueMask = "******"

// validateConfigDef ensures the type is known and, for select, that options is a
// non-empty JSON array — so bad definitions can't reach the DB/UI.
func validateConfigDef(typ, options string) error {
	if !allowedConfigTypes[typ] {
		return fmt.Errorf("不支持的配置类型: %s", typ)
	}
	if typ == "select" {
		if strings.TrimSpace(options) == "" {
			return errors.New("select 类型必须提供选项(options)")
		}
		var arr []json.RawMessage
		if err := json.Unmarshal([]byte(options), &arr); err != nil {
			return errors.New("options 必须是 JSON 数组")
		}
		if len(arr) == 0 {
			return errors.New("select 选项不能为空")
		}
	}
	return nil
}

const (
	minimumUploadFileSize = int64(1 << 20)
	maximumUploadFileSize = int64(100 << 30)
)

var (
	appPhonePattern = regexp.MustCompile(`^[0-9+() -]{5,32}$`)
	appColorPattern = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
)

func validateConfigValue(key, value string) error {
	value = strings.TrimSpace(value)
	switch key {
	case setting.APPNameKey:
		if value == "" || utf8.RuneCountInString(value) > 128 {
			return errors.New("应用名称不能为空且不能超过 128 个字符")
		}
	case setting.APPAboutKey:
		if utf8.RuneCountInString(value) > 5000 {
			return errors.New("关于我们不能超过 5000 个字符")
		}
	case setting.APPServicePhoneKey:
		if value != "" && !appPhonePattern.MatchString(value) {
			return errors.New("客服电话格式无效")
		}
	case setting.APPServiceEmailKey:
		if value != "" {
			address, err := mail.ParseAddress(value)
			if err != nil || !strings.EqualFold(address.Address, value) {
				return errors.New("客服邮箱格式无效")
			}
		}
	case setting.APPWebsiteKey:
		if value != "" {
			parsed, err := url.ParseRequestURI(value)
			if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
				return errors.New("官方网站必须是有效的 http 或 https 地址")
			}
		}
	case setting.APPThemeColorKey:
		if !appColorPattern.MatchString(value) {
			return errors.New("主题皮肤颜色必须使用 #RRGGBB 格式")
		}
	case setting.APPThemeModeKey:
		if value != "system" && value != "light" && value != "dark" {
			return errors.New("皮肤模式仅支持 system、light 或 dark")
		}
	case setting.APPLanguageKey:
		if value != "zh-CN" && value != "en-US" && value != "ja-JP" && value != "ko-KR" {
			return errors.New("默认语言不受支持")
		}
	case "upload.image_extensions", "upload.video_extensions":
		kind := upload.MediaImage
		label := "图片"
		if key == "upload.video_extensions" {
			kind, label = upload.MediaVideo, "视频"
		}
		if _, err := upload.PolicyForExtensions(kind, minimumUploadFileSize, splitConfigExtensions(value)); err != nil {
			return fmt.Errorf("%s上传格式配置无效: %w", label, err)
		}
	case "upload.image_max_file_size", "upload.video_max_file_size":
		size, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if err != nil || size < minimumUploadFileSize || size > maximumUploadFileSize {
			return errors.New("单文件大小必须在 1 MB 到 102400 MB 之间")
		}
	}
	return nil
}

func splitConfigExtensions(value string) []string {
	return strings.FieldsFunc(value, func(r rune) bool {
		switch r {
		case ',', ';', ' ', '\t', '\r', '\n':
			return true
		default:
			return false
		}
	})
}

func (s *ConfigService) List(ctx context.Context, group string) ([]model.VideoConfig, error) {
	q := &repository.QueryOptions{Order: []string{"sort ASC", "id ASC"}}
	if group != "" {
		q.Where = map[string]interface{}{"group": group}
	}
	list, err := s.repo.List(ctx, q)
	if err != nil {
		return nil, err
	}
	for i := range list {
		if list[i].Sensitive == 1 {
			list[i].Value = sensitiveValueMask
		}
	}
	return list, nil
}

type ConfigItem struct {
	Key   string `json:"key" binding:"required"`
	Value string `json:"value"`
}

// BatchUpdate writes many values in one transaction, then re-syncs the cache.
func (s *ConfigService) BatchUpdate(ctx context.Context, items []ConfigItem) error {
	if len(items) == 0 {
		return nil
	}
	for i := range items {
		items[i].Value = normalizeConfigValue(items[i].Key, items[i].Value)
		if err := validateConfigValue(items[i].Key, items[i].Value); err != nil {
			return err
		}
	}
	if err := repository.Transaction(ctx, func(ctx context.Context) error {
		for _, it := range items {
			config, err := s.repo.GetByKey(ctx, it.Key)
			if err != nil {
				return err
			}
			if config.Sensitive == 1 && it.Value == sensitiveValueMask {
				continue
			}
			if err := s.repo.UpdateValue(ctx, it.Key, it.Value); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	return setting.RefreshAll(context.Background())
}

type CreateConfigRequest struct {
	Group    string `json:"group"`
	Key      string `json:"key" binding:"required"`
	Name     string `json:"name" binding:"required"`
	Value    string `json:"value"`
	Type     string `json:"type"`
	Options  string `json:"options"`
	IsPublic bool   `json:"is_public"`
	Remark   string `json:"remark"`
	Sort     int    `json:"sort"`
}

func (s *ConfigService) Create(ctx context.Context, req *CreateConfigRequest) error {
	req.Value = normalizeConfigValue(req.Key, req.Value)
	typ := req.Type
	if typ == "" {
		typ = "string"
	}
	if err := validateConfigDef(typ, req.Options); err != nil {
		return err
	}
	if err := validateConfigValue(req.Key, req.Value); err != nil {
		return err
	}
	exists, err := s.repo.Exists(ctx, &repository.QueryOptions{
		Where: map[string]interface{}{"key": req.Key},
	})
	if err != nil {
		return err
	}
	if exists {
		return errors.New("配置键已存在")
	}
	sensitive := 0
	if typ == "password" {
		sensitive = 1
	}
	c := &model.VideoConfig{
		Group: req.Group, Key: req.Key, Name: req.Name, Value: req.Value,
		Type: typ, Options: req.Options, IsPublic: req.IsPublic,
		Sensitive: int8(sensitive), Remark: req.Remark, Sort: int64(req.Sort), Editable: 1, Builtin: 0,
	}
	if err := s.repo.Create(ctx, c); err != nil {
		return err
	}
	_ = setting.RefreshKey(context.Background(), req.Key)
	return nil
}

type UpdateConfigRequest struct {
	Group    string `json:"group"`
	Name     string `json:"name" binding:"required"`
	Value    string `json:"value"`
	Type     string `json:"type"`
	Options  string `json:"options"`
	IsPublic bool   `json:"is_public"`
	Remark   string `json:"remark"`
	Sort     int    `json:"sort"`
}

// Update edits a config's metadata and value (the key is immutable), then
// refreshes the cache for that key.
func (s *ConfigService) Update(ctx context.Context, id uint, req *UpdateConfigRequest) error {
	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return notFoundOr(err, "配置不存在")
	}
	req.Value = normalizeConfigValue(c.Key, req.Value)
	typ := req.Type
	if typ == "" {
		typ = "string"
	}
	if err := validateConfigDef(typ, req.Options); err != nil {
		return err
	}
	if err := validateConfigValue(c.Key, req.Value); err != nil {
		return err
	}
	c.Group = req.Group
	c.Name = req.Name
	if !(c.Sensitive == 1 && req.Value == sensitiveValueMask) {
		c.Value = req.Value
	}
	c.Type = typ
	c.Options = req.Options
	c.IsPublic = req.IsPublic
	c.Remark = req.Remark
	c.Sort = int64(req.Sort)
	if err := s.repo.Update(ctx, c, "Group", "Name", "Value", "Type", "Options", "IsPublic", "Remark", "Sort"); err != nil {
		return err
	}
	_ = setting.RefreshKey(context.Background(), c.Key)
	return nil
}

func normalizeConfigValue(key, value string) string {
	if strings.HasPrefix(key, "app.") {
		return strings.TrimSpace(value)
	}
	return value
}

func (s *ConfigService) Delete(ctx context.Context, id uint) error {
	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return notFoundOr(err, "配置不存在")
	}
	if c.Builtin == 1 {
		return errors.New("内置配置不可删除")
	}
	// Hard delete: configs are reference data, and a soft-deleted row would keep
	// the unique key, blocking re-creating the same key later.
	if err := s.repo.HardDelete(ctx, id); err != nil {
		return err
	}
	_ = setting.RefreshKey(context.Background(), c.Key) // drops the cache field
	return nil
}

// Refresh re-syncs the cache: a single key when given, otherwise everything.
func (s *ConfigService) Refresh(ctx context.Context, key string) error {
	if key != "" {
		return setting.RefreshKey(ctx, key)
	}
	return setting.RefreshAll(ctx)
}
