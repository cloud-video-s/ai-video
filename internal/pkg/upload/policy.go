package upload

import (
	"sort"
	"strings"
)

var supportedMediaTypes = map[MediaKind]map[string]string{
	MediaImage: {
		".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".png": "image/png",
		".gif": "image/gif", ".webp": "image/webp",
	},
	MediaVideo: {
		".mp4": "video/mp4", ".mov": "video/quicktime",
		".webm": "video/webm", ".mkv": "video/x-matroska",
	},
}

// SupportedFileExtensions returns the file extensions whose signatures and
// MIME types the upload validator knows how to verify.
func SupportedFileExtensions(kind MediaKind) []string {
	types := supportedMediaTypes[kind]
	result := make([]string, 0, len(types))
	for ext := range types {
		result = append(result, ext)
	}
	sort.Strings(result)
	return result
}

// PolicyForExtensions builds a complete policy from the admin-configurable
// extension list. MIME types are derived server-side so extensions and content
// signature validation cannot drift apart.
func PolicyForExtensions(kind MediaKind, maxFileSize int64, extensions []string) (Policy, error) {
	supported, ok := supportedMediaTypes[kind]
	if !ok {
		return Policy{}, uploadError(ErrInvalidRequest, "unknown media kind %q", kind)
	}
	if maxFileSize <= 0 {
		return Policy{}, uploadError(ErrInvalidRequest, "maximum file size must be positive")
	}
	extSeen := make(map[string]struct{}, len(extensions))
	mimeSeen := make(map[string]struct{}, len(extensions))
	exts := make([]string, 0, len(extensions))
	mimes := make([]string, 0, len(extensions))
	for _, value := range extensions {
		ext := strings.ToLower(strings.TrimSpace(value))
		if ext != "" && !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		mimeType, supportedType := supported[ext]
		if !supportedType {
			return Policy{}, uploadError(ErrUnsupportedType, "extension %q is not supported for %s uploads", ext, kind)
		}
		if _, exists := extSeen[ext]; !exists {
			extSeen[ext] = struct{}{}
			exts = append(exts, ext)
		}
		if _, exists := mimeSeen[mimeType]; !exists {
			mimeSeen[mimeType] = struct{}{}
			mimes = append(mimes, mimeType)
		}
	}
	if len(exts) == 0 {
		return Policy{}, uploadError(ErrInvalidRequest, "at least one file extension is required")
	}
	return Policy{MaxFileSize: maxFileSize, AllowedExts: exts, AllowedMIMETypes: mimes}, nil
}
