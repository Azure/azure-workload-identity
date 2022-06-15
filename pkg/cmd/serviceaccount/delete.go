package serviceaccount

import (
	"context"
	"fmt"

	"github.com/Azure/azure-workload-identity/pkg/cloud"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/auth"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/options"
	phases "github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/phases/delete"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/phases/workflow"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/util"
	"github.com/Azure/azure-workload-identity/pkg/kuberneteshelper"

	"github.com/microsoftgraph/msgraph-beta-sdk-go/models/microsoft/graph"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	roleAssignmentPhase    = phases.NewRoleAssignmentPhase()
	federatedIdentityPhase = phases.NewFederatedIdentityPhase()
	serviceAccountPhase    = phases.NewServiceAccountPhase()
	aadApplicationPhase    = phases.NewAADApplicationPhase()
)

func newDeleteCmd(authProvider auth.Provider) *cobra.Command {
	deleteRunner := workflow.NewPhaseRunner()
	data := &deleteData{
		authProvider: authProvider,
	}

	cmd := &cobra.Command{
		Use: "delete",
		RunE: func(cmd *cobra.Command, args []string) error {
			// if we are running the AAD application delete phase, we can
			// slightly optimize the delete command by skipping the federated-identity
			// phase because it will get removed when the AAD application is removed.
			if deleteRunner.IsPhaseActive(aadApplicationPhase) {
				deleteRunner.AppendSkipPhases(federatedIdentityPhase)
			}
			return deleteRunner.Run(data)
		},
	}

	f := cmd.Flags()
	f.StringVar(&data.serviceAccountName, options.ServiceAccountName.Flag, "", options.ServiceAccountName.Description)
	f.StringVar(&data.serviceAccountNamespace, options.ServiceAccountNamespace.Flag, "default", options.ServiceAccountNamespace.Description)
	f.StringVar(&data.serviceAccountIssuerURL, options.ServiceAccountIssuerURL.Flag, "", options.ServiceAccountIssuerURL.Description)
	f.StringVar(&data.aadApplicationName, options.AADApplicationName.Flag, "", options.AADApplicationName.Description)
	f.StringVar(&data.aadApplicationObjectID, options.AADApplicationObjectID.Flag, "", options.AADApplicationObjectID.Description)
	f.StringVar(&data.roleAssignmentID, options.RoleAssignmentID.Flag, "", options.RoleAssignmentID.Description)

	// append phases in order
	deleteRunner.AppendPhases(
		roleAssignmentPhase,
		federatedIdentityPhase,
		serviceAccountPhase,
		aadApplicationPhase,
	)
	deleteRunner.BindToCommand(cmd, data)

	return cmd
}

// deleteData is an implementation of phases.DeleteData in
// pkg/cmd/serviceaccount/phases/delete/data.go
type deleteData struct {
	serviceAccountName      string
	serviceAccountNamespace string
	serviceAccountIssuerURL string
	aadApplication          *graph.Application // cache
	aadApplicationName      string
	aadApplicationObjectID  string
	roleAssignmentID        string
	authProvider            auth.Provider
}

var _ phases.DeleteData = &deleteData{}

// ServiceAccountName returns the name of the service account.
func (d *deleteData) ServiceAccountName() string {
	return d.serviceAccountName
}

// ServiceAccountNamespace returns the namespace of the service account.
func (d *deleteData) ServiceAccountNamespace() string {
	return d.serviceAccountNamespace
}

// ServiceAccountIssuerURL returns the issuer URL of the service account.
func (d *deleteData) ServiceAccountIssuerURL() string {
	return d.serviceAccountIssuerURL
}

// AADApplication returns the AAD application object.
// This will return the cached value if it has been created.
func (d *deleteData) AADApplication() (*graph.Application, error) {
	if d.aadApplication == nil {
		app, err := d.AzureClient().GetApplication(context.Background(), d.AADApplicationName())
		if err != nil {
			return nil, err
		}
		d.aadApplication = app
	}
	return d.aadApplication, nil
}

// AADApplicationName returns the name of the AAD application.
func (d *deleteData) AADApplicationName() string {
	name := d.aadApplicationName
	if name == "" {
		if d.ServiceAccountNamespace() != "" && d.ServiceAccountName() != "" && d.ServiceAccountIssuerURL() != "" {
			log.Warn("--aad-application-name not specified, constructing name with service account namespace, name, and the hash of the issuer URL")
			name = fmt.Sprintf("%s-%s-%s", d.ServiceAccountNamespace(), d.serviceAccountName, util.GetIssuerHash(d.ServiceAccountIssuerURL()))
		}
	}
	return name
}

// AADApplicationObjectID returns the object ID of the AAD application.
// This will be used for creating or removing the federated identity credential.
func (d *deleteData) AADApplicationObjectID() string {
	if d.aadApplicationObjectID != "" {
		return d.aadApplicationObjectID
	}

	app, err := d.AADApplication()
	if err != nil {
		log.WithError(err).Error("failed to get AAD application object ID. Returning an empty string")
		return ""
	}
	return *app.GetId()
}

// AzureClient returns the Azure client.
func (d *deleteData) RoleAssignmentID() string {
	return d.roleAssignmentID
}

// AzureClient returns the Azure client.
func (d *deleteData) AzureClient() cloud.Interface {
	return d.authProvider.GetAzureClient()
}

// KubeClient returns the Kubernetes client.
func (d *deleteData) KubeClient() (client.Client, error) {
	return kuberneteshelper.GetKubeClient()
}
