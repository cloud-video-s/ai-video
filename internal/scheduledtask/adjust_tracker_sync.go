package scheduledtask

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"ai-video/internal/gen/model"
	"ai-video/internal/pkg/adjust"
	"ai-video/internal/pkg/task"
	"ai-video/internal/repository"
)

const (
	TypeSyncAdjustTrackers = "adjust:trackers:sync"
	AdjustTrackerSyncCron  = "*/15 * * * *"

	adjustTrackerSyncUniqueTTL   = 120 * time.Minute
	adjustTrackerRequestInterval = 100 * time.Millisecond
)

type adjustTrackerClient interface {
	GetTrackers(context.Context, string, adjust.ListOptions) (*adjust.TrackersResponse, error)
	GetTrackerChildren(context.Context, string, string, adjust.ListOptions) (*adjust.TrackersResponse, error)
}

type adjustTrackerRepository interface {
	UpsertByToken(context.Context, *model.VideoAdjustMediaAd) (uint64, error)
	SelectRootForSync(context.Context, []string) (string, error)
}

// AdjustTrackerSyncResult summarizes one root hierarchy synchronization.
type AdjustTrackerSyncResult struct {
	RootToken string
	Requests  int
	Trackers  int
}

type adjustTrackerSyncHandler struct {
	client          adjustTrackerClient
	repository      adjustTrackerRepository
	appToken        string
	onCompleted     func(context.Context, AdjustTrackerSyncResult)
	requestInterval time.Duration
	running         atomic.Bool
}

func (h *adjustTrackerSyncHandler) Handle(ctx context.Context, _ []byte) error {
	// A failed recurring task can coexist with a later cron enqueue while Asynq
	// is retrying it. Discard duplicates instead of letting them contend for the
	// same rows and multiply Campaign API calls.
	if !h.running.CompareAndSwap(false, true) {
		return nil
	}
	defer h.running.Store(false)

	result := AdjustTrackerSyncResult{}
	trackers, err := h.fetchPages(ctx, &result, func(ctx context.Context, cursor string) (*adjust.TrackersResponse, error) {
		return h.client.GetTrackers(ctx, h.appToken, adjust.ListOptions{Cursor: cursor})
	})
	if err == nil {
		err = h.syncNextRoot(ctx, &result, trackers)
	}
	if err != nil {
		return fmt.Errorf("sync Adjust tracker hierarchy: %w", err)
	}
	if h.onCompleted != nil {
		h.onCompleted(ctx, result)
	}
	return nil
}

type adjustTrackerPageFetcher func(context.Context, string) (*adjust.TrackersResponse, error)

func (h *adjustTrackerSyncHandler) fetchPages(ctx context.Context, result *AdjustTrackerSyncResult, fetch adjustTrackerPageFetcher) ([]adjust.Tracker, error) {
	cursor := ""
	seenCursors := make(map[string]struct{})
	var trackers []adjust.Tracker
	for {
		if err := waitForAdjustTrackerRequest(ctx, h.requestInterval); err != nil {
			return nil, err
		}
		response, err := fetch(ctx, cursor)
		result.Requests++
		if err != nil {
			return nil, err
		}
		if response == nil {
			return nil, errors.New("Adjust tracker API returned an empty response")
		}
		trackers = append(trackers, response.Data.Items...)

		nextCursor := strings.TrimSpace(response.Data.Paging.Cursor)
		if nextCursor == "" {
			return trackers, nil
		}
		if nextCursor == cursor {
			return nil, fmt.Errorf("Adjust tracker API repeated cursor %q", nextCursor)
		}
		if _, duplicate := seenCursors[nextCursor]; duplicate {
			return nil, fmt.Errorf("Adjust tracker API returned cursor cycle at %q", nextCursor)
		}
		seenCursors[nextCursor] = struct{}{}
		cursor = nextCursor
	}
}

func waitForAdjustTrackerRequest(ctx context.Context, interval time.Duration) error {
	if interval <= 0 {
		return nil
	}
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (h *adjustTrackerSyncHandler) syncNextRoot(ctx context.Context, result *AdjustTrackerSyncResult, trackers []adjust.Tracker) error {
	if len(trackers) == 0 {
		return nil
	}

	rootsByToken := make(map[string]adjust.Tracker, len(trackers))
	rootTokens := make([]string, 0, len(trackers))
	for _, tracker := range trackers {
		token := strings.TrimSpace(tracker.Token)
		if token == "" {
			return errors.New("Adjust tracker token is required")
		}
		if _, duplicate := rootsByToken[token]; duplicate {
			continue
		}
		rootsByToken[token] = tracker
		rootTokens = append(rootTokens, token)
	}

	selectedToken, err := h.repository.SelectRootForSync(ctx, rootTokens)
	if err != nil {
		return fmt.Errorf("select Adjust root tracker for sync: %w", err)
	}
	selectedToken = strings.TrimSpace(selectedToken)
	selected, exists := rootsByToken[selectedToken]
	if !exists {
		return fmt.Errorf("selected Adjust root tracker %q is not in the current root list", selectedToken)
	}

	result.RootToken = selectedToken
	return h.syncTracker(ctx, 0, nil, result, selected)
}

func (h *adjustTrackerSyncHandler) syncTracker(ctx context.Context, parentID uint64, ancestors map[string]struct{}, result *AdjustTrackerSyncResult, tracker adjust.Tracker) error {
	token := strings.TrimSpace(tracker.Token)
	if _, duplicate := ancestors[token]; duplicate {
		return fmt.Errorf("Adjust tracker token cycle at %q", token)
	}

	row, err := adjustTrackerRow(parentID, tracker)
	if err != nil {
		return err
	}
	if row == nil {
		return fmt.Errorf("Adjust tracker %q produced an empty database row", token)
	}
	row.ID, err = h.repository.UpsertByToken(ctx, row)
	if err != nil {
		return fmt.Errorf("upsert Adjust tracker %q: %w", token, err)
	}
	if row.ID == 0 {
		return fmt.Errorf("upsert Adjust tracker %q returned an empty ID", token)
	}
	result.Trackers++
	if !tracker.HasSubtrackers {
		return nil
	}

	children, err := h.fetchPages(ctx, result, func(ctx context.Context, cursor string) (*adjust.TrackersResponse, error) {
		return h.client.GetTrackerChildren(ctx, h.appToken, token, adjust.ListOptions{Cursor: cursor})
	})
	if err != nil {
		return fmt.Errorf("sync children of Adjust tracker %q: %w", token, err)
	}

	childAncestors := cloneTrackerTokens(ancestors)
	childAncestors[token] = struct{}{}
	for _, child := range children {
		if err = h.syncTracker(ctx, row.ID, childAncestors, result, child); err != nil {
			return err
		}
	}
	return nil
}

func adjustTrackerRow(parentID uint64, tracker adjust.Tracker) (*model.VideoAdjustMediaAd, error) {
	if tracker.Level < 0 || uint64(tracker.Level) > uint64(^uint32(0)) {
		return nil, fmt.Errorf("Adjust tracker %q has invalid level %d", tracker.Token, tracker.Level)
	}
	partnerID := uint32(0)
	if tracker.PartnerID != "" {
		id, _ := strconv.ParseInt(tracker.PartnerID, 10, 64)
		partnerID = uint32(id)
	}

	return &model.VideoAdjustMediaAd{
		Pid:             parentID,
		Name:            tracker.Name,
		Token:           strings.TrimSpace(tracker.Token),
		Label:           tracker.Label,
		Level:           uint32(tracker.Level),
		Archived:        boolUint32(tracker.Archived),
		HasSubtrackers:  boolUint32(tracker.HasSubtrackers),
		PartnerID:       partnerID,
		CostDataEnabled: boolUint32(tracker.CostDataEnabled),
		URL:             tracker.URL,
		ClickURL:        tracker.ClickURL,
		ImpressionURL:   tracker.ImpressionURL,
	}, nil
}

func boolUint32(value bool) uint32 {
	if value {
		return 1
	}
	return 0
}

func cloneTrackerTokens(source map[string]struct{}) map[string]struct{} {
	result := make(map[string]struct{}, len(source)+1)
	for token := range source {
		result[token] = struct{}{}
	}
	return result
}

// RegisterAdjustTrackerSync registers the shared worker handler and its
// fifteen-minute recurring enqueue. The uniqueness window prevents parallel app
// instances from enqueueing the same run at the same instant.
func RegisterAdjustTrackerSync(manager *task.Manager, scheduler *task.Scheduler, client adjustTrackerClient, appToken string, onCompleted func(context.Context, AdjustTrackerSyncResult)) error {
	if manager == nil || manager.Worker == nil {
		return errors.New("task manager and worker are required")
	}
	if scheduler == nil {
		return errors.New("task scheduler is required")
	}
	if client == nil {
		return errors.New("Adjust tracker client is required")
	}
	appToken = strings.TrimSpace(appToken)
	if appToken == "" {
		return errors.New("Adjust tracker app token is required")
	}

	handler := &adjustTrackerSyncHandler{
		client: client, repository: repository.NewAdjustMediaAdsRepo(),
		appToken: appToken, onCompleted: onCompleted,
		requestInterval: adjustTrackerRequestInterval,
	}
	manager.Worker.Handle(TypeSyncAdjustTrackers, handler.Handle)
	if _, err := scheduler.Register(task.CronTask{
		Cron: AdjustTrackerSyncCron, TypeName: TypeSyncAdjustTrackers,
		Payload: struct{}{}, Queue: "default", Unique: adjustTrackerSyncUniqueTTL,
	}); err != nil {
		return fmt.Errorf("register Adjust tracker sync schedule: %w", err)
	}
	return nil
}
