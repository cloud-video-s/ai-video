package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"ai-video/internal/gen/model"

	"gorm.io/gorm"
)

// AdjustMediaAdsRepo persists the Adjust tracker hierarchy. Tracker tokens are
// stable identifiers, while pid references the local ID returned by this
// repository for the parent tracker.
type AdjustMediaAdsRepo struct{}

func NewAdjustMediaAdsRepo() *AdjustMediaAdsRepo {
	return &AdjustMediaAdsRepo{}
}

// SelectRootForSync chooses an unseen root first, then the least recently
// updated root. The existing updated_at column makes the rotation survive
// process restarts and keeps multiple application instances on shared state.
func (r *AdjustMediaAdsRepo) SelectRootForSync(ctx context.Context, tokens []string) (string, error) {
	rootTokens := make([]string, 0, len(tokens))
	seen := make(map[string]struct{}, len(tokens))
	for _, token := range tokens {
		token = strings.TrimSpace(token)
		if token == "" {
			return "", errors.New("adjust root tracker token is required")
		}
		if _, duplicate := seen[token]; duplicate {
			continue
		}
		seen[token] = struct{}{}
		rootTokens = append(rootTokens, token)
	}
	if len(rootTokens) == 0 {
		return "", nil
	}

	q := qFrom(ctx).VideoAdjustMediaAd
	rows, err := q.WithContext(ctx).Unscoped().
		Select(q.Token, q.UpdatedAt).
		Where(q.Pid.Eq(0), q.Token.In(rootTokens...)).
		Find()
	if err != nil {
		return "", fmt.Errorf("find Adjust root tracker sync times: %w", err)
	}

	updatedByToken := make(map[string]time.Time, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		updatedAt, exists := updatedByToken[row.Token]
		if !exists || row.UpdatedAt.After(updatedAt) {
			updatedByToken[row.Token] = row.UpdatedAt
		}
	}

	var selected string
	var oldest time.Time
	for _, token := range rootTokens {
		updatedAt, exists := updatedByToken[token]
		if !exists {
			return token, nil
		}
		if selected == "" || updatedAt.Before(oldest) {
			selected = token
			oldest = updatedAt
		}
	}
	return selected, nil
}

// UpsertByToken creates a tracker the first time it is observed and updates it
// on subsequent synchronizations. A previously soft-deleted tracker is restored
// so its original ID (and therefore its children's pid values) remains stable.
func (r *AdjustMediaAdsRepo) UpsertByToken(ctx context.Context, item *model.VideoAdjustMediaAd) (uint64, error) {
	if item == nil {
		return 0, errors.New("adjust media ad is required")
	}
	item.Token = strings.TrimSpace(item.Token)
	if item.Token == "" {
		return 0, errors.New("adjust media ad token is required")
	}

	q := qFrom(ctx).VideoAdjustMediaAd
	// CREATE and UPDATE below are each a single atomic statement. Avoid GORM's
	// implicit transaction so a canceled task reports the real context error
	// instead of appending sql.ErrTxDone while the callback tries to commit.
	dao := q.WithContext(ctx).Unscoped().Session(&gorm.Session{SkipDefaultTransaction: true})
	existing, err := dao.Where(q.Token.Eq(item.Token)).Order(q.ID.Asc()).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if err = dao.Create(item); err != nil {
			return 0, fmt.Errorf("create adjust media ad %q: %w", item.Token, err)
		}
		return item.ID, nil
	}
	if err != nil {
		return 0, fmt.Errorf("find adjust media ad %q: %w", item.Token, err)
	}

	updates := map[string]any{
		"pid":               item.Pid,
		"name":              item.Name,
		"label":             item.Label,
		"level":             item.Level,
		"archived":          item.Archived,
		"has_subtrackers":   item.HasSubtrackers,
		"partner_id":        item.PartnerID,
		"cost_data_enabled": item.CostDataEnabled,
		"url":               item.URL,
		"click_url":         item.ClickURL,
		"impression_url":    item.ImpressionURL,
		"deleted_at":        nil,
	}
	if _, err = dao.Where(q.ID.Eq(existing.ID)).Updates(updates); err != nil {
		return 0, fmt.Errorf("update adjust media ad %q: %w", item.Token, err)
	}
	item.ID = existing.ID
	return existing.ID, nil
}
