package service

import (
	"context"
	"errors"
	"testing"

	"ai-video/internal/config"
	"ai-video/internal/domain"
	"ai-video/internal/pkg/cache"
	"ai-video/internal/pkg/jwt"
	"ai-video/internal/repository"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestAuthServiceLoginByAppleOrderIssuesAssociatedUserToken(t *testing.T) {
	service := newAppleOrderLoginTestService(t)
	db := config.DB

	if err := db.Exec(`INSERT INTO video_user (id, device_code, token_version, login_type, status)
		VALUES (7, 'old-device', 1, 1, 1), (8, 'linked-device', 3, 3, 1)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO video_order (id, user_id, payment_method, original_transaction_id)
		VALUES (10, 7, ?, 'original-1'), (11, 8, ?, 'original-1')`,
		domain.PaymentMethodAppleIAP, domain.PaymentMethodAppleIAP).Error; err != nil {
		t.Fatal(err)
	}

	result, err := service.LoginByAppleOrder(context.Background(), &AppleOrderLoginRequest{
		OrderCode: "  original-1  ",
	})
	if err != nil {
		t.Fatal(err)
	}
	claims, err := jwt.ParseApiToken(result.Token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != 8 || claims.DeviceCode != "linked-device" || claims.TokenVersion != 3 || claims.LoginType != 3 {
		t.Fatalf("unexpected Apple order login claims: %#v", claims)
	}
	if result.LoginType != 3 || result.TokenVersion != 3 || result.ExpireAt == 0 {
		t.Fatalf("unexpected Apple order login response: %#v", result)
	}
}

func TestAuthServiceLoginByAppleOrderReturnsSpecificLookupErrors(t *testing.T) {
	service := newAppleOrderLoginTestService(t)
	db := config.DB

	if err := db.Exec(`INSERT INTO video_order (id, user_id, payment_method, original_transaction_id)
		VALUES (20, 99, ?, 'missing-user'), (21, 1, 'stripe', 'not-apple')`,
		domain.PaymentMethodAppleIAP).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO video_user (id, device_code, token_version, login_type, status)
		VALUES (5, 'disabled-device', 0, 1, 0)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO video_order (id, user_id, payment_method, original_transaction_id)
		VALUES (22, 5, ?, 'disabled-user')`, domain.PaymentMethodAppleIAP).Error; err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		originalTransactionID string
		wantErr               error
	}{
		{originalTransactionID: "unknown", wantErr: ErrAppleOrderNotFound},
		{originalTransactionID: "not-apple", wantErr: ErrAppleOrderNotFound},
		{originalTransactionID: "missing-user", wantErr: ErrAppleOrderUserNotFound},
		{originalTransactionID: "disabled-user", wantErr: ErrAppleOrderUserDisabled},
	}
	for _, test := range tests {
		t.Run(test.originalTransactionID, func(t *testing.T) {
			_, err := service.LoginByAppleOrder(context.Background(), &AppleOrderLoginRequest{
				OrderCode: test.originalTransactionID,
			})
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("error = %v, want %v", err, test.wantErr)
			}
		})
	}
}

func newAppleOrderLoginTestService(t *testing.T) *AuthService {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	statements := []string{
		`CREATE TABLE video_user (
			id INTEGER PRIMARY KEY, device_code TEXT NOT NULL, token_version INTEGER NOT NULL,
			login_type INTEGER NOT NULL, status INTEGER NOT NULL, deleted_at DATETIME
		)`,
		`CREATE TABLE video_order (
			id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL, payment_method TEXT NOT NULL,
			original_transaction_id TEXT NOT NULL, deleted_at DATETIME
		)`,
		`CREATE TABLE video_config (
			id INTEGER PRIMARY KEY, key TEXT NOT NULL, value TEXT, deleted_at DATETIME
		)`,
		`INSERT INTO video_config (id, key, value) VALUES (1, 'user.single_device_login', 'false')`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}

	previousDB := config.DB
	previousJWT := config.Cfg.ApiJwt
	previousStore := cache.GetStore()
	config.DB = db
	config.Cfg.ApiJwt = config.JWTConfig{Secret: "apple-order-login-test-secret", Expire: 3600, Issuer: "api-test"}
	cache.InitStore(nil)
	t.Cleanup(func() {
		config.DB = previousDB
		config.Cfg.ApiJwt = previousJWT
		cache.InitStore(previousStore)
	})

	return &AuthService{userRepo: repository.NewAppUserRepo(), orderRepo: repository.NewOrderRepo()}
}
