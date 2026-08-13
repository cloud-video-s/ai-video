package generation

import (
	"context"
	"errors"
	"fmt"

	"ai-video/internal/config"
)

const (
	defaultDownloadConcurrency = 1
	defaultDownloadRetryCount  = 3
)

// generatedDownloadController limits complete generated-result download jobs.
// A job keeps its slot while downloading every URL and performing configured
// retries, so queued jobs cannot interleave with the job ahead of them.
type generatedDownloadController struct {
	slots      chan struct{}
	retryCount int
}

func newGeneratedDownloadController(concurrency, retryCount int) *generatedDownloadController {
	if concurrency <= 0 {
		concurrency = defaultDownloadConcurrency
	}
	if retryCount < 0 {
		retryCount = 0
	}
	return &generatedDownloadController{
		slots:      make(chan struct{}, concurrency),
		retryCount: retryCount,
	}
}

func configuredGeneratedDownloadController() *generatedDownloadController {
	concurrency := config.Cfg.Task.DownloadConcurrency
	retryCount := config.Cfg.Task.DownloadRetryCount
	// A zero concurrency means config has not been initialized (primarily in
	// focused unit tests). Keep the same safe defaults used by Viper.
	if concurrency <= 0 {
		concurrency = defaultDownloadConcurrency
		retryCount = defaultDownloadRetryCount
	}
	return newGeneratedDownloadController(concurrency, retryCount)
}

func (c *generatedDownloadController) run(ctx context.Context, operation func(retryCount int) error) error {
	if c == nil || operation == nil {
		return errors.New("generated download controller is not configured")
	}
	select {
	case c.slots <- struct{}{}:
		defer func() { <-c.slots }()
	case <-ctx.Done():
		return ctx.Err()
	}
	return operation(c.retryCount)
}

// downloadRetryExhaustedError identifies a download that used its initial
// attempt and every configured retry. Callers use it to move the generation
// task to a terminal failure state instead of polling it forever.
type downloadRetryExhaustedError struct {
	Attempts int
	Err      error
}

func (e *downloadRetryExhaustedError) Error() string {
	return fmt.Sprintf("download generated result failed after %d attempts: %v", e.Attempts, e.Err)
}

func (e *downloadRetryExhaustedError) Unwrap() error { return e.Err }

func retryDownload(ctx context.Context, retryCount int, operation func() error) error {
	if operation == nil {
		return errors.New("download operation is required")
	}
	if retryCount < 0 {
		retryCount = 0
	}
	var lastErr error
	for attempt := 1; attempt <= retryCount+1; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		lastErr = operation()
		if lastErr == nil {
			return nil
		}
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	return &downloadRetryExhaustedError{Attempts: retryCount + 1, Err: lastErr}
}

func isDownloadRetryExhausted(err error) bool {
	var exhausted *downloadRetryExhaustedError
	return errors.As(err, &exhausted)
}

// downloadController is initialized lazily because the shared Manager is
// constructed before application configuration is loaded.
func (m *Manager) downloadController() *generatedDownloadController {
	m.downloadMu.Lock()
	defer m.downloadMu.Unlock()
	if m.downloads == nil {
		m.downloads = configuredGeneratedDownloadController()
	}
	return m.downloads
}
