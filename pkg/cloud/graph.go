package cloud

import (
	"context"
	"fmt"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/microsoftgraph/msgraph-beta-sdk-go/applications"
	"github.com/microsoftgraph/msgraph-beta-sdk-go/applications/item/federatedidentitycredentials"
	"github.com/microsoftgraph/msgraph-beta-sdk-go/models"
	"github.com/microsoftgraph/msgraph-beta-sdk-go/serviceprincipals"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var (
	// ErrFederatedCredentialNotFound is returned when the federated credential is not found.
	ErrFederatedCredentialNotFound = errors.New("federated credential not found")
)

// CreateServicePrincipal creates a service principal for the given application.
// No secret or certificate is generated.
func (c *AzureClient) CreateServicePrincipal(ctx context.Context, appID string, tags []string) (models.ServicePrincipalable, error) {
	spPostOptions := &serviceprincipals.ServicePrincipalsRequestBuilderPostOptions{
		Body: models.NewServicePrincipal(),
	}
	spPostOptions.Body.SetAppId(to.StringPtr(appID))
	spPostOptions.Body.SetTags(tags)

	log.Debugf("Creating service principal for application with id=%s", appID)
	sp, err := c.graphServiceClient.ServicePrincipals().Post(spPostOptions)
	if err != nil {
		return nil, GetGraphError(err)
	}
	return sp, nil
}

// CreateApplication creates an application.
func (c *AzureClient) CreateApplication(ctx context.Context, displayName string) (models.Applicationable, error) {
	appPostOptions := &applications.ApplicationsRequestBuilderPostOptions{
		Body: models.NewApplication(),
	}
	appPostOptions.Body.SetDisplayName(to.StringPtr(displayName))

	log.Debugf("Creating application with display name=%s", displayName)
	app, err := c.graphServiceClient.Applications().Post(appPostOptions)
	if err != nil {
		return nil, GetGraphError(err)
	}
	return app, nil
}

// GetServicePrincipal gets a service principal by its display name.
func (c *AzureClient) GetServicePrincipal(ctx context.Context, displayName string) (models.ServicePrincipalable, error) {
	log.Debugf("Getting service principal with display name=%s", displayName)

	spGetOptions := &serviceprincipals.ServicePrincipalsRequestBuilderGetOptions{
		QueryParameters: &serviceprincipals.ServicePrincipalsRequestBuilderGetQueryParameters{
			Filter: to.StringPtr(getDisplayNameFilter(displayName)),
		},
	}

	resp, err := c.graphServiceClient.ServicePrincipals().Get(spGetOptions)
	if err != nil {
		return nil, GetGraphError(err)
	}
	if len(resp.GetValue()) == 0 {
		return nil, errors.Errorf("service principal %s not found", displayName)
	}
	return resp.GetValue()[0], nil
}

// GetApplication gets an application by its display name.
func (c *AzureClient) GetApplication(ctx context.Context, displayName string) (models.Applicationable, error) {
	log.Debugf("Getting application with display name=%s", displayName)

	appGetOptions := &applications.ApplicationsRequestBuilderGetOptions{
		QueryParameters: &applications.ApplicationsRequestBuilderGetQueryParameters{
			Filter: to.StringPtr(getDisplayNameFilter(displayName)),
		},
	}

	resp, err := c.graphServiceClient.Applications().Get(appGetOptions)
	if err != nil {
		return nil, GetGraphError(err)
	}
	if len(resp.GetValue()) == 0 {
		return nil, errors.Errorf("application with display name '%s' not found", displayName)
	}
	return resp.GetValue()[0], nil
}

// DeleteServicePrincipal deletes a service principal.
func (c *AzureClient) DeleteServicePrincipal(ctx context.Context, objectID string) error {
	log.Debugf("Deleting service principal with object id=%s", objectID)
	return c.graphServiceClient.ServicePrincipalsById(objectID).Delete(nil)
}

// DeleteApplication deletes an application.
func (c *AzureClient) DeleteApplication(ctx context.Context, objectID string) error {
	log.Debugf("Deleting application with object id=%s", objectID)
	return c.graphServiceClient.ApplicationsById(objectID).Delete(nil)
}

// AddFederatedCredential adds a federated credential to the cloud provider.
func (c *AzureClient) AddFederatedCredential(ctx context.Context, objectID string, fic models.FederatedIdentityCredentialable) error {
	log.Debugf("Adding federated credential for objectID=%s", objectID)

	ficPostOptions := &federatedidentitycredentials.FederatedIdentityCredentialsRequestBuilderPostOptions{
		Body: fic,
	}
	fic, err := c.graphServiceClient.ApplicationsById(objectID).FederatedIdentityCredentials().Post(ficPostOptions)
	if err != nil {
		return GetGraphError(err)
	}
	return nil
}

// GetFederatedCredential gets a federated credential from the cloud provider.
func (c *AzureClient) GetFederatedCredential(ctx context.Context, objectID, issuer, subject string) (models.FederatedIdentityCredentialable, error) {
	log.Debugf("Getting federated credential for objectID=%s, issuer=%s, subject=%s", objectID, issuer, subject)

	ficGetOptions := &federatedidentitycredentials.FederatedIdentityCredentialsRequestBuilderGetOptions{
		QueryParameters: &federatedidentitycredentials.FederatedIdentityCredentialsRequestBuilderGetQueryParameters{
			// Filtering on more than one resource is currently not supported.
			Filter: to.StringPtr(getSubjectFilter(subject)),
		},
	}

	resp, err := c.graphServiceClient.ApplicationsById(objectID).FederatedIdentityCredentials().Get(ficGetOptions)
	if err != nil {
		return nil, GetGraphError(err)
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
	log.Debugf("Deleting federated credential for objectID=%s, federatedCredentialID=%s", objectID, federatedCredentialID)
	return c.graphServiceClient.ApplicationsById(objectID).FederatedIdentityCredentialsById(federatedCredentialID).Delete(nil)
}

// getDisplayNameFilter returns a filter string for the given display name.
func getDisplayNameFilter(displayName string) string {
	return fmt.Sprintf("displayName eq '%s'", displayName)
}

// getSubjectFilter returns a filter string for the given subject.
func getSubjectFilter(subject string) string {
	return fmt.Sprintf("subject eq '%s'", subject)
}
