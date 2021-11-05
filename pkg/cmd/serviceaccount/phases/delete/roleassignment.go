package phases

import (
	"context"

	"github.com/Azure/azure-workload-identity/pkg/cloud"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/options"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/phases/workflow"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	roleAssignmentPhaseName = "role-assignment"
)

type roleAssignmentPhase struct {
}

// NewRoleAssignmentPhase creates a new phase to delete role assignment
func NewRoleAssignmentPhase() workflow.Phase {
	p := &roleAssignmentPhase{}
	return workflow.Phase{
		Name:        roleAssignmentPhaseName,
		Aliases:     []string{"ra"},
		Description: "Delete the role assignment between the AAD application and the Azure cloud resource",
		PreRun:      p.prerun,
		Run:         p.run,
		Flags:       []string{options.RoleAssignmentID},
	}
}

func (p *roleAssignmentPhase) prerun(data workflow.RunData) error {
	deleteData, ok := data.(DeleteData)
	if !ok {
		return errors.Errorf("invalid data type %T", data)
	}

	if deleteData.RoleAssignmentID() == "" {
		return options.FlagIsRequiredError(options.RoleAssignmentID)
	}

	return nil
}

func (p *roleAssignmentPhase) run(ctx context.Context, data workflow.RunData) error {
	deleteData := data.(DeleteData)

	// TODO(aramase): consider supporting deletion of role assignment with scope, role and application id
	// delete the role assignment
	l := log.WithField("roleAssignmentID", deleteData.RoleAssignmentID())
	if _, err := deleteData.AzureClient().DeleteRoleAssignment(ctx, deleteData.RoleAssignmentID()); err != nil {
		if !cloud.IsRoleAssignmentAlreadyDeleted(err) {
			return errors.Wrap(err, "failed to delete role assignment")
		}
		l.Warnf("[%s] role assignment not found", roleAssignmentPhaseName)
	} else {
		l.Infof("[%s] deleted role assignment", roleAssignmentPhaseName)
	}

	return nil
}
