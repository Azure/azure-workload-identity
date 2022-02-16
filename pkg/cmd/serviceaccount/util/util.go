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

// GetFederatedCredentialName returns a hash of
// the service account namespace, name, and issuer URL
func GetFederatedCredentialName(namespace, name, issuerURL string) string {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%s-%s-%s", namespace, name, issuerURL)))
	return base64.URLEncoding.EncodeToString(h.Sum(nil))
}

// GetFederatedCredentialSubject returns the subject of the federated credential
func GetFederatedCredentialSubject(namespace, name string) string {
	return fmt.Sprintf("system:serviceaccount:%s:%s", namespace, name)
}
