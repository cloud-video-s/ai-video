package jwt

import (
	"testing"

	"ai-video/internal/app"
)

func TestAdminAndClientTokensAreSeparated(t *testing.T) {
	previous := app.Cfg.JWT
	app.Cfg.JWT = app.JWTConfig{Secret: "test-secret-with-enough-entropy", Expire: 3600, Issuer: "test"}
	t.Cleanup(func() { app.Cfg.JWT = previous })

	clientToken, err := GenerateApiToken(42, "device-42", 3)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseApiToken(clientToken); err != nil {
		t.Fatalf("parse client token: %v", err)
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
