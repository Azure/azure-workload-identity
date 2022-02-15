package util

import "testing"

func TestGetIssuerHash(t *testing.T) {
	tests := []struct {
		name      string
		issuerURL string
		want      string
	}{
		{
			name:      "empty",
			issuerURL: "",
			want:      "47DEQpj8HBSa-_TImW-5JCeuQeRkm5NMpJWZG3hSuFU=",
		},
		{
			name:      "valid issuer",
			issuerURL: "https://test.blob.core.windows.net/oidc-test/",
			want:      "foWt5lYFJx_-XwBetmnSltvWY5J_nenUV-2c3Lqes3o=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetIssuerHash(tt.issuerURL)
			if got != tt.want {
				t.Errorf("GetIssuerHash() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestGetFederatedCredentialName(t *testing.T) {
	tests := []struct {
		name                    string
		serviceAccountNamespace string
		serviceAccountName      string
		issuerURL               string
		want                    string
	}{
		{
			name:                    "empty",
			serviceAccountNamespace: "",
			serviceAccountName:      "",
			issuerURL:               "",
			want:                    "2BVrrgxCQ9N0L8Tpd02KzqvgQQJJ1yDIVfmK_Ij_hGw=",
		},
		{
			name:                    "valid",
			serviceAccountNamespace: "oidc",
			serviceAccountName:      "pod-identity-sa",
			issuerURL:               "https://test.blob.core.windows.net/oidc-test/",
			want:                    "5Frx_q5PpeP09cXWfkbDVwCOg5IVRmmKE3BUKT4hP4I=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetFederatedCredentialName(tt.serviceAccountNamespace, tt.serviceAccountName, tt.issuerURL)
			if got != tt.want {
				t.Errorf("GetFederatedCredentialName() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestGetFederatedCredentialSubject(t *testing.T) {
	want := "system:serviceaccount:oidc:pod-identity-sa"
	got := GetFederatedCredentialSubject("oidc", "pod-identity-sa")
	if got != want {
		t.Errorf("GetFederatedCredentialSubject() = %s, want %s", got, want)
	}
}
