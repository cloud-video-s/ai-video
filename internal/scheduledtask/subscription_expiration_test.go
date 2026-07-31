package scheduledtask

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type subscriptionExpirationRepoStub struct {
	count int64
	err   error
	now   time.Time
}

func (s *subscriptionExpirationRepoStub) ExpireDueSubscriptions(_ context.Context, now time.Time) (int64, error) {
	s.now = now
	return s.count, s.err
}

func TestSubscriptionExpirationHandler(t *testing.T) {
	now := time.Date(2026, 7, 31, 16, 30, 0, 0, time.UTC)
	repo := &subscriptionExpirationRepoStub{count: 3}
	var completed int64
	handler := &subscriptionExpirationHandler{
		users: repo,
		now:   func() time.Time { return now },
		onCompleted: func(expiredUsers int64) {
			completed = expiredUsers
		},
	}

	if err := handler.Handle(context.Background(), []byte(`{}`)); err != nil {
		t.Fatalf("handle expiration task: %v", err)
	}
	if !repo.now.Equal(now) {
		t.Fatalf("repository now = %v, want %v", repo.now, now)
	}
	if completed != 3 {
		t.Fatalf("completed count = %d, want 3", completed)
	}
}

func TestSubscriptionExpirationHandlerReturnsRepositoryError(t *testing.T) {
	repoErr := errors.New("database unavailable")
	repo := &subscriptionExpirationRepoStub{err: repoErr}
	called := false
	handler := &subscriptionExpirationHandler{
		users: repo,
		now:   time.Now,
		onCompleted: func(int64) {
			called = true
		},
	}

	err := handler.Handle(context.Background(), nil)
	if !errors.Is(err, repoErr) || !strings.Contains(err.Error(), "expire due user subscriptions") {
		t.Fatalf("handler error = %v, want wrapped repository error", err)
	}
	if called {
		t.Fatal("completion callback called after repository error")
	}
}
