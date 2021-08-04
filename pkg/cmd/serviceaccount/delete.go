package serviceaccount

import (
	"context"
	"errors"
	"fmt"

	fic "github.com/Azure/azure-workload-identity/pkg/cloud/federatedcredentials"
	"github.com/Azure/azure-workload-identity/pkg/cloud/graph"
	"github.com/Azure/azure-workload-identity/pkg/cloud/roleassignments"
	"github.com/Azure/azure-workload-identity/pkg/kuberneteshelper"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
)

type deleteCmd struct {
	authProvider

	name             string
	namespace        string
	issuer           string
	roleAssignmentID string
	appObjectID      string

	graphClient                graph.Interface
	federatedCredentialsClient fic.Interface
	kubeClient                 kubernetes.Interface
	roleAssignmentsClient      roleassignments.Interface
}

func newDeleteCmd() *cobra.Command {
	dc := &deleteCmd{
		authProvider: &authArgs{},
	}

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a service account",
		Long:  "Delete a service account",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := dc.validate(); err != nil {
				return err
			}
			return dc.getAuthArgs().validate()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			if dc.graphClient, err = graph.NewGraphClient(dc.getAuthArgs().azureClientID, dc.getAuthArgs().azureClientSecret, dc.getAuthArgs().azureTenantID); err != nil {
				panic(err)
			}
			if dc.federatedCredentialsClient, err = fic.NewFederatedCredentialsClient(dc.getAuthArgs().azureClientID, dc.getAuthArgs().azureClientSecret, dc.getAuthArgs().azureTenantID); err != nil {
				return err
			}
			if dc.kubeClient, err = kuberneteshelper.GetKubeClient(); err != nil {
				panic(err)
			}
			if dc.roleAssignmentsClient, err = roleassignments.NewRoleAssignmentsClient(
				dc.getAuthArgs().azureSubscriptionID,
				dc.getAuthArgs().azureClientID,
				dc.getAuthArgs().azureClientSecret,
				dc.getAuthArgs().azureTenantID); err != nil {
				panic(err)
			}
			return dc.run(cmd, args)
		},
	}

	f := cmd.Flags()
	f.StringVarP(&dc.name, "name", "", "", "Name of the service account")
	f.StringVarP(&dc.namespace, "namespace", "", "", "Namespace of the service account")
	f.StringVarP(&dc.issuer, "issuer", "", "", "OpenID Connect (OIDC) issuer URL")
	f.StringVarP(&dc.roleAssignmentID, "azure-role-assignment-id", "r", "", "Azure role assignment ID")
	f.StringVarP(&dc.appObjectID, "azure-app-object-id", "", "", "Azure app object ID")

	addAuthFlags(dc.getAuthArgs(), f)

	return cmd
}

func (dc *deleteCmd) validate() error {
	if dc.name == "" {
		return errors.New("--name must be specified")
	}

	if dc.namespace == "" {
		return errors.New("--namespace must be specified")
	}

	if dc.issuer == "" {
		return errors.New("--issuer must be specified")
	}

	if dc.roleAssignmentID == "" {
		return errors.New("--azure-role-assignment-id must be specified")
	}

	if dc.appObjectID == "" {
		return errors.New("--azure-app-object-id must be specified")
	}

	return nil
}

func (dc *deleteCmd) run(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// delete the role assignment
	err := dc.roleAssignmentsClient.Delete(context.Background(), dc.roleAssignmentID)
	if err != nil {
		return err
	}
	log.Debugf("Deleted role assignment %s", dc.roleAssignmentID)

	subject := fmt.Sprintf("system:serviceaccount:%s:%s", dc.namespace, dc.name)
	// delete the federated identity credential
	fc, err := dc.federatedCredentialsClient.GetFederatedCredential(ctx, dc.appObjectID, dc.issuer, subject)
	if err != nil {
		// TODO(aramase) handle not found error
		return err
	}
	err = dc.federatedCredentialsClient.DeleteFederatedCredential(ctx, dc.appObjectID, fc.ID)
	if err != nil {
		return err
	}
	log.Debugf("Deleted federated identity credential %s", fc.ID)

	// delete the Kubernetes service account
	err = kuberneteshelper.DeleteServiceAccount(dc.kubeClient, dc.namespace, dc.name)
	if err != nil {
		return err
	}
	log.Debugf("deleted kubernetes service account: %s/%s", dc.namespace, dc.name)

	// delete the app registration
	err = dc.graphClient.DeleteApplication(ctx, dc.getAuthArgs().azureTenantID, dc.appObjectID)
	if err != nil {
		return err
	}
	log.Debugf("deleted app registration: %s", dc.appObjectID)

	return nil
}
