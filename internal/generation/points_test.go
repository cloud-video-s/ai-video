package generation

import (
	"context"
	"errors"
	"testing"
	"time"

	"ai-video/internal/commerce"
	"ai-video/internal/gen/model"
	"ai-video/internal/pkg/ucloud"
	"ai-video/internal/repository"

	"gorm.io/gorm"
)

func TestFreezeTaskPointsUsesVIPBeforeBalance(t *testing.T) {
	tests := []struct {
		name                          string
		vip, points                   int64
		score                         uint32
		wantVIP, wantPoints           int64
		wantFrozen                    uint64
		wantVIPScore, wantPointsScore uint32
		wantType                      uint32
	}{
		{name: "vip only", vip: 10, points: 10, score: 5, wantVIP: 5, wantPoints: 10, wantFrozen: 5, wantVIPScore: 5, wantType: TaskScoreTypeVIP},
		{name: "mixed", vip: 2, points: 10, score: 5, wantVIP: 0, wantPoints: 7, wantFrozen: 5, wantVIPScore: 2, wantPointsScore: 3, wantType: TaskScoreTypeMixed},
		{name: "balance only", points: 10, score: 5, wantPoints: 5, wantFrozen: 5, wantPointsScore: 5, wantType: TaskScoreTypeBalance},
		{name: "free", vip: 10, points: 10, wantVIP: 10, wantPoints: 10, wantType: TaskScoreTypeNone},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			user := &model.VideoUser{VipPoints: test.vip, PointsBalance: test.points}
			allocation, err := freezeTaskPoints(user, test.score)
			if err != nil {
				t.Fatal(err)
			}
			if user.VipPoints != test.wantVIP || user.PointsBalance != test.wantPoints || user.FrozenPoints != test.wantFrozen {
				t.Fatalf("balances = vip %d, points %d, frozen %d", user.VipPoints, user.PointsBalance, user.FrozenPoints)
			}
			if allocation.VIPScore != test.wantVIPScore || allocation.PointsScore != test.wantPointsScore || allocation.scoreType() != test.wantType {
				t.Fatalf("allocation = vip %d, points %d, type %d", allocation.VIPScore, allocation.PointsScore, allocation.scoreType())
			}
		})
	}
}

func TestCreateTaskFreezesMixedPointsAndIsIdempotent(t *testing.T) {
	db := newGenerationManagerTestDB(t)
	seedGenerationManagerVideoModel(t, db, "https://model.example.com")
	setGenerationTestBalances(t, db, 3, 10, 0)
	setGenerationTestModelScore(t, db, 5)
	manager := newPointsTestManager()

	request := &CreateTaskRequest{
		ModelCode: ucloud.ModelKlingO3, TaskType: TaskTypeVideo,
		ClientRequestID: "freeze-mixed-1", Input: map[string]any{"prompt": "mixed points"},
	}
	task, err := manager.CreateTask(context.Background(), 19, request)
	if err != nil {
		t.Fatal(err)
	}
	if task.Score != 5 || task.ScoreType != TaskScoreTypeMixed {
		t.Fatalf("task score allocation = score %d, type %d", task.Score, task.ScoreType)
	}
	assertGenerationTestBalances(t, db, 0, 8, 5)
	state, err := manager.taskRepo.GetPointStateForUpdate(context.Background(), task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if state.VIPScore != 3 || state.PointsScore != 2 {
		t.Fatalf("persisted allocation = vip %d, points %d", state.VIPScore, state.PointsScore)
	}

	again, err := manager.CreateTask(context.Background(), 19, request)
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != task.ID {
		t.Fatalf("idempotent task ID = %d, want %d", again.ID, task.ID)
	}
	assertGenerationTestBalances(t, db, 0, 8, 5)
}

func TestCompleteTaskConsumesFrozenPointsOnceAndCreatesLedger(t *testing.T) {
	db := newGenerationManagerTestDB(t)
	seedGenerationManagerVideoModel(t, db, "https://model.example.com")
	setGenerationTestBalances(t, db, 3, 10, 0)
	setGenerationTestModelScore(t, db, 5)
	manager := newPointsTestManager()
	task, err := manager.CreateTask(context.Background(), 19, &CreateTaskRequest{
		ModelCode: ucloud.ModelKlingO3, TaskType: TaskTypeVideo,
		ClientRequestID: "complete-points-1", Input: map[string]any{"prompt": "complete"},
	})
	if err != nil {
		t.Fatal(err)
	}

	task.Status = TaskStatusSuccess
	task.Progress = 100
	task.FinishedAt = time.Now()
	if err := manager.completeTask(context.Background(), task, "Status", "Progress", "FinishedAt"); err != nil {
		t.Fatal(err)
	}
	assertGenerationTestBalances(t, db, 0, 8, 0)
	assertGenerationLedger(t, db, task.TaskCode, 1, -5, 13, 8)

	if err := manager.completeTask(context.Background(), task, "Status", "Progress", "FinishedAt"); err != nil {
		t.Fatal(err)
	}
	assertGenerationTestBalances(t, db, 0, 8, 0)
	assertGenerationLedger(t, db, task.TaskCode, 1, -5, 13, 8)
}

func TestFailTaskReturnsMixedReservationOnce(t *testing.T) {
	db := newGenerationManagerTestDB(t)
	seedGenerationManagerVideoModel(t, db, "https://model.example.com")
	setGenerationTestBalances(t, db, 2, 10, 0)
	setGenerationTestModelScore(t, db, 5)
	manager := newPointsTestManager()
	task, err := manager.CreateTask(context.Background(), 19, &CreateTaskRequest{
		ModelCode: ucloud.ModelKlingO3, TaskType: TaskTypeVideo,
		ClientRequestID: "refund-points-1", Input: map[string]any{"prompt": "refund"},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertGenerationTestBalances(t, db, 0, 7, 5)

	if err := manager.failTask(context.Background(), task, "provider failed"); err == nil {
		t.Fatal("failTask must return the task failure")
	}
	assertGenerationTestBalances(t, db, 2, 10, 0)
	if task.Status != TaskStatusFailure {
		t.Fatalf("task status = %d", task.Status)
	}
	if err := manager.failTask(context.Background(), task, "duplicate callback"); err == nil {
		t.Fatal("repeated failTask must still report task failure")
	}
	assertGenerationTestBalances(t, db, 2, 10, 0)
	assertGenerationLedger(t, db, task.TaskCode, 0, 0, 0, 0)
}

func TestCreateTaskRejectsInsufficientCombinedPointsWithoutMutation(t *testing.T) {
	db := newGenerationManagerTestDB(t)
	seedGenerationManagerVideoModel(t, db, "https://model.example.com")
	setGenerationTestBalances(t, db, 2, 2, 0)
	setGenerationTestModelScore(t, db, 5)
	manager := newPointsTestManager()

	_, err := manager.CreateTask(context.Background(), 19, &CreateTaskRequest{
		ModelCode: ucloud.ModelKlingO3, TaskType: TaskTypeVideo,
		ClientRequestID: "insufficient-points-1", Input: map[string]any{"prompt": "too expensive"},
	})
	if !errors.Is(err, commerce.ErrInsufficientPoints) {
		t.Fatalf("CreateTask error = %v", err)
	}
	assertGenerationTestBalances(t, db, 2, 2, 0)
	var count int64
	if err := db.Table(model.TableNameVideoUserGenerationTask).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("created task count = %d", count)
	}
}

func TestCreateFreeTaskDoesNotChangePoints(t *testing.T) {
	db := newGenerationManagerTestDB(t)
	seedGenerationManagerVideoModel(t, db, "https://model.example.com")
	setGenerationTestBalances(t, db, 3, 10, 4)
	setGenerationTestModelScore(t, db, 0)
	manager := newPointsTestManager()

	task, err := manager.CreateTask(context.Background(), 19, &CreateTaskRequest{
		ModelCode: ucloud.ModelKlingO3, TaskType: TaskTypeVideo,
		ClientRequestID: "free-points-1", Input: map[string]any{"prompt": "free"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if task.Score != 0 || task.ScoreType != TaskScoreTypeNone {
		t.Fatalf("free task score = %d, type = %d", task.Score, task.ScoreType)
	}
	assertGenerationTestBalances(t, db, 3, 10, 4)
}

func newPointsTestManager() *Manager {
	return &Manager{
		AppUserRepo: repository.NewAppUserRepo(),
		modelRepo:   repository.NewModelRepo(), parameterRepo: repository.NewModelParameterRepo(),
		taskRepo: repository.NewUserGenerationTaskRepo(), hub: NewHub(),
	}
}

func setGenerationTestBalances(t *testing.T, db *gorm.DB, vip, points int64, frozen uint64) {
	t.Helper()
	if err := db.Table(model.TableNameVideoUser).Where("id = ?", 19).Updates(map[string]any{
		"vip_points": vip, "points_balance": points, "frozen_points": frozen,
	}).Error; err != nil {
		t.Fatal(err)
	}
}

func setGenerationTestModelScore(t *testing.T, db *gorm.DB, score int64) {
	t.Helper()
	if err := db.Table(model.TableNameVideoModel).Where("id = ?", 7).Update("score", score).Error; err != nil {
		t.Fatal(err)
	}
}

func assertGenerationTestBalances(t *testing.T, db *gorm.DB, vip, points int64, frozen uint64) {
	t.Helper()
	var got struct {
		VIP    int64  `gorm:"column:vip_points"`
		Points int64  `gorm:"column:points_balance"`
		Frozen uint64 `gorm:"column:frozen_points"`
	}
	if err := db.Table(model.TableNameVideoUser).Select("vip_points", "points_balance", "frozen_points").Where("id = ?", 19).Take(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got.VIP != vip || got.Points != points || got.Frozen != frozen {
		t.Fatalf("balances = vip %d, points %d, frozen %d; want %d, %d, %d", got.VIP, got.Points, got.Frozen, vip, points, frozen)
	}
}

func assertGenerationLedger(t *testing.T, db *gorm.DB, taskCode string, wantCount int64, change int64, before, after uint64) {
	t.Helper()
	var count int64
	query := db.Table(model.TableNameVideoUserPointsLedger).Where("order_code = ?", taskCode)
	if err := query.Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != wantCount {
		t.Fatalf("ledger count = %d, want %d", count, wantCount)
	}
	if wantCount == 0 {
		return
	}
	var ledger model.VideoUserPointsLedger
	if err := query.Take(&ledger).Error; err != nil {
		t.Fatal(err)
	}
	if ledger.PointsChange != change || ledger.BalanceBefore != before || ledger.BalanceAfter != after {
		t.Fatalf("ledger = change %d, balance %d -> %d", ledger.PointsChange, ledger.BalanceBefore, ledger.BalanceAfter)
	}
}
