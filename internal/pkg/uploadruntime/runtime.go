package uploadruntime

import (
	"crypto/sha256"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"ai-video/internal/app"
	"ai-video/internal/pkg/setting"
	"ai-video/internal/pkg/upload"
)

type storageFactory struct {
	mu          sync.Mutex
	fingerprint [sha256.Size]byte
	storage     upload.Storage
}

func ManagerConfig() (upload.Config, error) {
	config := app.UploadManagerConfig()
	factory := &storageFactory{}
	dynamic, err := upload.NewDynamicStorage(factory.resolve)
	if err != nil {
		return upload.Config{}, err
	}
	config.Storage = dynamic
	config.PolicyResolver = func(kind upload.MediaKind) (upload.Policy, error) {
		return configuredPolicy(kind, app.Cfg.Upload)
	}
	return config, nil
}

func configuredPolicy(kind upload.MediaKind, cfg app.UploadConfig) (upload.Policy, error) {
	var sizeKey, extensionsKey string
	var fallbackSize int64
	var fallbackExtensions []string
	switch kind {
	case upload.MediaImage:
		sizeKey, extensionsKey = "upload.image_max_file_size", "upload.image_extensions"
		fallbackSize, fallbackExtensions = cfg.ImageMaxFileSize, cfg.ImageExtensions
	case upload.MediaVideo:
		sizeKey, extensionsKey = "upload.video_max_file_size", "upload.video_extensions"
		fallbackSize, fallbackExtensions = cfg.VideoMaxFileSize, cfg.VideoExtensions
	default:
		return upload.Policy{}, fmt.Errorf("unsupported upload media kind %q", kind)
	}

	maxFileSize := fallbackSize
	if raw := strings.TrimSpace(setting.GetString(sizeKey)); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value <= 0 {
			return upload.Policy{}, fmt.Errorf("%s must be a positive byte count", sizeKey)
		}
		maxFileSize = value
	}
	extensions := fallbackExtensions
	if raw := strings.TrimSpace(setting.GetString(extensionsKey)); raw != "" {
		extensions = splitExtensions(raw)
	}
	return upload.PolicyForExtensions(kind, maxFileSize, extensions)
}

func splitExtensions(value string) []string {
	return strings.FieldsFunc(value, func(r rune) bool {
		switch r {
		case ',', ';', ' ', '\t', '\r', '\n':
			return true
		default:
			return false
		}
	})
}

func (f *storageFactory) resolve() (upload.Storage, error) {
	cfg := app.Cfg.Upload
	provider := configured("upload.storage_provider", cfg.StorageProvider)
	if provider == "" {
		provider = upload.StorageLocal
	}

	values := []string{
		provider,
		configured("upload.local_base_url", cfg.LocalBaseURL),
		configured("upload.oss.endpoint", cfg.OSSEndpoint),
		configured("upload.oss.access_key_id", cfg.OSSAccessKeyID),
		configured("upload.oss.access_key_secret", cfg.OSSAccessKeySecret),
		configured("upload.oss.bucket", cfg.OSSBucket),
		configured("upload.oss.object_prefix", cfg.OSSObjectPrefix),
		configured("upload.oss.base_url", cfg.OSSBaseURL),
	}
	fingerprint := sha256.Sum256([]byte(strings.Join(values, "\x00")))

	f.mu.Lock()
	defer f.mu.Unlock()
	if f.storage != nil && f.fingerprint == fingerprint {
		return f.storage, nil
	}

	var (
		storage upload.Storage
		err     error
	)
	switch provider {
	case upload.StorageLocal:
		storage, err = upload.NewLocalStorage(cfg.LocalRootDir, values[1])
	case upload.StorageAliyunOSS:
		storage, err = upload.NewOSSStorage(upload.OSSConfig{
			Endpoint: values[2], AccessKeyID: values[3], AccessKeySecret: values[4],
			Bucket: values[5], ObjectPrefix: values[6], BaseURL: values[7],
		})
	default:
		err = fmt.Errorf("unsupported upload storage provider %q", provider)
	}
	if err != nil {
		return nil, err
	}
	f.fingerprint = fingerprint
	f.storage = storage
	return storage, nil
}

func configured(key, fallback string) string {
	if value := strings.TrimSpace(setting.GetString(key)); value != "" {
		return value
	}
	return strings.TrimSpace(fallback)
}
