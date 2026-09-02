package cache

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"
)

var ErrStoreUnavailable = errors.New("cache store is unavailable")

// AcquireLock obtains a Redis-backed lock and returns its ownership token.
// Callers must pass that token to ReleaseLock so an expired/reacquired lock
// can never be released by its previous owner.
func AcquireLock(key string, ttl time.Duration) (token string, acquired bool, err error) {
	if store == nil {
		return "", false, ErrStoreUnavailable
	}
	if ttl <= 0 {
		return "", false, errors.New("lock TTL must be positive")
	}
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return "", false, err
	}
	token = hex.EncodeToString(random)
	acquired, err = store.SetNX(key, token, ttl)
	if err != nil || !acquired {
		return "", acquired, err
	}
	return token, true, nil
}

// AcquireLockWithRetry waits until a Redis-backed lock is acquired or ctx is
// canceled. retryInterval controls how often a contended lock is retried.
func AcquireLockWithRetry(ctx context.Context, key string, ttl, retryInterval time.Duration) (string, error) {
	if ctx == nil {
		return "", errors.New("lock context must not be nil")
	}
	if retryInterval <= 0 {
		return "", errors.New("lock retry interval must be positive")
	}

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		token, acquired, err := AcquireLock(key, ttl)
		if err != nil {
			return "", err
		}
		if acquired {
			return token, nil
		}

		timer := time.NewTimer(retryInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return "", ctx.Err()
		case <-timer.C:
		}
	}
}

func ReleaseLock(key, token string) error {
	if store == nil {
		return ErrStoreUnavailable
	}
	if token == "" {
		return nil
	}
	_, err := store.CompareAndDelete(key, token)
	return err
}
