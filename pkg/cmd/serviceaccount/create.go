package serviceaccount

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-workload-identity/pkg/cloud"
	"github.com/Azure/azure-workload-identity/pkg/kuberneteshelper"
	"github.com/Azure/azure-workload-identity/pkg/version"
	"github.com/Azure/azure-workload-identity/pkg/webhook"

	"github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-01-01-preview/authorization"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
)

type createCmd struct {
	authProvider

	name            string
	namespace       string
	issuer          string
	azureRole       string
	azureScope      string
	tokenExpiration time.Duration

	azureClient cloud.Interface
	kubeClient  kubernetes.Interface
}

func newCreateCmd() *cobra.Command {
	cc := createCmd{
		authProvider: &authArgs{},
	}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a workload identity",
		Long:  "This command provides the ability to create an app, add federated identity credential, create the Kubernetes service account and perform role assignment",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := cc.validate(); err != nil {
				return err
			}
			return cc.getAuthArgs().validate()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			if cc.azureClient, err = cc.getClient(); err != nil {
				return err
			}
			if cc.kubeClient, err = kuberneteshelper.GetKubeClient(); err != nil {
				return err
			}
			return cc.run()
		},
	}

	f := cmd.Flags()
	f.StringVar(&cc.name, "name", "", "Name of the service account")
	f.StringVar(&cc.namespace, "namespace", "default", "Namespace of the service account")
	f.StringVar(&cc.issuer, "issuer", "", "OpenID Connect (OIDC) issuer URL")
	f.StringVar(&cc.azureRole, "azure-role", "", "Azure Role name (see all available roles at https://docs.microsoft.com/en-us/azure/role-based-access-control/built-in-roles)")
	f.StringVar(&cc.azureScope, "azure-scope", "", "Scope at which the role assignment or definition applies to")
	f.DurationVar(&cc.tokenExpiration, "token-expiration", time.Duration(webhook.DefaultServiceAccountTokenExpiration)*time.Second, "Expiration time of the service account token. Must be between 1 hour and 24 hours")
	addAuthFlags(cc.getAuthArgs(), f)

	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("issuer")
	_ = cmd.MarkFlagRequired("azure-role")
	_ = cmd.MarkFlagRequired("azure-scope")

	return cmd
}

func (cc *createCmd) validate() error {
	minTokenExpirationDuration := time.Duration(webhook.MinServiceAccountTokenExpiration) * time.Second
	if cc.tokenExpiration < minTokenExpirationDuration {
		return errors.Errorf("--token-expiration must be greater than or equal to %s", minTokenExpirationDuration.String())
	}
	if cc.tokenExpiration > 24*time.Hour {
		return errors.Errorf("--token-expiration must be less than or equal to 24h")
	}
	return nil
}

func (cc *createCmd) run() error {
	ctx := context.Background()

	// the name of the app registration is of the format <service account namespace>-<service account name>-<issuer hash>
	appName := fmt.Sprintf("%s-%s-%s", cc.namespace, cc.name, getIssuerHash(cc.issuer))
	tags := []string{
		fmt.Sprintf("serviceAccount: %s-%s", cc.name, cc.namespace),
		fmt.Sprintf("azwi version: %s, commit: %s", version.BuildVersion, version.Vcs),
	}

	// Check if the application with the same name already exists
	app, err := cc.azureClient.GetApplication(ctx, appName)
	if err != nil {
		if !cloud.IsNotFound(err) {
			return errors.Wrap(err, "failed to get application")
		}
		// create the application as it doesn't exist
		app, err = cc.azureClient.CreateApplication(ctx, appName)
		if err != nil {
			return errors.Wrap(err, "failed to create application")
		}
	}
	log.Infof("created application with name: '%s', objectID: '%s'", appName, *app.ObjectID)

	// Check if the service principal with the same name already exists
	servicePrincipal, err := cc.azureClient.GetServicePrincipal(ctx, appName)
	if err != nil {
		if !cloud.IsNotFound(err) {
			return errors.Wrap(err, "failed to get service principal")
		}
		// create the service principal as it doesn't exist
		servicePrincipal, err = cc.azureClient.CreateServicePrincipal(ctx, *app.AppID, tags)
		if err != nil {
			return errors.Wrap(err, "failed to create service principal")
		}
	}
	log.Infof("created service principal with name: '%s', objectID: '%s'", *servicePrincipal.DisplayName, *servicePrincipal.ObjectID)

	// TODO(aramase) make the update behavior configurable. If the service account already exists, fail if --overwrite is not specified
	err = kuberneteshelper.CreateOrUpdateServiceAccount(ctx, cc.kubeClient, cc.namespace, cc.name, *app.AppID, cc.getAuthArgs().tenantID, cc.tokenExpiration)
	if err != nil {
		return errors.Wrap(err, "failed to create service account")
	}
	log.Debugf("created kubernetes service account: %s/%s", cc.namespace, cc.name)

	// add the federated credential
	subject := getSubject(cc.namespace, cc.name)
	description := fmt.Sprintf("Federated Service Account for %s/%s", cc.namespace, cc.name)
	audiences := []string{webhook.DefaultAudience}

	fc := cloud.NewFederatedCredential(*app.ObjectID, cc.issuer, subject, description, audiences)
	err = cc.azureClient.AddFederatedCredential(ctx, *app.ObjectID, fc)
	if err != nil && !cloud.IsAlreadyExists(err) {
		return errors.Wrap(err, "failed to add federated credential")
	}
	log.Infof("added federated credential for %s", subject)

	var ra authorization.RoleAssignment
	// create the role assignment using object id of the service principal
	if ra, err = cc.azureClient.CreateRoleAssignment(ctx, cc.azureScope, cc.azureRole, *servicePrincipal.ObjectID); err == nil {
		log.Infof("Created role assignment for scope=%s, role=%s, principal=%s, roleAssignmentID=%s", cc.azureScope, cc.azureRole, *servicePrincipal.ObjectID, *ra.ID)
		return nil
	}
	if err != nil && !cloud.IsAlreadyExists(err) {
		return errors.Wrap(err, "failed to create role assignment")
	}

	return nil
}
