package upload

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
)

const (
	minimumDirectUploadTTL = time.Minute
	maximumDirectUploadTTL = time.Hour
)

// DirectUploadRequest describes one file for which the API should issue a
// short-lived, size-bound OSS upload signature.
type DirectUploadRequest struct {
	MediaType   MediaKind `json:"media_type" binding:"required,oneof=image video"`
	FileName    string    `json:"file_name" binding:"required,max=255"`
	Size        int64     `json:"size" binding:"required,gt=0"`
	ContentType string    `json:"content_type" binding:"required,max=255"`
}

// DirectUploadCredential contains everything a client needs to PUT the raw
// file directly to OSS. All returned headers must be present on the PUT.
type DirectUploadCredential struct {
	UploadID   string            `json:"upload_id"`
	Provider   string            `json:"provider"`
	Method     string            `json:"method"`
	UploadURL  string            `json:"upload_url"`
	Headers    map[string]string `json:"headers"`
	ObjectKey  string            `json:"object_key"`
	FileURL    string            `json:"file_url"`
	PreviewURL string            `json:"preview_url,omitempty"`
	ExpiresAt  time.Time         `json:"expires_at"`
}

type DirectUploadSigner interface {
	Sign(ctx context.Context, request DirectUploadRequest) (*DirectUploadCredential, error)
}

type DirectUploadConfig struct {
	OSS            OSSConfig
	SignatureTTL   time.Duration
	Image          Policy
	Video          Policy
	PolicyResolver func(MediaKind) (Policy, error)
}

type ossPresignClient interface {
	Presign(context.Context, any, ...func(*oss.PresignOptions)) (*oss.PresignResult, error)
}

type OSSDirectUploadSigner struct {
	client         ossPresignClient
	bucket         string
	objectPrefix   string
	baseURL        string
	signatureTTL   time.Duration
	policies       map[MediaKind]normalizedPolicy
	policyResolver func(MediaKind) (Policy, error)
	now            func() time.Time
	newObjectID    func() (string, error)
}

func NewOSSDirectUploadSigner(config DirectUploadConfig) (*OSSDirectUploadSigner, error) {
	if config.SignatureTTL < minimumDirectUploadTTL || config.SignatureTTL > maximumDirectUploadTTL {
		return nil, uploadError(
			ErrInvalidRequest,
			"OSS direct upload signature TTL must be between %s and %s",
			minimumDirectUploadTTL,
			maximumDirectUploadTTL,
		)
	}
	normalized, err := normalizeOSSConfig(config.OSS)
	if err != nil {
		return nil, err
	}
	imagePolicy, err := normalizePolicy(config.Image)
	if err != nil {
		return nil, fmt.Errorf("image policy: %w", err)
	}
	videoPolicy, err := normalizePolicy(config.Video)
	if err != nil {
		return nil, fmt.Errorf("video policy: %w", err)
	}
	sdkConfig := oss.LoadDefaultConfig().
		WithCredentialsProvider(credentials.NewStaticCredentialsProvider(normalized.accessKeyID, normalized.accessKeySecret)).
		WithRegion(normalized.region).
		WithEndpoint(normalized.endpoint).
		WithSignatureVersion(oss.SignatureVersionV4).
		WithAdditionalHeaders([]string{"content-length"})
	return &OSSDirectUploadSigner{
		client:       oss.NewClient(sdkConfig),
		bucket:       normalized.bucket,
		objectPrefix: normalized.objectPrefix,
		baseURL:      normalized.baseURL,
		signatureTTL: config.SignatureTTL,
		policies: map[MediaKind]normalizedPolicy{
			MediaImage: imagePolicy,
			MediaVideo: videoPolicy,
		},
		policyResolver: config.PolicyResolver,
		now:            time.Now,
		newObjectID:    newUploadID,
	}, nil
}

func (s *OSSDirectUploadSigner) Sign(ctx context.Context, request DirectUploadRequest) (*DirectUploadCredential, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	policy, err := s.resolvePolicy(request.MediaType)
	if err != nil {
		return nil, err
	}
	fileName := safeBaseName(request.FileName)
	if fileName == "" || request.Size <= 0 {
		return nil, uploadError(ErrInvalidRequest, "file name and a positive size are required")
	}
	if request.Size > 0 && request.Size > policy.maxFileSize {
		return nil, uploadError(ErrFileTooLarge, "%s exceeds the %d byte limit", fileName, policy.maxFileSize)
	}
	extension := strings.ToLower(filepath.Ext(fileName))
	if _, allowed := policy.exts[extension]; !allowed {
		return nil, uploadError(ErrUnsupportedType, "%s has disallowed extension %q", fileName, extension)
	}
	contentType := normalizeMIME(request.ContentType)
	if contentType == "" {
		return nil, uploadError(ErrInvalidRequest, "content_type is required")
	}
	if _, allowed := policy.mimes[contentType]; !allowed {
		return nil, uploadError(ErrUnsupportedType, "%s has disallowed content type %q", fileName, contentType)
	}

	objectID, err := s.newObjectID()
	if err != nil {
		return nil, err
	}
	now := s.now()
	mediaDir := "images"
	if request.MediaType == MediaVideo {
		mediaDir = "videos"
	}
	objectKey := path.Join(mediaDir, now.Format("2006"), now.Format("01"), now.Format("02"), objectID+extension)
	if s.objectPrefix != "" {
		objectKey = path.Join(s.objectPrefix, objectKey)
	}
	expiresAt := now.Add(s.signatureTTL)
	expectedHeaders := map[string]string{
		"Content-Type":           contentType,
		"x-oss-forbid-overwrite": "true",
	}
	if request.Size > 0 {
		expectedHeaders["Content-Length"] = strconv.FormatInt(request.Size, 10)
	}

	result, err := s.client.Presign(ctx, &oss.PutObjectRequest{
		Bucket:        oss.Ptr(s.bucket),
		Key:           oss.Ptr(objectKey),
		RequestCommon: oss.RequestCommon{Headers: expectedHeaders},
	}, oss.PresignExpiration(expiresAt))
	if err != nil {
		return nil, fmt.Errorf("sign Aliyun OSS direct upload: %w", err)
	}
	if result.Method != "PUT" || strings.TrimSpace(result.URL) == "" {
		return nil, fmt.Errorf("sign Aliyun OSS direct upload: incomplete presign result")
	}
	for name, expected := range expectedHeaders {
		if actual, ok := headerValue(result.SignedHeaders, name); !ok || actual != expected {
			return nil, fmt.Errorf("sign Aliyun OSS direct upload: required header %s was not signed", name)
		}
	}
	return &DirectUploadCredential{
		UploadID: objectID, Provider: StorageAliyunOSS, Method: result.Method, UploadURL: result.URL,
		Headers: result.SignedHeaders, ObjectKey: objectKey,
		FileURL: HalfURL(joinPublicURL(s.baseURL, objectKey)), ExpiresAt: result.Expiration,
	}, nil
}

func (s *OSSDirectUploadSigner) resolvePolicy(kind MediaKind) (normalizedPolicy, error) {
	if kind != MediaImage && kind != MediaVideo {
		return normalizedPolicy{}, uploadError(ErrUnsupportedType, "unknown media type %q", kind)
	}
	if s.policyResolver == nil {
		return s.policies[kind], nil
	}
	policy, err := s.policyResolver(kind)
	if err != nil {
		return normalizedPolicy{}, err
	}
	return normalizePolicy(policy)
}

func headerValue(headers map[string]string, name string) (string, bool) {
	for key, value := range headers {
		if strings.EqualFold(key, name) {
			return value, true
		}
	}
	return "", false
}
