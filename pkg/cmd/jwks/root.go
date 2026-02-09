package jwks

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	jose "gopkg.in/go-jose/go-jose.v2"
	"k8s.io/client-go/util/keyutil"
	"monis.app/mlog"
)

type jwksCmd struct {
	publicKeys []string
	outputFile string
}

// NewJWKSCmd returns a new serviceaccount command
func NewJWKSCmd() *cobra.Command {
	jwksCmd := &jwksCmd{}

	cmd := &cobra.Command{
		Use:   "jwks",
		Short: "JSON Web Key Sets for the service account issuer keys",
		Long:  "This command provides the ability to generate a JSON Web Key Sets (JWKS) for the service account issuer keys",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return jwksCmd.validate()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return jwksCmd.run()
		},
	}

	f := cmd.Flags()
	f.StringSliceVar(&jwksCmd.publicKeys, "public-keys", nil, "List of public keys to include in the JWKS")
	f.StringVar(&jwksCmd.outputFile, "output-file", "", "The name of the file to write the JWKS to. If not provided, the default output is stdout")

	_ = cmd.MarkFlagRequired("public-keys")

	return cmd
}

func (jc *jwksCmd) validate() error {
	if len(jc.publicKeys) == 0 {
		return errors.New("no public keys provided")
	}
	return nil
}

func (jc *jwksCmd) run() error {
	mlog.Debug("generating JSON Web Key Set", "publicKeys", jc.publicKeys)

	var pubKeys []interface{}
	for _, file := range jc.publicKeys {
		pubKey, err := keyutil.PublicKeysFromFile(file)
		if err != nil {
			return errors.Wrap(err, "failed to read public key file")
		}
		pubKeys = append(pubKeys, pubKey...)
	}

	keySet, err := publicJWKSFromKeys(pubKeys)
	if err != nil {
		return errors.Wrap(err, "failed to construct JSONWebKeySet from a list of keys")
	}
	keysetJSON, err := json.MarshalIndent(keySet, "", "  ")
	if err != nil {
		return errors.Wrap(err, "failed to marshal JSONWebKeySet")
	}

	if jc.outputFile != "" {
		// write the keyset to the file
		if err = os.WriteFile(jc.outputFile, keysetJSON, 0600); err != nil {
			return errors.Wrap(err, "failed to write JWKS to file")
		}
		mlog.Debug("wrote JWKS", "file", jc.outputFile)
		return nil
	}

	mlog.Debug("writing JWKS to stdout")
	// write the keyset to stdout
	if _, err = os.Stdout.Write(keysetJSON); err != nil {
		return errors.Wrap(err, "failed to write JWKS to stdout")
	}
	return nil
}

// Most of the changes here have been vendored from pkg/serviceaccount/openidmetadata.go
//  * link: https://github.com/kubernetes/kubernetes/blob/ea0764452222146c47ec826977f49d7001b0ea8c/pkg/serviceaccount/openidmetadata.go

type publicKeyGetter interface {
	Public() crypto.PublicKey
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
			return nil, errors.New("the public key is not valid")
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
		return nil, errors.Errorf("JWK was not a public key! JWK: %v", jwk)
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
			return "", errors.New("unknown private key curve, must be 256, 384, or 521")
		}
	case jose.OpaqueSigner:
		return jose.SignatureAlgorithm(pk.Public().Algorithm), nil
	default:
		return "", errors.New("unknown public key type, must be *rsa.PublicKey, *ecdsa.PublicKey, or jose.OpaqueSigner")
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
		return "", errors.Wrap(err, "failed to serialize public key to DER format")
	}

	hasher := crypto.SHA256.New()
	hasher.Write(publicKeyDERBytes)
	publicKeyDERHash := hasher.Sum(nil)

	keyID := base64.RawURLEncoding.EncodeToString(publicKeyDERHash)

	return keyID, nil
}
