package adjustevent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ai-video/internal/config"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const pendingEventTable = "video_adjust_pending_event"

const (
	pendingEventStatusPending    = uint8(1)
	pendingEventStatusProcessing = uint8(2)
	pendingEventStatusProcessed  = uint8(3)
)

type pendingEventRow struct {
	ID         uint64         `gorm:"column:id;primaryKey"`
	EventID    string         `gorm:"column:event_id"`
	UserID     uint64         `gorm:"column:user_id"`
	Action     string         `gorm:"column:action"`
	ChannelID  uint64         `gorm:"column:channel_id"`
	OrderNo    string         `gorm:"column:order_no"`
	Status     uint8          `gorm:"column:status"`
	Payload    string         `gorm:"column:payload"`
	RequeuedAt *time.Time     `gorm:"column:requeued_at"`
	CreatedAt  time.Time      `gorm:"column:created_at"`
	UpdatedAt  time.Time      `gorm:"column:updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"column:deleted_at"`
}

func (*pendingEventRow) TableName() string { return pendingEventTable }

type pendingRepository struct{}

func (pendingRepository) Save(ctx context.Context, message Message) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("encode pending Adjust event: %w", err)
	}
	now := time.Now()
	row := &pendingEventRow{
		EventID: message.EventID, UserID: message.UserID, Action: string(message.Action),
		ChannelID: message.ChannelID, OrderNo: message.OrderNo, Status: pendingEventStatusPending,
		Payload:   string(payload),
		CreatedAt: now, UpdatedAt: now,
	}
	return config.DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "event_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"user_id": row.UserID, "action": row.Action, "channel_id": row.ChannelID,
			"order_no": row.OrderNo, "status": pendingEventStatusPending, "payload": row.Payload,
			"requeued_at": nil, "updated_at": now, "deleted_at": nil,
		}),
	}).Create(row).Error
}

func (pendingRepository) ListByUser(ctx context.Context, userID uint64, limit int) ([]Message, error) {
	return pendingRepository{}.list(ctx,
		"user_id = ? AND status = ? AND requeued_at IS NULL AND deleted_at IS NULL",
		[]any{userID, pendingEventStatusPending}, limit,
	)
}

func (pendingRepository) ListUnqueued(ctx context.Context, limit int) ([]Message, error) {
	return pendingRepository{}.list(ctx,
		"status = ? AND requeued_at IS NULL AND deleted_at IS NULL",
		[]any{pendingEventStatusPending}, limit,
	)
}

func (pendingRepository) list(ctx context.Context, where string, args []any, limit int) ([]Message, error) {
	if limit <= 0 {
		limit = 100
	}
	rows := make([]pendingEventRow, 0)
	db := config.DB.WithContext(ctx).Table(pendingEventTable).Where(where, args...).Order("id ASC").Limit(limit)
	if err := db.Find(&rows).Error; err != nil {
		return nil, err
	}
	messages := make([]Message, 0, len(rows))
	for _, row := range rows {
		var message Message
		if err := json.Unmarshal([]byte(row.Payload), &message); err != nil {
			return nil, fmt.Errorf("decode pending Adjust event %s: %w", row.EventID, err)
		}
		messages = append(messages, message)
	}
	return messages, nil
}

func (pendingRepository) MarkRequeued(ctx context.Context, eventID string, at time.Time) error {
	return config.DB.WithContext(ctx).Table(pendingEventTable).
		Where("event_id = ?", eventID).
		Updates(map[string]any{
			"status": gorm.Expr(
				"CASE WHEN status = ? THEN ? ELSE status END",
				pendingEventStatusPending, pendingEventStatusProcessing,
			),
			"requeued_at": at,
			"updated_at":  at,
		}).Error
}

// Delete marks a persisted event as processed and soft-deletes it so its
// lifecycle remains available for operational inspection.
func (pendingRepository) Delete(ctx context.Context, eventID string) error {
	now := time.Now()
	return config.DB.WithContext(ctx).Table(pendingEventTable).
		Where("event_id = ? AND deleted_at IS NULL", eventID).
		Updates(map[string]any{
			"status": pendingEventStatusProcessed, "updated_at": now, "deleted_at": now,
		}).Error
}
