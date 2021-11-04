package serviceaccount

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-workload-identity/pkg/cloud"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/auth"
	phases "github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/phases/create"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/phases/workflow"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/util"
	"github.com/Azure/azure-workload-identity/pkg/kuberneteshelper"
	"github.com/Azure/azure-workload-identity/pkg/webhook"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
)

func newCreateCmd(authProvider auth.Provider) *cobra.Command {
	createRunner := workflow.NewPhaseRunner()
	data := &createData{
		authProvider: authProvider,
	}

	cmd := &cobra.Command{
		Use: "create",
		RunE: func(cmd *cobra.Command, args []string) error {
			return createRunner.Run(data)
		},
	}

	f := cmd.Flags()
	f.StringVar(&data.serviceAccountName, "service-account-name", "", "Name of the service account")
	f.StringVar(&data.serviceAccountNamespace, "service-account-namespace", "default", "Namespace of the service account")
	f.StringVar(&data.serviceAccountIssuerURL, "service-account-issuer-url", "", "URL of the issuer")
	f.DurationVar(&data.serviceAccountTokenExpiration, "service-account-token-expiration", time.Duration(webhook.DefaultServiceAccountTokenExpiration)*time.Second, "Expiration time of the service account token. Must be between 1 hour and 24 hours")
	f.StringVar(&data.aadApplicationName, "aad-application-name", "", "Name of the AAD application, If not specified, the namespace, the name of the service account and the hash of the issuer URL will be used")
	f.StringVar(&data.aadApplicationClientID, "aad-application-client-id", "", "Client ID of the AAD application. If not specified, it will be fetched using the AAD application name")
	f.StringVar(&data.aadApplicationObjectID, "aad-application-object-id", "", "Object ID of the AAD application. If not specified, it will be fetched using the AAD application name")
	f.StringVar(&data.servicePrincipalName, "service-principal-name", "", "Name of the service principal that backs the AAD application. If this is not specified, the name of the AAD application will be used")
	f.StringVar(&data.servicePrincipalObjectID, "service-principal-object-id", "", "Object ID of the service principal that backs the AAD application. If not specified, it will be fetched using the service principal name")
	f.StringVar(&data.azureScope, "azure-scope", "", "Scope at which the role assignment or definition applies to")
	f.StringVar(&data.azureRole, "azure-role", "", "Role of the AAD application (see all available roles at https://docs.microsoft.com/en-us/azure/role-based-access-control/built-in-roles)")

	// append phases in order
	createRunner.AppendPhases(
		phases.NewAADApplicationPhase(),
		phases.NewServiceAccountPhase(),
		phases.NewFederatedIdentityPhase(),
		phases.NewRoleAssignmentPhase(),
	)
	createRunner.BindToCommand(cmd, data)

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
	servicePrincipal              *graphrbac.ServicePrincipal // cache
	servicePrincipalObjectID      string
	servicePrincipalName          string
	azureRole                     string
	azureScope                    string
	authProvider                  auth.Provider
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
func (c *createData) AADApplication() (*graphrbac.Application, error) {
	if c.aadApplication == nil {
		app, err := c.AzureClient().GetApplication(context.Background(), c.AADApplicationName())
		if err != nil {
			return nil, err
		}
		c.aadApplication = app
	}
	return c.aadApplication, nil
}

// AADApplicationName returns the name of the AAD application.
func (c *createData) AADApplicationName() string {
	name := c.aadApplicationName
	if name == "" {
		if c.ServiceAccountNamespace() != "" && c.ServiceAccountName() != "" && c.ServiceAccountIssuerURL() != "" {
			log.Warn("--aad-application-name not specified, constructing name with service account namespace, name, and the hash of the issuer URL")
			name = fmt.Sprintf("%s-%s-%s", c.ServiceAccountNamespace(), c.serviceAccountName, util.GetIssuerHash(c.ServiceAccountIssuerURL()))
		}
	}
	return name
}

// AADApplicationClientID returns the client ID of the AAD application.
// This will be used for annotating the service account.
func (c *createData) AADApplicationClientID() string {
	if c.aadApplicationClientID != "" {
		return c.aadApplicationClientID
	}

	app, err := c.AADApplication()
	if err != nil {
		log.WithError(err).Error("failed to get AAD application client ID. Returning an empty string")
		return ""
	}
	return *app.AppID
}

// AADApplicationObjectID returns the object ID of the AAD application.
// This will be used for creating or removing the federated identity credential.
func (c *createData) AADApplicationObjectID() string {
	if c.aadApplicationObjectID != "" {
		return c.aadApplicationObjectID
	}

	app, err := c.AADApplication()
	if err != nil {
		log.WithError(err).Error("failed to get AAD application object ID. Returning an empty string")
		return ""
	}
	return *app.ObjectID
}

// ServicePrincipal returns the service principal object.
// This will return the cached value if it has been created.
func (c *createData) ServicePrincipal() (*graphrbac.ServicePrincipal, error) {
	if c.servicePrincipal == nil {
		sp, err := c.AzureClient().GetServicePrincipal(context.Background(), c.ServicePrincipalName())
		if err != nil {
			return nil, err
		}
		c.servicePrincipal = sp
	}
	return c.servicePrincipal, nil
}

// ServicePrincipalName returns the name of the service principal.
func (c *createData) ServicePrincipalName() string {
	name := c.servicePrincipalName
	// fall back to the name of the AAD application
	if name == "" {
		log.Warn("--service-principal-name not specified, falling back to AAD application name")
		name = c.AADApplicationName()
	}
	return name
}

// ServicePrincipalObjectID returns the object ID of the service principal.
// This will be used for creating or removing the role assignment.
func (c *createData) ServicePrincipalObjectID() string {
	if c.servicePrincipalObjectID != "" {
		return c.servicePrincipalObjectID
	}

	sp, err := c.ServicePrincipal()
	if err != nil {
		log.WithError(err).Error("failed to get service principal object ID. Returning an empty string")
		return ""
	}
	return *sp.ObjectID
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
	return c.authProvider.GetAzureTenantID()
}

// AzureClient returns the Azure client.
func (c *createData) AzureClient() cloud.Interface {
	return c.authProvider.GetAzureClient()
}

// KubeClient returns the Kubernetes client.
func (c *createData) KubeClient() (kubernetes.Interface, error) {
	return kuberneteshelper.GetKubeClient()
}
