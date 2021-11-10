package cloud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Azure/azure-workload-identity/pkg/version"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	betagraph "github.com/microsoftgraph/msgraph-beta-sdk-go/models/microsoft/graph"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	federatedCredentialCreateRetryCount = 10
	federatedCredentialCreateRetryDelay = 6 * time.Second
)

var (
	// ErrFederatedCredentialNotFound is returned when the federated credential is not found.
	ErrFederatedCredentialNotFound = errors.New("federated credential not found")
)

// FederatedCredentials returns a list of federated credentials for the specified application.
type FederatedCredentials struct {
	Value []FederatedCredential `json:"value"`
}

// FederatedCredential is the definition of the federated credential.
type FederatedCredential struct {
	Name        string   `json:"name"`
	Issuer      string   `json:"issuer"`
	Subject     string   `json:"subject"`
	Description string   `json:"description"`
	Audiences   []string `json:"audiences"`
	ID          string   `json:"id"`
}

type FederatedCredentialsClient struct {
	autorest.Client

	baseURI string
}

func NewFederatedCredentialsClient(baseURI string) FederatedCredentialsClient {
	return FederatedCredentialsClient{
		Client: autorest.NewClientWithUserAgent(version.GetUserAgent("azwi")),
		// TODO(aramase): remove beta from baseURI when the API is stable
		baseURI: baseURI + "beta",
	}
}

// AddFederatedCredential adds a federated credential to the cloud provider.
func (c *AzureClient) AddFederatedCredential(ctx context.Context, objectID string, fc *betagraph.FederatedIdentityCredential) error {
	log.Debugf("Adding federated credential for objectID=%s", objectID)

	body, err := json.Marshal(fc)
	if err != nil {
		return err
	}

	federatedCredentialURL := fmt.Sprintf("%s/applications/%s/federatedIdentityCredentials", c.federatedCredentialsClient.baseURI, objectID)
	req, err := http.NewRequest(http.MethodPost, federatedCredentialURL, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")

	// Adding retries to handle the propagation delay of the service principal.
	// Trying to create federated identity credential immediately after service
	// principal is created might result in "PrincipalNotFound" error.
	var response *http.Response
	for i := 0; i < federatedCredentialCreateRetryCount; i++ {
		response, err = c.federatedCredentialsClient.Do(req)
		if err == nil {
			break
		}
		time.Sleep(federatedCredentialCreateRetryDelay)
	}
	if err != nil {
		return autorest.NewErrorWithError(err, "FederatedCredentialsClient", http.MethodPost, response, "Failure sending request")
	}
	if _, err := deleteResponder(response, http.StatusCreated); err != nil {
		return autorest.NewErrorWithError(err, "FederatedCredentialsClient", http.MethodPost, response, "Failure responding to request")
	}

	return nil
}

// GetFederatedCredential gets a federated credential from the cloud provider.
func (c *AzureClient) GetFederatedCredential(ctx context.Context, objectID, issuer, subject string) (FederatedCredential, error) {
	log.Debugf("Getting federated credential for objectID=%s, issuer=%s, subject=%s", objectID, issuer, subject)

	var fc FederatedCredential
	federatedCredentialURL := fmt.Sprintf("%s/applications/%s/federatedIdentityCredentials", c.federatedCredentialsClient.baseURI, objectID)
	req, err := http.NewRequest(http.MethodGet, federatedCredentialURL, nil)
	if err != nil {
		return fc, err
	}
	req.Header.Add("Content-Type", "application/json")

	response, err := c.federatedCredentialsClient.Do(req)
	if err != nil {
		return fc, autorest.NewErrorWithError(err, "FederatedCredentialsClient", http.MethodGet, response, "Failure sending request")
	}
	if response.StatusCode != http.StatusOK {
		if _, err := deleteResponder(response, http.StatusOK); err != nil {
			return fc, autorest.NewErrorWithError(err, "FederatedCredentialsClient", http.MethodGet, response, "Failure responding to request")
		}
	}

	var federatedCredentials FederatedCredentials
	err = json.NewDecoder(response.Body).Decode(&federatedCredentials)
	if err != nil {
		return fc, err
	}

	for _, fic := range federatedCredentials.Value {
		if fic.Issuer == issuer && fic.Subject == subject {
			return fic, nil
		}
	}
	return fc, ErrFederatedCredentialNotFound
}

// DeleteFederatedCredential deletes a federated credential from the cloud provider.
func (c *AzureClient) DeleteFederatedCredential(ctx context.Context, objectID, federatedCredentialID string) error {
	log.Debugf("Deleting federated credential for objectID=%s, federatedCredentialID=%s", objectID, federatedCredentialID)

	federatedCredentialURL := fmt.Sprintf("%s/applications/%s/federatedIdentityCredentials/%s", c.federatedCredentialsClient.baseURI, objectID, federatedCredentialID)
	req, err := http.NewRequest(http.MethodDelete, federatedCredentialURL, nil)
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")

	response, err := c.federatedCredentialsClient.Do(req)
	if err != nil {
		return autorest.NewErrorWithError(err, "FederatedCredentialsClient", http.MethodDelete, response, "Failure sending request")
	}
	if _, err := deleteResponder(response, http.StatusNoContent); err != nil {
		return autorest.NewErrorWithError(err, "FederatedCredentialsClient", http.MethodDelete, response, "Failure responding to request")
	}

	return nil
}

// NewFederatedCredential returns a new federated credential.
func NewFederatedCredential(objectID, issuer, subject, description string, audiences []string) FederatedCredential {
	return FederatedCredential{
		Name:        "federatedcredential-from-cli",
		Issuer:      issuer,
		Subject:     subject,
		Description: description,
		Audiences:   audiences,
	}
}

// deleteResponder handles the response to the http request. The method always
// closes the http.Response Body.
func deleteResponder(resp *http.Response, statusCodes ...int) (result autorest.Response, err error) {
	err = autorest.Respond(
		resp,
		azure.WithErrorUnlessStatusCode(statusCodes...),
		autorest.ByClosing())
	result.Response = resp
	return
}
