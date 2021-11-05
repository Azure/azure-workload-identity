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
			options.AzureScope,
			options.AzureRole,
			options.ServicePrincipalName,
			options.ServicePrincipalObjectID,
		},
	}
}

func (p *roleAssignmentPhase) prerun(data workflow.RunData) error {
	createData, ok := data.(CreateData)
	if !ok {
		return errors.Errorf("invalid data type %T", data)
	}

	if createData.AzureScope() == "" {
		return options.FlagIsRequiredError(options.AzureScope)
	}
	if createData.AzureRole() == "" {
		return options.FlagIsRequiredError(options.AzureRole)
	}
	if createData.ServicePrincipalName() == "" && createData.ServicePrincipalObjectID() == "" {
		return options.OneOfFlagsIsRequiredError(options.ServicePrincipalName, options.ServicePrincipalObjectID)
	}

	return nil
}

func (p *roleAssignmentPhase) run(ctx context.Context, data workflow.RunData) error {
	createData := data.(CreateData)

	// create the role assignment using object id of the service principal
	ra, err := createData.AzureClient().CreateRoleAssignment(ctx, createData.AzureScope(), createData.AzureRole(), createData.ServicePrincipalObjectID())
	if err != nil {
		if cloud.IsAlreadyExists(err) {
			log.WithFields(log.Fields{
				"scope":                    createData.AzureScope(),
				"role":                     createData.AzureRole(),
				"servicePrincipalObjectID": createData.ServicePrincipalObjectID(),
				"roleAssignmentID":         ra.ID,
			}).Debugf("[%s] role assignment has previously been created", roleAssignmentPhaseName)
		} else {
			return errors.Wrap(err, "failed to create role assignment")
		}
	}

	log.WithFields(log.Fields{
		"scope":                    createData.AzureScope(),
		"role":                     createData.AzureRole(),
		"servicePrincipalObjectID": createData.ServicePrincipalObjectID(),
		"roleAssignmentID":         ra.ID,
	}).Infof("[%s] created role assignment", roleAssignmentPhaseName)

	return nil
}
