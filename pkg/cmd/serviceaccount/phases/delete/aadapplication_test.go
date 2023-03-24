package phases

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"

	"github.com/Azure/azure-workload-identity/pkg/cloud/mock_cloud"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/phases/workflow"
)

func TestAADApplicationPreRun(t *testing.T) {
	tests := []struct {
		name     string
		phase    workflow.Phase
		data     interface{}
		errorMsg string
	}{
		{
			name:     "invalid data type",
			phase:    NewAADApplicationPhase(),
			data:     "test",
			errorMsg: "invalid data type string",
		},
		{
			name:     "valid data 1",
			phase:    NewAADApplicationPhase(),
			data:     &mockDeleteData{aadApplicationName: "test"},
			errorMsg: "",
		},
		{
			name:     "valid data 2",
			phase:    NewAADApplicationPhase(),
			data:     &mockDeleteData{aadApplicationObjectID: "test"},
			errorMsg: "",
		},
		{
			name:     "valid data 3",
			phase:    NewAADApplicationPhase(),
			data:     &mockDeleteData{serviceAccountNamespace: "test", serviceAccountName: "test", serviceAccountIssuerURL: "test"},
			errorMsg: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.phase.PreRun(test.data)
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

func TestAADApplicationRun(t *testing.T) {
	phase := NewAADApplicationPhase()
	data := &mockDeleteData{
		aadApplicationObjectID: "aad-application-object-id",
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAzureClient := mock_cloud.NewMockInterface(ctrl)
	mockAzureClient.EXPECT().DeleteApplication(gomock.Any(), data.aadApplicationObjectID).Return(nil)
	data.azureClient = mockAzureClient

	if err := phase.Run(context.Background(), data); err != nil {
		t.Errorf("expected no error but got: %s", err.Error())
	}

	// Test for scenario where it failed to delete aad application
	mockAzureClient.EXPECT().DeleteApplication(gomock.Any(), data.aadApplicationObjectID).Return(errors.New("random error"))
	if err := phase.Run(context.Background(), data); err == nil {
		t.Errorf("expected error but got nil")
	}
}
