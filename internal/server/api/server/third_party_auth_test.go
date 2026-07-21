package service

import (
	"ai-video/internal/config"
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"ai-video/internal/app"
	"ai-video/internal/domain"
	"ai-video/internal/gen/model"
	"ai-video/internal/middleware"
	apiJWT "ai-video/internal/pkg/jwt"
	"ai-video/internal/pkg/oidc"
	"ai-video/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type fakeIdentityVerifier struct {
	identity *oidc.Identity
	err      error
}

func (f fakeIdentityVerifier) Verify(context.Context, string, string) (*oidc.Identity, error) {
	return f.identity, f.err
}

func TestThirdPartyLoginUpgradesGuestAndReusesIdentity(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:third-party-login?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent), TranslateError: true})
	if err != nil {
		t.Fatal(err)
	}
	config.DB = db
	app.Cfg.JWT = app.JWTConfig{Secret: "test-third-party-jwt-secret", Expire: 3600, Issuer: "test"}
	createThirdPartyTestSchema(t, db)
	if err := db.Exec("INSERT INTO video_user (imei, username, login_type, user_type, subscription_status, status, registered, token_version, created_at, updated_at) VALUES (?, ?, 1, 1, 1, 1, 0, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)", "device-1", "guest_initial").Error; err != nil {
		t.Fatal(err)
	}
	guest, err := repository.NewAppUserRepo().GetByIMEI(context.Background(), "device-1", false)
	if err != nil {
		t.Fatal(err)
	}
	issuedAt := time.Now().Add(-time.Minute)
	identity := &oidc.Identity{
		Subject: "google-subject", Issuer: "https://accounts.google.com", Audience: "google-client",
		Email: "verified@example.com", EmailVerified: true, DisplayName: "Verified User", IssuedAt: &issuedAt,
	}
	svc := &AuthService{
		userRepo: repository.NewAppUserRepo(), attributionRepo: repository.NewUserAttributionRepo(),
		identityRepo: repository.NewUserIdentityRepo(), identityVerifiers: map[string]identityTokenVerifier{
			domain.IdentityProviderGoogle: fakeIdentityVerifier{identity: identity},
		},
	}

	response, err := svc.ThirdPartyLogin(thirdPartyTestContext(guest.ID), domain.IdentityProviderGoogle, &ThirdPartyLoginRequest{
		IDToken: "signed-token", IMEI: "device-1",
	}, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := apiJWT.ParseApiToken(response.Token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != guest.ID {
		t.Fatalf("token user ID = %d, want upgraded guest %d", claims.UserID, guest.ID)
	}
	updated, err := repository.NewAppUserRepo().GetByID(context.Background(), guest.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.LoginType != domain.AppUserLoginGoogle || !updated.Registered || updated.Email != identity.Email || updated.ThirdCode != identity.Subject {
		t.Fatalf("guest was not upgraded correctly: %+v", updated)
	}
	linked, err := repository.NewUserIdentityRepo().GetByProviderSubject(context.Background(), domain.IdentityProviderGoogle, identity.Subject, false)
	if err != nil {
		t.Fatal(err)
	}
	if linked.UserID != guest.ID {
		t.Fatalf("identity user ID = %d, want %d", linked.UserID, guest.ID)
	}

	second, err := svc.ThirdPartyLogin(thirdPartyTestContext(guest.ID), domain.IdentityProviderGoogle, &ThirdPartyLoginRequest{
		IDToken: "signed-token", IMEI: "another-device",
	}, "127.0.0.2", "test-agent-2")
	if err != nil {
		t.Fatal(err)
	}
	secondClaims, err := apiJWT.ParseApiToken(second.Token)
	if err != nil {
		t.Fatal(err)
	}
	if secondClaims.UserID != guest.ID {
		t.Fatalf("cross-device token user ID = %d, want %d", secondClaims.UserID, guest.ID)
	}
	var count int64
	if err := db.Model(&model.VideoUser{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("user count = %d, want 1", count)
	}
}

func thirdPartyTestContext(userID uint64) *gin.Context {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Set(middleware.HeaderUserIDKey, userID)
	return ctx
}

func TestUnbindIdentityRequiresAnotherLoginMethod(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:third-party-unbind?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent), TranslateError: true})
	if err != nil {
		t.Fatal(err)
	}
	config.DB = db
	createThirdPartyTestSchema(t, db)
	if err := db.Exec("INSERT INTO video_user (imei, username, login_type, user_type, subscription_status, status, registered, token_version, created_at, updated_at) VALUES (?, ?, 2, 1, 1, 1, 1, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)", "device", "user").Error; err != nil {
		t.Fatal(err)
	}
	user, err := repository.NewAppUserRepo().GetByIMEI(context.Background(), "device", false)
	if err != nil {
		t.Fatal(err)
	}
	identity := model.VideoUserIdentity{UserID: user.ID, Provider: domain.IdentityProviderGoogle, ProviderSubject: "sub", Issuer: "issuer", Audience: "audience"}
	if err := db.Create(&identity).Error; err != nil {
		t.Fatal(err)
	}
	svc := &AuthService{userRepo: repository.NewAppUserRepo(), identityRepo: repository.NewUserIdentityRepo()}
	if err := svc.UnbindIdentity(context.Background(), user.ID, domain.IdentityProviderGoogle); err == nil {
		t.Fatal("expected the last login method to be protected")
	}
}

func createThirdPartyTestSchema(t *testing.T, db *gorm.DB) {
	t.Helper()
	statements := []string{
		`CREATE TABLE video_user (
			id INTEGER PRIMARY KEY AUTOINCREMENT, imei TEXT NOT NULL UNIQUE, username TEXT NOT NULL,
			device_country TEXT DEFAULT '', channel_id TEXT DEFAULT '', app_version TEXT DEFAULT '', app_name TEXT DEFAULT '', phone_model TEXT DEFAULT '',
			first_opened_at DATETIME, last_opened_at DATETIME, attribution_clicked_at DATETIME,
			login_type INTEGER NOT NULL DEFAULT 1, user_type INTEGER NOT NULL DEFAULT 1, subscription_status INTEGER NOT NULL DEFAULT 1,
			active_days INTEGER NOT NULL DEFAULT 0, avg_daily_usage_seconds INTEGER NOT NULL DEFAULT 0, vip_expires_at DATETIME, points_balance INTEGER NOT NULL DEFAULT 0,
			re_registered_from_id INTEGER, email TEXT DEFAULT '', third_code TEXT DEFAULT '', package_code TEXT NOT NULL DEFAULT '',
			token_version INTEGER NOT NULL DEFAULT 0,
			status INTEGER NOT NULL DEFAULT 1, registered INTEGER NOT NULL DEFAULT 0, activated INTEGER NOT NULL DEFAULT 0,
			last_login_at DATETIME, last_login_ip TEXT DEFAULT '', login_account TEXT DEFAULT '', created_at DATETIME, updated_at DATETIME, deleted_at DATETIME,
			UNIQUE(imei)
		)`,
		`CREATE TABLE video_user_identity (
			id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL, provider TEXT NOT NULL, provider_subject TEXT NOT NULL,
			issuer TEXT NOT NULL, audience TEXT NOT NULL, email TEXT DEFAULT '', email_verified INTEGER NOT NULL DEFAULT 0,
			is_private_email INTEGER NOT NULL DEFAULT 0, display_name TEXT DEFAULT '', given_name TEXT DEFAULT '', family_name TEXT DEFAULT '', avatar_url TEXT DEFAULT '',
			last_login_at DATETIME, last_token_issued_at DATETIME, created_at DATETIME, updated_at DATETIME,
			UNIQUE(provider, provider_subject), UNIQUE(user_id, provider)
		)`,
		`CREATE TABLE video_user_attribution (
			id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL UNIQUE, channel_code TEXT DEFAULT '', oaid TEXT DEFAULT '', imei TEXT DEFAULT '', android_id TEXT DEFAULT '',
			ip TEXT DEFAULT '', user_agent TEXT DEFAULT '', activation_callback_count INTEGER DEFAULT 0, activation_deduct_count INTEGER DEFAULT 0,
			key_behavior_callback_count INTEGER DEFAULT 0, key_behavior_deduct_count INTEGER DEFAULT 0, payment_callback_count INTEGER DEFAULT 0, payment_deduct_count INTEGER DEFAULT 0,
			first_payment_callback_count INTEGER DEFAULT 0, first_payment_deduct_count INTEGER DEFAULT 0, registration_callback_count INTEGER DEFAULT 0, registration_deduct_count INTEGER DEFAULT 0,
			attributed_at DATETIME, last_operated_at DATETIME, remark TEXT DEFAULT '', created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
}
