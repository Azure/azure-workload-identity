package cloud

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	armpolicy "github.com/Azure/azure-sdk-for-go/sdk/azcore/arm/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions"
	"github.com/Azure/go-autorest/autorest/azure"
	kiotaauth "github.com/microsoft/kiota-authentication-azure-go"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/pkg/errors"
	"monis.app/mlog"
)

// ref: https://docs.microsoft.com/en-us/graph/migrate-azure-ad-graph-request-differences#basic-requests
var msGraphEndpoint = map[azure.Environment]string{
	azure.PublicCloud:       "https://graph.microsoft.com/",
	azure.USGovernmentCloud: "https://graph.microsoft.us/",
	azure.ChinaCloud:        "https://microsoftgraph.chinacloudapi.cn/",
	azure.GermanCloud:       "https://graph.microsoft.de/",
}

type Interface interface {
	CreateServicePrincipal(ctx context.Context, appID string, tags []string) (models.ServicePrincipalable, error)
	CreateApplication(ctx context.Context, displayName string) (models.Applicationable, error)
	DeleteServicePrincipal(ctx context.Context, objectID string) error
	DeleteApplication(ctx context.Context, objectID string) error
	GetServicePrincipal(ctx context.Context, displayName string) (models.ServicePrincipalable, error)
	GetApplication(ctx context.Context, displayName string) (models.Applicationable, error)

	// Role assignment methods
	CreateRoleAssignment(ctx context.Context, scope, roleName, principalID string) (armauthorization.RoleAssignment, error)
	DeleteRoleAssignment(ctx context.Context, roleAssignmentID string) (armauthorization.RoleAssignment, error)

	// Role definition methods
	GetRoleDefinitionIDByName(ctx context.Context, scope, roleName string) (armauthorization.RoleDefinition, error)

	// Federation methods
	AddFederatedCredential(ctx context.Context, objectID string, fic models.FederatedIdentityCredentialable) error
	GetFederatedCredential(ctx context.Context, objectID, issuer, subject string) (models.FederatedIdentityCredentialable, error)
	DeleteFederatedCredential(ctx context.Context, objectID, federatedCredentialID string) error
}

type AzureClient struct {
	environment    azure.Environment
	subscriptionID string

	graphServiceClient *msgraphsdk.GraphServiceClient

	roleAssignmentsClient *armauthorization.RoleAssignmentsClient
	roleDefinitionsClient *armauthorization.RoleDefinitionsClient
}

// NewAzureClientWithCLI creates an AzureClient configured from Azure CLI 2.0 for local development scenarios.
func NewAzureClientWithCLI(env azure.Environment, subscriptionID string, client *http.Client) (*AzureClient, error) {
	cred, err := azidentity.NewAzureCLICredential(nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create credential")
	}

	return getClient(env, subscriptionID, cred, client)
}

// NewAzureClientWithClientSecret returns an AzureClient via client_id and client_secret
func NewAzureClientWithClientSecret(env azure.Environment, subscriptionID, clientID, clientSecret, tenantID string, client *http.Client) (*AzureClient, error) {
	cred, err := azidentity.NewClientSecretCredential(tenantID, clientID, clientSecret,
		&azidentity.ClientSecretCredentialOptions{
			ClientOptions: azcore.ClientOptions{
				Transport: client,
			},
		})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create credential")
	}

	return getClient(env, subscriptionID, cred, client)
}

// NewAzureClientWithClientCertificateFile returns an AzureClient via client_id and jwt certificate assertion
func NewAzureClientWithClientCertificateFile(env azure.Environment, subscriptionID, clientID, tenantID, certificatePath, privateKeyPath string, client *http.Client) (*AzureClient, error) {
	certificateData, err := os.ReadFile(certificatePath)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to read certificate")
	}

	block, _ := pem.Decode(certificateData)
	if block == nil {
		return nil, errors.New("Failed to decode pem block from certificate")
	}

	certificate, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to parse certificate")
	}

	privateKey, err := parseRsaPrivateKey(privateKeyPath)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to parse rsa private key")
	}

	return NewAzureClientWithClientCertificate(env, subscriptionID, clientID, tenantID, certificate, privateKey, client)
}

// NewAzureClientWithClientCertificate returns an AzureClient via client_id and jwt certificate assertion
func NewAzureClientWithClientCertificate(env azure.Environment, subscriptionID, clientID, tenantID string, certificate *x509.Certificate, privateKey *rsa.PrivateKey, client *http.Client) (*AzureClient, error) {
	return newAzureClientWithCertificate(env, subscriptionID, clientID, tenantID, certificate, privateKey, client)
}

func newAzureClientWithCertificate(env azure.Environment, subscriptionID, clientID, tenantID string, certificate *x509.Certificate, privateKey *rsa.PrivateKey, client *http.Client) (*AzureClient, error) {
	if certificate == nil {
		return nil, errors.New("certificate should not be nil")
	}

	if privateKey == nil {
		return nil, errors.New("privateKey should not be nil")
	}

	cred, err := azidentity.NewClientCertificateCredential(tenantID, clientID, []*x509.Certificate{certificate}, privateKey,
		&azidentity.ClientCertificateCredentialOptions{
			ClientOptions: azcore.ClientOptions{
				Transport: client,
			},
		})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create credential")
	}

	return getClient(env, subscriptionID, cred, client)
}

func getClient(env azure.Environment, subscriptionID string, credential azcore.TokenCredential, client *http.Client) (*AzureClient, error) {
	auth, err := kiotaauth.NewAzureIdentityAuthenticationProviderWithScopes(credential, []string{getGraphScope(env)})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create authentication provider")
	}

	adapter, err := msgraphsdk.NewGraphRequestAdapterWithParseNodeFactoryAndSerializationWriterFactoryAndHttpClient(auth, nil, nil, client)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request adapter")
	}

	clientOpts := &armpolicy.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Transport: client,
		},
	}

	roleAssignmentsClient, err := armauthorization.NewRoleAssignmentsClient(subscriptionID, credential, clientOpts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create role assignments client")
	}

	roleDefinitionsClient, err := armauthorization.NewRoleDefinitionsClient(credential, clientOpts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create role definitions client")
	}

	azClient := &AzureClient{
		environment:    env,
		subscriptionID: subscriptionID,

		graphServiceClient: msgraphsdk.NewGraphServiceClient(adapter),

		roleAssignmentsClient: roleAssignmentsClient,
		roleDefinitionsClient: roleDefinitionsClient,
	}

	return azClient, nil
}

var _ azcore.TokenCredential = (*dummyCredential)(nil)

// dummyCredential is a dummy implementation of azcore.TokenCredential to be used
// when we only need to get the tenantID from a subscriptionID
type dummyCredential struct{}

func (d *dummyCredential) GetToken(_ context.Context, _ policy.TokenRequestOptions) (azcore.AccessToken, error) {
	return azcore.AccessToken{}, nil
}

// GetTenantID returns the tenantID for the given subscriptionID
// The tenantID is parsed from the WWW-Authenticate header of a failed request
func GetTenantID(subscriptionID string, client *http.Client) (string, error) {
	const hdrKey = "WWW-Authenticate"
	clientOpts := &armpolicy.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Transport: client,
		},
	}
	subscriptionsClient, err := armsubscriptions.NewClient(&dummyCredential{}, clientOpts)
	if err != nil {
		return "", errors.Wrap(err, "failed to create subscriptions client")
	}

	mlog.Debug("Resolving tenantID", "subscriptionID", subscriptionID)

	// we expect this request to fail (err != nil), but we are only interested
	// in headers, so surface the error if the Response is not present (i.e.
	// network error etc)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*150)
	defer cancel()

	_, err = subscriptionsClient.Get(ctx, subscriptionID, &armsubscriptions.ClientGetOptions{})
	var respErr *azcore.ResponseError
	if !errors.As(err, &respErr) {
		return "", errors.Errorf("unexpected response from get subscription: %v", err)
	}

	hdr := respErr.RawResponse.Header.Get(hdrKey)
	if hdr == "" {
		return "", errors.Errorf("header %q not found in get subscription response", hdrKey)
	}

	// Example value for hdr:
	//   Bearer authorization_uri="https://login.windows.net/996fe9d1-6171-40aa-945b-4c64b63bf655", error="invalid_token", error_description="The authentication failed because of missing 'Authorization' header."
	r := regexp.MustCompile(`authorization_uri=".*/([0-9a-f\-]+)"`)
	m := r.FindStringSubmatch(hdr)
	if m == nil {
		return "", errors.Errorf("Could not find the tenant ID in header: %s %q", hdrKey, hdr)
	}
	return m[1], nil
}

func parseRsaPrivateKey(path string) (*rsa.PrivateKey, error) {
	privateKeyData, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(privateKeyData)
	if block == nil {
		return nil, errors.New("Failed to decode a pem block from private key")
	}

	privatePkcs1Key, errPkcs1 := x509.ParsePKCS1PrivateKey(block.Bytes)
	if errPkcs1 == nil {
		return privatePkcs1Key, nil
	}

	privatePkcs8Key, errPkcs8 := x509.ParsePKCS8PrivateKey(block.Bytes)
	if errPkcs8 == nil {
		privatePkcs8RsaKey, ok := privatePkcs8Key.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("pkcs8 contained non-RSA key. Expected RSA key")
		}
		return privatePkcs8RsaKey, nil
	}

	return nil, errors.Errorf("failed to parse private key as Pkcs#1 or Pkcs#8. (%s). (%s)", errPkcs1, errPkcs8)
}

func getGraphScope(env azure.Environment) string {
	return fmt.Sprintf("%s.default", msGraphEndpoint[env])
}
