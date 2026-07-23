package cache

import (
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
