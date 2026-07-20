package upload

import (
	"bytes"
	"encoding/hex"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
)

type normalizedPolicy struct {
	maxFileSize int64
	exts        map[string]struct{}
	mimes       map[string]struct{}
}

func normalizePolicy(policy Policy) (normalizedPolicy, error) {
	if policy.MaxFileSize <= 0 || len(policy.AllowedExts) == 0 {
		return normalizedPolicy{}, uploadError(ErrInvalidRequest, "upload policy is incomplete")
	}
	normalized := normalizedPolicy{
		maxFileSize: policy.MaxFileSize,
		exts:        make(map[string]struct{}, len(policy.AllowedExts)),
		mimes:       make(map[string]struct{}, len(policy.AllowedMIMETypes)),
	}
	for _, ext := range policy.AllowedExts {
		ext = strings.ToLower(strings.TrimSpace(ext))
		if ext != "" && !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		if ext != "" {
			normalized.exts[ext] = struct{}{}
		}
	}
	for _, contentType := range policy.AllowedMIMETypes {
		contentType = normalizeMIME(contentType)
		if contentType != "" {
			normalized.mimes[contentType] = struct{}{}
		}
	}
	return normalized, nil
}

func normalizeMIME(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if mediaType, _, err := mime.ParseMediaType(value); err == nil {
		return mediaType
	}
	return value
}

func validateSHA256(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "", nil
	}
	decoded, err := hex.DecodeString(value)
	if err != nil || len(decoded) != 32 {
		return "", uploadError(ErrInvalidRequest, "sha256 must contain 64 hexadecimal characters")
	}
	return value, nil
}

func detectAndValidateContent(kind MediaKind, ext string, header []byte, policy normalizedPolicy) (string, error) {
	var detected string
	switch kind {
	case MediaImage:
		detected = http.DetectContentType(header)
	case MediaVideo:
		detected = detectVideoMIME(ext, header)
	default:
		return "", uploadError(ErrUnsupportedType, "unknown media kind %q", kind)
	}
	detected = normalizeMIME(detected)
	if _, ok := policy.mimes[detected]; !ok {
		return "", uploadError(ErrUnsupportedType, "detected content type %q is not allowed", detected)
	}
	return detected, nil
}

func detectVideoMIME(ext string, header []byte) string {
	if len(header) >= 12 && bytes.Equal(header[4:8], []byte("ftyp")) {
		if ext == ".mov" {
			return "video/quicktime"
		}
		return "video/mp4"
	}
	if len(header) >= 4 && bytes.Equal(header[:4], []byte{0x1a, 0x45, 0xdf, 0xa3}) {
		if ext == ".mkv" {
			return "video/x-matroska"
		}
		return "video/webm"
	}
	return "application/octet-stream"
}

func safeBaseName(name string) string {
	name = strings.TrimSpace(filepath.Base(strings.ReplaceAll(name, "\\", "/")))
	if name == "." || name == "" {
		return ""
	}
	return name
}
