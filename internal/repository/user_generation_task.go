package repository

import (
	"context"
	"time"

	"ai-video/internal/gen/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UserGenerationTaskRepo manages generation tasks owned by client users.
type UserGenerationTaskRepo struct {
	BaseRepo[model.VideoUserGenerationTask]
}

func NewUserGenerationTaskRepo() *UserGenerationTaskRepo {
	return &UserGenerationTaskRepo{}
}

// Create omits nullable lifecycle timestamps. The generated model uses
// time.Time for these columns, so writing its zero value would otherwise turn
// a SQL NULL into MySQL's zero date.
func (r *UserGenerationTaskRepo) Create(ctx context.Context, task *model.VideoUserGenerationTask) error {
	return dbFrom(ctx).Omit(
		"User", "Template", "ThirdTaskCode", "SubmittedAt", "StartedAt", "FinishedAt", "LastPolledAt",
	).Create(task).Error
}

// GenerationTaskPointState is the persisted point allocation for a generation
// task. VIPScore and PointsScore deliberately live beside the task so a failed
// mixed-balance reservation can always be returned to its original balances.
// These fields are kept here until the reviewed schema script is applied and
// GORM Gen is rerun against the updated schema.
type GenerationTaskPointState struct {
	ID          uint64 `gorm:"column:id"`
	UserID      uint64 `gorm:"column:user_id"`
	TaskCode    string `gorm:"column:task_code"`
	Status      int    `gorm:"column:status"`
	Score       uint32 `gorm:"column:score"`
	ScoreType   uint32 `gorm:"column:score_type"`
	VIPScore    uint32 `gorm:"column:vip_score"`
	PointsScore uint32 `gorm:"column:points_score"`
}

// SetPointAllocation stores the exact balance split selected while the user
// row is locked. It must be called in the same transaction that creates the
// task and updates the user's available and frozen balances.
func (r *UserGenerationTaskRepo) SetPointAllocation(ctx context.Context, id uint64, scoreType, vipScore, pointsScore uint32) error {
	result := dbFrom(ctx).Table(model.TableNameVideoUserGenerationTask).
		Where("id = ?", id).
		Updates(map[string]any{"score_type": scoreType, "vip_score": vipScore, "points_score": pointsScore})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		var count int64
		if err := dbFrom(ctx).Table(model.TableNameVideoUserGenerationTask).Where("id = ?", id).Count(&count).Error; err != nil {
			return err
		}
		if count != 1 {
			return gorm.ErrRecordNotFound
		}
	}
	return nil
}

// GetPointStateForUpdate serializes success settlement and failure release for
// one task. The task row is the idempotency guard that prevents a duplicate
// ledger entry or a second refund.
func (r *UserGenerationTaskRepo) GetPointStateForUpdate(ctx context.Context, id uint64) (*GenerationTaskPointState, error) {
	var state GenerationTaskPointState
	err := dbFrom(ctx).Table(model.TableNameVideoUserGenerationTask).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Select("id", "user_id", "task_code", "status", "score", "score_type", "vip_score", "points_score").
		Where("id = ? AND deleted_at IS NULL", id).
		Take(&state).Error
	return &state, err
}

func (r *UserGenerationTaskRepo) GetOwned(ctx context.Context, id, userID uint64) (*model.VideoUserGenerationTask, error) {
	q := qFrom(ctx).VideoUserGenerationTask
	return q.WithContext(ctx).Where(q.ID.Eq(id), q.UserID.Eq(userID)).First()
}

func (r *UserGenerationTaskRepo) GetOwnedByTaskCode(ctx context.Context, taskCode string, userID uint64) (*model.VideoUserGenerationTask, error) {
	q := qFrom(ctx).VideoUserGenerationTask
	return q.WithContext(ctx).Where(q.TaskCode.Eq(taskCode), q.UserID.Eq(userID)).First()
}

func (r *UserGenerationTaskRepo) GetByClientRequestID(ctx context.Context, userID uint64, requestID string) (*model.VideoUserGenerationTask, error) {
	q := qFrom(ctx).VideoUserGenerationTask
	return q.WithContext(ctx).Where(q.UserID.Eq(userID), q.ClientRequestID.Eq(requestID)).First()
}

func (r *UserGenerationTaskRepo) IDByClientRequestID(ctx context.Context, userID uint64, requestID string) (uint64, error) {
	q := qFrom(ctx).VideoUserGenerationTask
	row, err := q.WithContext(ctx).Select(q.ID).
		Where(q.UserID.Eq(userID), q.ClientRequestID.Eq(requestID)).First()
	if err != nil {
		return 0, err
	}
	return row.ID, nil
}

func (r *UserGenerationTaskRepo) PageOwned(ctx context.Context, userID uint64, page, pageSize int, status int) ([]model.VideoUserGenerationTask, int64, error) {
	q := qFrom(ctx).VideoUserGenerationTask
	dao := q.WithContext(ctx).Where(q.UserID.Eq(userID))
	if status > 0 {
		dao = dao.Where(q.Status.Eq(status))
	}
	total, err := dao.Count()
	if err != nil {
		return nil, 0, err
	}
	rows, err := dao.Order(q.ID.Desc()).Offset((page - 1) * pageSize).Limit(pageSize).Find()
	return valuesOf(rows), total, err
}

// UserGenerationTaskAdminFilter contains the read-only filters exposed by the
// admin task-management page.
type UserGenerationTaskAdminFilter struct {
	UserID      uint64
	ModelID     uint64
	TaskType    uint32
	Status      int
	TaskCode    string
	Keyword     string
	CreatedFrom *time.Time
	CreatedTo   *time.Time
}

// UserGenerationTaskAdminRecord keeps related user/model data beside the task
// without adding hand-written relationships to generated GORM models.
type UserGenerationTaskAdminRecord struct {
	Task  model.VideoUserGenerationTask
	User  *model.VideoUser
	Model *model.VideoModel
}

// PageAdmin lists generation tasks across all client users. Related rows are
// loaded in batches so historical tasks still remain readable if a user or
// model has since been soft-deleted.
func (r *UserGenerationTaskRepo) PageAdmin(ctx context.Context, page, pageSize int, filter *UserGenerationTaskAdminFilter) ([]UserGenerationTaskAdminRecord, int64, error) {
	taskTable := model.TableNameVideoUserGenerationTask
	userTable := model.TableNameVideoUser
	modelTable := model.TableNameVideoModel

	dao := dbFrom(ctx).Model(&model.VideoUserGenerationTask{}).
		Joins("LEFT JOIN " + userTable + " ON " + userTable + ".id = " + taskTable + ".user_id").
		Joins("LEFT JOIN " + modelTable + " ON " + modelTable + ".id = " + taskTable + ".model_id")
	if filter != nil {
		if filter.UserID != 0 {
			dao = dao.Where(taskTable+".user_id = ?", filter.UserID)
		}
		if filter.ModelID != 0 {
			dao = dao.Where(taskTable+".model_id = ?", filter.ModelID)
		}
		if filter.TaskType != 0 {
			dao = dao.Where(taskTable+".task_type = ?", filter.TaskType)
		}
		if filter.Status != 0 {
			dao = dao.Where(taskTable+".status = ?", filter.Status)
		}
		if filter.TaskCode != "" {
			dao = dao.Where(taskTable+".task_code = ?", filter.TaskCode)
		}
		if filter.CreatedFrom != nil {
			dao = dao.Where(taskTable+".created_at >= ?", *filter.CreatedFrom)
		}
		if filter.CreatedTo != nil {
			dao = dao.Where(taskTable+".created_at < ?", *filter.CreatedTo)
		}
		if filter.Keyword != "" {
			keyword := "%" + filter.Keyword + "%"
			dao = dao.Where("("+
				taskTable+".task_code LIKE ? OR "+taskTable+".client_request_id LIKE ? OR "+
				taskTable+".third_task_code LIKE ? OR "+taskTable+".prompt LIKE ? OR "+
				modelTable+".name LIKE ? OR "+modelTable+".code LIKE ? OR "+
				userTable+".username LIKE ? OR "+userTable+".email LIKE ? OR "+
				userTable+".login_account LIKE ? OR "+userTable+".imei LIKE ? OR "+
				"EXISTS (SELECT 1 FROM "+model.TableNameVideoUserIdentity+" identity_row "+
				"WHERE identity_row.user_id = "+taskTable+".user_id AND identity_row.email LIKE ?)"+
				")",
				keyword, keyword, keyword, keyword, keyword, keyword, keyword, keyword, keyword, keyword, keyword,
			)
		}
	}

	var total int64
	if err := dao.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var tasks []model.VideoUserGenerationTask
	if err := dao.Select(taskTable + ".*").
		Order(taskTable + ".created_at DESC").Order(taskTable + ".id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&tasks).Error; err != nil {
		return nil, 0, err
	}
	records, err := r.loadAdminRecords(ctx, tasks)
	return records, total, err
}

func (r *UserGenerationTaskRepo) GetAdminDetail(ctx context.Context, id uint64) (*UserGenerationTaskAdminRecord, error) {
	var task model.VideoUserGenerationTask
	if err := dbFrom(ctx).Where("id = ?", id).First(&task).Error; err != nil {
		return nil, err
	}
	records, err := r.loadAdminRecords(ctx, []model.VideoUserGenerationTask{task})
	if err != nil {
		return nil, err
	}
	return &records[0], nil
}

func (r *UserGenerationTaskRepo) loadAdminRecords(ctx context.Context, tasks []model.VideoUserGenerationTask) ([]UserGenerationTaskAdminRecord, error) {
	records := make([]UserGenerationTaskAdminRecord, 0, len(tasks))
	if len(tasks) == 0 {
		return records, nil
	}

	userIDs := make([]uint64, 0, len(tasks))
	modelIDs := make([]uint64, 0, len(tasks))
	for i := range tasks {
		userIDs = append(userIDs, tasks[i].UserID)
		modelIDs = append(modelIDs, tasks[i].ModelID)
	}

	var users []model.VideoUser
	if err := dbFrom(ctx).Unscoped().Where("id IN ?", userIDs).Find(&users).Error; err != nil {
		return nil, err
	}
	userByID := make(map[uint64]*model.VideoUser, len(users))
	for i := range users {
		userByID[users[i].ID] = &users[i]
	}

	var models []model.VideoModel
	if err := dbFrom(ctx).Unscoped().Where("id IN ?", modelIDs).Find(&models).Error; err != nil {
		return nil, err
	}
	modelByID := make(map[uint64]*model.VideoModel, len(models))
	for i := range models {
		if models[i].ID > 0 {
			modelByID[uint64(models[i].ID)] = &models[i]
		}
	}

	for i := range tasks {
		records = append(records, UserGenerationTaskAdminRecord{
			Task: tasks[i], User: userByID[tasks[i].UserID], Model: modelByID[tasks[i].ModelID],
		})
	}
	return records, nil
}

// ListActive returns recoverable tasks that still require polling or local
// result persistence.
func (r *UserGenerationTaskRepo) ListActive(ctx context.Context, limit int, statuses ...int) ([]model.VideoUserGenerationTask, error) {
	if len(statuses) == 0 {
		return []model.VideoUserGenerationTask{}, nil
	}
	q := qFrom(ctx).VideoUserGenerationTask
	rows, err := q.WithContext(ctx).
		Where(q.Status.In(statuses...)).
		Order(q.LastPolledAt.Asc(), q.CreatedAt.Asc()).Limit(limit).Find()
	return valuesOf(rows), err
}

func (r *UserGenerationTaskRepo) UpdateFields(ctx context.Context, task *model.VideoUserGenerationTask, fields ...string) error {
	return dbFrom(ctx).Model(task).Select(fields).Updates(task).Error
}

func (r *UserGenerationTaskRepo) MarkPolling(ctx context.Context, id uint64, at time.Time) error {
	q := qFrom(ctx).VideoUserGenerationTask
	_, err := q.WithContext(ctx).Where(q.ID.Eq(id)).Update(q.LastPolledAt, at)
	return err
}

// TryClaimSubmitting atomically leases a queued task to one worker. A stale
// lease can be reclaimed after a worker crash, while concurrent workers that
// fetched the same row cannot submit it to the provider more than once.
func (r *UserGenerationTaskRepo) TryClaimSubmitting(
	ctx context.Context,
	id uint64,
	expectedStatus int,
	claimedAt, staleBefore time.Time,
) (bool, error) {
	return r.tryClaim(ctx, id, expectedStatus, claimedAt, staleBefore)
}

// TryClaimPolling ensures only one worker instance queries the provider for a
// task in a polling interval. The status predicate also rejects stale rows
// fetched before another worker advanced the task.
func (r *UserGenerationTaskRepo) TryClaimPolling(
	ctx context.Context,
	id uint64,
	expectedStatus int,
	claimedAt, staleBefore time.Time,
) (bool, error) {
	return r.tryClaim(ctx, id, expectedStatus, claimedAt, staleBefore)
}

func (r *UserGenerationTaskRepo) tryClaim(
	ctx context.Context,
	id uint64,
	expectedStatus int,
	claimedAt, staleBefore time.Time,
) (bool, error) {
	result := dbFrom(ctx).Model(&model.VideoUserGenerationTask{}).
		Where("id = ? AND status = ? AND (last_polled_at IS NULL OR last_polled_at < ?)", id, expectedStatus, staleBefore).
		Update("last_polled_at", claimedAt)
	return result.RowsAffected == 1, result.Error
}

func (r *UserGenerationTaskRepo) CountByModel(ctx context.Context, modelID uint64) (int64, error) {
	q := qFrom(ctx).VideoUserGenerationTask
	return q.WithContext(ctx).Where(q.ModelID.Eq(modelID)).Count()
}

func (r *UserGenerationTaskRepo) DeleteOwned(ctx context.Context, id, userID uint64) error {
	q := qFrom(ctx).VideoUserGenerationTask
	result, err := q.WithContext(ctx).Where(q.ID.Eq(id), q.UserID.Eq(userID)).Delete()
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
