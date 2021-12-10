package phases

import (
	"context"
	"fmt"
	"testing"

	"github.com/Azure/azure-workload-identity/pkg/cloud"
	"github.com/Azure/azure-workload-identity/pkg/cloud/mock_cloud"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/phases/workflow"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/util"
	"github.com/Azure/azure-workload-identity/pkg/webhook"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	"github.com/microsoftgraph/msgraph-beta-sdk-go/models/microsoft/graph"
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
			name:     "valid data",
			data:     &mockCreateData{serviceAccountNamespace: "test", serviceAccountName: "test", serviceAccountIssuerURL: "test"},
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

	fic := graph.NewFederatedIdentityCredential()
	fic.SetAudiences([]string{webhook.DefaultAudience})
	fic.SetDescription(to.StringPtr(fmt.Sprintf("Federated Service Account for %s/%s", data.serviceAccountNamespace, data.serviceAccountName)))
	fic.SetIssuer(to.StringPtr(data.serviceAccountIssuerURL))
	fic.SetSubject(to.StringPtr(util.GetFederatedCredentialSubject(data.serviceAccountNamespace, data.serviceAccountName)))
	fic.SetName(to.StringPtr("federatedcredential-from-azwi-cli"))

	mockAzureClient := mock_cloud.NewMockInterface(ctrl)
	mockAzureClient.EXPECT().AddFederatedCredential(gomock.Any(), "aad-application-object-id", fic).Return(nil)
	data.azureClient = mockAzureClient

	err := phase.Run(context.Background(), data)
	if err != nil {
		t.Errorf("expected no error but got: %s", err.Error())
	}

	// Test for scenario where federated credential already exists
	graphError := cloud.GraphError{PublicError: &graph.PublicError{}}
	graphError.PublicError.SetCode(to.StringPtr(cloud.GraphErrorCodeMultipleObjectsWithSameKeyValue))
	graphError.PublicError.SetMessage(to.StringPtr("FederatedIdentityCredential with name federatedcredential-from-azwi-cli already exists."))
	mockAzureClient.EXPECT().AddFederatedCredential(gomock.Any(), "aad-application-object-id", gomock.Any()).Return(graphError)
	err = phase.Run(context.Background(), data)
	if err != nil {
		t.Errorf("expected no error but got: %s", err.Error())
	}
}
