package cache

import (
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
