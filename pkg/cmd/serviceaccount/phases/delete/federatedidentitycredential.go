package phases

import (
	"context"

	"github.com/pkg/errors"
	"monis.app/mlog"

	"github.com/Azure/azure-workload-identity/pkg/cloud"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/options"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/phases/workflow"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/util"
)

const (
	federatedIdentityPhaseName = "federated-identity"
)

type federatedIdentityPhase struct {
}

// NewFederatedIdentityPhase creates a new phase to delete federated identity.
func NewFederatedIdentityPhase() workflow.Phase {
	p := &federatedIdentityPhase{}
	return workflow.Phase{
		Name:        federatedIdentityPhaseName,
		Aliases:     []string{"fi"},
		Description: "Delete federated identity credential for the AAD application and the Kubernetes service account",
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
	deleteData, ok := data.(DeleteData)
	if !ok {
		return errors.Errorf("invalid data type %T", data)
	}

	if deleteData.ServiceAccountNamespace() == "" {
		return options.FlagIsRequiredError(options.ServiceAccountNamespace.Flag)
	}
	if deleteData.ServiceAccountName() == "" {
		return options.FlagIsRequiredError(options.ServiceAccountName.Flag)
	}
	if deleteData.ServiceAccountIssuerURL() == "" {
		return options.FlagIsRequiredError(options.ServiceAccountIssuerURL.Flag)
	}

	return nil
}

func (p *federatedIdentityPhase) run(ctx context.Context, data workflow.RunData) error {
	deleteData := data.(DeleteData)

	subject := util.GetFederatedCredentialSubject(deleteData.ServiceAccountNamespace(), deleteData.ServiceAccountName())
	l := mlog.WithValues(
		"subject", subject,
		"issuerURL", deleteData.ServiceAccountIssuerURL(),
	).WithName(federatedIdentityPhaseName)
	if fic, err := deleteData.AzureClient().GetFederatedCredential(ctx, deleteData.AADApplicationObjectID(), deleteData.ServiceAccountIssuerURL(), subject); err != nil {
		if !cloud.IsFederatedCredentialNotFound(err) {
			return errors.Wrap(err, "failed to get federated identity credential")
		}
		l.Warning("federated identity credential not found")
	} else {
		if err = deleteData.AzureClient().DeleteFederatedCredential(ctx, deleteData.AADApplicationObjectID(), *fic.GetId()); err != nil {
			return errors.Wrap(err, "failed to delete federated identity credential")
		}
		l.Info("deleted federated identity credential")
	}

	return nil
}
