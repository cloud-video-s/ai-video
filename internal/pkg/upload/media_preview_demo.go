package upload

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	maxPreviewSourcePixels      int64 = 100_000_000
	defaultRemoteImageMaxBytes  int64 = 20 << 20
	defaultRemoteRequestTimeout       = 30 * time.Minute
)

// VideoSnapshotURLBuilder converts a remote video URL into an image snapshot
// URL. The default builder uses Aliyun OSS video/snapshot processing.
type VideoSnapshotURLBuilder func(*url.URL) (*url.URL, error)

// MediaPreviewOptions controls the JPEG preview generated for an uploaded
// image or video. Zero values use DefaultMediaPreviewOptions.
type MediaPreviewOptions struct {
	MaxWidth         int
	MaxHeight        int
	JPEGQuality      int
	RemoteHTTPClient *http.Client
	MaxRemoteBytes   int64
	VideoSnapshotURL VideoSnapshotURLBuilder
}

// DefaultMediaPreviewOptions returns practical defaults for web thumbnails.
func DefaultMediaPreviewOptions() MediaPreviewOptions {
	return MediaPreviewOptions{
		MaxWidth:    1280,
		MaxHeight:   720,
		JPEGQuality: 75,
	}
}

// GenerateMediaPreview creates a compressed JPEG preview for an uploaded
// media file. Images are resized directly; videos use their first video frame.
// sourcePath may be a local path or an HTTP/HTTPS URL. The source file is never
// modified, and downloaded temporary files are removed after processing.
func GenerateMediaPreview(
	ctx context.Context,
	kind MediaKind,
	sourcePath string,
	outputPath string,
	options MediaPreviewOptions,
) error {
	switch kind {
	case MediaImage:
		return GenerateCompressedImagePreview(ctx, sourcePath, outputPath, options)
	case MediaVideo:
		return GenerateVideoFirstFramePreview(ctx, sourcePath, outputPath, options)
	default:
		return uploadError(ErrUnsupportedType, "cannot generate a preview for media kind %q", kind)
	}
}

// GenerateCompressedImagePreview resizes an image without enlarging it and
// writes a compressed JPEG preview. JPEG, PNG, and GIF inputs are supported by
// the standard library; animated GIFs use the first frame. sourcePath may be a
// local path or an HTTP/HTTPS URL.
func GenerateCompressedImagePreview(
	ctx context.Context,
	sourcePath string,
	outputPath string,
	options MediaPreviewOptions,
) error {
	options, err := normalizeMediaPreviewOptions(options)
	if err != nil {
		return err
	}
	remoteSourcePath, cleanup, err := preparePreviewSource(
		ctx,
		sourcePath,
		outputPath,
		options,
		defaultRemoteImageMaxBytes,
	)
	if err != nil {
		return err
	}
	defer cleanup()
	sourcePath = remoteSourcePath
	sourcePath, outputPath, err = preparePreviewPaths(sourcePath, outputPath)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("open preview source image: %w", err)
	}
	defer source.Close()

	config, _, err := image.DecodeConfig(source)
	if err != nil {
		return fmt.Errorf("decode preview source image config: %w", err)
	}
	if config.Width <= 0 || config.Height <= 0 ||
		int64(config.Width) > maxPreviewSourcePixels/int64(config.Height) {
		return uploadError(ErrInvalidRequest, "source image dimensions are invalid or exceed %d pixels", maxPreviewSourcePixels)
	}
	if _, err := source.Seek(0, 0); err != nil {
		return fmt.Errorf("rewind preview source image: %w", err)
	}
	decoded, _, err := image.Decode(source)
	if err != nil {
		return fmt.Errorf("decode preview source image: %w", err)
	}

	width, height := previewDimensions(config.Width, config.Height, options.MaxWidth, options.MaxHeight)
	preview := decoded
	if width != config.Width || height != config.Height {
		preview, err = resizePreviewBilinear(ctx, decoded, width, height)
		if err != nil {
			return err
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	// JPEG has no alpha channel. Composite transparent images onto white so
	// transparent pixels do not unexpectedly become black.
	flattened := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(flattened, flattened.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.Draw(flattened, flattened.Bounds(), preview, preview.Bounds().Min, draw.Over)
	if err := writeJPEGPreview(outputPath, flattened, options.JPEGQuality); err != nil {
		return fmt.Errorf("write compressed image preview: %w", err)
	}
	return nil
}

// GenerateVideoFirstFramePreview requests the first frame from a remote video
// snapshot service, resizes it without enlarging it, and writes a compressed
// JPEG preview. The default snapshot service is Aliyun OSS video/snapshot.
// Local video files are intentionally unsupported because Go's standard
// library does not include an MP4/H.264 decoder.
func GenerateVideoFirstFramePreview(
	ctx context.Context,
	sourcePath string,
	outputPath string,
	options MediaPreviewOptions,
) error {
	options, err := normalizeMediaPreviewOptions(options)
	if err != nil {
		return err
	}
	videoURL, remote, err := parseRemotePreviewURL(sourcePath)
	if err != nil {
		return err
	}
	if !remote {
		return uploadError(ErrUnsupportedType, "video preview requires an HTTP/HTTPS URL supported by a video snapshot service")
	}
	builder := options.VideoSnapshotURL
	if builder == nil {
		builder = buildAliyunOSSVideoSnapshotURL
	}
	snapshotURL, err := builder(videoURL)
	if err != nil {
		return fmt.Errorf("build video snapshot URL: %w", err)
	}
	if snapshotURL == nil {
		return uploadError(ErrInvalidRequest, "video snapshot URL builder returned no URL")
	}
	maxBytes := options.MaxRemoteBytes
	if maxBytes == 0 {
		maxBytes = defaultRemoteImageMaxBytes
	}
	client := options.RemoteHTTPClient
	if client == nil {
		client = &http.Client{Timeout: defaultRemoteRequestTimeout}
	}
	snapshotPath, cleanup, err := downloadPreviewSource(
		ctx, client, snapshotURL, maxBytes, filepath.Dir(outputPath),
	)
	if err != nil {
		return fmt.Errorf("download first video frame: %w", err)
	}
	defer cleanup()
	if err := GenerateCompressedImagePreview(ctx, snapshotPath, outputPath, options); err != nil {
		return fmt.Errorf("compress first video frame: %w", err)
	}
	return nil
}

func buildAliyunOSSVideoSnapshotURL(source *url.URL) (*url.URL, error) {
	if source == nil || (source.Scheme != "http" && source.Scheme != "https") || source.Host == "" {
		return nil, uploadError(ErrInvalidRequest, "Aliyun OSS video snapshot requires an HTTP/HTTPS source URL")
	}
	snapshot := *source
	query := snapshot.Query()
	query.Set("x-oss-process", "video/snapshot,t_0,f_jpg,m_fast")
	snapshot.RawQuery = query.Encode()
	return &snapshot, nil
}

func normalizeMediaPreviewOptions(options MediaPreviewOptions) (MediaPreviewOptions, error) {
	defaults := DefaultMediaPreviewOptions()
	if options.MaxWidth < 0 || options.MaxHeight < 0 || options.JPEGQuality < 0 || options.MaxRemoteBytes < 0 {
		return MediaPreviewOptions{}, uploadError(ErrInvalidRequest, "preview dimensions, JPEG quality and remote size limit cannot be negative")
	}
	if options.MaxWidth == 0 {
		options.MaxWidth = defaults.MaxWidth
	}
	if options.MaxHeight == 0 {
		options.MaxHeight = defaults.MaxHeight
	}
	if options.JPEGQuality == 0 {
		options.JPEGQuality = defaults.JPEGQuality
	}
	if options.JPEGQuality > 100 {
		return MediaPreviewOptions{}, uploadError(ErrInvalidRequest, "JPEG quality must be between 1 and 100")
	}
	return options, nil
}

func preparePreviewSource(
	ctx context.Context,
	source string,
	outputPath string,
	options MediaPreviewOptions,
	defaultMaxBytes int64,
) (string, func(), error) {
	parsed, remote, err := parseRemotePreviewURL(source)
	if err != nil {
		return "", func() {}, err
	}
	if !remote {
		return source, func() {}, nil
	}
	maxBytes := options.MaxRemoteBytes
	if maxBytes == 0 {
		maxBytes = defaultMaxBytes
	}
	client := options.RemoteHTTPClient
	if client == nil {
		client = &http.Client{Timeout: defaultRemoteRequestTimeout}
	}
	return downloadPreviewSource(ctx, client, parsed, maxBytes, filepath.Dir(outputPath))
}

func parseRemotePreviewURL(source string) (*url.URL, bool, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return nil, false, uploadError(ErrInvalidRequest, "preview source is required")
	}
	parsed, err := url.Parse(source)
	if err != nil {
		return nil, false, uploadError(ErrInvalidRequest, "invalid remote preview source URL")
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		if parsed.Host == "" {
			return nil, false, uploadError(ErrInvalidRequest, "remote preview source URL must include a host")
		}
		return parsed, true, nil
	case "":
		return nil, false, nil
	default:
		// A Windows path such as C:\\media\\file.mp4 is parsed with a scheme
		// but remains a local path because it does not contain ://.
		if !strings.Contains(source, "://") {
			return nil, false, nil
		}
		return nil, false, uploadError(ErrUnsupportedType, "remote preview sources must use HTTP or HTTPS")
	}
}

func downloadPreviewSource(
	ctx context.Context,
	client *http.Client,
	remoteURL *url.URL,
	maxBytes int64,
	temporaryDirectory string,
) (string, func(), error) {
	if client == nil || remoteURL == nil || maxBytes <= 0 {
		return "", func() {}, uploadError(ErrInvalidRequest, "remote preview client, URL and positive size limit are required")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, remoteURL.String(), nil)
	if err != nil {
		return "", func() {}, fmt.Errorf("create remote preview request: %w", err)
	}
	response, err := client.Do(request)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", func() {}, ctxErr
		}
		return "", func() {}, fmt.Errorf("download remote preview source: %w", err)
	}
	if response == nil {
		return "", func() {}, errors.New("download remote preview source: HTTP client returned no response")
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", func() {}, fmt.Errorf("download remote preview source: unexpected HTTP status %s", response.Status)
	}
	if response.ContentLength > maxBytes {
		return "", func() {}, uploadError(ErrFileTooLarge, "remote preview source exceeds the %d byte limit", maxBytes)
	}

	if strings.TrimSpace(temporaryDirectory) == "" {
		return "", func() {}, uploadError(ErrInvalidRequest, "remote preview temporary directory is required")
	}
	if err := os.MkdirAll(temporaryDirectory, 0o750); err != nil {
		return "", func() {}, fmt.Errorf("create remote preview temporary directory: %w", err)
	}
	temp, err := os.CreateTemp(temporaryDirectory, remotePreviewTempPattern(remoteURL.Path))
	if err != nil {
		return "", func() {}, fmt.Errorf("create remote preview temporary file: %w", err)
	}
	tempPath := temp.Name()
	cleanup := func() { _ = os.Remove(tempPath) }
	remoteBody := io.Reader(response.Body)
	if maxBytes < int64(^uint64(0)>>1) {
		remoteBody = io.LimitReader(response.Body, maxBytes+1)
	}
	copied, copyErr := copyWithContext(ctx, temp, remoteBody)
	if copyErr == nil && copied > maxBytes {
		copyErr = uploadError(ErrFileTooLarge, "remote preview source exceeds the %d byte limit", maxBytes)
	}
	if copyErr == nil && copied == 0 {
		copyErr = uploadError(ErrInvalidRequest, "remote preview source is empty")
	}
	if copyErr == nil {
		copyErr = temp.Sync()
	}
	if closeErr := temp.Close(); copyErr == nil {
		copyErr = closeErr
	}
	if copyErr != nil {
		cleanup()
		return "", func() {}, fmt.Errorf("store remote preview source: %w", copyErr)
	}
	return tempPath, cleanup, nil
}

func remotePreviewTempPattern(remotePath string) string {
	extension := strings.ToLower(filepath.Ext(remotePath))
	if len(extension) > 10 {
		extension = ""
	}
	for _, character := range strings.TrimPrefix(extension, ".") {
		if (character < 'a' || character > 'z') && (character < '0' || character > '9') {
			extension = ""
			break
		}
	}
	return ".remote-preview-*" + extension
}

func preparePreviewPaths(sourcePath, outputPath string) (string, string, error) {
	if strings.TrimSpace(sourcePath) == "" || strings.TrimSpace(outputPath) == "" {
		return "", "", uploadError(ErrInvalidRequest, "preview source and output paths are required")
	}
	if _, remote, err := parseRemotePreviewURL(outputPath); err != nil || remote {
		return "", "", uploadError(ErrInvalidRequest, "preview output must be a local file path")
	}
	sourcePath, err := filepath.Abs(sourcePath)
	if err != nil {
		return "", "", fmt.Errorf("resolve preview source path: %w", err)
	}
	outputPath, err = filepath.Abs(outputPath)
	if err != nil {
		return "", "", fmt.Errorf("resolve preview output path: %w", err)
	}
	if filepath.Clean(sourcePath) == filepath.Clean(outputPath) {
		return "", "", uploadError(ErrInvalidRequest, "preview output must differ from its source")
	}
	info, err := os.Stat(sourcePath)
	if err != nil {
		return "", "", fmt.Errorf("inspect preview source: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", "", uploadError(ErrInvalidRequest, "preview source must be a regular file")
	}
	return sourcePath, outputPath, nil
}

func previewDimensions(width, height, maxWidth, maxHeight int) (int, int) {
	scale := math.Min(float64(maxWidth)/float64(width), float64(maxHeight)/float64(height))
	if scale >= 1 {
		return width, height
	}
	return max(1, int(math.Round(float64(width)*scale))), max(1, int(math.Round(float64(height)*scale)))
}

func resizePreviewBilinear(ctx context.Context, source image.Image, width, height int) (*image.NRGBA, error) {
	bounds := source.Bounds()
	result := image.NewNRGBA(image.Rect(0, 0, width, height))
	xScale := float64(bounds.Dx()) / float64(width)
	yScale := float64(bounds.Dy()) / float64(height)

	for y := 0; y < height; y++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		sourceY := float64(bounds.Min.Y) + (float64(y)+0.5)*yScale - 0.5
		y0, y1, yWeight := interpolationCoordinates(sourceY, bounds.Min.Y, bounds.Max.Y-1)
		for x := 0; x < width; x++ {
			sourceX := float64(bounds.Min.X) + (float64(x)+0.5)*xScale - 0.5
			x0, x1, xWeight := interpolationCoordinates(sourceX, bounds.Min.X, bounds.Max.X-1)
			c00 := color.NRGBAModel.Convert(source.At(x0, y0)).(color.NRGBA)
			c10 := color.NRGBAModel.Convert(source.At(x1, y0)).(color.NRGBA)
			c01 := color.NRGBAModel.Convert(source.At(x0, y1)).(color.NRGBA)
			c11 := color.NRGBAModel.Convert(source.At(x1, y1)).(color.NRGBA)
			result.SetNRGBA(x, y, color.NRGBA{
				R: interpolateChannel(c00.R, c10.R, c01.R, c11.R, xWeight, yWeight),
				G: interpolateChannel(c00.G, c10.G, c01.G, c11.G, xWeight, yWeight),
				B: interpolateChannel(c00.B, c10.B, c01.B, c11.B, xWeight, yWeight),
				A: interpolateChannel(c00.A, c10.A, c01.A, c11.A, xWeight, yWeight),
			})
		}
	}
	return result, nil
}

func interpolationCoordinates(value float64, minimum, maximum int) (int, int, float64) {
	base := int(math.Floor(value))
	weight := value - float64(base)
	if base < minimum {
		return minimum, minimum, 0
	}
	if base >= maximum {
		return maximum, maximum, 0
	}
	return base, base + 1, weight
}

func interpolateChannel(c00, c10, c01, c11 uint8, xWeight, yWeight float64) uint8 {
	top := float64(c00)*(1-xWeight) + float64(c10)*xWeight
	bottom := float64(c01)*(1-xWeight) + float64(c11)*xWeight
	return uint8(math.Round(top*(1-yWeight) + bottom*yWeight))
}

func writeJPEGPreview(path string, source image.Image, quality int) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".image-preview-*.jpg")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := jpeg.Encode(temp, source, &jpeg.Options{Quality: quality}); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return replacePreviewFile(tempPath, path)
}

func replacePreviewFile(sourcePath, outputPath string) error {
	if err := os.Rename(sourcePath, outputPath); err == nil {
		return nil
	}
	if err := os.Remove(outputPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Rename(sourcePath, outputPath)
}
