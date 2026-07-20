package upload

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Manager struct {
	config   Config
	root     string
	policies map[MediaKind]normalizedPolicy
	locks    sync.Map
	now      func() time.Time
}

var (
	defaultManagerOnce sync.Once
	defaultManager     *Manager
	defaultManagerErr  error
)

func DefaultManager() (*Manager, error) {
	return SharedManager(DefaultConfig())
}

func SharedManager(config Config) (*Manager, error) {
	defaultManagerOnce.Do(func() {
		defaultManager, defaultManagerErr = NewManager(config)
	})
	return defaultManager, defaultManagerErr
}

func NewManager(config Config) (*Manager, error) {
	if strings.TrimSpace(config.RootDir) == "" || config.ChunkSize <= 0 || config.MaxBatchFiles <= 0 || config.SessionTTL <= 0 {
		return nil, uploadError(ErrInvalidRequest, "root directory, chunk size, batch limit and session TTL are required")
	}
	root, err := filepath.Abs(config.RootDir)
	if err != nil {
		return nil, fmt.Errorf("resolve upload root: %w", err)
	}
	imagePolicy, err := normalizePolicy(config.Image)
	if err != nil {
		return nil, fmt.Errorf("image policy: %w", err)
	}
	videoPolicy, err := normalizePolicy(config.Video)
	if err != nil {
		return nil, fmt.Errorf("video policy: %w", err)
	}
	if config.Storage == nil {
		config.Storage, err = NewLocalStorage(root, "/uploads")
		if err != nil {
			return nil, err
		}
	}
	return &Manager{
		config: config,
		root:   root,
		policies: map[MediaKind]normalizedPolicy{
			MediaImage: imagePolicy,
			MediaVideo: videoPolicy,
		},
		now: time.Now,
	}, nil
}

func (m *Manager) InitiateImages(ctx context.Context, files []FileSpec) ([]Session, error) {
	return m.InitiateBatch(ctx, MediaImage, files)
}

func (m *Manager) InitiateVideos(ctx context.Context, files []FileSpec) ([]Session, error) {
	return m.InitiateBatch(ctx, MediaVideo, files)
}

func (m *Manager) InitiateBatch(ctx context.Context, kind MediaKind, files []FileSpec) ([]Session, error) {
	return m.InitiateBatchForOwner(ctx, kind, files, UploadOwner{})
}

func (m *Manager) InitiateBatchForOwner(ctx context.Context, kind MediaKind, files []FileSpec, owner UploadOwner) ([]Session, error) {
	policy, err := m.resolvePolicy(kind)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, uploadError(ErrInvalidRequest, "at least one file is required")
	}
	if len(files) > m.config.MaxBatchFiles {
		return nil, uploadError(ErrBatchTooLarge, "maximum %d files per batch", m.config.MaxBatchFiles)
	}

	type preparedFile struct {
		name        string
		ext         string
		contentType string
		sha256      string
		size        int64
	}
	prepared := make([]preparedFile, len(files))
	for i, file := range files {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		name := safeBaseName(file.FileName)
		if name == "" || file.Size <= 0 {
			return nil, uploadError(ErrInvalidRequest, "file %d has an invalid name or size", i)
		}
		if file.Size > policy.maxFileSize {
			return nil, uploadError(ErrFileTooLarge, "%s exceeds the %d byte limit", name, policy.maxFileSize)
		}
		ext := strings.ToLower(filepath.Ext(name))
		if _, allowed := policy.exts[ext]; !allowed {
			return nil, uploadError(ErrUnsupportedType, "%s has disallowed extension %q", name, ext)
		}
		contentType := normalizeMIME(file.ContentType)
		if contentType != "" {
			if _, allowed := policy.mimes[contentType]; !allowed {
				return nil, uploadError(ErrUnsupportedType, "%s has disallowed content type %q", name, contentType)
			}
		}
		expectedSHA256, err := validateSHA256(file.SHA256)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		prepared[i] = preparedFile{name: name, ext: ext, contentType: contentType, sha256: expectedSHA256, size: file.Size}
	}

	now := m.now()
	sessions := make([]Session, 0, len(prepared))
	for _, file := range prepared {
		uploadID, err := newUploadID()
		if err != nil {
			m.removeSessions(sessions)
			return nil, err
		}
		totalChunks := int((file.size + m.config.ChunkSize - 1) / m.config.ChunkSize)
		session := Session{
			UploadID: uploadID, Kind: kind, OriginalName: file.name, Extension: file.ext,
			ContentType: file.contentType, TotalSize: file.size, ChunkSize: m.config.ChunkSize,
			TotalChunks: totalChunks, ExpectedSHA256: file.sha256,
			UploaderType: owner.Type, UploaderID: owner.ID,
			CreatedAt: now, ExpiresAt: now.Add(m.config.SessionTTL),
		}
		if err := m.saveSession(&session); err != nil {
			m.removeSessions(sessions)
			return nil, err
		}
		sessions = append(sessions, session)
	}
	return sessions, nil
}

func (m *Manager) Status(ctx context.Context, uploadID string) (*Session, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	session, err := m.loadSession(uploadID)
	if err != nil {
		return nil, err
	}
	if err := m.ensureActive(session); err != nil {
		return nil, err
	}
	if session.Completed {
		return session, nil
	}
	session.UploadedChunks, err = m.uploadedChunks(session)
	if err != nil {
		return nil, err
	}
	return session, nil
}

func (m *Manager) PutChunk(ctx context.Context, uploadID string, index int, reader io.Reader, expectedSHA256 string) (*Session, error) {
	if reader == nil {
		return nil, uploadError(ErrInvalidChunk, "chunk body is required")
	}
	checksum, err := validateSHA256(expectedSHA256)
	if err != nil {
		return nil, err
	}
	lock := m.uploadLock(uploadID)
	lock.Lock()
	defer lock.Unlock()

	session, err := m.loadSession(uploadID)
	if err != nil {
		return nil, err
	}
	if err := m.ensureActive(session); err != nil {
		return nil, err
	}
	if session.Completed || index < 0 || index >= session.TotalChunks {
		return nil, uploadError(ErrInvalidChunk, "chunk index %d is outside the upload range", index)
	}
	expectedSize := expectedChunkSize(session, index)
	chunkDir := m.chunkDir(uploadID)
	if err := os.MkdirAll(chunkDir, 0o750); err != nil {
		return nil, fmt.Errorf("create chunk directory: %w", err)
	}
	chunkPath := m.chunkPath(uploadID, index)
	if info, statErr := os.Stat(chunkPath); statErr == nil && info.Size() == expectedSize {
		if checksum == "" || fileSHA256Matches(chunkPath, checksum) {
			session.UploadedChunks, err = m.uploadedChunks(session)
			return session, err
		}
	}

	temp, err := os.CreateTemp(chunkDir, ".chunk-*")
	if err != nil {
		return nil, fmt.Errorf("create temporary chunk: %w", err)
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)

	hasher := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(temp, hasher), io.LimitReader(reader, expectedSize+1))
	if copyErr == nil {
		copyErr = ctx.Err()
	}
	if copyErr == nil && written != expectedSize {
		copyErr = uploadError(ErrInvalidChunk, "chunk %d has %d bytes, expected %d", index, written, expectedSize)
	}
	actualChecksum := hex.EncodeToString(hasher.Sum(nil))
	if copyErr == nil && checksum != "" && checksum != actualChecksum {
		copyErr = uploadError(ErrChecksumMismatch, "chunk %d checksum does not match", index)
	}
	if copyErr == nil {
		copyErr = temp.Sync()
	}
	if closeErr := temp.Close(); copyErr == nil {
		copyErr = closeErr
	}
	if copyErr != nil {
		return nil, copyErr
	}
	if err := os.Remove(chunkPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("replace chunk: %w", err)
	}
	if err := os.Rename(tempPath, chunkPath); err != nil {
		return nil, fmt.Errorf("store chunk: %w", err)
	}
	session.UploadedChunks, err = m.uploadedChunks(session)
	if err != nil {
		return nil, err
	}
	return session, nil
}

func (m *Manager) Complete(ctx context.Context, uploadID string) (*Session, error) {
	lock := m.uploadLock(uploadID)
	lock.Lock()
	defer lock.Unlock()

	session, err := m.loadSession(uploadID)
	if err != nil {
		return nil, err
	}
	if err := m.ensureActive(session); err != nil {
		return nil, err
	}
	if session.Completed {
		return session, nil
	}
	uploaded, err := m.uploadedChunks(session)
	if err != nil {
		return nil, err
	}
	if len(uploaded) != session.TotalChunks {
		return nil, uploadError(ErrMissingChunks, "%d of %d chunks are present", len(uploaded), session.TotalChunks)
	}

	mediaDir := "images"
	if session.Kind == MediaVideo {
		mediaDir = "videos"
	}
	objectKey := filepath.ToSlash(filepath.Join(
		mediaDir, session.CreatedAt.Format("2006"), session.CreatedAt.Format("01"), session.CreatedAt.Format("02"),
		session.UploadID+session.Extension,
	))
	mergeDir := filepath.Join(m.root, ".merging")
	if err := os.MkdirAll(mergeDir, 0o750); err != nil {
		return nil, fmt.Errorf("create upload merge directory: %w", err)
	}
	temp, err := os.CreateTemp(mergeDir, ".merge-*")
	if err != nil {
		return nil, fmt.Errorf("create merged upload: %w", err)
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	hasher := sha256.New()
	var mergedSize int64
	for index := 0; index < session.TotalChunks; index++ {
		if err := ctx.Err(); err != nil {
			temp.Close()
			return nil, err
		}
		chunk, openErr := os.Open(m.chunkPath(uploadID, index))
		if openErr != nil {
			temp.Close()
			return nil, fmt.Errorf("open chunk %d: %w", index, openErr)
		}
		copied, copyErr := io.Copy(io.MultiWriter(temp, hasher), chunk)
		closeErr := chunk.Close()
		if copyErr != nil {
			temp.Close()
			return nil, fmt.Errorf("merge chunk %d: %w", index, copyErr)
		}
		if closeErr != nil {
			temp.Close()
			return nil, closeErr
		}
		mergedSize += copied
	}
	if mergedSize != session.TotalSize {
		temp.Close()
		return nil, uploadError(ErrInvalidChunk, "merged file has %d bytes, expected %d", mergedSize, session.TotalSize)
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return nil, err
	}
	if err := temp.Close(); err != nil {
		return nil, err
	}
	actualSHA256 := hex.EncodeToString(hasher.Sum(nil))
	if session.ExpectedSHA256 != "" && actualSHA256 != session.ExpectedSHA256 {
		return nil, uploadError(ErrChecksumMismatch, "completed file checksum does not match")
	}
	header, err := readHeader(tempPath, 512)
	if err != nil {
		return nil, err
	}
	policy, err := m.resolvePolicy(session.Kind)
	if err != nil {
		return nil, err
	}
	if err := validateSessionPolicy(session, policy); err != nil {
		return nil, err
	}
	detectedType, err := detectAndValidateContent(session.Kind, session.Extension, header, policy)
	if err != nil {
		return nil, err
	}
	stored, err := m.config.Storage.Store(ctx, objectKey, tempPath, detectedType)
	if err != nil {
		return nil, err
	}

	session.ContentType = detectedType
	session.SHA256 = actualSHA256
	session.StorageProvider = stored.Provider
	session.Completed = true
	session.UploadedChunks = allChunkIndexes(session.TotalChunks)
	session.FilePath = stored.Path
	session.FileURL = stored.URL
	if err := m.saveSession(session); err != nil {
		return nil, err
	}
	if err := os.RemoveAll(m.chunkDir(uploadID)); err != nil {
		return nil, fmt.Errorf("remove completed chunks: %w", err)
	}
	return session, nil
}

func (m *Manager) CleanupExpired(ctx context.Context) (int, error) {
	entries, err := os.ReadDir(m.sessionDir())
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	removed := 0
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return removed, err
		}
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		uploadID := strings.TrimSuffix(entry.Name(), ".json")
		session, loadErr := m.loadSession(uploadID)
		if loadErr != nil || session.Completed || m.now().Before(session.ExpiresAt) {
			continue
		}
		lock := m.uploadLock(uploadID)
		lock.Lock()
		removeErr := os.RemoveAll(m.chunkDir(uploadID))
		if removeErr == nil {
			removeErr = os.Remove(m.sessionPath(uploadID))
		}
		lock.Unlock()
		if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return removed, removeErr
		}
		removed++
	}
	return removed, nil
}

func (m *Manager) ensureActive(session *Session) error {
	if !session.Completed && !m.now().Before(session.ExpiresAt) {
		return uploadError(ErrUploadExpired, "upload %s expired at %s", session.UploadID, session.ExpiresAt.Format(time.RFC3339))
	}
	return nil
}

func (m *Manager) uploadedChunks(session *Session) ([]int, error) {
	entries, err := os.ReadDir(m.chunkDir(session.UploadID))
	if errors.Is(err, os.ErrNotExist) {
		return []int{}, nil
	}
	if err != nil {
		return nil, err
	}
	uploaded := make([]int, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".part" {
			continue
		}
		index, parseErr := strconv.Atoi(strings.TrimSuffix(entry.Name(), ".part"))
		if parseErr != nil || index < 0 || index >= session.TotalChunks {
			continue
		}
		info, statErr := entry.Info()
		if statErr == nil && info.Size() == expectedChunkSize(session, index) {
			uploaded = append(uploaded, index)
		}
	}
	sort.Ints(uploaded)
	return uploaded, nil
}

func (m *Manager) saveSession(session *Session) error {
	if err := os.MkdirAll(m.sessionDir(), 0o750); err != nil {
		return fmt.Errorf("create session directory: %w", err)
	}
	temp, err := os.CreateTemp(m.sessionDir(), ".session-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	encoder := json.NewEncoder(temp)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(session); err != nil {
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
	target := m.sessionPath(session.UploadID)
	if err := os.Rename(tempPath, target); err != nil {
		if removeErr := os.Remove(target); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return err
		}
		if retryErr := os.Rename(tempPath, target); retryErr != nil {
			return retryErr
		}
	}
	return nil
}

func (m *Manager) loadSession(uploadID string) (*Session, error) {
	if !validUploadID(uploadID) {
		return nil, uploadError(ErrUploadNotFound, "invalid upload id")
	}
	file, err := os.Open(m.sessionPath(uploadID))
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrUploadNotFound
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var session Session
	if err := json.NewDecoder(file).Decode(&session); err != nil {
		return nil, fmt.Errorf("decode upload session: %w", err)
	}
	if session.UploadID != uploadID {
		return nil, ErrUploadNotFound
	}
	return &session, nil
}

func (m *Manager) removeSessions(sessions []Session) {
	for i := range sessions {
		_ = os.Remove(m.sessionPath(sessions[i].UploadID))
		_ = os.RemoveAll(m.chunkDir(sessions[i].UploadID))
	}
}

func (m *Manager) uploadLock(uploadID string) *sync.Mutex {
	value, _ := m.locks.LoadOrStore(uploadID, &sync.Mutex{})
	return value.(*sync.Mutex)
}

func (m *Manager) resolvePolicy(kind MediaKind) (normalizedPolicy, error) {
	if m.config.PolicyResolver == nil {
		policy, ok := m.policies[kind]
		if !ok {
			return normalizedPolicy{}, uploadError(ErrInvalidRequest, "unknown media kind %q", kind)
		}
		return policy, nil
	}
	policy, err := m.config.PolicyResolver(kind)
	if err != nil {
		return normalizedPolicy{}, err
	}
	normalized, err := normalizePolicy(policy)
	if err != nil {
		return normalizedPolicy{}, fmt.Errorf("%s policy: %w", kind, err)
	}
	return normalized, nil
}

func validateSessionPolicy(session *Session, policy normalizedPolicy) error {
	if session.TotalSize > policy.maxFileSize {
		return uploadError(ErrFileTooLarge, "%s exceeds the %d byte limit", session.OriginalName, policy.maxFileSize)
	}
	if _, allowed := policy.exts[strings.ToLower(session.Extension)]; !allowed {
		return uploadError(ErrUnsupportedType, "%s has disallowed extension %q", session.OriginalName, session.Extension)
	}
	return nil
}

func (m *Manager) sessionDir() string           { return filepath.Join(m.root, ".sessions") }
func (m *Manager) sessionPath(id string) string { return filepath.Join(m.sessionDir(), id+".json") }
func (m *Manager) chunkDir(id string) string    { return filepath.Join(m.root, ".chunks", id) }
func (m *Manager) chunkPath(id string, index int) string {
	return filepath.Join(m.chunkDir(id), fmt.Sprintf("%08d.part", index))
}

func newUploadID() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate upload id: %w", err)
	}
	return hex.EncodeToString(buffer), nil
}

func validUploadID(uploadID string) bool {
	if len(uploadID) != 32 {
		return false
	}
	decoded, err := hex.DecodeString(uploadID)
	return err == nil && len(decoded) == 16
}

func expectedChunkSize(session *Session, index int) int64 {
	if index < session.TotalChunks-1 {
		return session.ChunkSize
	}
	return session.TotalSize - int64(index)*session.ChunkSize
}

func fileSHA256Matches(path, expected string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return false
	}
	return hex.EncodeToString(hasher.Sum(nil)) == expected
}

func readHeader(path string, limit int64) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return io.ReadAll(io.LimitReader(file, limit))
}

func allChunkIndexes(total int) []int {
	indexes := make([]int, total)
	for i := range indexes {
		indexes[i] = i
	}
	return indexes
}
