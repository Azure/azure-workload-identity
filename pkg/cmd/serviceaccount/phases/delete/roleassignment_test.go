package phases

import (
	"context"
	"net/http"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization"
	"github.com/Azure/go-autorest/autorest"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"

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
			name:     "missing --role-assignment-id",
			data:     &mockDeleteData{},
			errorMsg: "--role-assignment-id is required",
		},
		{
			name:     "valid data",
			data:     &mockDeleteData{roleAssignmentID: "test"},
			errorMsg: "",
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
	data := &mockDeleteData{
		roleAssignmentID: "test",
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAzureClient := mock_cloud.NewMockInterface(ctrl)
	mockAzureClient.EXPECT().DeleteRoleAssignment(gomock.Any(), data.roleAssignmentID).Return(armauthorization.RoleAssignment{}, nil)
	data.azureClient = mockAzureClient

	if err := phase.Run(context.Background(), data); err != nil {
		t.Errorf("expected no error but got: %s", err.Error())
	}

	// Test for scenario where it failed to delete role assignment
	mockAzureClient.EXPECT().DeleteRoleAssignment(gomock.Any(), data.roleAssignmentID).Return(armauthorization.RoleAssignment{}, errors.New("random error"))
	if err := phase.Run(context.Background(), data); err == nil {
		t.Errorf("expected error but got nil")
	}

	// Test for scenario where role assignment is not found
	mockAzureClient.EXPECT().DeleteRoleAssignment(gomock.Any(), data.roleAssignmentID).Return(armauthorization.RoleAssignment{}, autorest.DetailedError{StatusCode: http.StatusNoContent})
	if err := phase.Run(context.Background(), data); err != nil {
		t.Errorf("expected no error but got: %s", err.Error())
	}
}
