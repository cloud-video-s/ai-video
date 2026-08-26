package config

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Timezone       string               `mapstructure:"timezone"`
	Server         ServerConfig         `mapstructure:"server"`
	Database       DatabaseConfig       `mapstructure:"database"`
	Redis          RedisConfig          `mapstructure:"redis"`
	JWT            JWTConfig            `mapstructure:"jwt"`
	ApiJwt         JWTConfig            `mapstructure:"api_jwt"`
	GeoIP          GeoIPConfig          `mapstructure:"geoip"`
	ThirdPartyAuth ThirdPartyAuthConfig `mapstructure:"third_party_auth"`
	Adjust         AdjustConfig         `mapstructure:"adjust"`
	AppStore       AppStoreConfig       `mapstructure:"app_store"`
	Upload         UploadConfig         `mapstructure:"upload"`
	Casbin         CasbinConfig         `mapstructure:"casbin"`
	Log            LogConfig            `mapstructure:"log"`
	Task           TaskConfig           `mapstructure:"task"`
}

type TaskConfig struct {
	Concurrency                  int      `mapstructure:"concurrency"`
	DownloadConcurrency          int      `mapstructure:"download_concurrency"`
	DownloadRetryCount           int      `mapstructure:"download_retry_count"`
	WorkerRestartDelaySeconds    int      `mapstructure:"worker_restart_delay_seconds"`
	WorkerRestartMaxDelaySeconds int      `mapstructure:"worker_restart_max_delay_seconds"`
	Queues                       []string `mapstructure:"queues"`
	SubscriptionExpirationAt     string   `mapstructure:"subscription_expiration_at"`
}

const (
	defaultWorkerRestartDelaySeconds    = 1
	defaultWorkerRestartMaxDelaySeconds = 30
)

func (c TaskConfig) WorkerRestartBackoff() (time.Duration, time.Duration) {
	initialSeconds := c.WorkerRestartDelaySeconds
	if initialSeconds <= 0 {
		initialSeconds = defaultWorkerRestartDelaySeconds
	}
	maxSeconds := c.WorkerRestartMaxDelaySeconds
	if maxSeconds <= 0 {
		maxSeconds = defaultWorkerRestartMaxDelaySeconds
	}
	if maxSeconds < initialSeconds {
		maxSeconds = initialSeconds
	}
	return time.Duration(initialSeconds) * time.Second, time.Duration(maxSeconds) * time.Second
}

const TaskExecuteAtLayout = "2006-01-02 15:04:05"

// SubscriptionExpirationTime parses the configured one-time execution time.
// A timestamp without an explicit offset is interpreted in loc. An empty value
// disables scheduling while still allowing workers to process an already
// queued task after a restart.
func (c TaskConfig) SubscriptionExpirationTime(loc *time.Location) (time.Time, bool, error) {
	value := strings.TrimSpace(c.SubscriptionExpirationAt)
	if value == "" {
		return time.Time{}, false, nil
	}
	if executeAt, err := time.Parse(time.RFC3339, value); err == nil {
		return executeAt, true, nil
	}
	if loc == nil {
		loc = time.Local
	}
	executeAt, err := time.ParseInLocation(TaskExecuteAtLayout, value, loc)
	if err != nil {
		return time.Time{}, false, fmt.Errorf(
			"task.subscription_expiration_at must use %q or RFC3339: %w",
			TaskExecuteAtLayout,
			err,
		)
	}
	return executeAt, true, nil
}

type ServerConfig struct {
	Port         int      `mapstructure:"port"`
	Mode         string   `mapstructure:"mode"`
	AllowOrigins []string `mapstructure:"allow_origins"`
}

type DatabaseConfig struct {
	Driver       string `mapstructure:"driver"`
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Username     string `mapstructure:"username"`
	Password     string `mapstructure:"password"`
	DBName       string `mapstructure:"dbname"`
	Charset      string `mapstructure:"charset"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	LogLevel     string `mapstructure:"log_level"`
}

// DSN builds the database connection string. timezone (an IANA name like
// Asia/Shanghai) is applied to Postgres directly; MySQL uses loc=Local, which
// follows the process-wide time.Local set by InitTimezone.
func (d *DatabaseConfig) DSN(timezone string) string {
	switch d.Driver {
	case "postgres":
		return fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable TimeZone=%s",
			d.Host, d.Port, d.Username, d.Password, d.DBName, timezone,
		)
	default:
		return fmt.Sprintf(
			"%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
			d.Username, d.Password, d.Host, d.Port, d.DBName, d.Charset,
		)
	}
}

type RedisConfig struct {
	Host      string `mapstructure:"host"`
	Port      int    `mapstructure:"port"`
	Username  string `mapstructure:"username"`
	Password  string `mapstructure:"password"`
	DB        int    `mapstructure:"db"`
	KeyPrefix string `mapstructure:"key_prefix"`
}

func (r *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

type JWTConfig struct {
	Secret string `mapstructure:"secret"`
	Expire int64  `mapstructure:"expire"`
	Issuer string `mapstructure:"issuer"`
}

type GeoIPConfig struct {
	CountryHeader string `mapstructure:"country_header"`
	LookupURL     string `mapstructure:"lookup_url"`
	CountryField  string `mapstructure:"country_field"`
	TimeoutMS     int    `mapstructure:"timeout_ms"`
}

type ThirdPartyAuthConfig struct {
	HTTPTimeoutMS    int                `mapstructure:"http_timeout_ms"`
	JWKSCacheSeconds int64              `mapstructure:"jwks_cache_seconds"`
	Google           OIDCProviderConfig `mapstructure:"google"`
	Apple            OIDCProviderConfig `mapstructure:"apple"`
}

type OIDCProviderConfig struct {
	ClientIDs []string `mapstructure:"client_ids"`
	Issuers   []string `mapstructure:"issuers"`
	JWKSURL   string   `mapstructure:"jwks_url"`
}

// AdjustConfig controls inbound attribution callbacks, outbound S2S events,
// and Campaign API tracker synchronization independently. CallbackToken is an
// application-defined callback credential; Adjust does not prescribe its
// length. Tracker channel keys may be Adjust tracker tokens or tracker names;
// token matches win.
type AdjustConfig struct {
	Enabled            bool              `mapstructure:"enabled"`
	EventEnabled       bool              `mapstructure:"event_enabled"`
	TrackerSyncEnabled bool              `mapstructure:"tracker_sync_enabled"`
	MaxBodyBytes       int64             `mapstructure:"max_body_bytes"`
	TrackerChannels    map[string]string `mapstructure:"tracker_channels"`
	CampaignAppToken   string            `mapstructure:"campaign_app_token"`
	CampaignBaseURL    string            `mapstructure:"campaign_base_url"`
	EventBaseURL       string            `mapstructure:"event_base_url"`
	EventEnvironment   string            `mapstructure:"event_environment"`
	EventAppTokens     map[string]string `mapstructure:"event_app_tokens"`
}

// AppStoreConfig contains the App Store Connect API key metadata used to call
// the App Store Server API. The private key signs outbound API JWTs only; Apple
// transaction and notification JWS values are verified with Apple's x5c chain.
type AppStoreConfig struct {
	BundleID       string `mapstructure:"bundle_id"`
	IssuerID       string `mapstructure:"issuer_id"`
	KeyID          string `mapstructure:"key_id"`
	PrivateKeyPath string `mapstructure:"private_key_path"`
	HTTPTimeoutMS  int    `mapstructure:"http_timeout_ms"`
}

type UploadConfig struct {
	RootDir                string   `mapstructure:"root_dir"`
	LocalRootDir           string   `mapstructure:"local_root_dir"`
	LocalBaseURL           string   `mapstructure:"local_base_url"`
	StorageProvider        string   `mapstructure:"storage_provider"`
	OSSRegion              string   `mapstructure:"oss_region"`
	OSSEndpoint            string   `mapstructure:"oss_endpoint"`
	OSSAccessKeyID         string   `mapstructure:"oss_access_key_id"`
	OSSAccessKeySecret     string   `mapstructure:"oss_access_key_secret"`
	OSSBucket              string   `mapstructure:"oss_bucket"`
	OSSObjectPrefix        string   `mapstructure:"oss_object_prefix"`
	OSSBaseURL             string   `mapstructure:"oss_base_url"`
	ProxyBaseURL           string   `mapstructure:"proxy_base_url"`
	OSSSignatureTTLSeconds int64    `mapstructure:"oss_signature_ttl_seconds"`
	ChunkSize              int64    `mapstructure:"chunk_size"`
	MaxBatchFiles          int      `mapstructure:"max_batch_files"`
	SessionTTLSeconds      int64    `mapstructure:"session_ttl_seconds"`
	ImageMaxFileSize       int64    `mapstructure:"image_max_file_size"`
	VideoMaxFileSize       int64    `mapstructure:"video_max_file_size"`
	ImageExtensions        []string `mapstructure:"image_extensions"`
	VideoExtensions        []string `mapstructure:"video_extensions"`
	ImageMIMETypes         []string `mapstructure:"image_mime_types"`
	VideoMIMETypes         []string `mapstructure:"video_mime_types"`
}

type CasbinConfig struct {
	ModelPath string `mapstructure:"model_path"`
}

type LogConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	Directory  string `mapstructure:"directory"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
}

var Cfg Config

const defaultJWTSecret = "frame-jwt-secret-key-change-in-production"

func InitConfig(cfgFile string) error {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath("./config")
		viper.AddConfigPath(".")
	}

	setConfigDefaults()

	// Allow nested config keys to be overridden via env vars, e.g. the env var
	// DATABASE_HOST overrides database.host (used by deploy/docker-compose.yaml).
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("read config failed: %w", err)
	}
	if err := viper.Unmarshal(&Cfg); err != nil {
		return fmt.Errorf("unmarshal config failed: %w", err)
	}
	if err := validateConfig(); err != nil {
		return err
	}
	return nil
}

func setConfigDefaults() {
	viper.SetDefault("timezone", "Asia/Shanghai")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.mode", "debug")
	viper.SetDefault("database.max_idle_conns", 10)
	viper.SetDefault("database.max_open_conns", 100)
	viper.SetDefault("jwt.expire", 7200)
	viper.SetDefault("geoip.country_field", "country_code")
	viper.SetDefault("geoip.timeout_ms", 3000)
	viper.SetDefault("third_party_auth.http_timeout_ms", 5000)
	viper.SetDefault("third_party_auth.jwks_cache_seconds", int64(21600))
	viper.SetDefault("third_party_auth.google.issuers", []string{"https://accounts.google.com", "accounts.google.com"})
	viper.SetDefault("third_party_auth.google.jwks_url", "https://www.googleapis.com/oauth2/v3/certs")
	viper.SetDefault("third_party_auth.apple.issuers", []string{"https://appleid.apple.com"})
	viper.SetDefault("third_party_auth.apple.jwks_url", "https://appleid.apple.com/auth/keys")
	viper.SetDefault("adjust.enabled", false)
	viper.SetDefault("adjust.event_enabled", true)
	viper.SetDefault("adjust.tracker_sync_enabled", false)
	viper.SetDefault("adjust.callback_token", "")
	viper.SetDefault("adjust.max_body_bytes", int64(65536))
	viper.SetDefault("adjust.tracker_channels", map[string]string{})
	viper.SetDefault("adjust.campaign_api_token", "")
	viper.SetDefault("adjust.campaign_app_token", "")
	viper.SetDefault("adjust.campaign_base_url", "https://api.adjust.com/public/v2")
	viper.SetDefault("adjust.event_auth_token", "")
	viper.SetDefault("adjust.event_base_url", "https://s2s.adjust.com")
	viper.SetDefault("adjust.event_environment", "production")
	viper.SetDefault("adjust.event_app_tokens", map[string]string{})
	viper.SetDefault("app_store.bundle_id", "")
	viper.SetDefault("app_store.issuer_id", "")
	viper.SetDefault("app_store.key_id", "")
	viper.SetDefault("app_store.private_key_path", "config/appkey.p8")
	viper.SetDefault("app_store.http_timeout_ms", 10000)
	viper.SetDefault("upload.root_dir", "storage/uploads/tmp")
	viper.SetDefault("upload.local_root_dir", "storage/uploads/files")
	viper.SetDefault("upload.local_base_url", "/uploads")
	viper.SetDefault("upload.storage_provider", "aliyun_oss")
	viper.SetDefault("upload.oss_base_url", "https://balaaitest.oss-ap-southeast-1.aliyuncs.com/")
	viper.SetDefault("upload.proxy_base_url", "https://test-cdn.zdrawai.com/")
	viper.SetDefault("upload.oss_signature_ttl_seconds", int64(600))
	viper.SetDefault("upload.chunk_size", int64(5<<20))
	viper.SetDefault("upload.max_batch_files", 20)
	viper.SetDefault("upload.session_ttl_seconds", int64(86400))
	viper.SetDefault("upload.image_max_file_size", int64(20<<20))
	viper.SetDefault("upload.video_max_file_size", int64(2<<30))
	viper.SetDefault("upload.image_extensions", []string{".jpg", ".jpeg", ".png", ".gif", ".webp"})
	viper.SetDefault("upload.video_extensions", []string{".mp4", ".mov", ".webm", ".mkv"})
	viper.SetDefault("upload.image_mime_types", []string{"image/jpeg", "image/png", "image/gif", "image/webp"})
	viper.SetDefault("upload.video_mime_types", []string{"video/mp4", "video/quicktime", "video/webm", "video/x-matroska"})
	viper.SetDefault("task.concurrency", 10)
	viper.SetDefault("task.download_concurrency", 1)
	viper.SetDefault("task.download_retry_count", 3)
	viper.SetDefault("task.worker_restart_delay_seconds", defaultWorkerRestartDelaySeconds)
	viper.SetDefault("task.worker_restart_max_delay_seconds", defaultWorkerRestartMaxDelaySeconds)
	viper.SetDefault("task.subscription_expiration_at", "")
}

// InitTimezone sets the process-wide time.Local from config so that every
// time.Now()/time.Local usage — logs, JWT, GORM timestamps, MySQL loc=Local,
// the asynq scheduler — shares one timezone. Call right after InitConfig.
func InitTimezone() error {
	loc, err := time.LoadLocation(Cfg.Timezone)
	if err != nil {
		return fmt.Errorf("load timezone %q: %w", Cfg.Timezone, err)
	}
	time.Local = loc
	return nil
}

// validateConfig fails fast on missing required fields and rejects weak or
// default JWT secrets when running in release mode.
func validateConfig() error {
	if Cfg.Database.Driver == "" {
		return fmt.Errorf("database.driver is required")
	}
	if Cfg.Database.DBName == "" {
		return fmt.Errorf("database.dbname is required")
	}
	if Cfg.JWT.Secret == "" {
		return fmt.Errorf("jwt.secret is required")
	}
	if Cfg.Upload.RootDir == "" || Cfg.Upload.LocalRootDir == "" || Cfg.Upload.ChunkSize <= 0 || Cfg.Upload.MaxBatchFiles <= 0 ||
		Cfg.Upload.SessionTTLSeconds <= 0 ||
		Cfg.Upload.ImageMaxFileSize <= 0 || Cfg.Upload.VideoMaxFileSize <= 0 {
		return fmt.Errorf("upload config values must be positive")
	}
	if filepath.Clean(Cfg.Upload.RootDir) == filepath.Clean(Cfg.Upload.LocalRootDir) {
		return fmt.Errorf("upload.root_dir must be a temporary directory separate from upload.local_root_dir")
	}
	if Cfg.Upload.OSSSignatureTTLSeconds < 60 || Cfg.Upload.OSSSignatureTTLSeconds > 3600 {
		return fmt.Errorf("upload.oss_signature_ttl_seconds must be between 60 and 3600")
	}
	if strings.TrimSpace(Cfg.Task.SubscriptionExpirationAt) != "" {
		loc, err := time.LoadLocation(Cfg.Timezone)
		if err != nil {
			return fmt.Errorf("load timezone %q for task.subscription_expiration_at: %w", Cfg.Timezone, err)
		}
		if _, _, err := Cfg.Task.SubscriptionExpirationTime(loc); err != nil {
			return err
		}
	}
	if Cfg.Task.DownloadConcurrency <= 0 {
		return fmt.Errorf("task.download_concurrency must be positive")
	}
	if Cfg.Task.DownloadRetryCount < 0 {
		return fmt.Errorf("task.download_retry_count cannot be negative")
	}
	if Cfg.Task.WorkerRestartDelaySeconds < 0 {
		return fmt.Errorf("task.worker_restart_delay_seconds cannot be negative")
	}
	if Cfg.Task.WorkerRestartMaxDelaySeconds < 0 {
		return fmt.Errorf("task.worker_restart_max_delay_seconds cannot be negative")
	}
	if Cfg.Task.WorkerRestartDelaySeconds > 0 && Cfg.Task.WorkerRestartMaxDelaySeconds > 0 &&
		Cfg.Task.WorkerRestartMaxDelaySeconds < Cfg.Task.WorkerRestartDelaySeconds {
		return fmt.Errorf("task.worker_restart_max_delay_seconds cannot be less than task.worker_restart_delay_seconds")
	}
	if Cfg.Adjust.Enabled {
		if Cfg.Adjust.MaxBodyBytes <= 0 || Cfg.Adjust.MaxBodyBytes > 1<<20 {
			return fmt.Errorf("adjust.max_body_bytes must be between 1 and 1048576")
		}
		for tracker, channel := range Cfg.Adjust.TrackerChannels {
			tracker = strings.TrimSpace(tracker)
			channel = strings.TrimSpace(channel)
			if tracker == "" || channel == "" {
				return fmt.Errorf("adjust.tracker_channels cannot contain empty tracker or channel codes")
			}
			if len(tracker) > 255 || len(channel) > 64 {
				return fmt.Errorf("adjust.tracker_channels tracker names must be at most 255 bytes and channel codes at most 64 bytes")
			}
		}
	}
	if Cfg.Adjust.EventEnabled {
		if len(Cfg.Task.Queues) > 0 {
			hasDefaultQueue := false
			for _, queue := range Cfg.Task.Queues {
				if strings.TrimSpace(queue) == "default" {
					hasDefaultQueue = true
					break
				}
			}
			if !hasDefaultQueue {
				return fmt.Errorf("task.queues must contain default when Adjust event reporting is enabled")
			}
		}
		environment := strings.ToLower(strings.TrimSpace(Cfg.Adjust.EventEnvironment))
		if environment != "production" && environment != "sandbox" {
			return fmt.Errorf("adjust.event_environment must be production or sandbox")
		}
		if strings.TrimSpace(Cfg.Adjust.EventBaseURL) == "" {
			return fmt.Errorf("adjust.event_base_url is required when Adjust event reporting is enabled")
		}
		for appCode, appToken := range Cfg.Adjust.EventAppTokens {
			if strings.TrimSpace(appCode) == "" || strings.TrimSpace(appToken) == "" {
				return fmt.Errorf("adjust.event_app_tokens cannot contain empty app codes or tokens")
			}
		}
	}
	if Cfg.Adjust.TrackerSyncEnabled {
		if len(Cfg.Task.Queues) > 0 {
			hasDefaultQueue := false
			for _, queue := range Cfg.Task.Queues {
				if strings.TrimSpace(queue) == "default" {
					hasDefaultQueue = true
					break
				}
			}
			if !hasDefaultQueue {
				return fmt.Errorf("task.queues must contain default when Adjust tracker sync is enabled")
			}
		}
		if !isConfigAlphaNumeric(strings.TrimSpace(Cfg.Adjust.CampaignAppToken)) {
			return fmt.Errorf("adjust.campaign_app_token must be a non-empty alphanumeric token when tracker sync is enabled")
		}
		if strings.TrimSpace(Cfg.Adjust.CampaignBaseURL) == "" {
			return fmt.Errorf("adjust.campaign_base_url is required when tracker sync is enabled")
		}
	}
	if Cfg.Server.Mode == "release" {
		if len(Cfg.JWT.Secret) < 32 {
			return fmt.Errorf("jwt.secret must be at least 32 bytes in release mode")
		}
	}
	return nil
}

func isConfigAlphaNumeric(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range value {
		if (character < 'a' || character > 'z') &&
			(character < 'A' || character > 'Z') &&
			(character < '0' || character > '9') {
			return false
		}
	}
	return true
}
