package phases

import (
	"context"

	"github.com/Azure/azure-workload-identity/pkg/cloud"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/phases/workflow"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/util"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	federatedIdentityPhaseName = "federated-identity"
)

type federatedIdentityPhase struct {
}

// NewFederatedIdentityPhase creates a new phase to create federated identity.
func NewFederatedIdentityPhase() workflow.Phase {
	p := &federatedIdentityPhase{}
	return workflow.Phase{
		Name:        federatedIdentityPhaseName,
		Description: "Delete federated identity between the AAD application and the Kubernetes service account",
		PreRun:      p.prerun,
		Run:         p.run,
	}
}

func (p *federatedIdentityPhase) prerun(data workflow.RunData) error {
	deleteData, ok := data.(DeleteData)
	if !ok {
		return errors.Errorf("invalid data type %T", data)
	}

	if deleteData.ServiceAccountNamespace() == "" {
		return errors.New("--service-account-namespace is required")
	}
	if deleteData.ServiceAccountName() == "" {
		return errors.New("--service-account-name is required")
	}
	if deleteData.ServiceAccountIssuerURL() == "" {
		return errors.New("--service-account-issuer-url is required")
	}

	return nil
}

func (p *federatedIdentityPhase) run(ctx context.Context, data workflow.RunData) error {
	deleteData := data.(DeleteData)

	subject := util.GetFederatedCredentialSubject(deleteData.ServiceAccountNamespace(), deleteData.ServiceAccountName())
	l := log.WithFields(log.Fields{
		"subject":   subject,
		"issuerURL": deleteData.ServiceAccountIssuerURL(),
	})
	if fc, err := deleteData.AzureClient().GetFederatedCredential(ctx, deleteData.AADApplicationObjectID(), deleteData.ServiceAccountIssuerURL(), subject); err != nil {
		if !cloud.IsResourceNotFound(err) {
			return errors.Wrap(err, "failed to get federated identity")
		}
		l.Warnf("[%s] federated identity not found", federatedIdentityPhaseName)
	} else {
		if err = deleteData.AzureClient().DeleteFederatedCredential(ctx, deleteData.AADApplicationObjectID(), fc.ID); err != nil {
			return errors.Wrap(err, "failed to delete federated identity")
		}
		l.Infof("[%s] deleted federated identity", federatedIdentityPhaseName)
	}

	return nil
}
