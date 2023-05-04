package phases

import (
	"context"
	"fmt"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/pkg/errors"
	"monis.app/mlog"

	"github.com/Azure/azure-workload-identity/pkg/cloud"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/options"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/phases/workflow"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/util"
	"github.com/Azure/azure-workload-identity/pkg/webhook"
)

const (
	federatedIdentityPhaseName = "federated-identity"
)

type federatedIdentityPhase struct {
}

// NewFederatedIdentityPhase creates a new phase to create federated identity credential.
func NewFederatedIdentityPhase() workflow.Phase {
	p := &federatedIdentityPhase{}
	return workflow.Phase{
		Name:        federatedIdentityPhaseName,
		Aliases:     []string{"fi"},
		Description: "Create federated identity credential between the AAD application and the Kubernetes service account",
		PreRun:      p.prerun,
		Run:         p.run,
		Flags: []string{
			options.ServiceAccountNamespace.Flag,
			options.ServiceAccountName.Flag,
			options.ServiceAccountIssuerURL.Flag,
			options.AADApplicationName.Flag,
			options.AADApplicationObjectID.Flag,
		},
	}
}

func (p *federatedIdentityPhase) prerun(data workflow.RunData) error {
	createData, ok := data.(CreateData)
	if !ok {
		return errors.Errorf("invalid data type %T", data)
	}

	if createData.ServiceAccountNamespace() == "" {
		return options.FlagIsRequiredError(options.ServiceAccountNamespace.Flag)
	}
	if createData.ServiceAccountName() == "" {
		return options.FlagIsRequiredError(options.ServiceAccountName.Flag)
	}
	if createData.ServiceAccountIssuerURL() == "" {
		return options.FlagIsRequiredError(options.ServiceAccountIssuerURL.Flag)
	}

	return nil
}

func (p *federatedIdentityPhase) run(ctx context.Context, data workflow.RunData) error {
	createData := data.(CreateData)

	serviceAccountNamespace, serviceAccountName := createData.ServiceAccountNamespace(), createData.ServiceAccountName()
	subject := util.GetFederatedCredentialSubject(serviceAccountNamespace, serviceAccountName)
	name := util.GetFederatedCredentialName(serviceAccountNamespace, serviceAccountName, createData.ServiceAccountIssuerURL())
	description := fmt.Sprintf("Federated Service Account for %s/%s", serviceAccountNamespace, serviceAccountName)
	audiences := []string{webhook.DefaultAudience}

	objectID := createData.AADApplicationObjectID()
	fic := models.NewFederatedIdentityCredential()
	fic.SetAudiences(audiences)
	fic.SetDescription(to.StringPtr(description))
	fic.SetIssuer(to.StringPtr(createData.ServiceAccountIssuerURL()))
	fic.SetSubject(to.StringPtr(subject))
	fic.SetName(to.StringPtr(name))

	err := createData.AzureClient().AddFederatedCredential(ctx, objectID, fic)
	if err != nil {
		if cloud.IsFederatedCredentialAlreadyExists(err) {
			mlog.WithValues(
				"objectID", objectID,
				"subject", subject,
			).WithName(federatedIdentityPhaseName).Warning("federated credential has been previously created")
		} else {
			return errors.Wrap(err, "failed to add federated credential")
		}
	}

	mlog.WithValues(
		"objectID", objectID,
		"subject", subject,
	).WithName(federatedIdentityPhaseName).Info("added federated credential")

	return nil
}
