package phases

import (
	"context"

	"github.com/pkg/errors"
	"monis.app/mlog"

	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/options"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/phases/workflow"
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
		Aliases:     []string{"app"},
		Description: "Delete the Azure Active Directory (AAD) application and its underlying service principal",
		PreRun:      p.prerun,
		Run:         p.run,
		Flags: []string{
			options.AADApplicationName.Flag,
			options.AADApplicationObjectID.Flag,
		},
	}
}

func (p *aadApplicationPhase) prerun(data workflow.RunData) error {
	deleteData, ok := data.(DeleteData)
	if !ok {
		return errors.Errorf("invalid data type %T", data)
	}

	if deleteData.AADApplicationName() == "" && deleteData.AADApplicationObjectID() == "" {
		return options.OneOfFlagsIsRequiredError(options.AADApplicationName.Flag, options.AADApplicationObjectID.Flag)
	}

	return nil
}

func (p *aadApplicationPhase) run(ctx context.Context, data workflow.RunData) error {
	deleteData := data.(DeleteData)

	l := mlog.WithValues(
		"name", deleteData.AADApplicationName(),
		"objectID", deleteData.AADApplicationObjectID(),
	).WithName(aadApplicationPhaseName)
	if err := deleteData.AzureClient().DeleteApplication(ctx, deleteData.AADApplicationObjectID()); err != nil {
		return errors.Wrap(err, "failed to delete application")
	}
	l.Info("deleted aad application")

	return nil
}
