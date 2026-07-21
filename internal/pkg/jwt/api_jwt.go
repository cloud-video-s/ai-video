package jwt

import (
	"ai-video/internal/config"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type ApiClaims struct {
	UserID       uint64 `json:"user_id"`
	IMEI         string `json:"imei"`
	TokenVersion int64  `json:"token_version"`
	TokenType    string `json:"token_type"`
	LoginType    uint32 `json:"login_type"`
	jwt.RegisteredClaims
}

func GenerateApiToken(userID uint64, imei string, tokenVersion int64, loginType uint32) (string, error) {
	cfg := config.Cfg.JWT
	claims := ApiClaims{
		UserID:       userID,
		IMEI:         imei,
		TokenVersion: tokenVersion,
		TokenType:    "client",
		LoginType:    loginType,
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
		return []byte(config.Cfg.JWT.Secret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*ApiClaims); ok && token.Valid && claims.TokenType == "client" {
		return claims, nil
	}
	return nil, errors.New("invalid token")
}
