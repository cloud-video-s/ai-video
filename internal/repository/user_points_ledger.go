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

type UserPointsLedgerRecord struct {
	model.VideoUserPointsLedger
	User          model.VideoUser           `json:"user"`
	PointsPackage *model.VideoPointsPackage `json:"points_package,omitempty"`
}

func (r *UserPointsLedgerRepo) Create(ctx context.Context, item *model.VideoUserPointsLedger) error {
	return qFrom(ctx).VideoUserPointsLedger.WithContext(ctx).Create(item)
}

func (r *UserPointsLedgerRepo) PageList(ctx context.Context, page, pageSize int, filter *UserPointsLedgerFilter) ([]UserPointsLedgerRecord, int64, UserPointsLedgerSummary, error) {
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
	rows, err := dao.Order(ledger.OccurredAt.Desc(), ledger.ID.Desc()).
		Offset((page - 1) * pageSize).Limit(pageSize).Find()
	if err != nil {
		return nil, 0, UserPointsLedgerSummary{}, err
	}
	records, err := r.loadRecords(ctx, valuesOf(rows))
	return records, total, summary, err
}

func (r *UserPointsLedgerRepo) GetDetail(ctx context.Context, id uint64) (*UserPointsLedgerRecord, error) {
	q := qFrom(ctx).VideoUserPointsLedger
	item, err := q.WithContext(ctx).Where(q.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}
	records, err := r.loadRecords(ctx, []model.VideoUserPointsLedger{*item})
	if err != nil {
		return nil, err
	}
	return &records[0], nil
}

func (r *UserPointsLedgerRepo) loadRecords(ctx context.Context, items []model.VideoUserPointsLedger) ([]UserPointsLedgerRecord, error) {
	result := make([]UserPointsLedgerRecord, 0, len(items))
	if len(items) == 0 {
		return result, nil
	}
	userIDs := make([]uint64, 0, len(items))
	packageIDs := make([]uint64, 0, len(items))
	for i := range items {
		userIDs = append(userIDs, items[i].UserID)
		if items[i].PointsPackageID != nil {
			packageIDs = append(packageIDs, *items[i].PointsPackageID)
		}
	}
	q := qFrom(ctx)
	users, err := q.VideoUser.WithContext(ctx).Where(q.VideoUser.ID.In(userIDs...)).Find()
	if err != nil {
		return nil, err
	}
	userByID := make(map[uint64]model.VideoUser, len(users))
	for _, user := range users {
		if user != nil {
			userByID[user.ID] = *user
		}
	}
	packageByID := make(map[uint64]*model.VideoPointsPackage, len(packageIDs))
	if len(packageIDs) > 0 {
		packages, err := q.VideoPointsPackage.WithContext(ctx).Where(q.VideoPointsPackage.ID.In(packageIDs...)).Find()
		if err != nil {
			return nil, err
		}
		for _, item := range packages {
			if item != nil {
				packageByID[item.ID] = item
			}
		}
	}
	for i := range items {
		record := UserPointsLedgerRecord{VideoUserPointsLedger: items[i], User: userByID[items[i].UserID]}
		if items[i].PointsPackageID != nil {
			record.PointsPackage = packageByID[*items[i].PointsPackageID]
		}
		result = append(result, record)
	}
	return result, nil
}
