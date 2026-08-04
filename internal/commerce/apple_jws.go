package commerce

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"strings"
	"time"
)

// Apple signs App Store Server API and Server Notifications data with a leaf
// certificate containing 1.2.840.113635.100.6.11.1. Its WWDR intermediate
// contains 1.2.840.113635.100.6.2.1. These checks mirror Apple's official
// App Store Server Library verifier and prevent accepting an unrelated Apple
// certificate for a valid ES256 signature.
var (
	appleReceiptSigningOID   = []int{1, 2, 840, 113635, 100, 6, 11, 1}
	appleWWDRIntermediateOID = []int{1, 2, 840, 113635, 100, 6, 2, 1}
	defaultAppleRootCAs      = mustAppleRootCAPool()
)

// Apple Root CA - G3 is the trust anchor used by the ECC certificate chain in
// current App Store signed data. Keep this root independent from the x5c chain
// supplied by the untrusted request.
const appleRootCAG3PEM = `-----BEGIN CERTIFICATE-----
MIICQzCCAcmgAwIBAgIILcX8iNLFS5UwCgYIKoZIzj0EAwMwZzEbMBkGA1UEAwwS
QXBwbGUgUm9vdCBDQSAtIEczMSYwJAYDVQQLDB1BcHBsZSBDZXJ0aWZpY2F0aW9u
IEF1dGhvcml0eTETMBEGA1UECgwKQXBwbGUgSW5jLjELMAkGA1UEBhMCVVMwHhcN
MTQwNDMwMTgxOTA2WhcNMzkwNDMwMTgxOTA2WjBnMRswGQYDVQQDDBJBcHBsZSBS
b290IENBIC0gRzMxJjAkBgNVBAsMHUFwcGxlIENlcnRpZmljYXRpb24gQXV0aG9y
aXR5MRMwEQYDVQQKDApBcHBsZSBJbmMuMQswCQYDVQQGEwJVUzB2MBAGByqGSM49
AgEGBSuBBAAiA2IABJjpLz1AcqTtkyJygRMc3RCV8cWjTnHcFBbZDuWmBSp3ZHtf
TjjTuxxEtX/1H7YyYl3J6YRbTzBPEVoA/VhYDKX1DyxNB0cTddqXl5dvMVztK517
IDvYuVTZXpmkOlEKMaNCMEAwHQYDVR0OBBYEFLuw3qFYM4iapIqZ3r6966/ayySr
MA8GA1UdEwEB/wQFMAMBAf8wDgYDVR0PAQH/BAQDAgEGMAoGCCqGSM49BAMDA2gA
MGUCMQCD6cHEFl4aXTQY2e3v9GwOAEZLuN+yRhHFD/3meoyhpmvOwgPUnPWTxnS4
at+qIxUCMG1mihDK1A3UT82NQz60imOlM27jbdoXt2QfyFMm+YhidDkLF1vLUagM
6BgD56KyKA==
-----END CERTIFICATE-----`

// mustAppleRootCAPool parses the embedded trust anchor at process startup and
// panics when the configured PEM is not exactly one X.509 certificate.
func mustAppleRootCAPool() *x509.CertPool {
	block, rest := pem.Decode([]byte(appleRootCAG3PEM))
	if block == nil || block.Type != "CERTIFICATE" || len(strings.TrimSpace(string(rest))) != 0 {
		panic("invalid embedded Apple Root CA - G3 certificate")
	}
	certificate, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		panic("invalid embedded Apple Root CA - G3 certificate: " + err.Error())
	}
	pool := x509.NewCertPool()
	pool.AddCert(certificate)
	return pool
}

// verifyAppleJWS verifies a compact Apple JWS with the production trust store
// and decodes its authenticated JSON payload into target.
func verifyAppleJWS(compact string, target any) error {
	return verifyAppleJWSWithRoots(compact, target, defaultAppleRootCAs)
}

// verifyAppleJWSWithRoots validates the protected header, Apple certificate
// chain and certificate OIDs, ES256 signature, and signed payload. The roots
// parameter permits isolated tests to supply their own trust anchor.
func verifyAppleJWSWithRoots(compact string, target any, roots *x509.CertPool) error {
	compact = strings.TrimSpace(compact)
	if compact == "" || target == nil || roots == nil {
		return ErrAppleSignatureInvalid
	}
	parts := strings.Split(compact, ".")
	if len(parts) != 3 {
		return fmt.Errorf("%w: compact JWS must contain exactly 3 segments, got %d", ErrAppleSignatureInvalid, len(parts))
	}

	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return ErrAppleSignatureInvalid
	}
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ErrAppleSignatureInvalid
	}
	var header struct {
		Alg string   `json:"alg"`
		X5C []string `json:"x5c"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil || header.Alg != "ES256" || len(header.X5C) != 3 {
		return ErrAppleSignatureInvalid
	}

	certificates := make([]*x509.Certificate, 0, len(header.X5C))
	for _, encoded := range header.X5C {
		der, decodeErr := base64.StdEncoding.DecodeString(encoded)
		if decodeErr != nil {
			return ErrAppleSignatureInvalid
		}
		certificate, parseErr := x509.ParseCertificate(der)
		if parseErr != nil {
			return ErrAppleSignatureInvalid
		}
		certificates = append(certificates, certificate)
	}
	leaf, intermediate := certificates[0], certificates[1]
	if !hasCertificateExtension(leaf, appleReceiptSigningOID) ||
		!hasCertificateExtension(intermediate, appleWWDRIntermediateOID) {
		return ErrAppleSignatureInvalid
	}

	var signedData struct {
		SignedDate int64 `json:"signedDate"`
	}
	if err := json.Unmarshal(payloadJSON, &signedData); err != nil {
		return ErrAppleSignatureInvalid
	}
	verificationTime := time.Now()
	if signedData.SignedDate > 0 {
		verificationTime = time.UnixMilli(signedData.SignedDate)
	}
	intermediates := x509.NewCertPool()
	intermediates.AddCert(intermediate)
	if _, err := leaf.Verify(x509.VerifyOptions{
		Roots:         roots,
		Intermediates: intermediates,
		CurrentTime:   verificationTime,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	}); err != nil {
		return fmt.Errorf("%w: %v", ErrAppleSignatureInvalid, err)
	}

	publicKey, ok := leaf.PublicKey.(*ecdsa.PublicKey)
	if !ok || publicKey.Curve != elliptic.P256() {
		return ErrAppleSignatureInvalid
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || len(signature) != 64 {
		return ErrAppleSignatureInvalid
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	r := new(big.Int).SetBytes(signature[:32])
	s := new(big.Int).SetBytes(signature[32:])
	if r.Sign() <= 0 || s.Sign() <= 0 || !ecdsa.Verify(publicKey, digest[:], r, s) {
		return ErrAppleSignatureInvalid
	}
	if err := json.Unmarshal(payloadJSON, target); err != nil {
		return fmt.Errorf("%w: invalid signed JSON", ErrAppleEvidenceInvalid)
	}
	return nil
}

// hasCertificateExtension reports whether a certificate carries the Apple
// purpose extension required for receipt signing or WWDR intermediates.
func hasCertificateExtension(certificate *x509.Certificate, oid []int) bool {
	if certificate == nil {
		return false
	}
	want := pkix.Extension{Id: oid}.Id
	for _, extension := range certificate.Extensions {
		if extension.Id.Equal(want) {
			return true
		}
	}
	return false
}
