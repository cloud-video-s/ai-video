package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"ai-video/internal/config"
	"ai-video/internal/gen/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestUserGenerationTaskRepoLifecycle(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:user-generation-task?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE video_user_generation_task (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		model_id INTEGER NOT NULL,
		client_request_id TEXT NOT NULL,
		task_code TEXT NOT NULL UNIQUE,
		third_task_code TEXT,
		status INTEGER NOT NULL,
		task_type INTEGER NOT NULL DEFAULT 1,
		progress INTEGER NOT NULL DEFAULT 0,
		prompt TEXT,
		request_payload TEXT NOT NULL,
		provider_response TEXT,
		remote_urls TEXT,
		local_urls TEXT,
		error_message TEXT,
		usage_duration INTEGER NOT NULL DEFAULT 0,
		submitted_at DATETIME NULL,
		started_at DATETIME NULL,
		finished_at DATETIME NULL,
		last_polled_at DATETIME NULL,
		template_id INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		deleted_at DATETIME NULL,
		UNIQUE (user_id, client_request_id),
		UNIQUE (third_task_code)
	)`).Error; err != nil {
		t.Fatal(err)
	}

	previousDB := config.DB
	config.DB = db
	t.Cleanup(func() { config.DB = previousDB })

	ctx := context.Background()
	repo := NewUserGenerationTaskRepo()
	task := &model.VideoUserGenerationTask{
		UserID: 7, ModelID: 11, ClientRequestID: "request-1", TaskCode: "task-1",
		Status: 1, RequestPayload: `{}`, RemoteUrls: `[]`, LocalUrls: `[]`,
	}
	if err := repo.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	if task.ID == 0 {
		t.Fatal("task ID was not populated")
	}

	var lifecycle struct {
		ThirdTaskCode sql.NullString
		SubmittedAt   sql.NullTime
		StartedAt     sql.NullTime
		FinishedAt    sql.NullTime
		LastPolledAt  sql.NullTime
	}
	if err := db.Table(model.TableNameVideoUserGenerationTask).Select(
		"third_task_code, submitted_at, started_at, finished_at, last_polled_at",
	).Where("id = ?", task.ID).Scan(&lifecycle).Error; err != nil {
		t.Fatal(err)
	}
	if lifecycle.ThirdTaskCode.Valid || lifecycle.SubmittedAt.Valid || lifecycle.StartedAt.Valid || lifecycle.FinishedAt.Valid || lifecycle.LastPolledAt.Valid {
		t.Fatalf("new task lifecycle timestamps must be NULL: %#v", lifecycle)
	}

	owned, err := repo.GetOwned(ctx, task.ID, task.UserID)
	if err != nil || owned.ClientRequestID != task.ClientRequestID {
		t.Fatalf("GetOwned() = %#v, %v", owned, err)
	}
	if _, err := repo.GetOwned(ctx, task.ID, task.UserID+1); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("cross-user GetOwned error = %v", err)
	}
	byCode, err := repo.GetOwnedByTaskCode(ctx, task.TaskCode, task.UserID)
	if err != nil || byCode.ID != task.ID {
		t.Fatalf("GetOwnedByTaskCode() = %#v, %v", byCode, err)
	}

	active, err := repo.ListActive(ctx, 10, 1, 2, 3, 4, 5)
	if err != nil || len(active) != 1 || active[0].ID != task.ID {
		t.Fatalf("ListActive() = %#v, %v", active, err)
	}
	items, total, err := repo.PageOwned(ctx, task.UserID, 1, 10, 1)
	if err != nil || total != 1 || len(items) != 1 {
		t.Fatalf("PageOwned() = %#v, %d, %v", items, total, err)
	}

	claimedAt := time.Now().Truncate(time.Millisecond)
	claimed, err := repo.TryClaimSubmitting(ctx, task.ID, 1, claimedAt, claimedAt.Add(-time.Minute))
	if err != nil || !claimed {
		t.Fatalf("first task claim = %v, %v", claimed, err)
	}
	claimed, err = repo.TryClaimPolling(ctx, task.ID, 1, claimedAt.Add(time.Second), claimedAt.Add(-time.Minute))
	if err != nil || claimed {
		t.Fatalf("active lease duplicate claim = %v, %v", claimed, err)
	}
	claimed, err = repo.TryClaimPolling(ctx, task.ID, 1, claimedAt.Add(3*time.Minute), claimedAt.Add(time.Second))
	if err != nil || !claimed {
		t.Fatalf("stale lease reclaim = %v, %v", claimed, err)
	}

	polledAt := time.Now().Truncate(time.Millisecond)
	if err := repo.MarkPolling(ctx, task.ID, polledAt); err != nil {
		t.Fatal(err)
	}
	if err := repo.DeleteOwned(ctx, task.ID, task.UserID); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetOwned(ctx, task.ID, task.UserID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("GetOwned after delete error = %v", err)
	}
}

func TestUserGenerationTaskRepoPageAdmin(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:user-generation-task-admin?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	statements := []string{
		`CREATE TABLE video_user (
			id INTEGER PRIMARY KEY, username TEXT, email TEXT, login_account TEXT,
			imei TEXT, device_code TEXT, deleted_at DATETIME NULL
		)`,
		`CREATE TABLE video_model (
			id INTEGER PRIMARY KEY, name TEXT, code TEXT, model_type INTEGER,
			version TEXT, deleted_at DATETIME NULL
		)`,
		`CREATE TABLE video_user_identity (
			id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL, email TEXT
		)`,
		`CREATE TABLE video_user_generation_task (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			model_id INTEGER NOT NULL,
			client_request_id TEXT NOT NULL,
			task_code TEXT NOT NULL UNIQUE,
			third_task_code TEXT,
			status INTEGER NOT NULL,
			task_type INTEGER NOT NULL DEFAULT 1,
			progress INTEGER NOT NULL DEFAULT 0,
			prompt TEXT,
			request_payload TEXT NOT NULL,
			provider_response TEXT,
			remote_urls TEXT,
			local_urls TEXT,
			error_message TEXT,
			usage_duration INTEGER NOT NULL DEFAULT 0,
			submitted_at DATETIME NULL,
			started_at DATETIME NULL,
			finished_at DATETIME NULL,
			last_polled_at DATETIME NULL,
			template_id INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			deleted_at DATETIME NULL
		)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Exec(`INSERT INTO video_user (id, username, email, login_account, imei, device_code)
		VALUES (7, 'Alice', 'alice@example.com', 'alice-login', 'imei-7', 'device-7')`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO video_user_identity (id, user_id, email)
		VALUES (1, 7, 'identity@example.com')`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO video_model (id, name, code, model_type, version)
		VALUES (11, 'Video Model', 'video-model', 2, 'v1')`).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now().Truncate(time.Second)
	if err := db.Exec(`INSERT INTO video_user_generation_task (
		user_id, model_id, client_request_id, task_code, third_task_code, status, progress,
		prompt, request_payload, remote_urls, local_urls, usage_duration, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		7, 11, "client-1", "task-admin-1", "third-1", 6, 100,
		"a sample video", `{}`, `["https://remote.example/video.mp4"]`,
		`["/storage/video.mp4"]`, 9, now, now,
	).Error; err != nil {
		t.Fatal(err)
	}

	previousDB := config.DB
	config.DB = db
	t.Cleanup(func() { config.DB = previousDB })

	repo := NewUserGenerationTaskRepo()
	records, total, err := repo.PageAdmin(context.Background(), 1, 20, &UserGenerationTaskAdminFilter{
		ModelType: 2, Status: 6, Keyword: "identity@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(records) != 1 {
		t.Fatalf("PageAdmin() returned %d records, total %d", len(records), total)
	}
	if records[0].User == nil || records[0].User.Username != "Alice" {
		t.Fatalf("PageAdmin() user = %#v", records[0].User)
	}
	if records[0].Model == nil || records[0].Model.Code != "video-model" {
		t.Fatalf("PageAdmin() model = %#v", records[0].Model)
	}

	detail, err := repo.GetAdminDetail(context.Background(), records[0].Task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Task.TaskCode != "task-admin-1" || detail.Model == nil || detail.User == nil {
		t.Fatalf("GetAdminDetail() = %#v", detail)
	}
}
