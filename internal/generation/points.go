package generation

import (
	"ai-video/internal/event"
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"ai-video/internal/domain"
	"ai-video/internal/gen/model"
	"ai-video/internal/repository"

	"gorm.io/gorm"
)

const (
	TaskScoreTypeNone    uint32 = 0
	TaskScoreTypeVIP     uint32 = 1
	TaskScoreTypeBalance uint32 = 2
	TaskScoreTypeMixed   uint32 = 3
)

type taskPointAllocation struct {
	VIPScore    uint32
	PointsScore uint32
}

func (a taskPointAllocation) total() uint64 {
	return uint64(a.VIPScore) + uint64(a.PointsScore)
}

func (a taskPointAllocation) scoreType() uint32 {
	switch {
	case a.VIPScore > 0 && a.PointsScore > 0:
		return TaskScoreTypeMixed
	case a.VIPScore > 0:
		return TaskScoreTypeVIP
	case a.PointsScore > 0:
		return TaskScoreTypeBalance
	default:
		return TaskScoreTypeNone
	}
}

// freezeTaskPoints moves the model-owned task cost from the user's available
// balances into frozen_points. The caller must hold the user row lock and must
// persist the returned allocation in the same transaction.
func freezeTaskPoints(user *model.VideoUser, score uint32) (taskPointAllocation, error) {
	if user == nil {
		return taskPointAllocation{}, errors.New("user is required")
	}
	if score == 0 {
		return taskPointAllocation{}, nil
	}
	remaining := int64(score)
	allocation := taskPointAllocation{
		VIPScore:    0,
		PointsScore: 0,
	}
	if user.SubscriptionStatus == 2 && user.VipExpiresAt.Unix() > time.Now().Unix() {
		if user.VipPoints+user.PointsBalance < remaining {
			return taskPointAllocation{}, errors.New("user points balance is invalid")
		}
		vipScore := min(user.VipPoints, remaining)
		remaining -= vipScore
		allocation.VIPScore = uint32(vipScore)
	} else {
		if user.PointsBalance < remaining {
			return taskPointAllocation{}, errors.New("user points balance is invalid")
		}
	}
	allocation.PointsScore = uint32(remaining)
	user.VipPoints -= int64(allocation.VIPScore)
	user.PointsBalance -= remaining
	user.FrozenPoints += uint64(score)
	return allocation, nil
}

// completeTask confirms a successful task and consumes its reservation. The
// locked task row is the idempotency boundary, so retries cannot create a
// second points ledger entry or consume frozen points twice.
func (m *Manager) completeTask(ctx context.Context, task *model.VideoUserGenerationTask, fields ...string) error {
	if task == nil || task.ID == 0 {
		return errors.New("generation task is required")
	}
	var lockedStatus int
	alreadyCompleted := false
	err := repository.Transaction(ctx, func(txCtx context.Context) error {
		state, err := m.taskRepo.GetPointStateForUpdate(txCtx, task.ID)
		if err != nil {
			return err
		}
		lockedStatus = state.Status
		switch state.Status {
		case TaskStatusSuccess:
			alreadyCompleted = true
			return nil
		case TaskStatusFailure:
			return errors.New("failed generation task cannot be completed")
		}

		if err = m.consumeFrozenTaskPoints(txCtx, state); err != nil {
			return err
		}
		return m.taskRepo.UpdateFields(txCtx, task, fields...)
	})
	if err != nil {
		if lockedStatus != 0 {
			task.Status = lockedStatus
		}
		return err
	}
	if alreadyCompleted {
		task.Status = TaskStatusSuccess
		task.Progress = 100
		go event.EventTask(ctx, task)
	}
	m.hub.Publish(task)
	return nil
}

func (m *Manager) consumeFrozenTaskPoints(ctx context.Context, state *repository.GenerationTaskPointState) error {
	if state.Score == 0 {
		return nil
	}
	_, reserved, err := pointAllocationFromState(state)
	if err != nil {
		return err
	}
	user, err := m.GetByIDForUpdate(ctx, state.UserID)
	if err != nil {
		return err
	}
	if user.VipPoints < 0 || user.PointsBalance < 0 {
		return errors.New("user points balance is invalid")
	}

	var beforeBalance, afterBalance uint64
	if reserved {
		if user.FrozenPoints < uint64(state.Score) {
			return errors.New("frozen points are less than the task reservation")
		}
		available := uint64(user.VipPoints) + uint64(user.PointsBalance)
		if available > math.MaxUint64-user.FrozenPoints {
			return errors.New("points ledger balance overflow")
		}
		beforeBalance = available + user.FrozenPoints
		afterBalance = beforeBalance - uint64(state.Score)
		user.FrozenPoints -= uint64(state.Score)
	} else {
		_, err = freezeTaskPoints(user, state.Score)
		if err != nil {
			return err
		}
		user.FrozenPoints -= uint64(state.Score)
		beforeBalance = uint64(user.VipPoints) + uint64(user.PointsBalance) + uint64(state.Score)
		afterBalance = uint64(user.VipPoints) + uint64(user.PointsBalance)
	}

	now := time.Now()
	ledger := &model.VideoUserPointsLedger{
		UserID:        state.UserID,
		Direction:     int8(domain.PointsDirectionExpense),
		PointsChange:  -int64(state.Score),
		BalanceBefore: beforeBalance,
		BalanceAfter:  afterBalance,
		SourceType:    domain.PointsSourceModelConsume,
		OrderCode:     state.TaskCode,
		Description:   "Generation task points consumed",
		OccurredAt:    now,
		CreatedAt:     now,
	}
	q := repository.QFrom(ctx)
	if err = q.VideoUserPointsLedger.WithContext(ctx).Create(ledger); err != nil {
		return fmt.Errorf("create generation points ledger: %w", err)
	}
	updates := map[string]any{"frozen_points": gorm.Expr("frozen_points - ?", state.Score)}
	if !reserved {
		updates["frozen_points"] = user.FrozenPoints
		updates["vip_points"] = user.VipPoints
		updates["points_balance"] = user.PointsBalance
	}
	if err = m.Update(ctx, user.ID, updates); err != nil {
		return fmt.Errorf("settle generation task points: %w", err)
	}
	return nil
}

// releaseFrozenTaskPoints returns a failed task's reservation to the exact
// balances it came from. A zero allocation identifies a legacy, unreserved
// task and therefore requires no release.
func (m *Manager) releaseFrozenTaskPoints(ctx context.Context, state *repository.GenerationTaskPointState) error {
	if state.Score == 0 {
		return nil
	}
	allocation, reserved, err := pointAllocationFromState(state)
	if err != nil {
		return err
	}
	if !reserved {
		return nil
	}
	user, err := m.GetByIDForUpdate(ctx, state.UserID)
	if err != nil {
		return err
	}
	if user.FrozenPoints < uint64(state.Score) {
		return errors.New("frozen points are less than the task reservation")
	}
	if user.VipPoints > math.MaxInt64-int64(allocation.VIPScore) ||
		user.PointsBalance > math.MaxInt64-int64(allocation.PointsScore) {
		return errors.New("points balance overflow while releasing reservation")
	}
	update := map[string]any{}
	if user.SubscriptionStatus == domain.AppUserSubscriptionSubscribed {
		update["vip_points"] = gorm.Expr("vip_points + ?", allocation.VIPScore)
	}
	update["points_balance"] = gorm.Expr("points_balance + ?", allocation.PointsScore)
	update["frozen_points"] = gorm.Expr("frozen_points - ?", state.Score)
	if err = m.Update(ctx, user.ID, update); err != nil {
		return fmt.Errorf("release generation task points: %w", err)
	}
	return nil
}

func pointAllocationFromState(state *repository.GenerationTaskPointState) (taskPointAllocation, bool, error) {
	allocation := taskPointAllocation{VIPScore: state.VIPScore, PointsScore: state.PointsScore}
	total := allocation.total()
	if total == 0 {
		return allocation, false, nil
	}
	if total != uint64(state.Score) || allocation.scoreType() != state.ScoreType {
		return taskPointAllocation{}, false, errors.New("generation task point allocation is invalid")
	}
	return allocation, true, nil
}
