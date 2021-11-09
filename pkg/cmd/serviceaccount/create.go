package serviceaccount

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-workload-identity/pkg/cloud"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/auth"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/options"
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
	f.StringVar(&data.serviceAccountName, options.ServiceAccountName.Flag, "", options.ServiceAccountName.Description)
	f.StringVar(&data.serviceAccountNamespace, options.ServiceAccountNamespace.Flag, "default", options.ServiceAccountNamespace.Description)
	f.StringVar(&data.serviceAccountIssuerURL, options.ServiceAccountIssuerURL.Flag, "", options.ServiceAccountIssuerURL.Description)
	f.DurationVar(&data.serviceAccountTokenExpiration, options.ServiceAccountTokenExpiration.Flag, time.Duration(webhook.DefaultServiceAccountTokenExpiration)*time.Second, options.ServiceAccountTokenExpiration.Description)
	f.StringVar(&data.aadApplicationName, options.AADApplicationName.Flag, "", options.AADApplicationName.Description)
	f.StringVar(&data.aadApplicationClientID, options.AADApplicationClientID.Flag, "", options.AADApplicationClientID.Description)
	f.StringVar(&data.aadApplicationObjectID, options.AADApplicationObjectID.Flag, "", options.AADApplicationObjectID.Description)
	f.StringVar(&data.servicePrincipalName, options.ServicePrincipalName.Flag, "", options.ServicePrincipalName.Description)
	f.StringVar(&data.servicePrincipalObjectID, options.ServicePrincipalObjectID.Flag, "", options.ServicePrincipalObjectID.Description)
	f.StringVar(&data.azureScope, options.AzureScope.Flag, "", options.AzureScope.Description)
	f.StringVar(&data.azureRole, options.AzureRole.Flag, "", options.AzureRole.Description)

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
