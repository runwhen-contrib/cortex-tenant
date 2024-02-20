package main

import (
    "crypto/x509"
    "encoding/pem"
    "net/url"
)

// ExtractCNFromCert extracts the Common Name from the given certificate
func ExtractCNFromCert(certStr string) (string, error) {
    decodedCert, err := url.QueryUnescape(certStr)
    if err != nil {
        return "", err
    }

    block, _ := pem.Decode([]byte(decodedCert))
    if block == nil {
        return "", err
    }

    cert, err := x509.ParseCertificate(block.Bytes)
    if err != nil {
        return "", err
    }

    return cert.Subject.CommonName, nil
}
