package phases

import (
	"context"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	"github.com/microsoftgraph/msgraph-beta-sdk-go/models/microsoft/graph"

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
			name:     "missing --aad-application-name",
			phase:    NewAADApplicationPhase(),
			data:     &mockCreateData{},
			errorMsg: "--aad-application-name is required",
		},
		{
			name:     "valid data 1",
			phase:    NewAADApplicationPhase(),
			data:     &mockCreateData{aadApplicationName: "test"},
			errorMsg: "",
		},
		{
			name:  "valid data 2",
			phase: NewAADApplicationPhase(),
			data:  &mockCreateData{serviceAccountNamespace: "test", serviceAccountName: "test", serviceAccountIssuerURL: "test"},
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
	data := &mockCreateData{
		serviceAccountNamespace: "service-account-namespace",
		serviceAccountName:      "service-account-name",
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAzureClient := mock_cloud.NewMockInterface(ctrl)
	mockAzureClient.EXPECT().CreateApplication(gomock.Any(), data.AADApplicationName()).Return(testApplication("client-id", "object-id", data.AADApplicationName()), nil)
	mockAzureClient.EXPECT().CreateServicePrincipal(gomock.Any(), "client-id", []string{
		"azwi version: , commit: ",
	}).Return(testServicePrincipal("client-id", "object-id", data.AADApplicationName()), nil)
	data.azureClient = mockAzureClient

	if err := phase.Run(context.Background(), data); err != nil {
		t.Errorf("expected no error but got: %s", err.Error())
	}
}

func testApplication(appID, objectID, displayName string) *graph.Application {
	app := graph.NewApplication()
	app.SetAppId(to.StringPtr(appID))
	app.SetId(to.StringPtr(objectID))
	app.SetDisplayName(to.StringPtr(displayName))
	return app
}

func testServicePrincipal(appID, objectID, displayName string) *graph.ServicePrincipal {
	sp := graph.NewServicePrincipal()
	sp.SetAppId(to.StringPtr(appID))
	sp.SetId(to.StringPtr(objectID))
	sp.SetDisplayName(to.StringPtr(displayName))
	return sp
}
