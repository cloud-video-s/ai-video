package model

import "time"

const (
	PointsDirectionIncome  int8 = 1
	PointsDirectionExpense int8 = 2
)

// VideoUserPointsLedger is an immutable audit entry for every points balance
// change. Corrections should be recorded as reversal entries, never edits.
type VideoUserPointsLedger struct {
	ID              uint64    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID          uint64    `json:"user_id" gorm:"not null;index;comment:client user ID"`
	Direction       int8      `json:"direction" gorm:"not null;index;comment:1 income, 2 expense"`
	PointsChange    int64     `json:"points_change" gorm:"not null;comment:signed points change"`
	BalanceBefore   uint64    `json:"balance_before" gorm:"not null;comment:balance before change"`
	BalanceAfter    uint64    `json:"balance_after" gorm:"not null;comment:balance after change"`
	SourceType      string    `json:"source_type" gorm:"size:32;not null;index;comment:purchase, consume, reward, refund, admin or other"`
	BusinessID      string    `json:"business_id" gorm:"size:191;index;comment:order or business reference"`
	PointsPackageID *uint64   `json:"points_package_id" gorm:"index;comment:related points package ID"`
	OperatorAdminID *uint     `json:"operator_admin_id" gorm:"index;comment:admin operator ID for manual changes"`
	Description     string    `json:"description" gorm:"size:1000;comment:change description"`
	OccurredAt      time.Time `json:"occurred_at" gorm:"not null;index;comment:business occurrence time"`
	CreatedAt       time.Time `json:"created_at" gorm:"not null;index"`

	User          VideoUser           `json:"user,omitempty" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	PointsPackage *VideoPointsPackage `json:"points_package,omitempty" gorm:"foreignKey:PointsPackageID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
}

func (VideoUserPointsLedger) TableName() string {
	return "video_user_points_ledger"
}
