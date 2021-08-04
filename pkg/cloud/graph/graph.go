package graph

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/pkg/errors"
)

const (
	// defaultGraphResource is the default resource for the graph API.
	defaultGraphResource = "https://graph.windows.net"
)

// Interface is the interface for the Azure Graph API.
type Interface interface {
	CreateServicePrincipal(ctx context.Context, appID, tenantID string, tags []string) (graphrbac.ServicePrincipal, error)
	CreateApplication(ctx context.Context, displayName, tenantID string) (graphrbac.Application, error)
	DeleteServicePrincipal(ctx context.Context, appID, objectID, tenantID string) error
	DeleteApplication(ctx context.Context, objectID, tenantID string) error
	GetServicePrincipal(ctx context.Context, displayName, tenantID string) (graphrbac.ServicePrincipal, error)
	GetApplication(ctx context.Context, displayName, tenantID string) (graphrbac.Application, error)
}

// client for the Azure Graph API.
type client struct {
	sp  graphrbac.ServicePrincipalsClient
	app graphrbac.ApplicationsClient
}

var _ Interface = &client{}

// NewGraphClient creates a new graph client.
func NewGraphClient(clientID, clientSecret, tenantID string) (Interface, error) {
	a, err := injectGraphAuthorizer(clientID, clientSecret, tenantID)
	if err != nil {
		return nil, err
	}
	sp, err := newServicePrincipalClient(a, tenantID)
	if err != nil {
		return nil, err
	}
	app, err := newApplicationClient(a, tenantID)
	if err != nil {
		return nil, err
	}
	return &client{sp: sp, app: app}, nil
}

func newServicePrincipalClient(a autorest.Authorizer, tenantID string) (graphrbac.ServicePrincipalsClient, error) {
	client := graphrbac.NewServicePrincipalsClient(tenantID)
	client.Authorizer = a
	return client, nil
}

func newApplicationClient(a autorest.Authorizer, tenantID string) (graphrbac.ApplicationsClient, error) {
	client := graphrbac.NewApplicationsClient(tenantID)
	client.Authorizer = a
	return client, nil
}

func injectGraphAuthorizer(clientID, clientSecret, tenantID string) (autorest.Authorizer, error) {
	client := auth.NewClientCredentialsConfig(clientID, clientSecret, tenantID)
	client.Resource = defaultGraphResource
	return client.Authorizer()
}

// CreateServicePrincipal creates a service principal for the given application.
// No secret or certificate is generated.
func (c *client) CreateServicePrincipal(ctx context.Context, appID, tenantID string, tags []string) (graphrbac.ServicePrincipal, error) {
	spCreateParameters := graphrbac.ServicePrincipalCreateParameters{
		AppID: &appID,
		Tags:  &tags,
	}
	return c.sp.Create(ctx, spCreateParameters)
}

// CreateApplication creates an application.
func (c *client) CreateApplication(ctx context.Context, displayName, tenantID string) (graphrbac.Application, error) {
	appCreateParameters := graphrbac.ApplicationCreateParameters{
		DisplayName: &displayName,
	}
	return c.app.Create(ctx, appCreateParameters)
}

// GetServicePrincipal gets a service principal by its display name.
func (c *client) GetServicePrincipal(ctx context.Context, displayName, tenantID string) (graphrbac.ServicePrincipal, error) {
	retServicePrincipal := graphrbac.ServicePrincipal{}

	filter := getDisplayNameFilter(displayName)
	spList, err := c.sp.List(ctx, filter)
	if err != nil {
		return retServicePrincipal, err
	}
	if len(spList.Values()) == 0 {
		return retServicePrincipal, errors.Errorf("service principal %s not found", displayName)
	}
	return spList.Values()[0], nil
}

// GetApplication gets an application by its display name.
func (c *client) GetApplication(ctx context.Context, displayName, tenantID string) (graphrbac.Application, error) {
	retApplication := graphrbac.Application{}

	filter := getDisplayNameFilter(displayName)
	appList, err := c.app.List(ctx, filter)
	if err != nil {
		return retApplication, err
	}
	if len(appList.Values()) == 0 {
		return retApplication, errors.Errorf("application with display name '%s' not found", displayName)
	}

	return appList.Values()[0], nil
}

// DeleteServicePrincipal deletes a service principal.
func (c *client) DeleteServicePrincipal(ctx context.Context, tenantID, appID, objectID string) error {
	_, err := c.sp.Delete(ctx, objectID)
	return err
}

// DeleteApplication deletes an application.
func (c *client) DeleteApplication(ctx context.Context, tenantID, objectID string) error {
	_, err := c.app.Delete(ctx, objectID)
	return err
}

// getDisplayNameFilter returns a filter string for the given display name.
func getDisplayNameFilter(displayName string) string {
	return fmt.Sprintf("startswith(displayName, '%s')", displayName)
}
