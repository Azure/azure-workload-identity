package federatedcredentials

import (
	"reflect"
	"testing"
)

func TestNewFederatedCredential(t *testing.T) {
	want := Federatedcredential{
		Name:        "federatedcredential-from-cli",
		Issuer:      "https://kubernetes.svc.local/",
		Audiences:   []string{"api://AzureADTokenExchange"},
		Subject:     "system:serviceaccount:oidc:pod-identity-sa",
		Description: "Federated credential created from CLI",
	}

	got := NewFederatedCredential("federatedcredential-from-cli", "https://kubernetes.svc.local/", "system:serviceaccount:oidc:pod-identity-sa", "Federated credential created from CLI", []string{"api://AzureADTokenExchange"})
	if !reflect.DeepEqual(got, want) {
		t.Errorf("NewFederatedCredential() = %v, want %v", got, want)
	}
}
