package cloud

import (
	"context"
	"fmt"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/microsoftgraph/msgraph-beta-sdk-go/applications"
	"github.com/microsoftgraph/msgraph-beta-sdk-go/applications/item/federatedidentitycredentials"
	"github.com/microsoftgraph/msgraph-beta-sdk-go/models/microsoft/graph"
	"github.com/microsoftgraph/msgraph-beta-sdk-go/serviceprincipals"
	"github.com/pkg/errors"
	"monis.app/mlog"
)

var (
	// ErrFederatedCredentialNotFound is returned when the federated credential is not found.
	ErrFederatedCredentialNotFound = errors.New("federated credential not found")
)

// CreateServicePrincipal creates a service principal for the given application.
// No secret or certificate is generated.
func (c *AzureClient) CreateServicePrincipal(ctx context.Context, appID string, tags []string) (*graph.ServicePrincipal, error) {
	spPostOptions := &serviceprincipals.ServicePrincipalsRequestBuilderPostOptions{
		Body: graph.NewServicePrincipal(),
	}
	spPostOptions.Body.SetAppId(to.StringPtr(appID))
	spPostOptions.Body.SetTags(tags)

	mlog.Debug("Creating service principal for application", "id", appID)
	sp, err := c.graphServiceClient.ServicePrincipals().Post(spPostOptions)
	if err != nil {
		return nil, err
	}
	graphErr, err := GetGraphError(sp.GetAdditionalData())
	if err != nil {
		return nil, err
	}
	if graphErr != nil {
		return nil, *graphErr
	}
	return sp, nil
}

// CreateApplication creates an application.
func (c *AzureClient) CreateApplication(ctx context.Context, displayName string) (*graph.Application, error) {
	appPostOptions := &applications.ApplicationsRequestBuilderPostOptions{
		Body: graph.NewApplication(),
	}
	appPostOptions.Body.SetDisplayName(to.StringPtr(displayName))

	mlog.Debug("Creating application", "displayName", displayName)
	app, err := c.graphServiceClient.Applications().Post(appPostOptions)
	if err != nil {
		return nil, err
	}
	graphErr, err := GetGraphError(app.GetAdditionalData())
	if err != nil {
		return nil, err
	}
	if graphErr != nil {
		return nil, *graphErr
	}
	return app, nil
}

// GetServicePrincipal gets a service principal by its display name.
func (c *AzureClient) GetServicePrincipal(ctx context.Context, displayName string) (*graph.ServicePrincipal, error) {
	mlog.Debug("Getting service principal", "displayName", displayName)

	spGetOptions := &serviceprincipals.ServicePrincipalsRequestBuilderGetOptions{
		Q: &serviceprincipals.ServicePrincipalsRequestBuilderGetQueryParameters{
			Filter: to.StringPtr(getDisplayNameFilter(displayName)),
		},
	}

	resp, err := c.graphServiceClient.ServicePrincipals().Get(spGetOptions)
	if err != nil {
		return nil, err
	}
	graphErr, err := GetGraphError(resp.GetAdditionalData())
	if err != nil {
		return nil, err
	}
	if graphErr != nil {
		return nil, *graphErr
	}
	if len(resp.GetValue()) == 0 {
		return nil, errors.Errorf("service principal %s not found", displayName)
	}
	return &resp.GetValue()[0], nil
}

// GetApplication gets an application by its display name.
func (c *AzureClient) GetApplication(ctx context.Context, displayName string) (*graph.Application, error) {
	mlog.Debug("Getting application", "displayName", displayName)

	appGetOptions := &applications.ApplicationsRequestBuilderGetOptions{
		Q: &applications.ApplicationsRequestBuilderGetQueryParameters{
			Filter: to.StringPtr(getDisplayNameFilter(displayName)),
		},
	}

	resp, err := c.graphServiceClient.Applications().Get(appGetOptions)
	if err != nil {
		return nil, err
	}
	graphErr, err := GetGraphError(resp.GetAdditionalData())
	if err != nil {
		return nil, err
	}
	if graphErr != nil {
		return nil, *graphErr
	}
	if len(resp.GetValue()) == 0 {
		return nil, errors.Errorf("application with display name '%s' not found", displayName)
	}
	return &resp.GetValue()[0], nil
}

// DeleteServicePrincipal deletes a service principal.
func (c *AzureClient) DeleteServicePrincipal(ctx context.Context, objectID string) error {
	mlog.Debug("Deleting service principal", "objectID", objectID)
	return c.graphServiceClient.ServicePrincipalsById(objectID).Delete(nil)
}

// DeleteApplication deletes an application.
func (c *AzureClient) DeleteApplication(ctx context.Context, objectID string) error {
	mlog.Debug("Deleting application", "objectID", objectID)
	return c.graphServiceClient.ApplicationsById(objectID).Delete(nil)
}

// AddFederatedCredential adds a federated credential to the cloud provider.
func (c *AzureClient) AddFederatedCredential(ctx context.Context, objectID string, fic *graph.FederatedIdentityCredential) error {
	mlog.Debug("Adding federated credential", "objectID", objectID)

	ficPostOptions := &federatedidentitycredentials.FederatedIdentityCredentialsRequestBuilderPostOptions{
		Body: fic,
	}
	fic, err := c.graphServiceClient.ApplicationsById(objectID).FederatedIdentityCredentials().Post(ficPostOptions)
	if err != nil {
		return err
	}
	graphErr, err := GetGraphError(fic.GetAdditionalData())
	if err != nil {
		return err
	}
	if graphErr != nil {
		return *graphErr
	}
	return nil
}

// GetFederatedCredential gets a federated credential from the cloud provider.
func (c *AzureClient) GetFederatedCredential(ctx context.Context, objectID, issuer, subject string) (*graph.FederatedIdentityCredential, error) {
	mlog.Debug("Getting federated credential",
		"objectID", objectID,
		"issuer", issuer,
		"subject", subject,
	)

	ficGetOptions := &federatedidentitycredentials.FederatedIdentityCredentialsRequestBuilderGetOptions{
		Q: &federatedidentitycredentials.FederatedIdentityCredentialsRequestBuilderGetQueryParameters{
			// Filtering on more than one resource is currently not supported.
			Filter: to.StringPtr(getSubjectFilter(subject)),
		},
	}

	resp, err := c.graphServiceClient.ApplicationsById(objectID).FederatedIdentityCredentials().Get(ficGetOptions)
	if err != nil {
		return nil, err
	}
	graphErr, err := GetGraphError(resp.GetAdditionalData())
	if err != nil {
		return nil, err
	}
	if graphErr != nil {
		return nil, *graphErr
	}
	for _, fic := range resp.GetValue() {
		if *fic.GetIssuer() == issuer {
			return &fic, nil
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
