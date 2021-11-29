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
	log "github.com/sirupsen/logrus"
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

	log.Debugf("Creating service principal for application with id=%s", appID)
	return c.graphServiceClient.ServicePrincipals().Post(spPostOptions)
}

// CreateApplication creates an application.
func (c *AzureClient) CreateApplication(ctx context.Context, displayName string) (*graph.Application, error) {
	appPostOptions := &applications.ApplicationsRequestBuilderPostOptions{
		Body: graph.NewApplication(),
	}
	appPostOptions.Body.SetDisplayName(to.StringPtr(displayName))

	log.Debugf("Creating application with display name=%s", displayName)
	return c.graphServiceClient.Applications().Post(appPostOptions)
}

// GetServicePrincipal gets a service principal by its display name.
func (c *AzureClient) GetServicePrincipal(ctx context.Context, displayName string) (*graph.ServicePrincipal, error) {
	log.Debugf("Getting service principal with display name=%s", displayName)

	spGetOptions := &serviceprincipals.ServicePrincipalsRequestBuilderGetOptions{
		Q: &serviceprincipals.ServicePrincipalsRequestBuilderGetQueryParameters{
			Filter: to.StringPtr(getDisplayNameFilter(displayName)),
		},
	}

	resp, err := c.graphServiceClient.ServicePrincipals().Get(spGetOptions)
	if err != nil {
		return nil, err
	}
	if len(resp.GetValue()) == 0 {
		return nil, errors.Errorf("service principal %s not found", displayName)
	}
	return &resp.GetValue()[0], nil
}

// GetApplication gets an application by its display name.
func (c *AzureClient) GetApplication(ctx context.Context, displayName string) (*graph.Application, error) {
	log.Infof("Getting application with display name=%s filter=%s", displayName, getDisplayNameFilter(displayName))

	appGetOptions := &applications.ApplicationsRequestBuilderGetOptions{
		Q: &applications.ApplicationsRequestBuilderGetQueryParameters{
			Filter: to.StringPtr(getDisplayNameFilter(displayName)),
		},
	}

	resp, err := c.graphServiceClient.Applications().Get(appGetOptions)
	if err != nil {
		return nil, err
	}
	if len(resp.GetValue()) == 0 {
		return nil, errors.Errorf("application with display name '%s' not found", displayName)
	}
	return &resp.GetValue()[0], nil
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

// getDisplayNameFilter returns a filter string for the given display name.
func getDisplayNameFilter(displayName string) string {
	return fmt.Sprintf("startswith(displayName, '%s')", displayName)
}

// AddFederatedCredential adds a federated credential to the cloud provider.
func (c *AzureClient) AddFederatedCredential(ctx context.Context, objectID string, fic *graph.FederatedIdentityCredential) error {
	log.Debugf("Adding federated credential for objectID=%s", objectID)

	ficPostOptions := &federatedidentitycredentials.FederatedIdentityCredentialsRequestBuilderPostOptions{
		Body: fic,
	}

	_, err := c.graphServiceClient.ApplicationsById(objectID).FederatedIdentityCredentials().Post(ficPostOptions)
	return err
}

// GetFederatedCredential gets a federated credential from the cloud provider.
func (c *AzureClient) GetFederatedCredential(ctx context.Context, objectID, issuer, subject string) (*graph.FederatedIdentityCredential, error) {
	log.Infof("Getting federated credential for objectID=%s, issuer=%s, subject=%s", objectID, issuer, subject)

	ficGetOptions := &federatedidentitycredentials.FederatedIdentityCredentialsRequestBuilderGetOptions{
		Q: &federatedidentitycredentials.FederatedIdentityCredentialsRequestBuilderGetQueryParameters{
			// TODO(aramase): compound filter with issuer and subject
			Filter: to.StringPtr(getSubjectFilter(subject, issuer)),
		},
	}

	resp, err := c.graphServiceClient.ApplicationsById(objectID).FederatedIdentityCredentials().Get(ficGetOptions)
	if err != nil {
		return nil, err
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
	log.Debugf("Deleting federated credential for objectID=%s, federatedCredentialID=%s", objectID, federatedCredentialID)
	return c.graphServiceClient.ApplicationsById(objectID).FederatedIdentityCredentialsById(federatedCredentialID).Delete(nil)
}

// getSubjectFilter returns a filter string for the given subject.
func getSubjectFilter(subject, issuer string) string {
	return fmt.Sprintf("subject eq '%s'", subject)
}
