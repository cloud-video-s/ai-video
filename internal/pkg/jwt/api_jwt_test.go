package jwt

import (
	"testing"

	"ai-video/internal/config"
)

func TestGenerateAPITokenCreatesUniqueTokenID(t *testing.T) {
	previous := config.Cfg.ApiJwt
	config.Cfg.ApiJwt = config.JWTConfig{Secret: "test-api-jwt-secret", Expire: 3600, Issuer: "api-test"}
	t.Cleanup(func() { config.Cfg.ApiJwt = previous })

	first, err := GenerateApiToken(7, "device-1", 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	second, err := GenerateApiToken(7, "device-1", 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("tokens issued with the same claims must still be unique")
	}

	claims, err := ParseApiToken(first)
	if err != nil {
		t.Fatal(err)
	}
	if claims.ID == "" || claims.UserID != 7 || claims.DeviceCode != "device-1" || claims.TokenVersion != 3 {
		t.Fatalf("unexpected claims: %#v", claims)
	}
}

func TestAdminAndClientTokensAreSeparated(t *testing.T) {
	previousAPI := config.Cfg.ApiJwt
	previousAdmin := config.Cfg.JWT
	config.Cfg.ApiJwt = config.JWTConfig{Secret: "test-client-secret-with-enough-entropy", Expire: 3600, Issuer: "client-test"}
	config.Cfg.JWT = config.JWTConfig{Secret: "test-admin-secret-with-enough-entropy", Expire: 3600, Issuer: "admin-test"}
	t.Cleanup(func() {
		config.Cfg.ApiJwt = previousAPI
		config.Cfg.JWT = previousAdmin
	})

	clientToken, err := GenerateApiToken(42, "device-42", 3, 1)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := ParseApiToken(clientToken)
	if err != nil {
		t.Fatalf("parse client token: %v", err)
	}
	if claims.DeviceCode != "device-42" {
		t.Fatalf("client token device_code=%q, want device-42", claims.DeviceCode)
	}
	if _, err := ParseToken(clientToken); err == nil {
		t.Fatal("client token must not be accepted as an admin token")
	}

	adminToken, err := GenerateToken(7, "admin", []string{"admin"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseToken(adminToken); err != nil {
		t.Fatalf("parse admin token: %v", err)
	}
	if _, err := ParseApiToken(adminToken); err == nil {
		t.Fatal("admin token must not be accepted as a client token")
	}
}
