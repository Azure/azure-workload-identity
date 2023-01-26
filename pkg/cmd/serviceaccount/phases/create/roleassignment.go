package phases

import (
	"context"

	"github.com/pkg/errors"
	"monis.app/mlog"

	"github.com/Azure/azure-workload-identity/pkg/cloud"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/options"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/phases/workflow"
)

const (
	roleAssignmentPhaseName = "role-assignment"
)

type roleAssignmentPhase struct {
}

// NewRoleAssignmentPhase creates a new phase to create role assignment
func NewRoleAssignmentPhase() workflow.Phase {
	p := &roleAssignmentPhase{}
	return workflow.Phase{
		Name:        roleAssignmentPhaseName,
		Aliases:     []string{"ra"},
		Description: "Create role assignment between the AAD application and the Azure cloud resource",
		PreRun:      p.prerun,
		Run:         p.run,
		Flags: []string{
			options.AzureScope.Flag,
			options.AzureRole.Flag,
			options.ServicePrincipalName.Flag,
			options.ServicePrincipalObjectID.Flag,
		},
	}
}

func (p *roleAssignmentPhase) prerun(data workflow.RunData) error {
	createData, ok := data.(CreateData)
	if !ok {
		return errors.Errorf("invalid data type %T", data)
	}

	if createData.AzureScope() == "" {
		return options.FlagIsRequiredError(options.AzureScope.Flag)
	}
	if createData.AzureRole() == "" {
		return options.FlagIsRequiredError(options.AzureRole.Flag)
	}
	if createData.ServicePrincipalName() == "" && createData.ServicePrincipalObjectID() == "" {
		return options.OneOfFlagsIsRequiredError(options.ServicePrincipalName.Flag, options.ServicePrincipalObjectID.Flag)
	}

	return nil
}

func (p *roleAssignmentPhase) run(ctx context.Context, data workflow.RunData) error {
	createData := data.(CreateData)

	// create the role assignment using object id of the service principal
	ra, err := createData.AzureClient().CreateRoleAssignment(ctx, createData.AzureScope(), createData.AzureRole(), createData.ServicePrincipalObjectID())
	if err != nil {
		if cloud.IsAlreadyExists(err) {
			mlog.WithValues(
				"scope", createData.AzureScope(),
				"role", createData.AzureRole(),
				"servicePrincipalObjectID", createData.ServicePrincipalObjectID(),
				"roleAssignmentID", ra.ID,
			).WithName(roleAssignmentPhaseName).Debug("role assignment has previously been created")
		} else {
			return errors.Wrap(err, "failed to create role assignment")
		}
	}

	mlog.WithValues(
		"scope", createData.AzureScope(),
		"role", createData.AzureRole(),
		"servicePrincipalObjectID", createData.ServicePrincipalObjectID(),
		"roleAssignmentID", ra.ID,
	).WithName(roleAssignmentPhaseName).Info("created role assignment")

	return nil
}
