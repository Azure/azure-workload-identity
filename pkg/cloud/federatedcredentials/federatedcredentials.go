package federatedcredentials

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

const (
	graphAPIURL = "https://graph.microsoft.com/beta"
	graphScope  = "https://graph.microsoft.com/.default"
)

// Interface for federated credential.
type Interface interface {
	AddFederatedCredential(ctx context.Context, objectID string, fc Federatedcredential) error
	GetFederatedCredential(ctx context.Context, objectID, issuer, subject string) (*Federatedcredential, error)
	DeleteFederatedCredential(ctx context.Context, objectID, federatedCredentialID string) error
}

// Federatedcredentials returns a list of federated credentials for the specified application.
type Federatedcredentials struct {
	Value []Federatedcredential `json:"value"`
}

// Federatedcredential is the definition of the federated credential.
type Federatedcredential struct {
	Name        string   `json:"name"`
	Issuer      string   `json:"issuer"`
	Subject     string   `json:"subject"`
	Description string   `json:"description"`
	Audiences   []string `json:"audiences"`
	ID          string   `json:"id"`
}

type client struct {
	tokenFn    func(ctx context.Context, resource string) (*azcore.AccessToken, error)
	httpClient *http.Client
}

var _ Interface = &client{}

func NewFederatedCredentialsClient(clientID, clientSecret, tenantID string) (Interface, error) {
	return &client{
		tokenFn: func(ctx context.Context, resource string) (*azcore.AccessToken, error) {
			return getToken(ctx, clientID, clientSecret, tenantID, resource)
		},
		httpClient: &http.Client{},
	}, nil
}

// AddFederatedCredential adds a federated credential to the cloud provider.
func (c *client) AddFederatedCredential(ctx context.Context, objectID string, fc Federatedcredential) error {
	accessToken, err := c.tokenFn(ctx, graphScope)
	if err != nil {
		return err
	}

	body, err := json.Marshal(fc)
	if err != nil {
		return err
	}

	federatedCredentialURL := fmt.Sprintf("%s/applications/%s/federatedIdentityCredentials", graphAPIURL, objectID)
	req, err := http.NewRequest(http.MethodPost, federatedCredentialURL, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	addRequestHeaders(req, accessToken)

	response, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		return fmt.Errorf("HTTP error: %s", response.Status)
	}
	return nil
}

// GetFederatedCredential gets a federated credential from the cloud provider.
func (c *client) GetFederatedCredential(ctx context.Context, objectID, issuer, subject string) (*Federatedcredential, error) {
	accessToken, err := c.tokenFn(ctx, graphScope)
	if err != nil {
		return nil, err
	}

	federatedCredentialURL := fmt.Sprintf("%s/applications/%s/federatedIdentityCredentials", graphAPIURL, objectID)
	req, err := http.NewRequest(http.MethodGet, federatedCredentialURL, nil)
	if err != nil {
		return nil, err
	}
	addRequestHeaders(req, accessToken)

	response, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %s", response.Status)
	}

	var federatedCredentials Federatedcredentials
	err = json.NewDecoder(response.Body).Decode(&federatedCredentials)
	if err != nil {
		return nil, err
	}

	for _, fic := range federatedCredentials.Value {
		if fic.Issuer == issuer && fic.Subject == subject {
			return &fic, nil
		}
	}
	return nil, fmt.Errorf("Federated credential not found")
}

// DeleteFederatedCredential deletes a federated credential from the cloud provider.
func (c *client) DeleteFederatedCredential(ctx context.Context, objectID, federatedCredentialID string) error {
	accessToken, err := c.tokenFn(ctx, graphScope)
	if err != nil {
		return err
	}

	federatedCredentialURL := fmt.Sprintf("%s/applications/%s/federatedIdentityCredentials/%s", graphAPIURL, objectID, federatedCredentialID)
	req, err := http.NewRequest(http.MethodDelete, federatedCredentialURL, nil)
	if err != nil {
		return err
	}
	addRequestHeaders(req, accessToken)

	response, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusNoContent {
		return fmt.Errorf("HTTP error: %s", response.Status)
	}
	return nil
}

// NewFederatedCredential returns a new federated credential.
func NewFederatedCredential(objectID, issuer, subject, description string, audiences []string) Federatedcredential {
	return Federatedcredential{
		Name:        "federatedcredential-from-cli",
		Issuer:      issuer,
		Subject:     subject,
		Description: description,
		Audiences:   audiences,
	}
}

// getToken gets a token from the cloud provider.
func getToken(ctx context.Context, clientID, clientSecret, tenantID, resource string) (*azcore.AccessToken, error) {
	cred, err := azidentity.NewClientSecretCredential(tenantID, clientID, clientSecret, nil)
	if err != nil {
		return nil, err
	}
	tokenOptions := policy.TokenRequestOptions{
		Scopes: []string{resource},
	}

	return cred.GetToken(ctx, tokenOptions)
}

func addRequestHeaders(req *http.Request, accessToken *azcore.AccessToken) {
	req.Header.Add("Content-Type", "application/json")
	// add token for authorization
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken.Token))
}
