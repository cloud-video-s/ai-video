package repository

import (
	"context"
	"time"

	"ai-video/internal/gen/model"

	"gorm.io/gen/field"
)

type UserPointsLedgerRepo struct{}

func NewUserPointsLedgerRepo() *UserPointsLedgerRepo { return &UserPointsLedgerRepo{} }

type UserPointsLedgerFilter struct {
	UserID          uint64
	Direction       int8
	SourceType      string
	PointsPackageID uint64
	BusinessID      string
	Keyword         string
	OccurredFrom    *time.Time
	OccurredTo      *time.Time
}

type UserPointsLedgerSummary struct {
	IncomeTotal  int64 `json:"income_total"`
	ExpenseTotal int64 `json:"expense_total"`
}

func (r *UserPointsLedgerRepo) Create(ctx context.Context, item *model.VideoUserPointsLedger) error {
	return qFrom(ctx).VideoUserPointsLedger.WithContext(ctx).Create(item)
}

func (r *UserPointsLedgerRepo) PageList(ctx context.Context, page, pageSize int, filter *UserPointsLedgerFilter) ([]model.VideoUserPointsLedger, int64, UserPointsLedgerSummary, error) {
	q := qFrom(ctx)
	ledger := q.VideoUserPointsLedger
	user := q.VideoUser
	dao := ledger.WithContext(ctx).LeftJoin(user, user.ID.EqCol(ledger.UserID))
	if filter != nil {
		if filter.UserID != 0 {
			dao = dao.Where(ledger.UserID.Eq(filter.UserID))
		}
		if filter.Direction != 0 {
			dao = dao.Where(ledger.Direction.Eq(filter.Direction))
		}
		if filter.SourceType != "" {
			dao = dao.Where(ledger.SourceType.Eq(filter.SourceType))
		}
		if filter.PointsPackageID != 0 {
			dao = dao.Where(ledger.PointsPackageID.Eq(filter.PointsPackageID))
		}
		if filter.BusinessID != "" {
			dao = dao.Where(ledger.BusinessID.Eq(filter.BusinessID))
		}
		if filter.OccurredFrom != nil {
			dao = dao.Where(ledger.OccurredAt.Gte(*filter.OccurredFrom))
		}
		if filter.OccurredTo != nil {
			dao = dao.Where(ledger.OccurredAt.Lt(*filter.OccurredTo))
		}
		if filter.Keyword != "" {
			keyword := "%" + filter.Keyword + "%"
			conditions := []field.Expr{
				ledger.BusinessID.Like(keyword), ledger.Description.Like(keyword),
				user.Username.Like(keyword), user.IMEI.Like(keyword),
				user.LoginAccount.Like(keyword), user.Email.Like(keyword),
			}
			identity := q.VideoUserIdentity
			var identityUserIDs []uint64
			if err := identity.WithContext(ctx).Where(identity.Email.Like(keyword)).
				Pluck(identity.UserID, &identityUserIDs); err != nil {
				return nil, 0, UserPointsLedgerSummary{}, err
			}
			if len(identityUserIDs) > 0 {
				conditions = append(conditions, ledger.UserID.In(identityUserIDs...))
			}
			dao = dao.Where(field.Or(conditions...))
		}
	}

	total, err := dao.Count()
	if err != nil {
		return nil, 0, UserPointsLedgerSummary{}, err
	}
	var summary UserPointsLedgerSummary
	if err := dao.Select(
		field.NewUnsafeFieldRaw("COALESCE(SUM(CASE WHEN video_user_points_ledger.points_change > 0 THEN video_user_points_ledger.points_change ELSE 0 END), 0)").As("income_total"),
		field.NewUnsafeFieldRaw("COALESCE(SUM(CASE WHEN video_user_points_ledger.points_change < 0 THEN -video_user_points_ledger.points_change ELSE 0 END), 0)").As("expense_total"),
	).Scan(&summary); err != nil {
		return nil, 0, UserPointsLedgerSummary{}, err
	}
	rows, err := dao.Preload(ledger.User, ledger.PointsPackage).
		Order(ledger.OccurredAt.Desc(), ledger.ID.Desc()).Offset((page - 1) * pageSize).Limit(pageSize).Find()
	return valuesOf(rows), total, summary, err
}

func (r *UserPointsLedgerRepo) GetDetail(ctx context.Context, id uint64) (*model.VideoUserPointsLedger, error) {
	q := qFrom(ctx).VideoUserPointsLedger
	return q.WithContext(ctx).Preload(q.User, q.PointsPackage).Where(q.ID.Eq(id)).First()
}
