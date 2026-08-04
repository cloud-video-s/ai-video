package apple

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"strings"
)

// AppleIAPConfig 保存 App Store Connect 下发的 API 凭据，用于签发
// App Store Server API V2 所需的 ES256 JWT bearer token。
// StoreKit 2 交易与 Server Notifications V2 的 JWS 验签无需这些凭据，
// 已在 ai-video/internal/commerce/apple_notification_v2.go 中使用 Apple 根证书链本地完成。
type AppleIAPConfig struct {
	IssuerID      string
	BundleID      string
	KeyID         string
	PrivateKeyPEM string
	IsProduction  bool

	privateKey *ecdsa.PrivateKey
}

// NewAppleIAPConfig 解析并校验 .p8 私钥。私钥必须包含 BEGIN/END PRIVATE KEY 头部。
func NewAppleIAPConfig(issuerID, bundleID, keyID, privateKeyPEM string, isProduction bool) (*AppleIAPConfig, error) {
	issuerID = strings.TrimSpace(issuerID)
	bundleID = strings.TrimSpace(bundleID)
	keyID = strings.TrimSpace(keyID)
	privateKeyPEM = strings.TrimSpace(privateKeyPEM)
	if issuerID == "" || bundleID == "" || keyID == "" || privateKeyPEM == "" {
		return nil, errors.New("all Apple IAP parameters are required")
	}
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return nil, errors.New("invalid Apple private key PEM")
	}
	raw, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	ecdsaKey, ok := raw.(*ecdsa.PrivateKey)
	if !ok {
		return nil, errors.New("Apple private key is not an EC private key")
	}
	return &AppleIAPConfig{
		IssuerID: issuerID, BundleID: bundleID, KeyID: keyID,
		PrivateKeyPEM: privateKeyPEM, IsProduction: isProduction,
		privateKey: ecdsaKey,
	}, nil
}

// SigningKey 返回已解析的 ES256 私钥，供后续生成 App Store Server API bearer token。
func (c *AppleIAPConfig) SigningKey() *ecdsa.PrivateKey { return c.privateKey }

// AppStoreServerAPIBaseURL 返回当前环境对应的 App Store Server API 根路径。
func (c *AppleIAPConfig) AppStoreServerAPIBaseURL() string {
	if c.IsProduction {
		return "https://api.storekit.itunes.apple.com"
	}
	return "https://api.storekit-sandbox.itunes.apple.com"
}
