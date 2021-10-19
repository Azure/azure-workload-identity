package util

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// GetIssuerHash returns a hash of the issuer URL
func GetIssuerHash(issuerURL string) string {
	h := sha256.New()
	h.Write([]byte(issuerURL))
	return base64.URLEncoding.EncodeToString(h.Sum(nil))
}

// GetFederatedCredentialSubject returns the subject of the federated credential
func GetFederatedCredentialSubject(namespace, name string) string {
	return fmt.Sprintf("system:serviceaccount:%s:%s", namespace, name)
}
