package serviceaccount

import "testing"

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
