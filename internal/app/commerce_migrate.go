package app

import (
	"fmt"
	"time"

	"ai-video/internal/gen/model"

	"gorm.io/gorm"
)

// commercePointsLedgerColumns keeps commerce-only columns nullable for legacy
// ledger rows that predate orders and generation-mode tracking.
type commercePointsLedgerColumns struct {
	OrderID        *uint64 `gorm:"column:order_id;type:bigint unsigned;index:idx_video_user_points_ledger_order_id"`
	WorkID         *string `gorm:"column:work_id;type:varchar(191);index:idx_video_user_points_ledger_work_id"`
	ModeKey        *string `gorm:"column:mode_key;type:varchar(64);index:idx_video_user_points_ledger_mode_key"`
	IdempotencyKey *string `gorm:"column:idempotency_key;type:varchar(191);uniqueIndex:uk_video_user_points_ledger_idempotency"`
}

func (*commercePointsLedgerColumns) TableName() string { return model.TableNameVideoUserPointsLedger }

// commerceOrderTable intentionally omits the generated User association so a
// commerce migration never attempts to alter the much larger video_user table.
type commerceOrderTable struct {
	ID                    uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	OrderNo               string     `gorm:"column:order_no;type:varchar(40);not null;uniqueIndex:uk_video_order_order_no"`
	ClientRequestID       string     `gorm:"column:client_request_id;type:varchar(64);not null;uniqueIndex:uk_video_order_client_request"`
	UserID                uint64     `gorm:"column:user_id;type:bigint unsigned;not null;index:idx_video_order_user_id"`
	ProductType           string     `gorm:"column:product_type;type:varchar(32);not null;index:idx_video_order_product,priority:1"`
	ProductID             uint64     `gorm:"column:product_id;type:bigint unsigned;not null;index:idx_video_order_product,priority:2"`
	ProductCode           string     `gorm:"column:product_code;type:varchar(191);not null;index:idx_video_order_product_code"`
	ProductName           string     `gorm:"column:product_name;type:varchar(128);not null"`
	Currency              string     `gorm:"column:currency;type:varchar(8);not null;default:USD"`
	ProductAmount         float64    `gorm:"column:product_amount;type:decimal(12,2);not null;default:0"`
	DiscountAmount        float64    `gorm:"column:discount_amount;type:decimal(12,2);not null;default:0"`
	PayableAmount         float64    `gorm:"column:payable_amount;type:decimal(12,2);not null;default:0"`
	PaidAmount            float64    `gorm:"column:paid_amount;type:decimal(12,2);not null;default:0"`
	RefundedAmount        float64    `gorm:"column:refunded_amount;type:decimal(12,2);not null;default:0"`
	BonusPoints           uint64     `gorm:"column:bonus_points;type:bigint unsigned;not null;default:0"`
	VipLevel              uint       `gorm:"column:vip_level;type:int unsigned;not null;default:0"`
	VipDurationDays       uint       `gorm:"column:vip_duration_days;type:int unsigned;not null;default:0"`
	Status                string     `gorm:"column:status;type:varchar(20);not null;default:pending;index:idx_video_order_status"`
	PaymentMethod         string     `gorm:"column:payment_method;type:varchar(32);not null;uniqueIndex:uk_video_order_payment_transaction,priority:1"`
	ProviderTransactionID *string    `gorm:"column:provider_transaction_id;type:varchar(191);uniqueIndex:uk_video_order_payment_transaction,priority:2"`
	OriginalTransactionID *string    `gorm:"column:original_transaction_id;type:varchar(191);index:idx_video_order_original_transaction"`
	PaymentEvidence       string     `gorm:"column:payment_evidence;type:text"`
	FailureCode           string     `gorm:"column:failure_code;type:varchar(64)"`
	FailureMessage        string     `gorm:"column:failure_message;type:varchar(500)"`
	CancelReason          string     `gorm:"column:cancel_reason;type:varchar(500)"`
	PaidAt                *time.Time `gorm:"column:paid_at;type:datetime;index:idx_video_order_paid_at"`
	CancelledAt           *time.Time `gorm:"column:cancelled_at;type:datetime;index:idx_video_order_cancelled_at"`
	ExpiresAt             *time.Time `gorm:"column:expires_at;type:datetime;index:idx_video_order_expires_at"`
	CreatedAt             time.Time  `gorm:"column:created_at;type:datetime;not null;index:idx_video_order_created_at"`
	UpdatedAt             time.Time  `gorm:"column:updated_at;type:datetime;not null"`
}

func (*commerceOrderTable) TableName() string { return model.TableNameVideoOrder }

// MigrateCommerceTables is safe to run at every API startup. It creates the
// order table and adds idempotency/business columns to the existing ledger.
func MigrateCommerceTables(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	migratorDB := db.Session(&gorm.Session{})
	migratorDB.Config.IgnoreRelationshipsWhenMigrating = true
	migratorDB.Config.DisableForeignKeyConstraintWhenMigrating = true
	if err := migratorDB.AutoMigrate(&commerceOrderTable{}); err != nil {
		return fmt.Errorf("migrate video_order: %w", err)
	}
	if !migratorDB.Migrator().HasTable(&model.VideoUserPointsLedger{}) {
		if err := migratorDB.AutoMigrate(&model.VideoUserPointsLedger{}); err != nil {
			return fmt.Errorf("create video_user_points_ledger: %w", err)
		}
		return nil
	}
	if err := migratorDB.AutoMigrate(&commercePointsLedgerColumns{}); err != nil {
		return fmt.Errorf("migrate video_user_points_ledger commerce columns: %w", err)
	}
	return nil
}
