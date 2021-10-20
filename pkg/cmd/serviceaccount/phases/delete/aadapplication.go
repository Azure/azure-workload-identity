package phases

import (
	"context"

	"github.com/Azure/azure-workload-identity/pkg/cloud"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/phases/workflow"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	aadApplicationPhaseName = "aad-application"
)

type aadApplicationPhase struct {
}

// NewAADApplicationPhase creates a new phase to delete an AAD application
func NewAADApplicationPhase() workflow.Phase {
	p := &aadApplicationPhase{}
	return workflow.Phase{
		Name:        aadApplicationPhaseName,
		Description: "Delete the Azure Active Directory (AAD) application and its underlying service principal",
		PreRun:      p.prerun,
		Run:         p.run,
	}
}

func (p *aadApplicationPhase) prerun(data workflow.RunData) error {
	deleteData, ok := data.(DeleteData)
	if !ok {
		return errors.Errorf("invalid data type %T", data)
	}

	if deleteData.AADApplicationName() == "" && deleteData.AADApplicationObjectID() == "" {
		return errors.New("--aad-application-name or --aad-application-object-id is required")
	}

	return nil
}

func (p *aadApplicationPhase) run(ctx context.Context, data workflow.RunData) error {
	deleteData := data.(DeleteData)

	l := log.WithField("objectID", deleteData.AADApplicationObjectID())
	if _, err := deleteData.AzureClient().DeleteApplication(ctx, deleteData.AADApplicationObjectID()); err != nil {
		if !cloud.IsResourceNotFound(err) {
			return errors.Wrap(err, "failed to delete application")
		}
		l.Warnf("[%s] aad application not found", aadApplicationPhaseName)
	} else {
		l.Infof("[%s] deleted aad application", aadApplicationPhaseName)
	}

	return nil
}
