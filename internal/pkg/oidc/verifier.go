package oidc

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("invalid identity token")

type Config struct {
	Issuers    []string
	Audiences  []string
	JWKSURL    string
	HTTPClient *http.Client
	CacheTTL   time.Duration
	Now        func() time.Time
}

type Identity struct {
	Subject        string
	Issuer         string
	Audience       string
	Email          string
	EmailVerified  bool
	IsPrivateEmail bool
	DisplayName    string
	GivenName      string
	FamilyName     string
	AvatarURL      string
	Nonce          string
	IssuedAt       *time.Time
}

type Verifier struct {
	config    Config
	mu        sync.Mutex
	keys      map[string]*rsa.PublicKey
	expiresAt time.Time
}

func NewVerifier(config Config) *Verifier {
	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{Timeout: 5 * time.Second}
	}
	if config.CacheTTL <= 0 {
		config.CacheTTL = 6 * time.Hour
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	return &Verifier{config: config, keys: make(map[string]*rsa.PublicKey)}
}

func (v *Verifier) Verify(ctx context.Context, rawToken, expectedNonce string) (*Identity, error) {
	if strings.TrimSpace(rawToken) == "" {
		return nil, fmt.Errorf("%w: token is required", ErrInvalidToken)
	}
	if len(v.config.Audiences) == 0 || len(v.config.Issuers) == 0 || strings.TrimSpace(v.config.JWKSURL) == "" {
		return nil, errors.New("identity provider is not configured")
	}
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(rawToken, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != jwt.SigningMethodRS256.Alg() {
			return nil, fmt.Errorf("%w: unsupported signing algorithm", ErrInvalidToken)
		}
		kid, _ := token.Header["kid"].(string)
		if kid == "" {
			return nil, fmt.Errorf("%w: signing key ID is missing", ErrInvalidToken)
		}
		return v.signingKey(ctx, kid)
	}, jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}), jwt.WithExpirationRequired(), jwt.WithLeeway(30*time.Second), jwt.WithTimeFunc(v.config.Now))
	if err != nil || token == nil || !token.Valid {
		if err == nil {
			err = ErrInvalidToken
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	issuer, err := claims.GetIssuer()
	if err != nil || !contains(v.config.Issuers, issuer) {
		return nil, fmt.Errorf("%w: issuer is not allowed", ErrInvalidToken)
	}
	audiences, err := claims.GetAudience()
	if err != nil {
		return nil, fmt.Errorf("%w: audience is invalid", ErrInvalidToken)
	}
	audience := firstMatch(v.config.Audiences, audiences)
	if audience == "" {
		return nil, fmt.Errorf("%w: audience is not allowed", ErrInvalidToken)
	}
	if authorizedParty := stringClaim(claims, "azp"); authorizedParty != "" && !contains(v.config.Audiences, authorizedParty) {
		return nil, fmt.Errorf("%w: authorized party is not allowed", ErrInvalidToken)
	}
	subject, err := claims.GetSubject()
	if err != nil || strings.TrimSpace(subject) == "" {
		return nil, fmt.Errorf("%w: subject is missing", ErrInvalidToken)
	}
	nonce := stringClaim(claims, "nonce")
	if expectedNonce != "" && nonce != expectedNonce {
		return nil, fmt.Errorf("%w: nonce does not match", ErrInvalidToken)
	}
	issuedAt, err := claims.GetIssuedAt()
	if err != nil || issuedAt == nil {
		return nil, fmt.Errorf("%w: issued-at time is missing", ErrInvalidToken)
	}
	if issuedAt.Time.After(v.config.Now().Add(30 * time.Second)) {
		return nil, fmt.Errorf("%w: issued-at time is in the future", ErrInvalidToken)
	}
	issued := issuedAt.Time
	return &Identity{
		Subject: subject, Issuer: issuer, Audience: audience, Email: strings.TrimSpace(stringClaim(claims, "email")),
		EmailVerified: boolClaim(claims, "email_verified"), IsPrivateEmail: boolClaim(claims, "is_private_email"),
		DisplayName: stringClaim(claims, "name"), GivenName: stringClaim(claims, "given_name"),
		FamilyName: stringClaim(claims, "family_name"), AvatarURL: stringClaim(claims, "picture"), Nonce: nonce,
		IssuedAt: &issued,
	}, nil
}

func (v *Verifier) signingKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if key := v.keys[kid]; key != nil && v.config.Now().Before(v.expiresAt) {
		return key, nil
	}
	if err := v.refreshKeys(ctx); err != nil {
		return nil, err
	}
	key := v.keys[kid]
	if key == nil {
		return nil, fmt.Errorf("%w: signing key was not found", ErrInvalidToken)
	}
	return key, nil
}

type jwkSet struct {
	Keys []jwk `json:"keys"`
}
type jwk struct {
	KTY string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func (v *Verifier) refreshKeys(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.config.JWKSURL, nil)
	if err != nil {
		return err
	}
	resp, err := v.config.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch identity provider keys: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("fetch identity provider keys: HTTP %d", resp.StatusCode)
	}
	var set jwkSet
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&set); err != nil {
		return fmt.Errorf("decode identity provider keys: %w", err)
	}
	keys := make(map[string]*rsa.PublicKey)
	for _, value := range set.Keys {
		if value.KTY != "RSA" || value.Kid == "" || (value.Alg != "" && value.Alg != "RS256") || (value.Use != "" && value.Use != "sig") {
			continue
		}
		key, err := rsaKey(value.N, value.E)
		if err == nil {
			keys[value.Kid] = key
		}
	}
	if len(keys) == 0 {
		return errors.New("identity provider returned no usable signing keys")
	}
	v.keys = keys
	ttl := cacheTTL(resp.Header.Get("Cache-Control"), v.config.CacheTTL)
	v.expiresAt = v.config.Now().Add(ttl)
	return nil
}

func rsaKey(encodedN, encodedE string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(encodedN)
	if err != nil || len(nBytes) == 0 {
		return nil, errors.New("invalid RSA modulus")
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(encodedE)
	if err != nil || len(eBytes) == 0 || len(eBytes) > 8 {
		return nil, errors.New("invalid RSA exponent")
	}
	exponent := 0
	for _, value := range eBytes {
		exponent = (exponent << 8) | int(value)
	}
	if exponent < 3 {
		return nil, errors.New("invalid RSA exponent")
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: exponent}, nil
}

func cacheTTL(cacheControl string, fallback time.Duration) time.Duration {
	for _, directive := range strings.Split(cacheControl, ",") {
		parts := strings.SplitN(strings.TrimSpace(directive), "=", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "max-age") {
			seconds, err := strconv.ParseInt(strings.Trim(parts[1], `"`), 10, 64)
			if err == nil && seconds > 0 {
				return time.Duration(seconds) * time.Second
			}
		}
	}
	return fallback
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
func firstMatch(allowed, actual []string) string {
	for _, value := range actual {
		if contains(allowed, value) {
			return value
		}
	}
	return ""
}
func stringClaim(claims jwt.MapClaims, key string) string {
	value, _ := claims[key].(string)
	return value
}
func boolClaim(claims jwt.MapClaims, key string) bool {
	switch value := claims[key].(type) {
	case bool:
		return value
	case string:
		parsed, _ := strconv.ParseBool(value)
		return parsed
	default:
		return false
	}
}
