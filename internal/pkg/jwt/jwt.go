package jwt

import (
	"ai-video/internal/app"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type AdminClaims struct {
	UserID       uint     `json:"user_id"`
	Username     string   `json:"username"`
	RoleCodes    []string `json:"role_codes"`
	TokenVersion int      `json:"token_version"`
	TokenType    string   `json:"token_type"`
	jwt.RegisteredClaims
}

func GenerateToken(userID uint, username string, roleCodes []string, tokenVersion int) (string, error) {
	cfg := app.Cfg.JWT
	claims := AdminClaims{
		UserID:       userID,
		Username:     username,
		RoleCodes:    roleCodes,
		TokenVersion: tokenVersion,
		TokenType:    "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(cfg.Expire) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    cfg.Issuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.Secret))
}

func ParseToken(tokenString string) (*AdminClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AdminClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(app.Cfg.JWT.Secret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*AdminClaims); ok && token.Valid && claims.TokenType == "admin" {
		return claims, nil
	}
	return nil, errors.New("invalid token")
}
