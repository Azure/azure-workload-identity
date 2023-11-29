package cloud

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization"
	"github.com/pkg/errors"
	"monis.app/mlog"
)

// GetRoleDefinitionIDByName returns the role definition ID for the given role name.
func (c *AzureClient) GetRoleDefinitionIDByName(ctx context.Context, scope, roleName string) (armauthorization.RoleDefinition, error) {
	mlog.Debug("Get role definition ID", "name", roleName)

	filter := getRoleNameFilter(roleName)
	pager := c.roleDefinitionsClient.NewListPager(scope, &armauthorization.RoleDefinitionsClientListOptions{
		Filter: &filter,
	})

	for pager.More() {
		nextResult, err := pager.NextPage(ctx)
		if err != nil {
			return armauthorization.RoleDefinition{}, errors.Wrap(err, "failed to list role definitions")
		}
		if len(nextResult.Value) > 0 {
			return *nextResult.Value[0], nil
		}
	}

	return armauthorization.RoleDefinition{}, errors.Errorf("role definition %s not found", roleName)
}

// getRoleNameFilter returns a filter string for the given role name.
// Supported filters are either roleName eq '{value}' or type eq 'BuiltInRole|CustomRole'."
func getRoleNameFilter(roleName string) string {
	return fmt.Sprintf("roleName eq '%s'", roleName)
}
