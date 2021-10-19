package serviceaccount

import (
	"context"
	"time"

	"github.com/Azure/azure-workload-identity/pkg/cloud"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/auth"
	phases "github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/phases/create"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/phases/workflow"
	"github.com/Azure/azure-workload-identity/pkg/kuberneteshelper"
	"github.com/Azure/azure-workload-identity/pkg/webhook"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
)

func newCreateCmd() *cobra.Command {
	authProvider := auth.NewProvider()
	createRunner := workflow.NewPhaseRunner()
	data := &createData{}
	cmd := &cobra.Command{
		Use: "create",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return authProvider.Validate()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// inject various clients and fields at runtime
			var err error
			if data.azureClient, err = authProvider.GetAzureClient(); err != nil {
				return err
			}
			if data.kubeClient, err = kuberneteshelper.GetKubeClient(); err != nil {
				return err
			}
			data.azureTenantID = authProvider.GetAzureTenantID()

			return createRunner.Run(data)
		},
	}

	f := cmd.Flags()
	authProvider.AddFlags(f)
	f.StringVar(&data.serviceAccountName, "service-account-name", "", "Name of the service account")
	f.StringVar(&data.serviceAccountNamespace, "service-account-namespace", "default", "Namespace of the service account")
	f.StringVar(&data.serviceAccountIssuerURL, "service-account-issuer-url", "", "URL of the issuer")
	f.DurationVar(&data.serviceAccountTokenExpiration, "service-account-token-expiration", time.Duration(webhook.DefaultServiceAccountTokenExpiration)*time.Second, "Expiration time of the service account token. Must be between 1 hour and 24 hours")
	f.StringVar(&data.aadApplicationName, "aad-application-name", "", "Name of the AAD application, If not specified, the name of the service account will be used")
	f.StringVar(&data.aadApplicationClientID, "aad-application-client-id", "", "Client ID of the AAD application. If not specified, it will be fetched using the AAD application name")
	f.StringVar(&data.aadApplicationObjectID, "aad-application-object-id", "", "Object ID of the AAD application. If not specified, it will be fetched using the AAD application name")
	f.StringVar(&data.servicePrincipalName, "service-principal-name", "", "Name of the service principal that backs the AAD application. If this is not specified, the name of the AAD application will be used")
	f.StringVar(&data.servicePrincipalObjectID, "service-principal-object-id", "", "Object ID of the service principal that backs the AAD application. If not specified, it will be fetched using the service principal name")
	f.StringVar(&data.azureScope, "azure-scope", "", "Scope of the AAD application")
	f.StringVar(&data.azureRole, "azure-role", "", "Role of the AAD application")

	// append phases in order
	createRunner.AppendPhases(
		phases.NewAADApplicationPhase(),
		phases.NewServiceAccountPhase(),
		phases.NewFederatedIdentityPhase(),
		phases.NewRoleAssignmentPhase(),
	)
	createRunner.BindToCommand(cmd)

	return cmd
}

// createData is an implementation of phases.CreateData in
// pkg/cmd/serviceaccount/phases/create/data.go
type createData struct {
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

var _ phases.CreateData = &createData{}

// ServiceAccountName returns the name of the service account.
func (c *createData) ServiceAccountName() string {
	return c.serviceAccountName
}

// ServiceAccountNamespace returns the namespace of the service account.
func (c *createData) ServiceAccountNamespace() string {
	return c.serviceAccountNamespace
}

// ServiceAccountIssuerURL returns the issuer URL of the service account.
func (c *createData) ServiceAccountIssuerURL() string {
	return c.serviceAccountIssuerURL
}

// ServiceAccountTokenExpiration returns the expiration time of the service account token.
func (c *createData) ServiceAccountTokenExpiration() time.Duration {
	return c.serviceAccountTokenExpiration
}

// AADApplication returns the AAD application object.
// This will return the cached value if it has been created.
func (c *createData) AADApplication() *graphrbac.Application {
	if c.aadApplication == nil {
		app, err := c.AzureClient().GetApplication(context.Background(), c.AADApplicationName())
		if err != nil {
			if cloud.IsNotFound(err) {
				log.WithField("name", c.AADApplicationName()).Debug("AAD application not found")
			} else {
				log.WithError(err).Debug("failed to get AAD application")
			}
			return nil
		}
		c.aadApplication = app
	}
	return c.aadApplication
}

// AADApplicationName returns the name of the AAD application.
func (c *createData) AADApplicationName() string {
	name := c.aadApplicationName
	if name == "" {
		name = c.ServiceAccountName()
		log.WithField("name", name).Debug("AAD application name not specified, falling back to service account name")
	}
	return name
}

// AADApplicationClientID returns the client ID of the AAD application.
// This will be used for annotating the service account.
func (c *createData) AADApplicationClientID() string {
	if c.aadApplicationClientID != "" {
		return c.aadApplicationClientID
	}

	app := c.AADApplication()
	if app == nil {
		return ""
	}
	return *app.AppID
}

// AADApplicationObjectID returns the object ID of the AAD application.
// This will be used for creating or removing the federated identity.
func (c *createData) AADApplicationObjectID() string {
	if c.aadApplicationObjectID != "" {
		return c.aadApplicationObjectID
	}

	app := c.AADApplication()
	if app == nil {
		return ""
	}
	return *app.ObjectID
}

// ServicePrincipal returns the service principal object.
// This will return the cached value if it has been created.
func (c *createData) ServicePrincipal() *graphrbac.ServicePrincipal {
	if c.servicePrincipal == nil {
		sp, err := c.AzureClient().GetServicePrincipal(context.Background(), c.ServicePrincipalName())
		if err != nil {
			if cloud.IsNotFound(err) {
				log.WithField("name", c.ServiceAccountName()).Debug("service principal not found")
			}
			return nil
		}
		c.servicePrincipal = sp
	}
	return c.servicePrincipal
}

// ServicePrincipalName returns the name of the service principal.
func (c *createData) ServicePrincipalName() string {
	name := c.servicePrincipalName
	// fall back to the name of the AAD application
	if name == "" {
		name = c.AADApplicationName()
		log.WithField("name", name).Debug("service principal name not specified, falling back to AAD application name")
	}
	return name
}

// ServicePrincipalObjectID returns the object ID of the service principal.
// This will be used for creating or removing the role assignment.
func (c *createData) ServicePrincipalObjectID() string {
	if c.servicePrincipalObjectID != "" {
		return c.servicePrincipalObjectID
	}

	sp := c.ServicePrincipal()
	if sp == nil {
		return ""
	}
	return *c.servicePrincipal.ObjectID
}

// AzureRole returns the Azure role.
func (c *createData) AzureRole() string {
	return c.azureRole
}

// AzureScope returns the Azure scope.
func (c *createData) AzureScope() string {
	return c.azureScope
}

// AzureTenantID returns the Azure tenant ID.
func (c *createData) AzureTenantID() string {
	return c.azureTenantID
}

// AzureClient returns the Azure client.
func (c *createData) AzureClient() cloud.Interface {
	return c.azureClient
}

// KubeClient returns the Kubernetes client.
func (c *createData) KubeClient() kubernetes.Interface {
	return c.kubeClient
}
