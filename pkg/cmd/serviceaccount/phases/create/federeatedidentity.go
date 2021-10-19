package phases

import (
	"context"
	"fmt"

	"github.com/Azure/azure-workload-identity/pkg/cloud"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/phases/workflow"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/util"
	"github.com/Azure/azure-workload-identity/pkg/webhook"

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
		Description: "Create a federated identity between the AAD application and the Kubernetes service account",
		PreRun:      p.prerun,
		Run:         p.run,
	}
}

func (p *federatedIdentityPhase) prerun(data workflow.RunData) error {
	createData, ok := data.(CreateData)
	if !ok {
		return errors.Errorf("invalid data type %T", data)
	}

	if createData.ServiceAccountNamespace() == "" {
		return errors.New("--service-account-namespace is required")
	}
	if createData.ServiceAccountName() == "" {
		return errors.New("--service-account-name is required")
	}
	if createData.ServiceAccountIssuerURL() == "" {
		return errors.New("--service-account-issuer-url is required")
	}
	if createData.AADApplicationName() == "" && createData.AADApplicationObjectID() == "" {
		return errors.New("--aad-application-name or --aad-application-object-id is required")
	}

	return nil
}

func (p *federatedIdentityPhase) run(ctx context.Context, data workflow.RunData) error {
	createData := data.(CreateData)

	serviceAccountNamespace, serviceAccountName := createData.ServiceAccountNamespace(), createData.ServiceAccountName()
	subject := util.GetFederatedCredentialSubject(serviceAccountNamespace, serviceAccountName)
	description := fmt.Sprintf("Federated Service Account for %s/%s", serviceAccountNamespace, serviceAccountName)
	audiences := []string{webhook.DefaultAudience}

	objectID := createData.AADApplicationObjectID()
	fc := cloud.NewFederatedCredential(objectID, createData.ServiceAccountIssuerURL(), subject, description, audiences)
	err := createData.AzureClient().AddFederatedCredential(ctx, objectID, fc)
	if err != nil {
		if cloud.IsAlreadyExists(err) {
			log.WithFields(log.Fields{
				"objecID": objectID,
				"subject": subject,
			}).Debugf("[%s] federated credential has been previously established", federatedIdentityPhaseName)
		} else {
			return errors.Wrap(err, "failed to add federated credential")
		}
	}

	log.WithFields(log.Fields{
		"objecID": objectID,
		"subject": subject,
	}).Infof("[%s] added federated credential", federatedIdentityPhaseName)

	return nil
}
