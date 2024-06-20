package cloud

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/microsoftgraph/msgraph-sdk-go/applications"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/microsoftgraph/msgraph-sdk-go/serviceprincipals"
	"github.com/pkg/errors"
	"monis.app/mlog"
)

var (
	// ErrFederatedCredentialNotFound is returned when the federated credential is not found.
	ErrFederatedCredentialNotFound = errors.New("federated credential not found")
)

// CreateServicePrincipal creates a service principal for the given application.
// No secret or certificate is generated.
func (c *AzureClient) CreateServicePrincipal(ctx context.Context, appID string, tags []string) (models.ServicePrincipalable, error) {
	body := models.NewServicePrincipal()
	body.SetAppId(to.Ptr(appID))
	body.SetTags(tags)

	mlog.Debug("Creating service principal for application", "id", appID)
	sp, err := c.graphServiceClient.ServicePrincipals().Post(ctx, body, nil)
	if err != nil {
		return nil, maybeExtractGraphError(err)
	}

	return sp, nil
}

// CreateApplication creates an application.
func (c *AzureClient) CreateApplication(ctx context.Context, displayName string) (models.Applicationable, error) {
	body := models.NewApplication()
	body.SetDisplayName(to.Ptr(displayName))

	mlog.Debug("Creating application", "displayName", displayName)
	app, err := c.graphServiceClient.Applications().Post(ctx, body, nil)
	if err != nil {
		return nil, maybeExtractGraphError(err)
	}

	return app, nil
}

// GetServicePrincipal gets a service principal by its display name.
func (c *AzureClient) GetServicePrincipal(ctx context.Context, displayName string) (models.ServicePrincipalable, error) {
	mlog.Debug("Getting service principal", "displayName", displayName)

	spGetOptions := &serviceprincipals.ServicePrincipalsRequestBuilderGetRequestConfiguration{
		QueryParameters: &serviceprincipals.ServicePrincipalsRequestBuilderGetQueryParameters{
			Filter: to.Ptr(getDisplayNameFilter(displayName)),
		},
	}

	resp, err := c.graphServiceClient.ServicePrincipals().Get(ctx, spGetOptions)
	if err != nil {
		return nil, maybeExtractGraphError(err)
	}

	if len(resp.GetValue()) == 0 {
		return nil, errors.Errorf("service principal %s not found", displayName)
	}
	return resp.GetValue()[0], nil
}

// GetApplication gets an application by its display name.
func (c *AzureClient) GetApplication(ctx context.Context, displayName string) (models.Applicationable, error) {
	mlog.Debug("Getting application", "displayName", displayName)

	appGetOptions := &applications.ApplicationsRequestBuilderGetRequestConfiguration{
		QueryParameters: &applications.ApplicationsRequestBuilderGetQueryParameters{
			Filter: to.Ptr(getDisplayNameFilter(displayName)),
		},
	}

	resp, err := c.graphServiceClient.Applications().Get(ctx, appGetOptions)
	if err != nil {
		return nil, maybeExtractGraphError(err)
	}

	if len(resp.GetValue()) == 0 {
		return nil, errors.Errorf("application with display name '%s' not found", displayName)
	}
	return resp.GetValue()[0], nil
}

// DeleteServicePrincipal deletes a service principal.
func (c *AzureClient) DeleteServicePrincipal(ctx context.Context, objectID string) error {
	mlog.Debug("Deleting service principal", "objectID", objectID)
	return c.graphServiceClient.ServicePrincipals().ByServicePrincipalId(objectID).Delete(ctx, nil)
}

// DeleteApplication deletes an application.
func (c *AzureClient) DeleteApplication(ctx context.Context, objectID string) error {
	mlog.Debug("Deleting application", "objectID", objectID)
	return c.graphServiceClient.Applications().ByApplicationId(objectID).Delete(ctx, nil)
}

// AddFederatedCredential adds a federated credential to the cloud provider.
func (c *AzureClient) AddFederatedCredential(ctx context.Context, objectID string, fic models.FederatedIdentityCredentialable) error {
	mlog.Debug("Adding federated credential", "objectID", objectID)

	if _, err := c.graphServiceClient.Applications().ByApplicationId(objectID).FederatedIdentityCredentials().Post(ctx, fic, nil); err != nil {
		return maybeExtractGraphError(err)
	}

	return nil
}

// GetFederatedCredential gets a federated credential from the cloud provider.
func (c *AzureClient) GetFederatedCredential(ctx context.Context, objectID, issuer, subject string) (models.FederatedIdentityCredentialable, error) {
	mlog.Debug("Getting federated credential",
		"objectID", objectID,
		"issuer", issuer,
		"subject", subject,
	)

	ficGetOptions := &applications.ItemFederatedidentitycredentialsFederatedIdentityCredentialsRequestBuilderGetRequestConfiguration{
		QueryParameters: &applications.ItemFederatedidentitycredentialsFederatedIdentityCredentialsRequestBuilderGetQueryParameters{
			// Filtering on more than one resource is currently not supported.
			Filter: to.Ptr(getSubjectFilter(subject)),
		},
	}

	resp, err := c.graphServiceClient.Applications().ByApplicationId(objectID).FederatedIdentityCredentials().Get(ctx, ficGetOptions)
	if err != nil {
		return nil, maybeExtractGraphError(err)
	}

	for _, fic := range resp.GetValue() {
		if *fic.GetIssuer() == issuer {
			return fic, nil
		}
	}
	return nil, ErrFederatedCredentialNotFound
}

// DeleteFederatedCredential deletes a federated credential from the cloud provider.
func (c *AzureClient) DeleteFederatedCredential(ctx context.Context, objectID, federatedCredentialID string) error {
	mlog.Debug("Deleting federated credential",
		"objectID", objectID,
		"federatedCredentialID", federatedCredentialID,
	)
	return c.graphServiceClient.Applications().ByApplicationId(objectID).FederatedIdentityCredentials().ByFederatedIdentityCredentialId(federatedCredentialID).Delete(ctx, nil)
}

// getDisplayNameFilter returns a filter string for the given display name.
func getDisplayNameFilter(displayName string) string {
	return fmt.Sprintf("displayName eq '%s'", displayName)
}

// getSubjectFilter returns a filter string for the given subject.
func getSubjectFilter(subject string) string {
	return fmt.Sprintf("subject eq '%s'", subject)
}
