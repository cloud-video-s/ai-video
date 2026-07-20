package oidc

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestVerifierVerifyRS256IdentityToken(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	kid := "test-key"
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		w.Header().Set("Cache-Control", "public, max-age=3600")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"keys": []map[string]string{{
			"kty": "RSA", "kid": kid, "use": "sig", "alg": "RS256",
			"n": base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.PublicKey.E)).Bytes()),
		}}})
	}))
	defer server.Close()

	now := time.Date(2026, time.July, 17, 10, 0, 0, 0, time.UTC)
	verifier := NewVerifier(Config{
		Issuers: []string{"https://issuer.example"}, Audiences: []string{"client-id"},
		JWKSURL: server.URL, HTTPClient: server.Client(), Now: func() time.Time { return now },
	})
	token := signedToken(t, privateKey, kid, jwt.MapClaims{
		"iss": "https://issuer.example", "aud": "client-id", "sub": "stable-subject",
		"iat": now.Unix(), "exp": now.Add(5 * time.Minute).Unix(), "nonce": "nonce-1",
		"email": "user@example.com", "email_verified": "true", "name": "Test User",
	})
	identity, err := verifier.Verify(t.Context(), token, "nonce-1")
	if err != nil {
		t.Fatal(err)
	}
	if identity.Subject != "stable-subject" || identity.Email != "user@example.com" || !identity.EmailVerified {
		t.Fatalf("unexpected identity: %+v", identity)
	}
	if requests != 1 {
		t.Fatalf("JWKS requests = %d, want 1", requests)
	}
	if _, err := verifier.Verify(t.Context(), token, "nonce-1"); err != nil {
		t.Fatal(err)
	}
	if requests != 1 {
		t.Fatalf("cached JWKS requests = %d, want 1", requests)
	}
}

func TestVerifierRejectsAudienceNonceAndExpiry(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	kid := "test-key"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"keys": []map[string]string{{
			"kty": "RSA", "kid": kid, "alg": "RS256",
			"n": base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.PublicKey.E)).Bytes()),
		}}})
	}))
	defer server.Close()
	now := time.Date(2026, time.July, 17, 10, 0, 0, 0, time.UTC)
	verifier := NewVerifier(Config{Issuers: []string{"issuer"}, Audiences: []string{"client"}, JWKSURL: server.URL, HTTPClient: server.Client(), Now: func() time.Time { return now }})

	tests := []struct {
		name   string
		claims jwt.MapClaims
		nonce  string
	}{
		{name: "wrong audience", claims: jwt.MapClaims{"iss": "issuer", "aud": "other", "sub": "sub", "iat": now.Unix(), "exp": now.Add(time.Minute).Unix()}},
		{name: "wrong nonce", claims: jwt.MapClaims{"iss": "issuer", "aud": "client", "sub": "sub", "iat": now.Unix(), "exp": now.Add(time.Minute).Unix(), "nonce": "signed"}, nonce: "requested"},
		{name: "expired", claims: jwt.MapClaims{"iss": "issuer", "aud": "client", "sub": "sub", "iat": now.Add(-time.Hour).Unix(), "exp": now.Add(-time.Minute).Unix()}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := verifier.Verify(t.Context(), signedToken(t, privateKey, kid, test.claims), test.nonce); err == nil {
				t.Fatal("expected token verification failure")
			}
		})
	}
}

func signedToken(t *testing.T, key *rsa.PrivateKey, kid string, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	value, err := token.SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return value
}
