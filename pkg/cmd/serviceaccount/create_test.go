package serviceaccount

import (
	"errors"
	"testing"
	"time"

	"github.com/Azure/azure-workload-identity/pkg/cloud"
	"github.com/Azure/azure-workload-identity/pkg/cloud/mock_cloud"
	"github.com/microsoftgraph/msgraph-sdk-go/models/microsoft/graph"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	"github.com/spf13/pflag"
)

const (
	serviceAccountNamespace = "service-account-namespace"
	serviceAccountName      = "service-account-name"
	serviceAccountIssuerURL = "service-account-issuer-url"
	appID                   = "app-id"
	objectID                = "object-id"
	appName                 = "aad-application-name"
)

type mockAuthProvider struct {
	azureClient   *mock_cloud.MockInterface
	azureTenantID string
}

func (m *mockAuthProvider) AddFlags(f *pflag.FlagSet)       {}
func (m *mockAuthProvider) GetAzureClient() cloud.Interface { return m.azureClient }
func (m *mockAuthProvider) GetAzureTenantID() string        { return m.azureTenantID }
func (m *mockAuthProvider) Validate() error                 { return nil }

func TestCreateDataServiceAccountName(t *testing.T) {
	createData := &createData{
		serviceAccountName: serviceAccountName,
	}
	if createData.ServiceAccountName() != serviceAccountName {
		t.Errorf("Expected ServiceAccountName() to be 'service-account-name', got %s", createData.ServiceAccountName())
	}
}

func TestCreateDataServiceAccountNamespace(t *testing.T) {
	createData := &createData{
		serviceAccountNamespace: serviceAccountNamespace,
	}
	if createData.ServiceAccountNamespace() != serviceAccountNamespace {
		t.Errorf("Expected ServiceAccountNamespace() to be 'service-account-namespace', got %s", createData.ServiceAccountNamespace())
	}
}

func TestCreateDataServiceAccountIssuerURL(t *testing.T) {
	createData := &createData{
		serviceAccountIssuerURL: serviceAccountIssuerURL,
	}
	if createData.ServiceAccountIssuerURL() != serviceAccountIssuerURL {
		t.Errorf("Expected ServiceAccountIssuerURL() to be 'service-account-issuer-url', got %s", createData.ServiceAccountIssuerURL())
	}
}

func TestCreateDataServiceAccountTokenExpiration(t *testing.T) {
	createData := &createData{
		serviceAccountTokenExpiration: 1 * time.Hour,
	}
	if createData.ServiceAccountTokenExpiration() != 1*time.Hour {
		t.Errorf("Expected ServiceAccountTokenExpiration() to be '1h', got %s", createData.ServiceAccountTokenExpiration())
	}
}

func TestCreateDataAADApplication(t *testing.T) {
	tests := []struct {
		name       string
		createData *createData
		expect     func(m *mock_cloud.MockInterfaceMockRecorder)
		verify     func(t *testing.T, createData *createData)
	}{
		{
			name: "random error",
			createData: &createData{
				aadApplicationName: appName,
			},
			expect: func(m *mock_cloud.MockInterfaceMockRecorder) {
				m.GetApplication(gomock.Any(), appName).Return(nil, errors.New("random error")).Times(3)
			},
			verify: func(t *testing.T, createData *createData) {
				if _, err := createData.AADApplication(); err == nil {
					t.Error("Expected AADApplication() to return error")
				}
				if createData.AADApplicationClientID() != "" {
					t.Errorf("Expected AADApplicationClientID() to be empty, got %s", createData.AADApplicationClientID())
				}
				if createData.AADApplicationObjectID() != "" {
					t.Errorf("Expected AADApplicationObjectID() to be empty, got %s", createData.AADApplicationObjectID())
				}
			},
		},
		{
			name: "no cache",
			createData: &createData{
				aadApplicationName: appName,
			},
			expect: func(m *mock_cloud.MockInterfaceMockRecorder) {
				m.GetApplication(gomock.Any(), appName).Return(&graphrbac.Application{
					AppID:    to.StringPtr(appID),
					ObjectID: to.StringPtr(objectID),
				}, nil)
			},
			verify: func(t *testing.T, createData *createData) {
				if _, err := createData.AADApplication(); err != nil {
					t.Error("Expected AADApplication() to not return error")
				}
				if createData.AADApplicationClientID() != appID {
					t.Errorf("Expected AADApplicationClientID() to be 'client-id', got %s", createData.AADApplicationClientID())
				}
				if createData.AADApplicationObjectID() != objectID {
					t.Errorf("Expected AADApplicationObjectID() to be 'object-id', got %s", createData.AADApplicationObjectID())
				}
			},
		},
		{
			name: "cache",
			createData: &createData{
				aadApplicationName: appName,
				aadApplication:     testApplication(appID, objectID),
			},
			expect: func(m *mock_cloud.MockInterfaceMockRecorder) {},
			verify: func(t *testing.T, createData *createData) {
				if _, err := createData.AADApplication(); err != nil {
					t.Error("Expected AADApplication() to not return error")
				}
				if createData.AADApplicationClientID() != appID {
					t.Errorf("Expected AADApplicationClientID() to be 'client-id', got %s", createData.AADApplicationClientID())
				}
				if createData.AADApplicationObjectID() != objectID {
					t.Errorf("Expected AADApplicationObjectID() to be 'object-id', got %s", createData.AADApplicationObjectID())
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
			test.createData.authProvider = authProvider
			test.verify(t, test.createData)
		})
	}
}

func TestCreateDataAADApplicationName(t *testing.T) {
	createData := &createData{
		aadApplicationName: appName,
	}
	if createData.AADApplicationName() != appName {
		t.Errorf("Expected aadApplicationName() to be 'aad-application-name', got %s", createData.AADApplicationName())
	}
	createData.aadApplicationName = ""
	createData.serviceAccountNamespace = serviceAccountNamespace
	createData.serviceAccountName = serviceAccountName
	createData.serviceAccountIssuerURL = serviceAccountIssuerURL
	if createData.AADApplicationName() != "service-account-namespace-service-account-name-t4BxHnnPeJsOfTLIBFbdKeRHdVMaIRdxwkxwF13SvKw=" {
		t.Errorf("Expected aadApplicationName() to be 'service-account-namespace-service-account-name-t4BxHnnPeJsOfTLIBFbdKeRHdVMaIRdxwkxwF13SvKw=', got %s", createData.AADApplicationName())
	}
}

func TestCreateDataAADApplicationClientID(t *testing.T) {
	createData := &createData{
		aadApplicationClientID: appID,
	}
	if createData.AADApplicationClientID() != appID {
		t.Errorf("Expected aadApplicationClientID() to be 'client-id', got %s", createData.AADApplicationClientID())
	}
}

func TestCreateDataAADApplicationObjectID(t *testing.T) {
	createData := &createData{
		aadApplicationObjectID: objectID,
	}
	if createData.AADApplicationObjectID() != objectID {
		t.Errorf("Expected aadApplicationObjectID() to be 'object-id', got %s", createData.AADApplicationObjectID())
	}
}

func TestCreateDataServicePrincipal(t *testing.T) {
	tests := []struct {
		name       string
		createData *createData
		expect     func(m *mock_cloud.MockInterfaceMockRecorder)
		verify     func(t *testing.T, createData *createData)
	}{
		{
			name: "random error",
			createData: &createData{
				servicePrincipalName: "service-principal-name",
			},
			expect: func(m *mock_cloud.MockInterfaceMockRecorder) {
				m.GetServicePrincipal(gomock.Any(), "service-principal-name").Return(nil, errors.New("random error")).Times(2)
			},
			verify: func(t *testing.T, createData *createData) {
				if _, err := createData.ServicePrincipal(); err == nil {
					t.Error("Expected ServicePrincipal() to return error")
				}
				if createData.ServicePrincipalObjectID() != "" {
					t.Errorf("Expected ServicePrincipalObjectID() to be '', got %s", createData.ServicePrincipalObjectID())
				}
			},
		},
		{
			name: "cache",
			createData: &createData{
				servicePrincipalName: "service-principal-name",
				servicePrincipal:     testServicePrincipal("", objectID),
			},
			expect: func(m *mock_cloud.MockInterfaceMockRecorder) {},
			verify: func(t *testing.T, createData *createData) {
				if _, err := createData.ServicePrincipal(); err != nil {
					t.Error("Expected ServicePrincipal() to not return error")
				}
				if createData.ServicePrincipalObjectID() != objectID {
					t.Errorf("Expected ServicePrincipalObjectID() to be 'object-id', got %s", createData.ServicePrincipalObjectID())
				}
			},
		},
		{
			name: "no cache",
			createData: &createData{
				servicePrincipalName: "service-principal-name",
			},
			expect: func(m *mock_cloud.MockInterfaceMockRecorder) {
				m.GetServicePrincipal(gomock.Any(), "service-principal-name").Return(&graphrbac.ServicePrincipal{
					ObjectID: to.StringPtr(objectID),
				}, nil)
			},
			verify: func(t *testing.T, createData *createData) {
				if _, err := createData.ServicePrincipal(); err != nil {
					t.Error("Expected ServicePrincipal() to not return error")
				}
				if createData.ServicePrincipalObjectID() != objectID {
					t.Errorf("Expected ServicePrincipalObjectID() to be 'object-id', got %s", createData.ServicePrincipalObjectID())
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
			test.createData.authProvider = authProvider
			test.verify(t, test.createData)
		})
	}
}

func TestCreateDataServicePrincipalName(t *testing.T) {
	createData := &createData{
		servicePrincipalName: "service-principal-name",
	}
	if createData.ServicePrincipalName() != "service-principal-name" {
		t.Errorf("Expected servicePrincipalName() to be 'service-principal-name', got %s", createData.ServicePrincipalName())
	}
	createData.servicePrincipalName = ""
	createData.aadApplicationName = appName
	if createData.ServicePrincipalName() != appName {
		t.Errorf("Expected servicePrincipalName() to be 'aad-application-name', got %s", createData.ServicePrincipalName())
	}
}

func TestCreateDataServicePrincipalObjectID(t *testing.T) {
	createData := &createData{
		servicePrincipalObjectID: objectID,
	}
	if createData.ServicePrincipalObjectID() != objectID {
		t.Errorf("Expected servicePrincipalObjectID() to be 'object-id', got %s", createData.ServicePrincipalObjectID())
	}
}

func TestCreateDataAzureRole(t *testing.T) {
	createData := &createData{
		azureRole: "azure-role",
	}
	if createData.AzureRole() != "azure-role" {
		t.Errorf("Expected AzureRole() to be 'azure-role', got %s", createData.AzureRole())
	}
}

func TestCreateDataAzureScope(t *testing.T) {
	createData := &createData{
		azureScope: "azure-scope",
	}
	if createData.AzureScope() != "azure-scope" {
		t.Errorf("Expected AzureScope() to be 'azure-scope', got %s", createData.AzureScope())
	}
}

func TestCreateDataAzureTenantID(t *testing.T) {
	createData := &createData{
		authProvider: &mockAuthProvider{
			azureTenantID: "azure-tenant-id",
		},
	}
	if createData.AzureTenantID() != "azure-tenant-id" {
		t.Errorf("Expected AzureTenantID() to be 'azure-tenant-id', got %s", createData.AzureTenantID())
	}
}

func testApplication(appID, objectID string) *graph.Application {
	app := graph.NewApplication()
	app.SetAppId(to.StringPtr(appID))
	app.SetId(to.StringPtr(objectID))
	return app
}

func testServicePrincipal(appID, objectID string) *graph.ServicePrincipal {
	sp := graph.NewServicePrincipal()
	sp.SetAppId(to.StringPtr(appID))
	sp.SetId(to.StringPtr(objectID))
	return sp
}
