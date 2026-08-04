package commerce

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"strings"
	"testing"
	"time"
)

type appleJWSTestSigner struct {
	privateKey *ecdsa.PrivateKey
	x5c        []string
	roots      *x509.CertPool
	now        time.Time
}

func newAppleJWSTestSigner(t *testing.T) *appleJWSTestSigner {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Millisecond)
	rootKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	rootTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Apple test root"},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}
	rootDER, err := x509.CreateCertificate(rand.Reader, rootTemplate, rootTemplate, &rootKey.PublicKey, rootKey)
	if err != nil {
		t.Fatal(err)
	}
	rootCertificate, err := x509.ParseCertificate(rootDER)
	if err != nil {
		t.Fatal(err)
	}

	intermediateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	intermediateTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(2),
		Subject:               pkix.Name{CommonName: "Apple test WWDR intermediate"},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(12 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		ExtraExtensions: []pkix.Extension{{
			Id: asn1.ObjectIdentifier(appleWWDRIntermediateOID), Value: []byte{0x05, 0x00},
		}},
	}
	intermediateDER, err := x509.CreateCertificate(
		rand.Reader, intermediateTemplate, rootCertificate, &intermediateKey.PublicKey, rootKey,
	)
	if err != nil {
		t.Fatal(err)
	}
	intermediateCertificate, err := x509.ParseCertificate(intermediateDER)
	if err != nil {
		t.Fatal(err)
	}

	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	leafTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject:      pkix.Name{CommonName: "Apple test receipt signer"},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(6 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtraExtensions: []pkix.Extension{{
			Id: asn1.ObjectIdentifier(appleReceiptSigningOID), Value: []byte{0x05, 0x00},
		}},
	}
	leafDER, err := x509.CreateCertificate(
		rand.Reader, leafTemplate, intermediateCertificate, &leafKey.PublicKey, intermediateKey,
	)
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	roots.AddCert(rootCertificate)
	return &appleJWSTestSigner{
		privateKey: leafKey,
		x5c: []string{
			base64.StdEncoding.EncodeToString(leafDER),
			base64.StdEncoding.EncodeToString(intermediateDER),
			base64.StdEncoding.EncodeToString(rootDER),
		},
		roots: roots,
		now:   now,
	}
}

func (s *appleJWSTestSigner) sign(t *testing.T, payload any) string {
	t.Helper()
	headerJSON, err := json.Marshal(map[string]any{"alg": "ES256", "x5c": s.x5c})
	if err != nil {
		t.Fatal(err)
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	header := base64.RawURLEncoding.EncodeToString(headerJSON)
	body := base64.RawURLEncoding.EncodeToString(payloadJSON)
	digest := sha256.Sum256([]byte(header + "." + body))
	rValue, sValue, err := ecdsa.Sign(rand.Reader, s.privateKey, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	signature := make([]byte, 64)
	rValue.FillBytes(signature[:32])
	sValue.FillBytes(signature[32:])
	return header + "." + body + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func TestDecodeAppleNotificationPayloadVerifiesNestedData(t *testing.T) {
	signer := newAppleJWSTestSigner(t)
	signedDate := signer.now.UnixMilli()
	signedTransaction := signer.sign(t, map[string]any{
		"transactionId": "tx-1", "originalTransactionId": "original-1",
		"bundleId": "com.example.video", "productId": "vip.monthly",
		"purchaseDate": signedDate - 1_000, "expiresDate": signedDate + 3_600_000,
		"environment": "Sandbox", "signedDate": signedDate,
	})
	signedRenewalInfo := signer.sign(t, map[string]any{
		"environment": "Sandbox", "signedDate": signedDate,
	})
	signedPayload := signer.sign(t, map[string]any{
		"notificationType": AppleNotificationDidRenew,
		"notificationUUID": "notification-1", "version": "2.0", "signedDate": signedDate,
		"data": map[string]any{
			"appAppleId": int64(123456789), "bundleId": "com.example.video",
			"environment": "Sandbox", "status": 1,
			"signedTransactionInfo": signedTransaction, "signedRenewalInfo": signedRenewalInfo,
		},
	})

	decoded, err := decodeAppleNotificationPayload(signedPayload, signer.roots)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.NotificationType != AppleNotificationDidRenew || decoded.NotificationUUID != "notification-1" ||
		decoded.Version != "2.0" || decoded.SignedDate != signedDate || decoded.AppAppleID != 123456789 ||
		decoded.SubscriptionStatus != 1 || decoded.TransactionID != "tx-1" ||
		decoded.OriginalTransaction != "original-1" || decoded.ProductID != "vip.monthly" {
		t.Fatalf("unexpected decoded notification: %#v", decoded)
	}
}

func TestDecodeAppleNotificationPayloadRejectsInvalidCompactJWS(t *testing.T) {
	for _, value := range []string{"one-part", "a.b", "a.b.c.d", "header.payload.signature"} {
		if _, err := decodeAppleNotificationPayload(value, newAppleJWSTestSigner(t).roots); !errors.Is(err, ErrAppleSignatureInvalid) {
			t.Fatalf("payload %q error=%v, want signature error", value, err)
		}
	}
	if _, err := decodeAppleNotificationPayload("a.b.c.d.e", newAppleJWSTestSigner(t).roots); !errors.Is(err, ErrAppleSignatureInvalid) || !strings.Contains(err.Error(), "exactly 3 segments, got 5") {
		t.Fatalf("five-part payload error=%v", err)
	}
}

func TestDecodeAppleNotificationPayloadRejectsTamperingAndBundleMismatch(t *testing.T) {
	signer := newAppleJWSTestSigner(t)
	signedDate := signer.now.UnixMilli()
	transaction := signer.sign(t, map[string]any{
		"transactionId": "tx-mismatch", "originalTransactionId": "original-mismatch",
		"bundleId": "com.other.app", "productId": "vip.monthly",
		"environment": "Sandbox", "signedDate": signedDate,
	})
	payload := signer.sign(t, map[string]any{
		"notificationType": AppleNotificationDidRenew, "signedDate": signedDate,
		"data": map[string]any{
			"bundleId": "com.example.video", "environment": "Sandbox",
			"signedTransactionInfo": transaction,
		},
	})
	if _, err := decodeAppleNotificationPayload(payload, signer.roots); !errors.Is(err, ErrAppleBundleMismatch) {
		t.Fatalf("bundle mismatch error=%v", err)
	}

	transaction = signer.sign(t, map[string]any{
		"transactionId": "tx-environment", "originalTransactionId": "original-environment",
		"bundleId": "com.example.video", "productId": "vip.monthly",
		"environment": "Production", "signedDate": signedDate,
	})
	environmentMismatch := signer.sign(t, map[string]any{
		"notificationType": AppleNotificationDidRenew, "signedDate": signedDate,
		"data": map[string]any{
			"bundleId": "com.example.video", "environment": "Sandbox",
			"signedTransactionInfo": transaction,
		},
	})
	if _, err := decodeAppleNotificationPayload(environmentMismatch, signer.roots); !errors.Is(err, ErrAppleEnvironmentMismatch) {
		t.Fatalf("environment mismatch error=%v", err)
	}

	parts := strings.Split(payload, ".")
	parts[2] = strings.Repeat("A", len(parts[2]))
	if _, err := decodeAppleNotificationPayload(strings.Join(parts, "."), signer.roots); !errors.Is(err, ErrAppleSignatureInvalid) {
		t.Fatalf("tampered signature error=%v", err)
	}
}

func TestDecodeAppleNotificationPayloadSupportsSummaryShape(t *testing.T) {
	signer := newAppleJWSTestSigner(t)
	signedPayload := signer.sign(t, map[string]any{
		"notificationType": AppleNotificationRenewalExtension,
		"subtype":          AppleSubtypeSummary, "notificationUUID": "summary-1",
		"version": "2.0", "signedDate": signer.now.UnixMilli(),
		"summary": map[string]any{
			"environment": "Sandbox", "appAppleId": int64(123456789),
			"bundleId": "com.example.video", "productId": "vip.monthly",
		},
	})
	decoded, err := decodeAppleNotificationPayload(signedPayload, signer.roots)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.BundleID != "com.example.video" || decoded.Environment != "Sandbox" ||
		decoded.ProductID != "vip.monthly" || decoded.Subtype != AppleSubtypeSummary {
		t.Fatalf("unexpected summary notification: %#v", decoded)
	}
}

func TestDecodeAppleNotificationV2PayloadVerifiesAppData(t *testing.T) {
	signer := newAppleJWSTestSigner(t)
	signedDate := signer.now.UnixMilli()
	signedAppTransaction := signer.sign(t, map[string]any{
		"environment": "Sandbox",
		"bundleId":    "com.example.video", "signedDate": signedDate,
	})
	signedPayload := signer.sign(t, map[string]any{
		"notificationType": AppleNotificationMigration,
		"notificationUUID": "migration-1", "version": "2.0", "signedDate": signedDate,
		"appData": map[string]any{
			"environment": "Sandbox", "appAppleId": int64(123456789),
			"bundleId": "com.example.video", "signedAppTransactionInfo": signedAppTransaction,
		},
	})

	decoded, err := decodeAppleNotificationV2Payload(signedPayload, signer.roots)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.NotificationType != AppleNotificationMigration || decoded.BundleID != "com.example.video" ||
		decoded.Environment != "Sandbox" || decoded.AppAppleID != 123456789 {
		t.Fatalf("unexpected appData notification: %#v", decoded)
	}

	mismatchedAppTransaction := signer.sign(t, map[string]any{
		"environment": "Production",
		"bundleId":    "com.example.video", "signedDate": signedDate,
	})
	mismatchedPayload := signer.sign(t, map[string]any{
		"notificationType": AppleNotificationMigration, "signedDate": signedDate,
		"appData": map[string]any{
			"environment": "Sandbox", "appAppleId": int64(123456789),
			"bundleId": "com.example.video", "signedAppTransactionInfo": mismatchedAppTransaction,
		},
	})
	if _, err := decodeAppleNotificationV2Payload(mismatchedPayload, signer.roots); !errors.Is(err, ErrAppleEnvironmentMismatch) {
		t.Fatalf("app transaction environment mismatch error=%v", err)
	}
}

func TestDecodeAppleNotificationV2PayloadSupportsExternalPurchaseToken(t *testing.T) {
	signer := newAppleJWSTestSigner(t)
	signedPayload := signer.sign(t, map[string]any{
		"notificationType": AppleNotificationExternalPurchaseToken,
		"notificationUUID": "external-purchase-1", "version": "2.0", "signedDate": signer.now.UnixMilli(),
		"externalPurchaseToken": map[string]any{
			"externalPurchaseId": "SANDBOX-123456", "tokenCreationDate": signer.now.UnixMilli(),
			"appAppleId": int64(123456789), "bundleId": "com.example.video",
		},
	})

	decoded, err := decodeAppleNotificationV2Payload(signedPayload, signer.roots)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Environment != "Sandbox" || decoded.BundleID != "com.example.video" || decoded.AppAppleID != 123456789 {
		t.Fatalf("unexpected external purchase notification: %#v", decoded)
	}
}
