package jwks

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestJWKSCmdValidate(t *testing.T) {
	tests := []struct {
		name    string
		jwksCmd *jwksCmd
		verify  func(t *testing.T, err error)
	}{
		{
			name:    "no public keys",
			jwksCmd: &jwksCmd{},
			verify: func(t *testing.T, err error) {
				if err == nil {
					t.Error("expected error, got nil")
				}
			},
		},
		{
			name: "valid command",
			jwksCmd: &jwksCmd{
				publicKeys: []string{"testdata/public.key"},
			},
			verify: func(t *testing.T, err error) {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.verify(t, tt.jwksCmd.validate())
		})
	}
}

func TestJWKSCmdRun(t *testing.T) {
	testPublicKey := `
-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA1QJE2YmLbvMLP6FtzcfP
zGbSDbHEEtA0mH6kwgrOrlKs83zj2vr6Y5k/ZcGdIbsdm5vDj2IxtSkE+pSDtgFM
2iq0sJ7xuE6RYmlrtBm+H2WHvXrP9RrG1EfO7iWs6Czj4A/Ddxg3kNUiQCtQEJww
H2pfrUkh8STQhST/T86pq5AIFCuQiQSrkfC80eD9bUFypV3CLB2M9Fa1hbvOWbzS
F93/I0toUK2+oPgVW6m2EwMyy8Fh/3KRixrAJO8g+D4d537C1fa1vJJRlMRFtLMA
/bo6k1fAtNsVQuQoML5CmRrvNT7ZpXRLaQy64OSFrVLD3Pb7wct7b4g2xQECixQo
dwIDAQAB
-----END PUBLIC KEY-----`

	expectedJWKS := `
{
  "keys": [
		{
		"use": "sig",
		"kty": "RSA",
		"kid": "2A3FPpix2keOV1SGPQiM0_wVemz4XOIgQyJJnpu5sPE",
		"alg": "RS256",
		"n": "1QJE2YmLbvMLP6FtzcfPzGbSDbHEEtA0mH6kwgrOrlKs83zj2vr6Y5k_ZcGdIbsdm5vDj2IxtSkE-pSDtgFM2iq0sJ7xuE6RYmlrtBm-H2WHvXrP9RrG1EfO7iWs6Czj4A_Ddxg3kNUiQCtQEJwwH2pfrUkh8STQhST_T86pq5AIFCuQiQSrkfC80eD9bUFypV3CLB2M9Fa1hbvOWbzSF93_I0toUK2-oPgVW6m2EwMyy8Fh_3KRixrAJO8g-D4d537C1fa1vJJRlMRFtLMA_bo6k1fAtNsVQuQoML5CmRrvNT7ZpXRLaQy64OSFrVLD3Pb7wct7b4g2xQECixQodw",
		"e": "AQAB"
		}
	]
}`

	tmpDir, err := os.MkdirTemp("", "jwks")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	publicKeyFile := filepath.Join(tmpDir, "public.key")
	expectedOutputFile := filepath.Join(tmpDir, "jwks.json")

	if err = os.WriteFile(publicKeyFile, []byte(testPublicKey), 0600); err != nil {
		t.Errorf("Error writing file: %v", err)
	}

	jwksCmd := &jwksCmd{
		publicKeys: []string{publicKeyFile},
		outputFile: expectedOutputFile,
	}
	if err = jwksCmd.run(); err != nil {
		t.Errorf("Error running jwksCmd: %v", err)
	}

	if _, err := os.Stat(expectedOutputFile); os.IsNotExist(err) {
		t.Errorf("Error creating jwks.json file: %v", err)
	}
	body, err := os.ReadFile(expectedOutputFile)
	if err != nil {
		t.Errorf("Error reading jwks.json file: %v", err)
	}

	var o1, o2 interface{}
	if err := json.Unmarshal(body, &o1); err != nil {
		t.Errorf("Error unmarshalling jwks.json: %v", err)
	}
	if err := json.Unmarshal([]byte(expectedJWKS), &o2); err != nil {
		t.Errorf("Error unmarshalling expected jwks.json: %v", err)
	}
	if !reflect.DeepEqual(o1, o2) {
		t.Errorf("expected jwks: %v, got: %v", o2, o1)
	}

	// Test jwks is written to stdout
	jwksCmd.outputFile = ""

	old := os.Stdout // keep backup of the real stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = jwksCmd.run()

	outC := make(chan string)
	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		outC <- strings.TrimSpace(buf.String())
	}()

	// back to normal state
	w.Close()
	os.Stdout = old // restoring the real stdout
	out := <-outC

	if err != nil {
		t.Errorf("Error running jwksCmd: %v", err)
	}
	if err := json.Unmarshal([]byte(out), &o1); err != nil {
		t.Errorf("Error unmarshalling jwks.json: %v", err)
	}
	if !reflect.DeepEqual(o1, o2) {
		t.Errorf("expected jwks: %v, got: %v", o2, o1)
	}
}
