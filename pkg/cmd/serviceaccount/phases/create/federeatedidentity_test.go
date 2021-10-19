package phases

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/Azure/azure-workload-identity/pkg/cloud"
	"github.com/Azure/azure-workload-identity/pkg/cloud/mock_cloud"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/phases/workflow"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/util"
	"github.com/Azure/azure-workload-identity/pkg/webhook"

	"github.com/Azure/go-autorest/autorest"
	"github.com/golang/mock/gomock"
)

func TestFederatedIdentityPreRun(t *testing.T) {
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
			name:     "missing --service-account-issuer-url",
			data:     &mockCreateData{serviceAccountNamespace: "test", serviceAccountName: "test"},
			errorMsg: "--service-account-issuer-url is required",
		},
		{
			name:     "missing --aad-application-name and --aad-application-object-id",
			data:     &mockCreateData{serviceAccountNamespace: "test", serviceAccountName: "test", serviceAccountIssuerURL: "test"},
			errorMsg: "--aad-application-name or --aad-application-object-id is required",
		},
		{
			name:     "valid data",
			data:     &mockCreateData{serviceAccountNamespace: "test", serviceAccountName: "test", serviceAccountIssuerURL: "test", aadApplicationName: "test"},
			errorMsg: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := NewFederatedIdentityPhase().PreRun(test.data)
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

func TestFederatedIdentityRun(t *testing.T) {
	phase := NewFederatedIdentityPhase()
	data := &mockCreateData{
		serviceAccountNamespace: "service-account-namespace",
		serviceAccountName:      "service-account-name",
		serviceAccountIssuerURL: "service-account-issuer-url",
		aadApplicationObjectID:  "aad-application-object-id",
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAzureClient := mock_cloud.NewMockInterface(ctrl)
	mockAzureClient.EXPECT().AddFederatedCredential(gomock.Any(), "aad-application-object-id", cloud.FederatedCredential{
		Name:        "federatedcredential-from-cli",
		Issuer:      data.serviceAccountIssuerURL,
		Subject:     util.GetFederatedCredentialSubject(data.serviceAccountNamespace, data.serviceAccountName),
		Description: fmt.Sprintf("Federated Service Account for %s/%s", data.serviceAccountNamespace, data.serviceAccountName),
		Audiences:   []string{webhook.DefaultAudience},
	}).Return(nil)
	data.azureClient = mockAzureClient

	err := phase.Run(context.Background(), data)
	if err != nil {
		t.Errorf("expected no error but got: %s", err.Error())
	}

	// Test for scenario where federated credential already exists
	mockAzureClient.EXPECT().AddFederatedCredential(gomock.Any(), "aad-application-object-id", gomock.Any()).Return(autorest.DetailedError{StatusCode: http.StatusConflict})
	err = phase.Run(context.Background(), data)
	if err != nil {
		t.Errorf("expected no error but got: %s", err.Error())
	}
}
