package phases

import (
	"context"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	"github.com/microsoftgraph/msgraph-beta-sdk-go/models/microsoft/graph"
	"github.com/pkg/errors"

	"github.com/Azure/azure-workload-identity/pkg/cloud"
	"github.com/Azure/azure-workload-identity/pkg/cloud/mock_cloud"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/phases/workflow"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/util"
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
			data:     &mockDeleteData{},
			errorMsg: "--service-account-namespace is required",
		},
		{
			name:     "missing --service-account-name",
			data:     &mockDeleteData{serviceAccountNamespace: "test"},
			errorMsg: "--service-account-name is required",
		},
		{
			name:     "missing --service-account-issuer-url",
			data:     &mockDeleteData{serviceAccountNamespace: "test", serviceAccountName: "test"},
			errorMsg: "--service-account-issuer-url is required",
		},
		{
			name:     "valid data",
			data:     &mockDeleteData{serviceAccountNamespace: "test", serviceAccountName: "test", serviceAccountIssuerURL: "test"},
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
	data := &mockDeleteData{
		serviceAccountNamespace: "service-account-namespace",
		serviceAccountName:      "service-account-name",
		serviceAccountIssuerURL: "service-account-issuer-url",
		aadApplicationObjectID:  "aad-application-object-id",
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fic := graph.NewFederatedIdentityCredential()
	fic.SetId(to.StringPtr("federated-identity-credential-id"))

	mockAzureClient := mock_cloud.NewMockInterface(ctrl)
	mockAzureClient.EXPECT().GetFederatedCredential(
		gomock.Any(),
		"aad-application-object-id",
		data.serviceAccountIssuerURL,
		util.GetFederatedCredentialSubject(data.serviceAccountNamespace, data.serviceAccountName),
	).Return(fic, nil)
	mockAzureClient.EXPECT().DeleteFederatedCredential(gomock.Any(), "aad-application-object-id", "federated-identity-credential-id").Return(nil)
	data.azureClient = mockAzureClient

	err := phase.Run(context.Background(), data)
	if err != nil {
		t.Errorf("expected no error but got: %s", err.Error())
	}

	// Test for scenario where it failed to delete federated credential
	mockAzureClient.EXPECT().GetFederatedCredential(
		gomock.Any(),
		"aad-application-object-id",
		data.serviceAccountIssuerURL,
		util.GetFederatedCredentialSubject(data.serviceAccountNamespace, data.serviceAccountName),
	).Return(fic, nil)
	mockAzureClient.EXPECT().DeleteFederatedCredential(gomock.Any(), "aad-application-object-id", "federated-identity-credential-id").Return(errors.New("random error"))
	err = phase.Run(context.Background(), data)
	if err == nil {
		t.Errorf("expected error but got nil")
	}

	// Test for scenario where federated credential is not found
	graphError := cloud.GraphError{PublicError: &graph.PublicError{}}
	graphError.PublicError.SetCode(to.StringPtr(cloud.GraphErrorCodeResourceNotFound))
	graphError.PublicError.SetMessage(to.StringPtr("FederatedIdentityCredential with name federatedcredential-from-azwi-cli not found."))
	mockAzureClient.EXPECT().GetFederatedCredential(
		gomock.Any(),
		"aad-application-object-id",
		data.serviceAccountIssuerURL,
		util.GetFederatedCredentialSubject(data.serviceAccountNamespace, data.serviceAccountName),
	).Return(nil, graphError)
	err = phase.Run(context.Background(), data)
	if err != nil {
		t.Errorf("expected no error but got: %s", err.Error())
	}
}
