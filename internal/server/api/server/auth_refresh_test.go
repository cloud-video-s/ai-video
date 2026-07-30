package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"ai-video/internal/config"
	"ai-video/internal/gen/model"
	"ai-video/internal/pkg/cache"
	"ai-video/internal/pkg/jwt"
	"ai-video/internal/repository"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type refreshTokenStore struct {
	cache.Store
	values      map[string]string
	expirations map[string]time.Duration
}

func (s *refreshTokenStore) Get(key string) (string, error) {
	value, ok := s.values[key]
	if !ok {
		return "", errors.New("cache miss")
	}
	return value, nil
}

func (s *refreshTokenStore) Set(key, value string, expiration time.Duration) error {
	s.values[key] = value
	s.expirations[key] = expiration
	return nil
}

func TestAuthServiceRefreshRotatesAPIToken(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:auth-refresh?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE video_user (
		id INTEGER PRIMARY KEY,
		device_code TEXT NOT NULL,
		token_version INTEGER NOT NULL,
		login_type INTEGER NOT NULL,
		status INTEGER NOT NULL,
		is_frozen INTEGER NOT NULL,
		is_blacklisted INTEGER NOT NULL,
		deleted_at DATETIME
	)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO video_user
		(id, device_code, token_version, login_type, status, is_frozen, is_blacklisted)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, 7, "device-1", 3, 2, 1, 0, 0).Error; err != nil {
		t.Fatal(err)
	}

	previousDB := config.DB
	previousJWT := config.Cfg.ApiJwt
	previousStore := cache.GetStore()
	store := &refreshTokenStore{values: make(map[string]string), expirations: make(map[string]time.Duration)}
	config.DB = db
	config.Cfg.ApiJwt = config.JWTConfig{Secret: "test-api-jwt-secret", Expire: 3600, Issuer: "api-test"}
	cache.InitStore(store)
	t.Cleanup(func() {
		config.DB = previousDB
		config.Cfg.ApiJwt = previousJWT
		cache.InitStore(previousStore)
	})

	current, err := issueToken(&model.VideoUser{ID: 7, DeviceCode: "device-1", TokenVersion: 3, LoginType: 2}, 2)
	if err != nil {
		t.Fatal(err)
	}
	service := &AuthService{userRepo: repository.NewAppUserRepo()}
	refreshed, err := service.Refresh(context.Background(), 7, 3, current.Token)
	if err != nil {
		t.Fatal(err)
	}
	if refreshed.Token == current.Token {
		t.Fatal("refresh returned the current token")
	}
	if !cache.IsTokenBlacklisted(current.Token) {
		t.Fatal("current token was not revoked")
	}
	if store.expirations["token:blacklist:"+current.Token] <= 0 {
		t.Fatal("revoked token must have a positive blacklist TTL")
	}
	claims, err := jwt.ParseApiToken(refreshed.Token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != 7 || claims.DeviceCode != "device-1" || claims.TokenVersion != 3 || claims.LoginType != 2 {
		t.Fatalf("unexpected refreshed claims: %#v", claims)
	}
}
