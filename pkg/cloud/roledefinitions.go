package cloud

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-01-01-preview/authorization"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// GetRoleDefinitionIDByName returns the role definition ID for the given role name.
func (c *AzureClient) GetRoleDefinitionIDByName(ctx context.Context, scope, roleName string) (authorization.RoleDefinition, error) {
	log.Debugf("Get role definition ID by name=%s", roleName)

	roleDefinitionList, err := c.roleDefinitionsClient.List(ctx, scope, getRoleNameFilter(roleName))
	if err != nil {
		return authorization.RoleDefinition{}, errors.Wrap(err, "failed to list role definitions")
	}
	if len(roleDefinitionList.Values()) == 0 {
		return authorization.RoleDefinition{}, errors.Errorf("role definition %s not found", roleName)
	}

	return roleDefinitionList.Values()[0], nil
}

// getRoleNameFilter returns a filter string for the given role name.
// Supported filters are either roleName eq '{value}' or type eq 'BuiltInRole|CustomRole'."
func getRoleNameFilter(roleName string) string {
	return fmt.Sprintf("roleName eq '%s'", roleName)
}
