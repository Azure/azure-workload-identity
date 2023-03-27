package phases

import (
	"time"

	"github.com/microsoftgraph/msgraph-beta-sdk-go/models/microsoft/graph"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/Azure/azure-workload-identity/pkg/cloud"
)

// CreateData is the interface to use for create phase.
// The "createData" type from cmd/create.go must satisfy this interface.
type CreateData interface {
	// ServiceAccountName returns the name of the service account.
	ServiceAccountName() string

	// ServiceAccountNamespace returns the namespace of the service account.
	ServiceAccountNamespace() string

	// ServiceAccountIssuerURL returns the issuer URL of the service account.
	ServiceAccountIssuerURL() string

	// ServiceAccountTokenExpiration returns the expiration time of the service account token.
	ServiceAccountTokenExpiration() time.Duration

	// AADApplication returns the AAD application object.
	// This will return the cached value if it has been created.
	AADApplication() (*graph.Application, error)

	// AADApplicationName returns the name of the AAD application.
	AADApplicationName() string

	// AADApplicationClientID returns the client ID of the AAD application.
	// This will be used for annotating the service account.
	AADApplicationClientID() string

	// AADApplicationObjectID returns the object ID of the AAD application.
	// This will be used for creating or removing the federated identity credential.
	AADApplicationObjectID() string

	// ServicePrincipal returns the service principal object.
	// This will return the cached value if it has been created.
	ServicePrincipal() (*graph.ServicePrincipal, error)

	// ServicePrincipalName returns the name of the service principal.
	ServicePrincipalName() string

	// ServicePrincipalObjectID returns the object ID of the service principal.
	// This will be used for creating or removing the role assignment.
	ServicePrincipalObjectID() string

	// AzureRole returns the Azure role.
	AzureRole() string

	// AzureScope returns the Azure scope.
	AzureScope() string

	// AzureTenantID returns the Azure tenant ID.
	AzureTenantID() string

	// AzureClient returns the Azure client.
	AzureClient() cloud.Interface

	// KubeClient returns the Kubernetes client.
	KubeClient() (client.Client, error)
}
