package cache

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type memoryLockStore struct {
	Store
	mu    sync.Mutex
	locks map[string]string
}

func (s *memoryLockStore) SetNX(key, value string, _ time.Duration) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.locks[key]; exists {
		return false, nil
	}
	s.locks[key] = value
	return true, nil
}

func (s *memoryLockStore) CompareAndDelete(key, value string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.locks[key] != value {
		return false, nil
	}
	delete(s.locks, key)
	return true, nil
}

func TestAcquireLockAllowsOnlyOneConcurrentOwner(t *testing.T) {
	previous := GetStore()
	memory := &memoryLockStore{locks: make(map[string]string)}
	InitStore(memory)
	t.Cleanup(func() { InitStore(previous) })

	const workers = 32
	start := make(chan struct{})
	tokens := make(chan string, workers)
	var acquired int32
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			token, ok, err := AcquireLock("lock:favorite:1:9", time.Second)
			if err != nil {
				t.Errorf("AcquireLock: %v", err)
				return
			}
			if ok {
				atomic.AddInt32(&acquired, 1)
				tokens <- token
			}
		}()
	}
	close(start)
	wg.Wait()
	close(tokens)
	if acquired != 1 {
		t.Fatalf("acquired owners = %d, want 1", acquired)
	}
	ownerToken := <-tokens
	if err := ReleaseLock("lock:favorite:1:9", "not-the-owner"); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := AcquireLock("lock:favorite:1:9", time.Second); err != nil || ok {
		t.Fatalf("non-owner released the lock: ok=%v err=%v", ok, err)
	}
	if err := ReleaseLock("lock:favorite:1:9", ownerToken); err != nil {
		t.Fatal(err)
	}
	_, ok, err := AcquireLock("lock:favorite:1:9", time.Second)
	if err != nil || !ok {
		t.Fatalf("lock was not available after owner release: ok=%v err=%v", ok, err)
	}
}

func TestAcquireLockWithRetryWaitsForCurrentOwner(t *testing.T) {
	previous := GetStore()
	memory := &memoryLockStore{locks: make(map[string]string)}
	InitStore(memory)
	t.Cleanup(func() { InitStore(previous) })

	const lockKey = "lock:day-count:2026-09-01"
	ownerToken, ok, err := AcquireLock(lockKey, time.Second)
	if err != nil || !ok {
		t.Fatalf("acquire initial lock: ok=%v err=%v", ok, err)
	}

	released := make(chan struct{})
	go func() {
		defer close(released)
		time.Sleep(10 * time.Millisecond)
		if releaseErr := ReleaseLock(lockKey, ownerToken); releaseErr != nil {
			t.Errorf("release initial lock: %v", releaseErr)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	token, err := AcquireLockWithRetry(ctx, lockKey, time.Second, time.Millisecond)
	if err != nil {
		t.Fatalf("AcquireLockWithRetry: %v", err)
	}
	if token == "" || token == ownerToken {
		t.Fatalf("new owner token = %q, initial token = %q", token, ownerToken)
	}
	<-released
}

func TestAcquireLockWithRetryStopsWhenContextCanceled(t *testing.T) {
	previous := GetStore()
	memory := &memoryLockStore{locks: make(map[string]string)}
	InitStore(memory)
	t.Cleanup(func() { InitStore(previous) })

	const lockKey = "lock:day-count:2026-09-01"
	if _, ok, err := AcquireLock(lockKey, time.Second); err != nil || !ok {
		t.Fatalf("acquire initial lock: ok=%v err=%v", ok, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := AcquireLockWithRetry(ctx, lockKey, time.Second, time.Millisecond)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("AcquireLockWithRetry error = %v, want context.Canceled", err)
	}
}
