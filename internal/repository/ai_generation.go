package repository

import (
	"context"
	"strings"
	"time"

	"ai-video/internal/gen/model"

	"gorm.io/gen/field"
	"gorm.io/gorm"
)

// AIModelRepo 管理可配置的第三方视频生成模型。
type AIModelRepo struct{ BaseRepo[model.VideoAiModel] }

func NewAIModelRepo() *AIModelRepo { return &AIModelRepo{} }

type AIModelListFilter struct {
	Keyword  string
	Provider string
	Status   *int8
}

func (r *AIModelRepo) PageList(ctx context.Context, page, pageSize int, filter *AIModelListFilter) ([]model.VideoAiModel, int64, error) {
	q := qFrom(ctx).VideoAiModel
	dao := q.WithContext(ctx)
	if filter != nil {
		if filter.Provider != "" {
			dao = dao.Where(q.Provider.Eq(filter.Provider))
		}
		if filter.Status != nil {
			dao = dao.Where(q.Status.Eq(*filter.Status))
		}
		if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
			value := "%" + keyword + "%"
			dao = dao.Where(field.Or(q.Code.Like(value), q.Name.Like(value), q.ModelName.Like(value), q.Description.Like(value)))
		}
	}
	total, err := dao.Count()
	if err != nil {
		return nil, 0, err
	}
	rows, err := dao.Order(q.ID.Desc()).Offset((page - 1) * pageSize).Limit(pageSize).Find()
	return valuesOf(rows), total, err
}

func (r *AIModelRepo) GetByCode(ctx context.Context, code string) (*model.VideoAiModel, error) {
	q := qFrom(ctx).VideoAiModel
	return q.WithContext(ctx).Where(q.Code.Eq(code)).First()
}

func (r *AIModelRepo) GetEnabledByCode(ctx context.Context, code string) (*model.VideoAiModel, error) {
	q := qFrom(ctx).VideoAiModel
	return q.WithContext(ctx).Where(q.Code.Eq(code), q.Status.Eq(1)).First()
}

func (r *AIModelRepo) ListEnabled(ctx context.Context) ([]model.VideoAiModel, error) {
	q := qFrom(ctx).VideoAiModel
	rows, err := q.WithContext(ctx).Where(q.Status.Eq(1)).Order(q.ID.Asc()).Find()
	return valuesOf(rows), err
}

func (r *AIModelRepo) UpdateFields(ctx context.Context, item *model.VideoAiModel) error {
	q := qFrom(ctx).VideoAiModel
	_, err := q.WithContext(ctx).Where(q.ID.Eq(item.ID)).Select(
		q.Code, q.Name, q.Provider, q.ModelName, q.BaseURL, q.SubmitPath, q.StatusPath,
		q.APIKey, q.DefaultParameters, q.HTTPTimeoutSeconds, q.PollIntervalSeconds,
		q.TaskTimeoutSeconds, q.Status, q.Description,
	).Updates(item)
	return err
}

func (r *AIModelRepo) TaskCount(ctx context.Context, id uint64) (int64, error) {
	q := qFrom(ctx).VideoGenerationTask
	return q.WithContext(ctx).Where(q.ModelConfigID.Eq(id)).Count()
}

// GenerationTaskRepo 管理客户端用户的视频生成任务。
type GenerationTaskRepo struct {
	BaseRepo[model.VideoGenerationTask]
}

func NewGenerationTaskRepo() *GenerationTaskRepo { return &GenerationTaskRepo{} }

func (r *GenerationTaskRepo) GetOwned(ctx context.Context, id, userID uint64) (*model.VideoGenerationTask, error) {
	q := qFrom(ctx).VideoGenerationTask
	return q.WithContext(ctx).Where(q.ID.Eq(id), q.UserID.Eq(userID)).First()
}

func (r *GenerationTaskRepo) GetByClientRequestID(ctx context.Context, userID uint64, requestID string) (*model.VideoGenerationTask, error) {
	q := qFrom(ctx).VideoGenerationTask
	return q.WithContext(ctx).Where(q.UserID.Eq(userID), q.ClientRequestID.Eq(requestID)).First()
}

func (r *GenerationTaskRepo) IDByClientRequestID(ctx context.Context, userID uint64, requestID string) (uint64, error) {
	q := qFrom(ctx).VideoGenerationTask
	row, err := q.WithContext(ctx).Select(q.ID).Where(q.UserID.Eq(userID), q.ClientRequestID.Eq(requestID)).First()
	if err != nil {
		return 0, err
	}
	return row.ID, nil
}

func (r *GenerationTaskRepo) PageOwned(ctx context.Context, userID uint64, page, pageSize int, status string) ([]model.VideoGenerationTask, int64, error) {
	q := qFrom(ctx).VideoGenerationTask
	dao := q.WithContext(ctx).Where(q.UserID.Eq(userID))
	if status != "" {
		dao = dao.Where(q.Status.Eq(status))
	}
	total, err := dao.Count()
	if err != nil {
		return nil, 0, err
	}
	rows, err := dao.Order(q.ID.Desc()).Offset((page - 1) * pageSize).Limit(pageSize).Find()
	return valuesOf(rows), total, err
}

// ListActive 返回需要恢复或继续轮询的非终态任务。
func (r *GenerationTaskRepo) ListActive(ctx context.Context, limit int) ([]model.VideoGenerationTask, error) {
	q := qFrom(ctx).VideoGenerationTask
	rows, err := q.WithContext(ctx).Where(q.Status.In("submitted", "pending", "running", "downloading")).
		Order(q.LastPolledAt.Asc(), q.CreatedAt.Asc()).Limit(limit).Find()
	return valuesOf(rows), err
}

func (r *GenerationTaskRepo) UpdateFields(ctx context.Context, item *model.VideoGenerationTask, fields ...string) error {
	return dbFrom(ctx).Model(item).Select(fields).Updates(item).Error
}

func (r *GenerationTaskRepo) MarkPolling(ctx context.Context, id uint64, at time.Time) error {
	q := qFrom(ctx).VideoGenerationTask
	_, err := q.WithContext(ctx).Where(q.ID.Eq(id)).Update(q.LastPolledAt, at)
	return err
}

func (r *GenerationTaskRepo) DeleteOwned(ctx context.Context, id, userID uint64) error {
	q := qFrom(ctx).VideoGenerationTask
	result, err := q.WithContext(ctx).Where(q.ID.Eq(id), q.UserID.Eq(userID)).Delete()
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
