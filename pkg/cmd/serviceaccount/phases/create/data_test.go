package phases

import (
	"time"

	"github.com/Azure/azure-workload-identity/pkg/cloud"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"k8s.io/client-go/kubernetes"
)

type mockCreateData struct {
	serviceAccountName            string
	serviceAccountNamespace       string
	serviceAccountIssuerURL       string
	serviceAccountTokenExpiration time.Duration
	aadApplication                *graphrbac.Application // cache
	aadApplicationName            string
	aadApplicationClientID        string
	aadApplicationObjectID        string
	servicePrincipal              *graphrbac.ServicePrincipal
	servicePrincipalObjectID      string
	servicePrincipalName          string
	azureRole                     string
	azureScope                    string
	azureTenantID                 string
	azureClient                   cloud.Interface
	kubeClient                    kubernetes.Interface
}

var _ CreateData = &mockCreateData{}

func (c *mockCreateData) ServiceAccountName() string {
	return c.serviceAccountName
}

func (c *mockCreateData) ServiceAccountNamespace() string {
	return c.serviceAccountNamespace
}

func (c *mockCreateData) ServiceAccountIssuerURL() string {
	return c.serviceAccountIssuerURL
}

func (c *mockCreateData) ServiceAccountTokenExpiration() time.Duration {
	return c.serviceAccountTokenExpiration
}

func (c *mockCreateData) AADApplication() *graphrbac.Application {
	return c.aadApplication
}

func (c *mockCreateData) AADApplicationName() string {
	return c.aadApplicationName
}

func (c *mockCreateData) AADApplicationClientID() string {
	return c.aadApplicationClientID
}

func (c *mockCreateData) AADApplicationObjectID() string {
	return c.aadApplicationObjectID
}

func (c *mockCreateData) ServicePrincipal() *graphrbac.ServicePrincipal {
	return c.servicePrincipal
}

func (c *mockCreateData) ServicePrincipalName() string {
	return c.serviceAccountName
}

func (c *mockCreateData) ServicePrincipalObjectID() string {
	return c.servicePrincipalObjectID
}

func (c *mockCreateData) AzureRole() string {
	return c.azureRole
}

func (c *mockCreateData) AzureScope() string {
	return c.azureScope
}

func (c *mockCreateData) AzureTenantID() string {
	return c.azureTenantID
}

func (c *mockCreateData) AzureClient() cloud.Interface {
	return c.azureClient
}

func (c *mockCreateData) KubeClient() kubernetes.Interface {
	return c.kubeClient
}
