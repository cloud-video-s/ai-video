package commerce

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"ai-video/internal/config"
	"ai-video/internal/pkg/tracing"

	"github.com/golang-jwt/jwt/v5"
)

const (
	appleServerAPIProductionURL = "https://api.storekit.apple.com"
	appleServerAPISandboxURL    = "https://api.storekit-sandbox.apple.com"
	appleServerAPIAudience      = "appstoreconnect-v1"
	appleServerAPITokenTTL      = 5 * time.Minute
	appleServerAPIResponseLimit = 1 << 20
	applePrivateKeySizeLimit    = 64 << 10
	applePayChannelOne          = 1
	applePayChannelTwo          = 2
)

var (
	ErrAppleServerAPIConfig        = errors.New("Apple App Store Server API is not configured")
	ErrAppleServerAPIAuthorization = errors.New("Apple App Store Server API authorization failed")
	ErrAppleServerAPIRequest       = errors.New("Apple App Store Server API request failed")
	ErrAppleTransactionNotFound    = errors.New("Apple transaction was not found")
)

// appleTransactionInfoProvider resolves a client-supplied transaction ID to
// Apple's authoritative signedTransactionInfo response.
type appleTransactionInfoProvider interface {
	LookupTransaction(context.Context, string, string) (string, int, error)
}

type appleServerAPIClient struct {
	config        config.AppStoreConfig
	httpClient    *http.Client
	now           func() time.Time
	productionURL string
	sandboxURL    string

	privateKeyOnce sync.Once
	privateKey     *ecdsa.PrivateKey
	privateKeyErr  error
}

func newAppleServerAPIClient(cfg config.AppStoreConfig) *appleServerAPIClient {
	timeout := time.Duration(cfg.HTTPTimeoutMS) * time.Millisecond
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &appleServerAPIClient{
		config: cfg, httpClient: &http.Client{Timeout: timeout}, now: time.Now,
		productionURL: appleServerAPIProductionURL, sandboxURL: appleServerAPISandboxURL,
	}
}

// verifyApplePurchaseEvidence uses a client-provided StoreKit JWS when it has
// the required three-part compact form. Other client formats (including the
// five-part value reported by some SDK integrations) are never trusted or
// decrypted; the transaction ID is resolved through Apple's Server API and
// the returned signedTransactionInfo is then verified normally.
func (s *Service) verifyApplePurchaseEvidence(
	ctx context.Context,
	req ApplePurchaseRequest,
	expectedBundle string,
) (*verifiedAppleTransaction, error) {
	evidence := strings.TrimSpace(req.SignedTransactionInfo)
	if len(strings.Split(evidence, ".")) == 3 {
		return verifyApplePurchase(req, expectedBundle, s.appleRootCAs)
	}
	if s.appleServerAPI == nil {
		return nil, ErrAppleServerAPIConfig
	}
	signedTransaction, payChannel, err := s.appleServerAPI.LookupTransaction(
		ctx, strings.TrimSpace(req.TransactionID), strings.TrimSpace(expectedBundle),
	)
	if err != nil {
		return nil, err
	}
	req.SignedTransactionInfo = signedTransaction
	purchase, err := verifyApplePurchase(req, expectedBundle, s.appleRootCAs)
	if err != nil {
		return purchase, err
	}
	purchase.payChannel = payChannel
	return purchase, nil
}

// LookupTransaction calls Apple's current Get Transaction Info endpoint. It
// checks Production first and retries Sandbox when Apple reports that the
// transaction ID doesn't exist there. Some sandbox-only apps return 401 from
// the Production host even though the same token is valid in Sandbox, so an
// authorization failure from Production is also retried once in Sandbox.
func (c *appleServerAPIClient) LookupTransaction(
	ctx context.Context,
	transactionID string,
	bundleID string,
) (string, int, error) {
	transactionID = strings.TrimSpace(transactionID)
	bundleID = strings.TrimSpace(bundleID)
	if transactionID == "" || bundleID == "" {
		return "", 0, ErrAppleEvidenceInvalid
	}
	configuredBundleID := strings.TrimSpace(c.config.BundleID)
	if configuredBundleID != "" {
		if bundleID != configuredBundleID {
			return "", 0, ErrAppleBundleMismatch
		}
		bundleID = configuredBundleID
	}
	token, err := c.authorizationToken(bundleID)
	if err != nil {
		return "", 0, err
	}

	signedTransaction, err := c.lookupAt(ctx, c.productionURL, transactionID, token)
	if err == nil {
		return signedTransaction, applePayChannelTwo, nil
	}
	//if !errors.Is(err, ErrAppleTransactionNotFound) &&
	//	!errors.Is(err, ErrAppleServerAPIAuthorization) {
	//	return "", err
	//}
	at, err := c.lookupAt(ctx, c.sandboxURL, transactionID, token)
	return at, applePayChannelOne, err
}

func (c *appleServerAPIClient) lookupAt(
	ctx context.Context,
	baseURL string,
	transactionID string,
	token string,
) (string, error) {
	endpoint := strings.TrimRight(baseURL, "/") + "/inApps/v1/transactions/" + url.PathEscape(transactionID)
	request, err := tracing.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("%w: create request", ErrAppleServerAPIRequest)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)

	response, err := c.httpClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrAppleServerAPIRequest, err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return "", ErrAppleTransactionNotFound
	}
	if response.StatusCode != http.StatusOK {
		var apiError struct {
			ErrorCode    int64  `json:"errorCode"`
			ErrorMessage string `json:"errorMessage"`
		}
		_ = json.NewDecoder(io.LimitReader(response.Body, appleServerAPIResponseLimit)).Decode(&apiError)
		if response.StatusCode == http.StatusUnauthorized {
			return "", fmt.Errorf("%w: verify app_store.bundle_id, issuer_id, key_id, and the matching In-App Purchase .p8 key", ErrAppleServerAPIAuthorization)
		}
		if response.StatusCode == http.StatusBadRequest {
			return "", fmt.Errorf("%w: App Store Server API rejected transaction ID (Apple error %d)", ErrAppleEvidenceInvalid, apiError.ErrorCode)
		}
		if apiError.ErrorCode != 0 {
			return "", fmt.Errorf("%w: status %d, Apple error %d", ErrAppleServerAPIRequest, response.StatusCode, apiError.ErrorCode)
		}
		return "", fmt.Errorf("%w: status %d", ErrAppleServerAPIRequest, response.StatusCode)
	}

	var result struct {
		SignedTransactionInfo string `json:"signedTransactionInfo"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, appleServerAPIResponseLimit)).Decode(&result); err != nil {
		return "", fmt.Errorf("%w: invalid transaction response", ErrAppleServerAPIRequest)
	}
	result.SignedTransactionInfo = strings.TrimSpace(result.SignedTransactionInfo)
	if len(strings.Split(result.SignedTransactionInfo, ".")) != 3 {
		return "", fmt.Errorf("%w: Apple returned invalid signedTransactionInfo", ErrAppleServerAPIRequest)
	}
	return result.SignedTransactionInfo, nil
}

func (c *appleServerAPIClient) authorizationToken(bundleID string) (string, error) {
	issuerID := strings.TrimSpace(c.config.IssuerID)
	keyID := strings.TrimSpace(c.config.KeyID)
	privateKeyPath := strings.TrimSpace(c.config.PrivateKeyPath)
	if issuerID == "" || keyID == "" || privateKeyPath == "" {
		return "", fmt.Errorf("%w: app_store.issuer_id, key_id, and private_key_path are required", ErrAppleServerAPIConfig)
	}

	privateKey, err := c.loadPrivateKey()
	if err != nil {
		return "", err
	}
	now := c.now().UTC()
	token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{
		"iss": issuerID,
		"iat": now.Unix(),
		"exp": now.Add(appleServerAPITokenTTL).Unix(),
		"aud": appleServerAPIAudience,
		"bid": strings.TrimSpace(bundleID),
	})
	token.Header["kid"] = keyID
	token.Header["typ"] = "JWT"
	signed, err := token.SignedString(privateKey)
	if err != nil {
		return "", fmt.Errorf("%w: sign authorization token", ErrAppleServerAPIConfig)
	}
	return signed, nil
}

func (c *appleServerAPIClient) loadPrivateKey() (*ecdsa.PrivateKey, error) {
	c.privateKeyOnce.Do(func() {
		path := strings.TrimSpace(c.config.PrivateKeyPath)
		contents, err := os.ReadFile(path)
		if err != nil {
			c.privateKeyErr = fmt.Errorf("%w: read private key: %v", ErrAppleServerAPIConfig, err)
			return
		}
		if len(contents) == 0 || len(contents) > applePrivateKeySizeLimit {
			c.privateKeyErr = fmt.Errorf("%w: invalid private key file", ErrAppleServerAPIConfig)
			return
		}
		privateKey, err := jwt.ParseECPrivateKeyFromPEM(contents)
		if err != nil || privateKey.Curve != elliptic.P256() {
			c.privateKeyErr = fmt.Errorf("%w: private key must be an ES256 .p8 key", ErrAppleServerAPIConfig)
			return
		}
		c.privateKey = privateKey
	})
	return c.privateKey, c.privateKeyErr
}
