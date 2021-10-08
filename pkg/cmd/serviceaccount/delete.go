package serviceaccount

import (
	"context"

	"github.com/Azure/azure-workload-identity/pkg/cloud"
	"github.com/Azure/azure-workload-identity/pkg/kuberneteshelper"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
)

type deleteCmd struct {
	authProvider

	name             string
	namespace        string
	issuer           string
	roleAssignmentID string
	appObjectID      string

	azureClient cloud.Interface
	kubeClient  kubernetes.Interface
}

func newDeleteCmd() *cobra.Command {
	dc := &deleteCmd{
		authProvider: &authArgs{},
	}

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a service account",
		Long:  "This command provides the ability to remove the role assignment, federated identity credential, Kubernetes service account, and application",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return dc.getAuthArgs().validate()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			if dc.azureClient, err = dc.getClient(); err != nil {
				return err
			}
			if dc.kubeClient, err = kuberneteshelper.GetKubeClient(); err != nil {
				return err
			}
			return dc.run()
		},
	}

	f := cmd.Flags()
	f.StringVar(&dc.name, "name", "", "Name of the service account")
	f.StringVar(&dc.namespace, "namespace", "default", "Namespace of the service account")
	f.StringVar(&dc.issuer, "issuer", "", "OpenID Connect (OIDC) issuer URL")
	f.StringVar(&dc.roleAssignmentID, "role-assignment-id", "", "Azure role assignment ID")
	f.StringVar(&dc.appObjectID, "application-object-id", "", "Azure application object ID")

	addAuthFlags(dc.getAuthArgs(), f)

	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("issuer")
	_ = cmd.MarkFlagRequired("role-assignment-id")
	_ = cmd.MarkFlagRequired("application-object-id")

	return cmd
}

func (dc *deleteCmd) run() error {
	ctx := context.Background()
	var err error

	// TODO(aramase): consider supporting deletion of role assignment with scope, role and application id
	// delete the role assignment
	if _, err := dc.azureClient.DeleteRoleAssignment(ctx, dc.roleAssignmentID); err != nil {
		if !cloud.IsRoleAssignmentAlreadyDeleted(err) {
			return errors.Wrap(err, "failed to delete role assignment")
		}
		log.Debugf("Role assignment with id=%s already deleted", dc.roleAssignmentID)
	} else {
		log.Infof("Deleted role assignment %s", dc.roleAssignmentID)
	}

	var fc cloud.FederatedCredential
	subject := getSubject(dc.namespace, dc.name)
	// delete the federated identity credential
	if fc, err = dc.azureClient.GetFederatedCredential(ctx, dc.appObjectID, dc.issuer, subject); err != nil {
		if !cloud.IsResourceNotFound(err) {
			return errors.Wrap(err, "failed to get federated credential")
		}
		log.Debugf("Federated credential for subject=%s, issuer=%s not found", subject, dc.issuer)
	} else {
		if err = dc.azureClient.DeleteFederatedCredential(ctx, dc.appObjectID, fc.ID); err != nil {
			return errors.Wrap(err, "failed to delete federated credential")
		}
		log.Infof("Deleted federated identity credential for subject=%s, issuer=%s with id=%s", subject, dc.issuer, fc.ID)
	}

	// delete the Kubernetes service account
	if err = kuberneteshelper.DeleteServiceAccount(ctx, dc.kubeClient, dc.namespace, dc.name); err != nil {
		if !apierrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to delete service account")
		}
		log.Debugf("Service account %s/%s not found", dc.namespace, dc.name)
	} else {
		log.Infof("Deleted kubernetes service account=%s/%s", dc.namespace, dc.name)
	}

	// TODO(aramase): consider deleting the application using the display name
	// delete the application with the objectID
	// this is cascading delete of all resources associated with the application
	if _, err = dc.azureClient.DeleteApplication(ctx, dc.appObjectID); err != nil {
		if !cloud.IsResourceNotFound(err) {
			return errors.Wrap(err, "failed to delete application")
		}
		log.Debugf("Application with objectID=%s not found", dc.appObjectID)
	} else {
		log.Debugf("Deleted application with objectID=%s", dc.appObjectID)
	}

	return nil
}
