package main

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/url"
)

// ExtractCNFromCert returns the Common Name (CN) from the Subject of a
// PEM-encoded X.509 certificate.
//
// The input may either be:
//   - a raw PEM string (typical when an upstream proxy injects the client
//     cert directly), or
//   - a URL-encoded PEM string (typical when the cert is forwarded in an
//     HTTP header, since the header transport often percent-encodes the
//     PEM block delimiters and newlines).
//
// The function tries the raw form first and falls back to URL-decoding
// only if the raw decode fails — keeping the happy path allocation-free.
//
// Used only when CN validation is enabled (see Config.CNValidation.Enabled).
// When that feature is disabled the function is never called.
func ExtractCNFromCert(certStr string) (string, error) {
	block, _ := pem.Decode([]byte(certStr))
	if block == nil {
		decodedCert, err := url.QueryUnescape(certStr)
		if err != nil {
			return "", fmt.Errorf("failed to URL decode certificate header: %w", err)
		}
		block, _ = pem.Decode([]byte(decodedCert))
		if block == nil {
			return "", fmt.Errorf("certificate header is not valid PEM (tried both raw and URL-decoded)")
		}
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("failed to parse certificate: %w", err)
	}

	return cert.Subject.CommonName, nil
}
