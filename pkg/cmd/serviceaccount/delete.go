package serviceaccount

import (
	"context"
	"fmt"

	"github.com/Azure/azure-workload-identity/pkg/cloud"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/auth"
	phases "github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/phases/delete"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/phases/workflow"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/util"
	"github.com/Azure/azure-workload-identity/pkg/kuberneteshelper"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
)

func newDeleteCmd(authProvider auth.Provider) *cobra.Command {
	deleteRunner := workflow.NewPhaseRunner()
	data := &deleteData{
		authProvider: authProvider,
	}

	cmd := &cobra.Command{
		Use: "delete",
		RunE: func(cmd *cobra.Command, args []string) error {
			return deleteRunner.Run(data)
		},
	}

	f := cmd.Flags()
	authProvider.AddFlags(f)
	f.StringVar(&data.serviceAccountName, "service-account-name", "", "Name of the service account")
	f.StringVar(&data.serviceAccountNamespace, "service-account-namespace", "default", "Namespace of the service account")
	f.StringVar(&data.serviceAccountIssuerURL, "service-account-issuer-url", "", "URL of the issuer")
	f.StringVar(&data.aadApplicationName, "aad-application-name", "", "Name of the AAD application. If not specified, the namespace, the name of the service account and the hash of the issuer URL will be used")
	f.StringVar(&data.aadApplicationObjectID, "aad-application-object-id", "", "Object ID of the AAD application. If not specified, it will be fetched using the AAD application name")
	f.StringVar(&data.roleAssignmentID, "role-assignment-id", "", "Azure role assignment ID")

	// append phases in order
	deleteRunner.AppendPhases(
		phases.NewRoleAssignmentPhase(),
		phases.NewFederatedIdentityPhase(),
		phases.NewServiceAccountPhase(),
		phases.NewAADApplicationPhase(),
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
	aadApplication          *graphrbac.Application // cache
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
func (d *deleteData) AADApplication() (*graphrbac.Application, error) {
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
		log.Warn("--aad-application-name not specified, constructing name with service account namespace, name, and the hash of the issuer URL")
		if d.ServiceAccountNamespace() != "" && d.ServiceAccountName() != "" && d.ServiceAccountIssuerURL() != "" {
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
	return *app.ObjectID
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
func (d *deleteData) KubeClient() (kubernetes.Interface, error) {
	return kuberneteshelper.GetKubeClient()
}
