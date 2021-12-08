package serviceaccount

import (
	"testing"

	"github.com/Azure/azure-workload-identity/pkg/cloud/mock_cloud"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
)

func TestDeleteDataServiceAccountName(t *testing.T) {
	deleteData := &deleteData{
		serviceAccountName: serviceAccountName,
	}
	if deleteData.ServiceAccountName() != serviceAccountName {
		t.Errorf("Expected ServiceAccountName() to be 'service-account-name', got %s", deleteData.ServiceAccountName())
	}
}

func TestDeleteDataServiceAccountNamespace(t *testing.T) {
	deleteData := &deleteData{
		serviceAccountNamespace: serviceAccountNamespace,
	}
	if deleteData.ServiceAccountNamespace() != serviceAccountNamespace {
		t.Errorf("Expected ServiceAccountNamespace() to be 'service-account-namespace', got %s", deleteData.ServiceAccountNamespace())
	}
}

func TestDeleteDataServiceAccountIssuerURL(t *testing.T) {
	deleteData := &deleteData{
		serviceAccountIssuerURL: serviceAccountIssuerURL,
	}
	if deleteData.ServiceAccountIssuerURL() != serviceAccountIssuerURL {
		t.Errorf("Expected ServiceAccountIssuerURL() to be 'service-account-issuer-url', got %s", deleteData.ServiceAccountIssuerURL())
	}
}

func TestDeleteDataAADApplication(t *testing.T) {
	tests := []struct {
		name       string
		deleteData *deleteData
		expect     func(m *mock_cloud.MockInterfaceMockRecorder)
		verify     func(t *testing.T, deleteData *deleteData)
	}{
		{
			name: "random error",
			deleteData: &deleteData{
				aadApplicationName: appName,
			},
			expect: func(m *mock_cloud.MockInterfaceMockRecorder) {
				m.GetApplication(gomock.Any(), appName).Return(nil, errors.New("random error")).Times(2)
			},
			verify: func(t *testing.T, deleteData *deleteData) {
				if _, err := deleteData.AADApplication(); err == nil {
					t.Error("Expected AADApplication() to return error")
				}
				if deleteData.AADApplicationObjectID() != "" {
					t.Errorf("Expected AADApplicationObjectID() to be empty, got %s", deleteData.AADApplicationObjectID())
				}
			},
		},
		{
			name: "no cache",
			deleteData: &deleteData{
				aadApplicationName: appName,
			},
			expect: func(m *mock_cloud.MockInterfaceMockRecorder) {
				m.GetApplication(gomock.Any(), appName).Return(testApplication(appID, objectID), nil)
			},
			verify: func(t *testing.T, deleteData *deleteData) {
				if _, err := deleteData.AADApplication(); err != nil {
					t.Error("Expected AADApplication() to not return error")
				}
				if deleteData.AADApplicationObjectID() != objectID {
					t.Errorf("Expected AADApplicationObjectID() to be 'object-id', got %s", deleteData.AADApplicationObjectID())
				}
			},
		},
		{
			name: "cache",
			deleteData: &deleteData{
				aadApplicationName: appName,
				aadApplication:     testApplication(appID, objectID),
			},
			expect: func(m *mock_cloud.MockInterfaceMockRecorder) {},
			verify: func(t *testing.T, deleteData *deleteData) {
				if _, err := deleteData.AADApplication(); err != nil {
					t.Error("Expected AADApplication() to not return error")
				}
				if deleteData.AADApplicationObjectID() != objectID {
					t.Errorf("Expected AADApplicationObjectID() to be 'object-id', got %s", deleteData.AADApplicationObjectID())
				}
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			authProvider := &mockAuthProvider{
				azureClient: mock_cloud.NewMockInterface(ctrl),
			}
			test.expect(authProvider.azureClient.EXPECT())
			test.deleteData.authProvider = authProvider
			test.verify(t, test.deleteData)
		})
	}
}

func TestDeleteDataAADApplicationName(t *testing.T) {
	deleteData := &deleteData{
		aadApplicationName: appName,
	}
	if deleteData.AADApplicationName() != appName {
		t.Errorf("Expected AADApplicationName() to be 'aad-application-name', got %s", deleteData.AADApplicationName())
	}
	deleteData.aadApplicationName = ""
	deleteData.serviceAccountNamespace = serviceAccountNamespace
	deleteData.serviceAccountName = serviceAccountName
	deleteData.serviceAccountIssuerURL = serviceAccountIssuerURL
	if deleteData.AADApplicationName() != "service-account-namespace-service-account-name-t4BxHnnPeJsOfTLIBFbdKeRHdVMaIRdxwkxwF13SvKw=" {
		t.Errorf("Expected AADApplicationName() to be 'service-account-namespace-service-account-name-t4BxHnnPeJsOfTLIBFbdKeRHdVMaIRdxwkxwF13SvKw=', got %s", deleteData.AADApplicationName())
	}
}

func TestDeleteDataAADApplicationObjectID(t *testing.T) {
	deleteData := &deleteData{
		aadApplicationObjectID: objectID,
	}
	if deleteData.AADApplicationObjectID() != objectID {
		t.Errorf("Expected AADApplicationObjectID() to be 'object-id', got %s", deleteData.AADApplicationObjectID())
	}
}

func TestDeleteDataRoleAssignmentID(t *testing.T) {
	deleteData := &deleteData{
		roleAssignmentID: "role-assignment-id",
	}
	if deleteData.RoleAssignmentID() != "role-assignment-id" {
		t.Errorf("Expected RoleAssignmentID() to be 'role-assignment-id', got %s", deleteData.AADApplicationObjectID())
	}
}
