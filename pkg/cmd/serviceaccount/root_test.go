package serviceaccount

import (
	"errors"
	"testing"

	"github.com/Azure/azure-workload-identity/pkg/cloud"
	"github.com/Azure/azure-workload-identity/pkg/cloud/mock_cloud"

	"github.com/spf13/cobra"
)

//mockAuthProvider implements AuthProvider and allows in particular to stub out getClient()
type mockAuthProvider struct {
	getClientMock cloud.Interface
	*authArgs
}

func (provider *mockAuthProvider) getClient() (cloud.Interface, error) {
	provider.getClientMock = &mock_cloud.MockInterface{}
	return provider.getClientMock, nil
}

func (provider *mockAuthProvider) getAuthArgs() *authArgs {
	return provider.authArgs
}

func TestNewServiceAccountCmd(t *testing.T) {
	command := NewServiceAccountCmd()
	// The commands need to be listed in alphabetical order
	expectedCommands := []*cobra.Command{newCreateCmd(), newDeleteCmd()}
	cmds := command.Commands()

	for i, c := range expectedCommands {
		if cmds[i].Use != c.Use {
			t.Errorf("serviceaccount command should have command %s, but found %s", c.Use, cmds[i].Use)
		}
	}
}

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
				authMethod: "",
			},
			wantErr: errors.New("--auth-method is a required parameter"),
		},
		{
			name: "AlwaysExpectValidClientID",
			authArgs: authArgs{
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
				rawSubscriptionID:   validID,
				rawClientID:         validID,
				clientSecret:        "secret",
				authMethod:          "client_secret",
				rawAzureEnvironment: "AZUREPUBLICCLOUD",
			},
			wantErr: nil,
		},
		{
			name: "ClientCertificateAuthExpectsCertificatePath",
			authArgs: authArgs{
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
				rawSubscriptionID:   validID,
				rawClientID:         validID,
				certificatePath:     "/a/path",
				privateKeyPath:      "/a/path",
				authMethod:          "client_certificate",
				rawAzureEnvironment: "AZUREPUBLICCLOUD",
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.authArgs.validate()
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

func TestGetIssuerHash(t *testing.T) {
	tests := []struct {
		name        string
		inputIssuer string
		want        string
	}{
		{
			name:        "empty",
			inputIssuer: "",
			want:        "47DEQpj8HBSa-_TImW-5JCeuQeRkm5NMpJWZG3hSuFU=",
		},
		{
			name:        "valid issuer",
			inputIssuer: "https://test.blob.core.windows.net/oidc-test/",
			want:        "foWt5lYFJx_-XwBetmnSltvWY5J_nenUV-2c3Lqes3o=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getIssuerHash(tt.inputIssuer)
			if got != tt.want {
				t.Errorf("getIssuerHash() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestGetSubject(t *testing.T) {
	want := "system:serviceaccount:oidc:pod-identity-sa"
	got := getSubject("oidc", "pod-identity-sa")
	if got != want {
		t.Errorf("getSubject() = %s, want %s", got, want)
	}
}
