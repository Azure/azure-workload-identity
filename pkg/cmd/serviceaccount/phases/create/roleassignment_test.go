package phases

import (
	"context"
	"net/http"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-01-01-preview/authorization"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"

	"github.com/Azure/azure-workload-identity/pkg/cloud/mock_cloud"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/phases/workflow"
)

func TestRoleAssignmentPreRun(t *testing.T) {
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
			name:     "missing --azure-scope",
			data:     &mockCreateData{},
			errorMsg: "--azure-scope is required",
		},
		{
			name:     "missing --azure-role",
			data:     &mockCreateData{azureScope: "test"},
			errorMsg: "--azure-role is required",
		},
		{
			name:     "missing --service-principal-name or --service-principal-object-id",
			data:     &mockCreateData{azureScope: "test", azureRole: "test"},
			errorMsg: "--service-principal-name or --service-principal-object-id is required",
		},
		{
			name:     "valid data 1",
			data:     &mockCreateData{azureScope: "test", azureRole: "test", servicePrincipalName: "test"},
			errorMsg: "",
		},
		{
			name:  "valid data 2",
			phase: NewAADApplicationPhase(),
			data:  &mockCreateData{azureScope: "test", azureRole: "test", serviceAccountNamespace: "test", serviceAccountName: "test", serviceAccountIssuerURL: "test"},
		},
		{
			name:  "valid data 3",
			phase: NewAADApplicationPhase(),
			data:  &mockCreateData{azureScope: "test", azureRole: "test", servicePrincipalObjectID: "test"},
		},
		{
			name:  "valid data 4",
			phase: NewAADApplicationPhase(),
			data:  &mockCreateData{azureScope: "test", azureRole: "test", aadApplicationName: "test"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := NewRoleAssignmentPhase().PreRun(test.data)
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

func TestRoleAssignmentRun(t *testing.T) {
	phase := NewRoleAssignmentPhase()
	data := &mockCreateData{
		azureRole:                "azure-role",
		azureScope:               "azure-scope",
		servicePrincipalObjectID: "service-principal-object-id",
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAzureClient := mock_cloud.NewMockInterface(ctrl)
	mockAzureClient.EXPECT().CreateRoleAssignment(context.Background(), data.azureScope, data.azureRole, data.servicePrincipalObjectID).Return(authorization.RoleAssignment{
		ID: to.StringPtr("id"),
	}, nil)
	data.azureClient = mockAzureClient

	if err := phase.Run(context.Background(), data); err != nil {
		t.Errorf("expected no error but got: %s", err.Error())
	}

	// Test for scenario where role assignment already exists
	mockAzureClient.EXPECT().CreateRoleAssignment(context.Background(), data.azureScope, data.azureRole, data.servicePrincipalObjectID).Return(authorization.RoleAssignment{
		ID: to.StringPtr("id"),
	}, autorest.DetailedError{StatusCode: http.StatusConflict})
	if err := phase.Run(context.Background(), data); err != nil {
		t.Errorf("expected no error but got: %s", err.Error())
	}
}
