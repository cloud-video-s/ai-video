package jwt

import (
	"ai-video/internal/app"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type ApiClaims struct {
	UserID       uint64 `json:"user_id"`
	PhoneCode    string `json:"phone_code"`
	TokenVersion int    `json:"token_version"`
	TokenType    string `json:"token_type"`
	jwt.RegisteredClaims
}

func GenerateApiToken(userID uint64, phoneCode string, tokenVersion int) (string, error) {
	cfg := app.Cfg.JWT
	claims := ApiClaims{
		UserID:       userID,
		PhoneCode:    phoneCode,
		TokenVersion: tokenVersion,
		TokenType:    "client",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(cfg.Expire) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    cfg.Issuer,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.Secret))
}

func ParseApiToken(tokenString string) (*ApiClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &ApiClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(app.Cfg.JWT.Secret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*ApiClaims); ok && token.Valid && claims.TokenType == "client" {
		return claims, nil
	}
	return nil, errors.New("invalid token")
}
