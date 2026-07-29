package uploadruntime

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"ai-video/internal/config"
	"ai-video/internal/pkg/setting"
	"ai-video/internal/pkg/upload"
)

type storageFactory struct {
	mu          sync.Mutex
	fingerprint [sha256.Size]byte
	storage     upload.Storage
}

type directSignerFactory struct {
	mu          sync.Mutex
	fingerprint [sha256.Size]byte
	signer      upload.DirectUploadSigner
}

var sharedStorageFactory = &storageFactory{}

func ManagerConfig() (upload.Config, error) {
	managerConfig := config.UploadManagerConfig()
	dynamic, err := Storage()
	if err != nil {
		return upload.Config{}, err
	}
	managerConfig.Storage = dynamic
	managerConfig.PolicyResolver = func(kind upload.MediaKind) (upload.Policy, error) {
		return configuredPolicy(kind, config.Cfg.Upload)
	}
	return managerConfig, nil
}

// Storage returns the dynamic storage used by both regular uploads and
// generated task results. The active provider is resolved for every Store
// call, so changes made in the upload settings take effect without a restart.
func Storage() (upload.Storage, error) {
	return upload.NewDynamicStorage(sharedStorageFactory.resolve)
}

func DirectSigner() upload.DirectUploadSigner {
	return &directSignerFactory{}
}

func (f *directSignerFactory) Sign(ctx context.Context, request upload.DirectUploadRequest) (*upload.DirectUploadCredential, error) {
	signer, err := f.resolve()
	if err != nil {
		return nil, err
	}
	return signer.Sign(ctx, request)
}

func (f *directSignerFactory) resolve() (upload.DirectUploadSigner, error) {
	cfg := config.Cfg.Upload
	provider := configured("upload.storage_provider", cfg.StorageProvider)
	if provider != upload.StorageAliyunOSS {
		return nil, fmt.Errorf("%w: active storage provider is %q", upload.ErrDirectUploadUnavailable, provider)
	}
	ttlSeconds, err := configuredPositiveInt64("upload.oss.signature_ttl_seconds", cfg.OSSSignatureTTLSeconds)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", upload.ErrDirectUploadUnavailable, err)
	}
	values := []string{
		configured("upload.oss.region", cfg.OSSRegion),
		configured("upload.oss.endpoint", cfg.OSSEndpoint),
		configured("upload.oss.access_key_id", cfg.OSSAccessKeyID),
		configured("upload.oss.access_key_secret", cfg.OSSAccessKeySecret),
		configured("upload.oss.bucket", cfg.OSSBucket),
		configured("upload.oss.object_prefix", cfg.OSSObjectPrefix),
		configured("upload.oss.base_url", cfg.OSSBaseURL),
		strconv.FormatInt(ttlSeconds, 10),
	}
	fingerprint := sha256.Sum256([]byte(strings.Join(values, "\x00")))

	f.mu.Lock()
	defer f.mu.Unlock()
	if f.signer != nil && f.fingerprint == fingerprint {
		return f.signer, nil
	}
	baseConfig := config.UploadManagerConfig()
	signer, err := upload.NewOSSDirectUploadSigner(upload.DirectUploadConfig{
		OSS: upload.OSSConfig{
			Region: values[0], Endpoint: values[1], AccessKeyID: values[2], AccessKeySecret: values[3],
			Bucket: values[4], ObjectPrefix: values[5], BaseURL: values[6],
		},
		SignatureTTL: time.Duration(ttlSeconds) * time.Second,
		Image:        baseConfig.Image,
		Video:        baseConfig.Video,
		PolicyResolver: func(kind upload.MediaKind) (upload.Policy, error) {
			return configuredPolicy(kind, config.Cfg.Upload)
		},
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", upload.ErrDirectUploadUnavailable, err)
	}
	f.fingerprint = fingerprint
	f.signer = signer
	return signer, nil
}

func configuredPolicy(kind upload.MediaKind, cfg config.UploadConfig) (upload.Policy, error) {
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
	cfg := config.Cfg.Upload
	provider := configured("upload.storage_provider", cfg.StorageProvider)
	if provider == "" {
		provider = upload.StorageLocal
	}

	values := []string{
		provider,
		configured("upload.local_base_url", cfg.LocalBaseURL),
		configured("upload.oss.region", cfg.OSSRegion),
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
			Region: values[2], Endpoint: values[3], AccessKeyID: values[4], AccessKeySecret: values[5],
			Bucket: values[6], ObjectPrefix: values[7], BaseURL: values[8],
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

func configuredPositiveInt64(key string, fallback int64) (int64, error) {
	value := fallback
	if raw := strings.TrimSpace(setting.GetString(key)); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("%s must be a positive integer", key)
		}
		value = parsed
	}
	if value <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", key)
	}
	return value, nil
}
