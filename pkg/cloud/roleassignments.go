package cloud

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"monis.app/mlog"
)

const (
	roleAssignmentCreateRetryCount = 7
	roleAssignmentCreateRetryDelay = 5 * time.Second
)

// CreateRoleAssignment creates a role assignment.
func (c *AzureClient) CreateRoleAssignment(ctx context.Context, scope, roleName, principalID string) (armauthorization.RoleAssignment, error) {
	var result armauthorization.RoleAssignment

	roleDefinitionID, err := c.GetRoleDefinitionIDByName(ctx, "", roleName)
	if err != nil {
		return result, errors.Wrapf(err, "failed to get role definition id for role %s", roleName)
	}

	mlog.Debug("Creating role assignment",
		"principalID", principalID,
		"role", roleName,
	)

	parameters := armauthorization.RoleAssignmentCreateParameters{
		Properties: &armauthorization.RoleAssignmentProperties{
			RoleDefinitionID: roleDefinitionID.ID,
			PrincipalID:      to.Ptr(principalID),
		},
	}

	// Adding retries to handle the propagation delay of the service principal.
	// Trying to create role assignment immediately after service principal is created
	// results in "PrincipalNotFound" error.
	for i := 0; i < roleAssignmentCreateRetryCount; i++ {
		resp, err := c.roleAssignmentsClient.Create(ctx, scope, uuid.New().String(), parameters, nil)
		if err == nil {
			return resp.RoleAssignment, nil
		}

		if IsRoleAssignmentExists(err) {
			mlog.Warning("Role assignment already exists", "principalID", principalID, "role", roleName)
			return result, err
		}
		time.Sleep(roleAssignmentCreateRetryDelay)
	}

	return result, err
}

// DeleteRoleAssignment deletes a role assignment.
func (c *AzureClient) DeleteRoleAssignment(ctx context.Context, roleAssignmentID string) (armauthorization.RoleAssignment, error) {
	mlog.Debug("Deleting role assignment", "id", roleAssignmentID)
	resp, err := c.roleAssignmentsClient.DeleteByID(ctx, roleAssignmentID, nil)
	if err != nil {
		return armauthorization.RoleAssignment{}, err
	}
	return resp.RoleAssignment, nil
}
