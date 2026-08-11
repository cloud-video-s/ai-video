package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"ai-video/internal/pkg/upload"

	"github.com/google/uuid"
)

const ConfigFileMaxSize = int64(5 << 20)

var (
	ErrInvalidConfigFile     = errors.New("配置文件无效")
	ErrUnsupportedConfigFile = errors.New("不支持的配置文件类型")
	ErrConfigFileTooLarge    = errors.New("配置文件超过大小限制")

	configFileKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[._-][a-z0-9]+)*$`)
)

type ConfigFile struct {
	OriginalName string `json:"original_name"`
	ContentType  string `json:"content_type"`
	Size         int64  `json:"size"`
	FilePath     string `json:"file_path"`
	FileURL      string `json:"file_url"`
}

type configFileType struct {
	contentType string
	text        bool
}

var configFileTypes = map[string]configFileType{
	".txt":  {contentType: "text/plain", text: true},
	".html": {contentType: "text/html", text: true},
	".htm":  {contentType: "text/html", text: true},
	".md":   {contentType: "text/markdown", text: true},
	".json": {contentType: "application/json", text: true},
	".xml":  {contentType: "application/xml", text: true},
	".pdf":  {contentType: "application/pdf"},
}

type ConfigFileService struct {
	storage upload.Storage
	now     func() time.Time
	newID   func() string
}

func NewConfigFileService(storage upload.Storage) *ConfigFileService {
	return &ConfigFileService{storage: storage, now: time.Now, newID: uuid.NewString}
}

// Store validates a small public document and writes it to the configured
// upload storage. It deliberately does not update a configuration value: the
// admin must still click Save, keeping file selection and config publication
// as two explicit operations.
func (s *ConfigFileService) Store(ctx context.Context, configKey, originalName string, source io.Reader) (*ConfigFile, error) {
	configKey = strings.TrimSpace(configKey)
	if !configFileKeyPattern.MatchString(configKey) || len(configKey) > 128 {
		return nil, fmt.Errorf("%w: 配置键格式错误", ErrInvalidConfigFile)
	}
	if source == nil {
		return nil, fmt.Errorf("%w: 文件内容不能为空", ErrInvalidConfigFile)
	}

	name := filepath.Base(strings.ReplaceAll(strings.TrimSpace(originalName), `\`, "/"))
	ext := strings.ToLower(filepath.Ext(name))
	fileType, supported := configFileTypes[ext]
	if name == "" || name == "." || !supported {
		return nil, fmt.Errorf("%w: 仅支持 TXT、HTML、Markdown、JSON、XML 或 PDF 文件", ErrUnsupportedConfigFile)
	}

	data, err := io.ReadAll(io.LimitReader(source, ConfigFileMaxSize+1))
	if err != nil {
		return nil, fmt.Errorf("读取配置文件: %w", err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("%w: 文件内容不能为空", ErrInvalidConfigFile)
	}
	if int64(len(data)) > ConfigFileMaxSize {
		return nil, fmt.Errorf("%w: 最大允许 5 MB", ErrConfigFileTooLarge)
	}
	if err := validateConfigFileContent(ext, fileType, data); err != nil {
		return nil, err
	}
	if s.storage == nil {
		return nil, errors.New("配置文件存储未初始化")
	}

	temp, err := os.CreateTemp("", "app-config-file-*")
	if err != nil {
		return nil, fmt.Errorf("创建配置文件临时文件: %w", err)
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if _, err = temp.Write(data); err == nil {
		err = temp.Sync()
	}
	if closeErr := temp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return nil, fmt.Errorf("写入配置文件临时文件: %w", err)
	}

	now := s.now()
	keyPath := strings.ReplaceAll(configKey, ".", "/")
	objectKey := filepath.ToSlash(filepath.Join(
		"config-files", keyPath, now.Format("2006"), now.Format("01"), s.newID()+ext,
	))
	stored, err := s.storage.Store(ctx, objectKey, tempPath, fileType.contentType)
	if err != nil {
		return nil, err
	}
	if stored == nil || strings.TrimSpace(stored.Path) == "" || strings.TrimSpace(stored.URL) == "" {
		return nil, errors.New("配置文件存储未返回有效地址")
	}
	return &ConfigFile{
		OriginalName: name,
		ContentType:  fileType.contentType,
		Size:         int64(len(data)),
		FilePath:     stored.Path,
		FileURL:      stored.URL,
	}, nil
}

func validateConfigFileContent(ext string, fileType configFileType, data []byte) error {
	if fileType.text {
		if !utf8.Valid(data) || bytes.IndexByte(data, 0) >= 0 {
			return fmt.Errorf("%w: %s 必须是 UTF-8 文本文件", ErrUnsupportedConfigFile, ext)
		}
		detected := strings.ToLower(strings.TrimSpace(strings.SplitN(http.DetectContentType(data), ";", 2)[0]))
		if detected != "text/plain" && detected != "text/html" && detected != "application/json" &&
			detected != "text/xml" && detected != "application/xml" {
			return fmt.Errorf("%w: 文件内容与扩展名 %s 不匹配", ErrUnsupportedConfigFile, ext)
		}
		return nil
	}
	if ext == ".pdf" && !bytes.HasPrefix(data, []byte("%PDF-")) {
		return fmt.Errorf("%w: 文件内容与扩展名 .pdf 不匹配", ErrUnsupportedConfigFile)
	}
	return nil
}
