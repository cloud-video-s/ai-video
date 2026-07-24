// Package menuexport 提供独立、只读的 video_menu 数据快照生成能力。
package menuexport

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"ai-video/internal/gen/model"

	"gorm.io/gorm"
)

// Snapshot 是 video_menu 的只读快照，不包含任何 GORM 关系字段。
type Snapshot struct {
	SourceTable    string     `json:"source_table"`
	GeneratedAt    time.Time  `json:"generated_at"`
	IncludeDeleted bool       `json:"include_deleted"`
	Count          int        `json:"count"`
	Menus          []MenuItem `json:"menus"`
}

// MenuItem 对应 video_menu 的基础字段。
type MenuItem struct {
	ID         uint64     `gorm:"column:id" json:"id"`
	ParentID   uint64     `gorm:"column:parent_id" json:"parent_id"`
	Name       string     `gorm:"column:name" json:"name"`
	Path       string     `gorm:"column:path" json:"path"`
	Component  string     `gorm:"column:component" json:"component"`
	Icon       string     `gorm:"column:icon" json:"icon"`
	Sort       uint64     `gorm:"column:sort" json:"sort"`
	Type       uint8      `gorm:"column:type" json:"type"`
	Permission string     `gorm:"column:permission" json:"permission"`
	Visible    uint8      `gorm:"column:visible" json:"visible"`
	Status     uint8      `gorm:"column:status" json:"status"`
	CreatedAt  time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt  time.Time  `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt  *time.Time `gorm:"column:deleted_at" json:"deleted_at,omitempty"`
}

// Generate 从 video_menu 读取基础列并写出 JSON。
// 该方法只有 SELECT，不会创建、更新或删除数据库记录，也不会解析模型关联。
func Generate(ctx context.Context, db *gorm.DB, writer io.Writer, includeDeleted bool) (int, error) {
	if db == nil {
		return 0, fmt.Errorf("database connection is required")
	}
	if writer == nil {
		return 0, fmt.Errorf("menu snapshot writer is required")
	}

	rows := make([]MenuItem, 0)
	query := db.WithContext(ctx).Table(model.TableNameVideoMenu).
		Select("id", "parent_id", "name", "path", "component", "icon", "sort", "type", "permission", "visible", "status", "created_at", "updated_at", "deleted_at")
	if !includeDeleted {
		query = query.Where("deleted_at IS NULL")
	}
	if err := query.Order("parent_id ASC").Order("sort ASC").Order("id ASC").Scan(&rows).Error; err != nil {
		return 0, fmt.Errorf("read video_menu: %w", err)
	}

	snapshot := Snapshot{
		SourceTable: model.TableNameVideoMenu, GeneratedAt: time.Now(),
		IncludeDeleted: includeDeleted, Count: len(rows), Menus: rows,
	}
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(snapshot); err != nil {
		return 0, fmt.Errorf("encode menu snapshot: %w", err)
	}
	return len(rows), nil
}
