package phases

import (
	"context"
	"testing"
	"time"

	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/phases/workflow"
	"github.com/Azure/azure-workload-identity/pkg/webhook"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestServiceAccountPreRun(t *testing.T) {
	tests := []struct {
		name     string
		phase    workflow.Phase
		data     interface{}
		errorMsg string
	}{
		{
			name:     "invalid data type",
			data:     "test",
			errorMsg: "invalid data type string",
		},
		{
			name:     "missing --service-account-namespace",
			data:     &mockCreateData{},
			errorMsg: "--service-account-namespace is required",
		},
		{
			name:     "missing --service-account-name",
			data:     &mockCreateData{serviceAccountNamespace: "test"},
			errorMsg: "--service-account-name is required",
		},
		{
			name: "token expiration >= minimum token expiration",
			data: &mockCreateData{
				serviceAccountNamespace:       "test",
				serviceAccountName:            "test",
				serviceAccountTokenExpiration: 1 * time.Hour,
				kubeClient:                    fake.NewClientBuilder().Build(),
			},
			errorMsg: "",
		},
		{
			name: "token expiration < minimum token expiration",
			data: &mockCreateData{
				serviceAccountNamespace:       "test",
				serviceAccountName:            "test",
				serviceAccountTokenExpiration: 1 * time.Minute,
			},
			errorMsg: "--service-account-token-expiration must be greater than or equal to 1h0m0s",
		},
		{
			name: "token expiration > maximum token expiration",
			data: &mockCreateData{
				serviceAccountNamespace:       "test",
				serviceAccountName:            "test",
				serviceAccountTokenExpiration: 25 * time.Hour,
			},
			errorMsg: "--service-account-token-expiration must be less than or equal to 24h0m0s",
		},
		{
			name: "valid data",
			data: &mockCreateData{
				serviceAccountNamespace:       "test",
				serviceAccountName:            "test",
				serviceAccountTokenExpiration: 1 * time.Hour,
				kubeClient:                    fake.NewClientBuilder().Build(),
			},
			errorMsg: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := NewServiceAccountPhase().PreRun(test.data)
			if err == nil {
				if test.errorMsg != "" {
					t.Errorf("expected error but got nil")
				}
			} else if err.Error() != test.errorMsg {
				t.Errorf("expected error message: %s, but got: %s", test.errorMsg, err.Error())
			}
		})
	}
}

func TestServiceAccountRun(t *testing.T) {
	phase := NewServiceAccountPhase()
	kubeClient := fake.NewClientBuilder().Build()
	data := &mockCreateData{
		serviceAccountNamespace:       "service-account-namespace",
		serviceAccountName:            "service-account-name",
		serviceAccountTokenExpiration: 2 * time.Hour,
		aadApplicationClientID:        "aad-application-client-id",
		azureTenantID:                 "azure-tenant-id",
		kubeClient:                    kubeClient,
	}

	err := phase.PreRun(data)
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}
	err = phase.Run(context.Background(), data)
	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	sa := &corev1.ServiceAccount{}
	if err = kubeClient.Get(context.TODO(), types.NamespacedName{Name: "service-account-name", Namespace: "service-account-namespace"}, sa); err != nil {
		t.Errorf("expected service account to be created")
	}
	if sa.Labels[webhook.UseWorkloadIdentityLabel] != "true" {
		t.Errorf("expected service account to have workload identity label but got: %s", sa.Labels[webhook.UseWorkloadIdentityLabel])
	}
	if sa.Annotations[webhook.ClientIDAnnotation] != "aad-application-client-id" {
		t.Errorf("expected service account to have client id annotation but got: %s", sa.Annotations[webhook.ClientIDAnnotation])
	}
	if sa.Annotations[webhook.TenantIDAnnotation] != "azure-tenant-id" {
		t.Errorf("expected service account to have tenant id annotation but got: %s", sa.Annotations[webhook.TenantIDAnnotation])
	}
	if sa.Annotations[webhook.ServiceAccountTokenExpiryAnnotation] != "7200" {
		t.Errorf("expected service account to have token expiration label but got: %s", sa.Labels[webhook.ServiceAccountTokenExpiryAnnotation])
	}
}
