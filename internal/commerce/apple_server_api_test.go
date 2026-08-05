package commerce

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ai-video/internal/config"

	"github.com/golang-jwt/jwt/v5"
)

type appleTransactionInfoProviderFunc func(context.Context, string, string) (string, error)

func (f appleTransactionInfoProviderFunc) LookupTransaction(
	ctx context.Context,
	transactionID string,
	bundleID string,
) (string, error) {
	return f(ctx, transactionID, bundleID)
}

func TestVerifyApplePurchaseEvidenceResolvesFivePartValueThroughServerAPI(t *testing.T) {
	signer := newAppleJWSTestSigner(t)
	purchaseAt := signer.now
	expiresAt := purchaseAt.Add(30 * 24 * time.Hour)
	transaction := appleSignedTransaction{
		TransactionID: "server-api-tx", OriginalTransactionID: "server-api-tx",
		BundleID: "com.example.video", ProductID: "vip.monthly",
		PurchaseDate: purchaseAt.UnixMilli(), ExpiresDate: expiresAt.UnixMilli(),
		Quantity: 1, Type: "Auto-Renewable Subscription", SignedDate: signer.now.UnixMilli(),
		Environment: "Sandbox", TransactionReason: "PURCHASE", Price: 9990, Currency: "USD",
	}
	signedTransaction := signer.sign(t, transaction)
	lookupCalls := 0
	service := &Service{
		appleRootCAs: signer.roots,
		appleServerAPI: appleTransactionInfoProviderFunc(func(_ context.Context, transactionID, bundleID string) (string, error) {
			lookupCalls++
			if transactionID != transaction.TransactionID || bundleID != transaction.BundleID {
				t.Fatalf("lookup transaction=%q bundle=%q", transactionID, bundleID)
			}
			return signedTransaction, nil
		}),
	}
	verified, err := service.verifyApplePurchaseEvidence(context.Background(), ApplePurchaseRequest{
		ShopType: 1, BundleID: transaction.BundleID, ExpirationDate: &expiresAt,
		OriginalTransactionID: transaction.OriginalTransactionID, ProductID: transaction.ProductID,
		PurchaseDate: purchaseAt, TransactionID: transaction.TransactionID,
		SignedTransactionInfo: "one.two.three.four.five",
	}, transaction.BundleID)
	if err != nil {
		t.Fatal(err)
	}
	if lookupCalls != 1 || verified.TransactionID != transaction.TransactionID ||
		verified.SignedTransaction != signedTransaction || verified.EvidenceMode != "jws" {
		t.Fatalf("unexpected verified transaction: calls=%d transaction=%#v", lookupCalls, verified)
	}
}

func TestAppleServerAPIClientFallsBackToSandboxAndSignsJWT(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	encodedKey, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatal(err)
	}
	privateKeyPath := filepath.Join(t.TempDir(), "appkey.p8")
	if err := os.WriteFile(privateKeyPath, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: encodedKey}), 0o600); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	paths := make([]string, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		paths = append(paths, request.URL.EscapedPath())
		authorization := strings.TrimPrefix(request.Header.Get("Authorization"), "Bearer ")
		parsed, parseErr := jwt.Parse(
			authorization,
			func(token *jwt.Token) (any, error) {
				if token.Header["kid"] != "key-id" || token.Header["typ"] != "JWT" {
					t.Errorf("unexpected JWT header: %#v", token.Header)
				}
				return &privateKey.PublicKey, nil
			},
			jwt.WithValidMethods([]string{jwt.SigningMethodES256.Alg()}),
			jwt.WithAudience(appleServerAPIAudience),
			jwt.WithIssuer("issuer-id"),
			jwt.WithTimeFunc(func() time.Time { return now }),
		)
		if parseErr != nil || !parsed.Valid {
			t.Errorf("invalid API authorization JWT: %v", parseErr)
			writer.WriteHeader(http.StatusUnauthorized)
			return
		}
		claims := parsed.Claims.(jwt.MapClaims)
		if claims["bid"] != "com.example.video" {
			t.Errorf("JWT bundle claim=%v", claims["bid"])
			writer.WriteHeader(http.StatusUnauthorized)
			return
		}

		writer.Header().Set("Content-Type", "application/json")
		if strings.HasPrefix(request.URL.Path, "/production/") {
			writer.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(writer).Encode(map[string]any{"errorCode": 4040010})
			return
		}
		_ = json.NewEncoder(writer).Encode(map[string]string{"signedTransactionInfo": "header.payload.signature"})
	}))
	defer server.Close()

	client := newAppleServerAPIClient(config.AppStoreConfig{
		IssuerID: "issuer-id", KeyID: "key-id", PrivateKeyPath: privateKeyPath, HTTPTimeoutMS: 1000,
	})
	client.httpClient = server.Client()
	client.now = func() time.Time { return now }
	client.productionURL = server.URL + "/production"
	client.sandboxURL = server.URL + "/sandbox"

	signedTransaction, err := client.LookupTransaction(context.Background(), "tx/123", "com.example.video")
	if err != nil {
		t.Fatal(err)
	}
	if signedTransaction != "header.payload.signature" {
		t.Fatalf("signedTransactionInfo=%q", signedTransaction)
	}
	wantPaths := []string{
		"/production/inApps/v1/transactions/tx%2F123",
		"/sandbox/inApps/v1/transactions/tx%2F123",
	}
	if len(paths) != len(wantPaths) {
		t.Fatalf("paths=%v", paths)
	}
	for index := range wantPaths {
		if paths[index] != wantPaths[index] {
			t.Fatalf("paths=%v, want %v", paths, wantPaths)
		}
	}
}

func TestAppleServerAPIClientRequiresKeyMetadata(t *testing.T) {
	client := newAppleServerAPIClient(config.AppStoreConfig{PrivateKeyPath: "config/appkey.p8"})
	if _, err := client.LookupTransaction(context.Background(), "tx-1", "com.example.video"); !errors.Is(err, ErrAppleServerAPIConfig) {
		t.Fatalf("configuration error=%v", err)
	}
}

func TestAppleServerAPIClientRejectsBundleMismatchBeforeRequest(t *testing.T) {
	client := newAppleServerAPIClient(config.AppStoreConfig{BundleID: "com.expected.app"})
	if _, err := client.LookupTransaction(context.Background(), "tx-1", "com.other.app"); !errors.Is(err, ErrAppleBundleMismatch) {
		t.Fatalf("bundle mismatch error=%v", err)
	}
}

func TestAppleServerAPIClientClassifiesInvalidTransactionID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(writer).Encode(map[string]any{"errorCode": 4000006})
	}))
	defer server.Close()
	client := newAppleServerAPIClient(config.AppStoreConfig{})
	client.httpClient = server.Client()
	if _, err := client.lookupAt(context.Background(), server.URL, "bad-transaction", "test-token"); !errors.Is(err, ErrAppleEvidenceInvalid) {
		t.Fatalf("invalid transaction error=%v", err)
	}
}

func TestAppleServerAPIClientClassifiesAuthorizationFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()
	client := newAppleServerAPIClient(config.AppStoreConfig{})
	client.httpClient = server.Client()
	if _, err := client.lookupAt(context.Background(), server.URL, "tx-1", "test-token"); !errors.Is(err, ErrAppleServerAPIAuthorization) {
		t.Fatalf("authorization error=%v", err)
	}
}

func TestAppleServerAPIClientFallsBackToSandboxAfterProductionAuthorizationFailure(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	encodedKey, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatal(err)
	}
	privateKeyPath := filepath.Join(t.TempDir(), "appkey.p8")
	if err := os.WriteFile(privateKeyPath, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: encodedKey}), 0o600); err != nil {
		t.Fatal(err)
	}

	paths := make([]string, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		paths = append(paths, request.URL.Path)
		if strings.HasPrefix(request.URL.Path, "/production/") {
			writer.WriteHeader(http.StatusUnauthorized)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(map[string]string{"signedTransactionInfo": "header.payload.signature"})
	}))
	defer server.Close()

	client := newAppleServerAPIClient(config.AppStoreConfig{
		IssuerID: "issuer-id", KeyID: "key-id", PrivateKeyPath: privateKeyPath, HTTPTimeoutMS: 1000,
	})
	client.httpClient = server.Client()
	client.productionURL = server.URL + "/production"
	client.sandboxURL = server.URL + "/sandbox"

	signedTransaction, err := client.LookupTransaction(context.Background(), "sandbox-tx", "com.example.video")
	if err != nil {
		t.Fatal(err)
	}
	if signedTransaction != "header.payload.signature" {
		t.Fatalf("signedTransactionInfo=%q", signedTransaction)
	}
	wantPaths := []string{
		"/production/inApps/v1/transactions/sandbox-tx",
		"/sandbox/inApps/v1/transactions/sandbox-tx",
	}
	if len(paths) != len(wantPaths) {
		t.Fatalf("paths=%v, want %v", paths, wantPaths)
	}
	for index := range wantPaths {
		if paths[index] != wantPaths[index] {
			t.Fatalf("paths=%v, want %v", paths, wantPaths)
		}
	}
}
