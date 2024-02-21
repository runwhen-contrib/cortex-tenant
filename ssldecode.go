package main

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/url"
)

// ExtractCNFromCert extracts the Common Name from the given certificate
func ExtractCNFromCert(certStr string) (string, error) {
    // First, try to directly decode the PEM. This will work if the input is already a proper PEM string.
    block, _ := pem.Decode([]byte(certStr))
    if block == nil {
        // If direct decoding fails, it might be URL-encoded. Attempt to unescape.
        decodedCert, err := url.QueryUnescape(certStr)
        if err != nil {
            return "", fmt.Errorf("failed to URL decode the certificate: %v", err)
        }
        // Try decoding again after unescaping.
        block, _ = pem.Decode([]byte(decodedCert))
        if block == nil {
            // If decoding fails again, the certificate is not valid PEM.
            return "", fmt.Errorf("failed to parse certificate PEM after URL decoding")
        }
    }

    // Parse the decoded PEM block to extract the certificate.
    cert, err := x509.ParseCertificate(block.Bytes)
    if err != nil {
        return "", fmt.Errorf("failed to parse certificate: %v", err)
    }

    // Return the Common Name from the certificate.
    return cert.Subject.CommonName, nil
}
