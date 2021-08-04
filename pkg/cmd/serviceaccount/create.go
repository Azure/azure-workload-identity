package serviceaccount

import (
	"context"
	"errors"
	"fmt"
	"strings"

	fic "github.com/Azure/azure-workload-identity/pkg/cloud/federatedcredentials"
	"github.com/Azure/azure-workload-identity/pkg/cloud/graph"
	"github.com/Azure/azure-workload-identity/pkg/cloud/roleassignments"
	"github.com/Azure/azure-workload-identity/pkg/kuberneteshelper"
	"github.com/Azure/azure-workload-identity/pkg/version"
	"github.com/Azure/azure-workload-identity/pkg/webhook"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
)

type createCmd struct {
	authProvider

	name       string
	namespace  string
	issuer     string
	azureRole  string
	azureScope string

	graphClient                graph.Interface
	federatedCredentialsClient fic.Interface
	kubeClient                 kubernetes.Interface
	roleAssignmentsClient      roleassignments.Interface
}

func newCreateCmd() *cobra.Command {
	cc := createCmd{
		authProvider: &authArgs{},
	}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a workload identity",
		Long:  "This command provides the ability to create an app registration, add federated identity credential, create the Kubernetes service account and perform role assignment",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := cc.validate(); err != nil {
				return err
			}
			return cc.getAuthArgs().validate()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			if cc.graphClient, err = graph.NewGraphClient(cc.getAuthArgs().azureClientID, cc.getAuthArgs().azureClientSecret, cc.getAuthArgs().azureTenantID); err != nil {
				return err
			}
			if cc.federatedCredentialsClient, err = fic.NewFederatedCredentialsClient(cc.getAuthArgs().azureClientID, cc.getAuthArgs().azureClientSecret, cc.getAuthArgs().azureTenantID); err != nil {
				return err
			}
			if cc.kubeClient, err = kuberneteshelper.GetKubeClient(); err != nil {
				return err
			}
			if cc.roleAssignmentsClient, err = roleassignments.NewRoleAssignmentsClient(
				cc.getAuthArgs().azureSubscriptionID,
				cc.getAuthArgs().azureClientID,
				cc.getAuthArgs().azureClientSecret,
				cc.getAuthArgs().azureTenantID); err != nil {
				return err
			}
			return cc.run()
		},
	}

	f := cmd.Flags()
	f.StringVarP(&cc.name, "name", "", "", "Name of the service account")
	f.StringVarP(&cc.namespace, "namespace", "", "", "Namespace of the service account")
	f.StringVarP(&cc.issuer, "issuer", "", "", "OpenID Connect (OIDC) issuer URL")
	f.StringVarP(&cc.azureRole, "azure-role", "", "", "Azure Role")
	f.StringVarP(&cc.azureScope, "azure-scope", "", "", "Azure Scope")

	addAuthFlags(cc.getAuthArgs(), f)

	return cmd
}

func (cc *createCmd) validate() error {
	if cc.name == "" {
		return errors.New("--name must be specified")
	}

	if cc.namespace == "" {
		return errors.New("--namespace must be specified")
	}

	if cc.issuer == "" {
		return errors.New("--issuer must be specified")
	}

	if cc.azureRole == "" {
		return errors.New("--azure-role must be specified")
	}

	if cc.azureScope == "" {
		return errors.New("--azure-scope must be specified")
	}

	return nil
}

func (cc *createCmd) run() error {
	ctx := context.Background()

	// the name of the app registration is of the format <service account namespace>-<service account name>-<issuer hash>
	refName := fmt.Sprintf("%s-%s-%s", cc.namespace, cc.name, getIssuerHash(cc.issuer))
	tags := []string{
		fmt.Sprintf("serviceAccount: %s-%s", cc.name, cc.namespace),
		fmt.Sprintf("azwi version: %s, commit: %s", version.BuildVersion, version.Vcs),
	}
	// Check if the application with the same name already exists
	app, err := cc.graphClient.GetApplication(ctx, refName, cc.getAuthArgs().azureTenantID)
	if err != nil {
		if !strings.Contains(err.Error(), "not found") {
			return err
		}
		// create the application as it doesn't exist
		app, err = cc.graphClient.CreateApplication(ctx, refName, cc.getAuthArgs().azureTenantID)
		if err != nil {
			return err
		}
		log.Debugf("created app registration with name: '%s', objectID: '%s'", refName, *app.ObjectID)
	}

	// Check if the service principal with the same name already exists
	servicePrincipal, err := cc.graphClient.GetServicePrincipal(ctx, refName, cc.getAuthArgs().azureTenantID)
	if err != nil {
		if !strings.Contains(err.Error(), "not found") {
			return err
		}
		// create the service principal as it doesn't exist
		servicePrincipal, err = cc.graphClient.CreateServicePrincipal(ctx, *app.AppID, cc.getAuthArgs().azureTenantID, tags)
		if err != nil {
			return err
		}
		log.Debugf("created service principal with name: '%s', objectID: '%s'", *servicePrincipal.DisplayName, *servicePrincipal.ObjectID)
	}

	err = kuberneteshelper.CreateServiceAccount(cc.kubeClient, cc.namespace, cc.name, *app.AppID, cc.getAuthArgs().azureTenantID)
	if err != nil {
		return err
	}
	log.Debugf("created kubernetes service account: %s/%s", cc.namespace, cc.name)

	// add the federated credential
	subject := getSubject(cc.namespace, cc.name)
	description := fmt.Sprintf(`Federated Service Account for %s/%s`, cc.namespace, cc.name)
	audiences := []string{webhook.DefaultAudience}

	fc := fic.NewFederatedCredential(*app.ObjectID, cc.issuer, subject, description, audiences)
	err = cc.federatedCredentialsClient.AddFederatedCredential(ctx, *app.ObjectID, fc)
	if err != nil {
		return err
	}

	log.Debugf("added federated credential for %s", subject)

	// create the role assignment using object id of the service principal
	assignmentID, err := cc.roleAssignmentsClient.Create(context.Background(), cc.azureScope, cc.azureRole, *servicePrincipal.ObjectID)
	if err != nil {
		return err
	}
	log.Debugf("created role assignment with id: '%s'", assignmentID)

	return nil
}
