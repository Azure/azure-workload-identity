package cloud

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-01-01-preview/authorization"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	azureBuiltInContributorID    = "b24988ac-6180-42a0-ab88-20f7382dd24c"
	azureBuiltInReaderID         = "acdd72a7-3385-48ef-bd42-f606fba81ae7"
	azureKeyvaultAdministratorID = "00482a5a-887f-4fb3-b363-3b7fe8e74483"
	// TODO (aramase): Dynamically get the role definition id based on the role name.

	roleDefinitionIDFormat = "/subscriptions/%s/providers/Microsoft.Authorization/roleDefinitions/%s"

	roleAssignmentCreateRetryCount = 7
	roleAssignmentCreateRetryDelay = 5 * time.Second
)

// CreateRoleAssignment creates a role assignment.
func (c *AzureClient) CreateRoleAssignment(ctx context.Context, scope, roleName, principalID string) (authorization.RoleAssignment, error) {
	var result authorization.RoleAssignment
	roleDefinitionID, err := getRoleDefinitionID(c.subscriptionID, roleName)
	if err != nil {
		return result, err
	}

	log.Debugf("Creating role assignment for principalID=%s with role=%s", principalID, roleName)
	parameters := authorization.RoleAssignmentCreateParameters{
		RoleAssignmentProperties: &authorization.RoleAssignmentProperties{
			RoleDefinitionID: to.StringPtr(roleDefinitionID),
			PrincipalID:      to.StringPtr(principalID),
		},
	}

	// Adding retries to handle the propagation delay of the service principal.
	// Trying to create role assignment immediately after service principal is created
	// results in "PrincipalNotFound" error.
	for i := 0; i < roleAssignmentCreateRetryCount; i++ {
		if result, err = c.authorizationClient.Create(ctx, scope, uuid.New().String(), parameters); err == nil {
			return result, nil
		}
		if IsAlreadyExists(err) {
			log.Warnf("Role assignment already exists for principalID=%s with role=%s", principalID, roleName)
			return result, err
		}
		time.Sleep(roleAssignmentCreateRetryDelay)
	}

	return result, err
}

// DeleteRoleAssignment deletes a role assignment.
func (c *AzureClient) DeleteRoleAssignment(ctx context.Context, roleAssignmentID string) (authorization.RoleAssignment, error) {
	log.Debugf("Deleting role assignment with id=%s", roleAssignmentID)
	return c.authorizationClient.DeleteByID(ctx, roleAssignmentID)
}

// getRoleDefinitionID gets the role definition id for the given role name.
func getRoleDefinitionID(subscriptionID, roleName string) (string, error) {
	switch strings.ToLower(roleName) {
	case "contributor":
		return fmt.Sprintf(roleDefinitionIDFormat, subscriptionID, azureBuiltInContributorID), nil
	case "reader":
		return fmt.Sprintf(roleDefinitionIDFormat, subscriptionID, azureBuiltInReaderID), nil
	case "administrator":
		return fmt.Sprintf(roleDefinitionIDFormat, subscriptionID, azureKeyvaultAdministratorID), nil
	default:
		return "", errors.Errorf("role %s is not supported", roleName)
	}
}
