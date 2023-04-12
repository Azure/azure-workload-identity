package auth

import (
	"net/http"
	"testing"

	"github.com/pkg/errors"
)

func TestValidateAuthArgs(t *testing.T) {
	validID := "cc6b141e-6afc-4786-9bf6-e3b9a5601460"
	invalidID := "invalidID"

	tests := []struct {
		name     string
		authArgs authArgs
		wantErr  error
	}{
		{
			name: "AuthMethodIsRequired",
			authArgs: authArgs{
				client:     http.DefaultClient,
				authMethod: "",
			},
			wantErr: errors.New("--auth-method is a required parameter"),
		},
		{
			name: "AlwaysExpectValidClientID",
			authArgs: authArgs{
				client:              http.DefaultClient,
				rawSubscriptionID:   validID,
				rawClientID:         invalidID,
				clientSecret:        "secret",
				authMethod:          "client_secret",
				rawAzureEnvironment: "AZUREPUBLICCLOUD",
			},
			wantErr: errors.New(`parsing --client-id: invalid UUID length: 9`),
		},
		{
			name: "AlwaysExpectValidClientID",
			authArgs: authArgs{
				client:              http.DefaultClient,
				rawSubscriptionID:   validID,
				rawClientID:         invalidID,
				clientSecret:        "secret",
				authMethod:          "client_certificate",
				rawAzureEnvironment: "AZUREPUBLICCLOUD",
			},
			wantErr: errors.New(`parsing --client-id: invalid UUID length: 9`),
		},
		{
			name: "ClientSecretAuthExpectsClientSecret",
			authArgs: authArgs{
				client:              http.DefaultClient,
				rawSubscriptionID:   validID,
				rawClientID:         validID,
				clientSecret:        "",
				authMethod:          "client_secret",
				rawAzureEnvironment: "AZUREPUBLICCLOUD",
			},
			wantErr: errors.New(`--client-secret must be specified when --auth-method="client_secret"`),
		},
		{
			name: "ValidClientSecretAuth",
			authArgs: authArgs{
				client:              http.DefaultClient,
				rawSubscriptionID:   validID,
				rawClientID:         validID,
				clientSecret:        "secret",
				authMethod:          "client_secret",
				rawAzureEnvironment: "AZUREPUBLICCLOUD",
			},
			wantErr: errors.New("Unexpected response from Get Subscription: 404"),
		},
		{
			name: "ClientCertificateAuthExpectsCertificatePath",
			authArgs: authArgs{
				client:              http.DefaultClient,
				rawSubscriptionID:   validID,
				rawClientID:         validID,
				certificatePath:     "",
				privateKeyPath:      "/a/path",
				authMethod:          "client_certificate",
				rawAzureEnvironment: "AZUREPUBLICCLOUD",
			},
			wantErr: errors.New(`--certificate-path and --private-key-path must be specified when --auth-method="client_certificate"`),
		},
		{
			name: "ClientCertificateAuthExpectsPrivateKeyPath",
			authArgs: authArgs{
				client:              http.DefaultClient,
				rawSubscriptionID:   validID,
				rawClientID:         validID,
				certificatePath:     "/a/path",
				privateKeyPath:      "",
				authMethod:          "client_certificate",
				rawAzureEnvironment: "AZUREPUBLICCLOUD",
			},
			wantErr: errors.New(`--certificate-path and --private-key-path must be specified when --auth-method="client_certificate"`),
		},
		{
			name: "ValidClientCertificateAuth",
			authArgs: authArgs{
				client:              http.DefaultClient,
				rawSubscriptionID:   validID,
				rawClientID:         validID,
				certificatePath:     "/a/path",
				privateKeyPath:      "/a/path",
				authMethod:          "client_certificate",
				rawAzureEnvironment: "AZUREPUBLICCLOUD",
			},
			wantErr: errors.New("Unexpected response from Get Subscription: 404"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.authArgs.Validate()
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("validate() = %v, want %v", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("validate() = %v, want %v", err, tt.wantErr)
				}
			}
		})
	}
}
