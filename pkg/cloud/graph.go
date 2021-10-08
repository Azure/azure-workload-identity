package cloud

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// CreateServicePrincipal creates a service principal for the given application.
// No secret or certificate is generated.
func (c *AzureClient) CreateServicePrincipal(ctx context.Context, appID string, tags []string) (graphrbac.ServicePrincipal, error) {
	spCreateParameters := graphrbac.ServicePrincipalCreateParameters{
		AppID: &appID,
		Tags:  &tags,
	}
	log.Debugf("Creating service principal for application with id=%s", appID)

	return c.servicePrincipalsClient.Create(ctx, spCreateParameters)
}

// CreateApplication creates an application.
func (c *AzureClient) CreateApplication(ctx context.Context, displayName string) (graphrbac.Application, error) {
	appCreateParameters := graphrbac.ApplicationCreateParameters{
		DisplayName: to.StringPtr(displayName),
	}
	log.Debugf("Creating application with display name=%s", displayName)

	return c.applicationsClient.Create(ctx, appCreateParameters)
}

// GetServicePrincipal gets a service principal by its display name.
func (c *AzureClient) GetServicePrincipal(ctx context.Context, displayName string) (graphrbac.ServicePrincipal, error) {
	retServicePrincipal := graphrbac.ServicePrincipal{}

	log.Debugf("Getting service principal with display name=%s", displayName)
	filter := getDisplayNameFilter(displayName)
	spList, err := c.servicePrincipalsClient.List(ctx, filter)
	if err != nil {
		return retServicePrincipal, err
	}
	if len(spList.Values()) == 0 {
		return retServicePrincipal, errors.Errorf("service principal %s not found", displayName)
	}
	return spList.Values()[0], nil
}

// GetApplication gets an application by its display name.
func (c *AzureClient) GetApplication(ctx context.Context, displayName string) (graphrbac.Application, error) {
	retApplication := graphrbac.Application{}

	log.Debugf("Getting application with display name=%s", displayName)
	filter := getDisplayNameFilter(displayName)
	appList, err := c.applicationsClient.List(ctx, filter)
	if err != nil {
		return retApplication, err
	}
	if len(appList.Values()) == 0 {
		return retApplication, errors.Errorf("application with display name '%s' not found", displayName)
	}

	return appList.Values()[0], nil
}

// DeleteServicePrincipal deletes a service principal.
func (c *AzureClient) DeleteServicePrincipal(ctx context.Context, objectID string) (autorest.Response, error) {
	log.Debugf("Deleting service principal with object id=%s", objectID)
	return c.servicePrincipalsClient.Delete(ctx, objectID)
}

// DeleteApplication deletes an application.
func (c *AzureClient) DeleteApplication(ctx context.Context, objectID string) (autorest.Response, error) {
	log.Debugf("Deleting application with object id=%s", objectID)
	return c.applicationsClient.Delete(ctx, objectID)
}

// getDisplayNameFilter returns a filter string for the given display name.
func getDisplayNameFilter(displayName string) string {
	return fmt.Sprintf("startswith(displayName, '%s')", displayName)
}
