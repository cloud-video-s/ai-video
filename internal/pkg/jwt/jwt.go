package jwt

import (
	"ai-video/internal/config"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type AdminClaims struct {
	UserID       uint64   `json:"user_id"`
	Username     string   `json:"username"`
	RoleCodes    []string `json:"role_codes"`
	TokenVersion int64    `json:"token_version"`
	TokenType    string   `json:"token_type"`
	jwt.RegisteredClaims
}

const AdminTokenRenewalWindow = 30 * time.Minute

func GenerateToken(userID uint64, username string, roleCodes []string, tokenVersion int64) (string, error) {
	cfg := config.Cfg.JWT
	now := time.Now()
	claims := AdminClaims{
		UserID:       userID,
		Username:     username,
		RoleCodes:    roleCodes,
		TokenVersion: tokenVersion,
		TokenType:    "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(cfg.Expire) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    cfg.Issuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.Secret))
}

// NeedsAdminTokenRenewal reports whether a valid admin token has entered the
// sliding renewal window. Expired tokens are never eligible for renewal.
func NeedsAdminTokenRenewal(claims *AdminClaims, now time.Time) bool {
	if claims == nil || claims.ExpiresAt == nil {
		return false
	}
	remaining := claims.ExpiresAt.Time.Sub(now)
	return remaining > 0 && remaining <= AdminTokenRenewalWindow
}

func ParseToken(tokenString string) (*AdminClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AdminClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(config.Cfg.JWT.Secret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*AdminClaims); ok && token.Valid && claims.TokenType == "admin" {
		return claims, nil
	}
	return nil, errors.New("invalid token")
}
