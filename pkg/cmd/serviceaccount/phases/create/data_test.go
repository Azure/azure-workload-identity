package phases

import (
	"fmt"
	"time"

	"github.com/Azure/azure-workload-identity/pkg/cloud"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/util"

	"github.com/microsoftgraph/msgraph-beta-sdk-go/models"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type mockCreateData struct {
	serviceAccountName            string
	serviceAccountNamespace       string
	serviceAccountIssuerURL       string
	serviceAccountTokenExpiration time.Duration
	aadApplication                models.Applicationable // cache
	aadApplicationName            string
	aadApplicationClientID        string
	aadApplicationObjectID        string
	servicePrincipal              models.ServicePrincipalable
	servicePrincipalObjectID      string
	servicePrincipalName          string
	azureRole                     string
	azureScope                    string
	azureTenantID                 string
	azureClient                   cloud.Interface
	kubeClient                    client.Client
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

func (c *mockCreateData) AADApplication() (models.Applicationable, error) {
	if c.aadApplication == nil {
		return nil, errors.New("not found")
	}
	return c.aadApplication, nil
}

func (c *mockCreateData) AADApplicationName() string {
	if c.aadApplicationName == "" && c.ServiceAccountNamespace() != "" && c.ServiceAccountName() != "" && c.ServiceAccountIssuerURL() != "" {
		return fmt.Sprintf("%s-%s-%s", c.ServiceAccountNamespace(), c.serviceAccountName, util.GetIssuerHash(c.ServiceAccountIssuerURL()))
	}
	return c.aadApplicationName
}

func (c *mockCreateData) AADApplicationClientID() string {
	return c.aadApplicationClientID
}

func (c *mockCreateData) AADApplicationObjectID() string {
	return c.aadApplicationObjectID
}

func (c *mockCreateData) ServicePrincipal() (models.ServicePrincipalable, error) {
	if c.servicePrincipal == nil {
		return nil, errors.New("not found")
	}
	return c.servicePrincipal, nil
}

func (c *mockCreateData) ServicePrincipalName() string {
	if c.servicePrincipalName == "" {
		return c.AADApplicationName()
	}
	return c.servicePrincipalName
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

func (c *mockCreateData) KubeClient() (client.Client, error) {
	if c.kubeClient == nil {
		return nil, errors.New("not found")
	}
	return c.kubeClient, nil
}
