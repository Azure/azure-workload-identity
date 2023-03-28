package phases

import (
	"github.com/microsoftgraph/msgraph-beta-sdk-go/models/microsoft/graph"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/Azure/azure-workload-identity/pkg/cloud"
)

// DeleteData is the interface to use for create phase.
// The "deleteData" type from cmd/delete.go must satisfy this interface.
type DeleteData interface {
	// ServiceAccountName returns the name of the service account.
	ServiceAccountName() string

	// ServiceAccountNamespace returns the namespace of the service account.
	ServiceAccountNamespace() string

	// ServiceAccountIssuerURL returns the issuer URL of the service account.
	ServiceAccountIssuerURL() string

	// AADApplication returns the AAD application object.
	// This will return the cached value if it has been created.
	AADApplication() (*graph.Application, error)

	// AADApplicationName returns the name of the AAD application.
	AADApplicationName() string

	// AADApplicationObjectID returns the object ID of the AAD application.
	// This will be used for creating or removing the federated identity credential.
	AADApplicationObjectID() string

	// RoleDefinitionID returns the role definition ID.
	RoleAssignmentID() string

	// AzureClient returns the Azure client.
	AzureClient() cloud.Interface

	// KubeClient returns the Kubernetes client.
	KubeClient() (client.Client, error)
}
