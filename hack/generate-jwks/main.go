package main

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	jose "gopkg.in/square/go-jose.v2"
	"k8s.io/client-go/util/keyutil"
	"k8s.io/klog/v2"
)

// Most of the changes here have been vendored from pkg/serviceaccount/openidmetadata.go
//  * link: https://github.com/kubernetes/kubernetes/blob/ea0764452222146c47ec826977f49d7001b0ea8c/pkg/serviceaccount/openidmetadata.go

type publicKeyGetter interface {
	Public() crypto.PublicKey
}

func main() {
	var publicKeys string
	flag.StringVar(&publicKeys, "public-keys", "", "List of public key files")
	flag.Parse()

	if publicKeys == "" {
		klog.Infof("no public keys provided")
		os.Exit(0)
	}

	var pubKeys []interface{}
	files := strings.Split(strings.TrimSpace(publicKeys), ",")
	for _, file := range files {
		pubKey, err := keyutil.PublicKeysFromFile(file)
		if err != nil {
			klog.Fatalf("failed to read public key: %v", err)
		}
		pubKeys = append(pubKeys, pubKey...)
	}

	keyset, err := publicJWKSFromKeys(pubKeys)
	if err != nil {
		klog.Fatalf("failed to generate jwks: %v", err)
	}
	keysetJSON, err := json.MarshalIndent(keyset, "", "  ")
	if err != nil {
		klog.Fatalf("failed to marshal service account issuer JWKS: %v", err)
	}
	// write the jwks to stdout
	fmt.Println(string(keysetJSON))
}

// publicJWKSFromKeys constructs a JSONWebKeySet from a list of keys. The key
// set will only contain the public keys associated with the input keys.
func publicJWKSFromKeys(in []interface{}) (*jose.JSONWebKeySet, error) {
	// Decode keys into a JWKS.
	var keys jose.JSONWebKeySet
	for _, key := range in {
		var pubkey *jose.JSONWebKey
		var err error

		switch k := key.(type) {
		case publicKeyGetter:
			// This is a private key. Get its public key
			pubkey, err = jwkFromPublicKey(k.Public())
		default:
			pubkey, err = jwkFromPublicKey(k)
		}
		if err != nil {
			return nil, err
		}

		if !pubkey.Valid() {
			return nil, fmt.Errorf("the public key is not valid")
		}
		keys.Keys = append(keys.Keys, *pubkey)
	}
	return &keys, nil
}

func jwkFromPublicKey(publicKey crypto.PublicKey) (*jose.JSONWebKey, error) {
	alg, err := algorithmFromPublicKey(publicKey)
	if err != nil {
		return nil, err
	}

	keyID, err := keyIDFromPublicKey(publicKey)
	if err != nil {
		return nil, err
	}

	jwk := &jose.JSONWebKey{
		Algorithm: string(alg),
		Key:       publicKey,
		KeyID:     keyID,
		Use:       "sig",
	}

	if !jwk.IsPublic() {
		return nil, fmt.Errorf("JWK was not a public key! JWK: %v", jwk)
	}

	return jwk, nil
}

func algorithmFromPublicKey(publicKey crypto.PublicKey) (jose.SignatureAlgorithm, error) {
	switch pk := publicKey.(type) {
	case *rsa.PublicKey:
		// IMPORTANT: If this function is updated to support additional key sizes,
		// signerFromRSAPrivateKey in serviceaccount/jwt.go must also be
		// updated to support the same key sizes. Today we only support RS256.
		return jose.RS256, nil
	case *ecdsa.PublicKey:
		switch pk.Curve {
		case elliptic.P256():
			return jose.ES256, nil
		case elliptic.P384():
			return jose.ES384, nil
		case elliptic.P521():
			return jose.ES512, nil
		default:
			return "", fmt.Errorf("unknown private key curve, must be 256, 384, or 521")
		}
	case jose.OpaqueSigner:
		return jose.SignatureAlgorithm(pk.Public().Algorithm), nil
	default:
		return "", fmt.Errorf("unknown public key type, must be *rsa.PublicKey, *ecdsa.PublicKey, or jose.OpaqueSigner")
	}
}

// keyIDFromPublicKey derives a key ID non-reversibly from a public key.
//
// The Key ID is field on a given on JWTs and JWKs that help relying parties
// pick the correct key for verification when the identity party advertises
// multiple keys.
//
// Making the derivation non-reversible makes it impossible for someone to
// accidentally obtain the real key from the key ID and use it for token
// validation.
func keyIDFromPublicKey(publicKey interface{}) (string, error) {
	publicKeyDERBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", fmt.Errorf("failed to serialize public key to DER format: %v", err)
	}

	hasher := crypto.SHA256.New()
	hasher.Write(publicKeyDERBytes)
	publicKeyDERHash := hasher.Sum(nil)

	keyID := base64.RawURLEncoding.EncodeToString(publicKeyDERHash)

	return keyID, nil
}
