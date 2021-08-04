package roleassignments

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/authorization/mgmt/authorization"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

const (
	azureBuiltInContributorID = "b24988ac-6180-42a0-ab88-20f7382dd24c"
	azureBuiltInReaderID      = "acdd72a7-3385-48ef-bd42-f606fba81ae7"
	// TODO (aramase): Dynamically get the role definition id based on the role name.

	roleDefinitionIDFormat = "/subscriptions/%s/providers/Microsoft.Authorization/roleDefinitions/%s"
)

// Interface is the interface for role assignment operations.
type Interface interface {
	Create(ctx context.Context, scope, roleName, principalID string) (string, error)
	Delete(ctx context.Context, roleAssignmentID string) error
}

type client struct {
	authorization.RoleAssignmentsClient
}

var _ Interface = &client{}

// NewRoleAssignmentClient creates an instance of the RoleAssignmentClient client.
func NewRoleAssignmentsClient(subscriptionID, clientID, clientSecret, tenantID string) (Interface, error) {
	roleAssignmentsClient := authorization.NewRoleAssignmentsClient(subscriptionID)
	cfg := auth.NewClientCredentialsConfig(clientID, clientSecret, tenantID)
	authorizer, err := cfg.Authorizer()
	if err != nil {
		return nil, err
	}
	roleAssignmentsClient.Authorizer = authorizer
	return &client{roleAssignmentsClient}, nil
}

// Create creates a role assignment.
func (c *client) Create(ctx context.Context, scope, roleName, principalID string) (string, error) {
	roleDefinitionID, err := getRoleDefinitionID(c.SubscriptionID, roleName)
	if err != nil {
		return "", err
	}

	log.Debugf("Creating role assignment for %s with role %s\n", principalID, roleName)
	log.Debugf("Role definition id: %s\n", roleDefinitionID)

	parameters := authorization.RoleAssignmentCreateParameters{
		Properties: &authorization.RoleAssignmentProperties{
			RoleDefinitionID: to.StringPtr(roleDefinitionID),
			PrincipalID:      to.StringPtr(principalID),
		},
	}

	// Adding retries to handle the propagation delay of the service principal.
	// Trying to create role assignment immediately after service principal is created
	// results in "PrincipalNotFound" error.
	for i := 0; i < 7; i++ {
		result, err := c.RoleAssignmentsClient.Create(ctx, scope, uuid.NewV1().String(), parameters)
		if err == nil {
			return *result.ID, nil
		}
		log.Debugf("Error creating role assignment: %v\n", err)
		time.Sleep(5 * time.Second)
	}

	return "", errors.Wrap(err, "failed to create role assignment")
}

// Delete deletes a role assignment.
func (c *client) Delete(ctx context.Context, roleAssignmentID string) error {
	log.Debugf("Deleting role assignment %s\n", roleAssignmentID)
	_, err := c.RoleAssignmentsClient.DeleteByID(ctx, roleAssignmentID)
	return err
}

// getRoleDefinitionID gets the role definition id for the given role name.
func getRoleDefinitionID(subscriptionID, roleName string) (string, error) {
	switch strings.ToLower(roleName) {
	case "contributor":
		return fmt.Sprintf(roleDefinitionIDFormat, subscriptionID, azureBuiltInContributorID), nil
	case "reader":
		return fmt.Sprintf(roleDefinitionIDFormat, subscriptionID, azureBuiltInReaderID), nil
	default:
		return "", fmt.Errorf("role %s is not supported", roleName)
	}
}
