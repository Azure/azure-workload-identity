package cloud

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"net/http"
	"regexp"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-01-01-preview/authorization"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2016-06-01/subscriptions"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/cli"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// ref: https://docs.microsoft.com/en-us/graph/migrate-azure-ad-graph-request-differences#basic-requests
var msGraphEndpoint = map[azure.Environment]string{
	azure.PublicCloud:       "https://graph.microsoft.com/",
	azure.USGovernmentCloud: "https://graph.microsoft.us/",
	azure.ChinaCloud:        "https://microsoftgraph.chinacloudapi.cn/",
	azure.GermanCloud:       "https://graph.microsoft.de/",
}

type Interface interface {
	CreateServicePrincipal(ctx context.Context, appID string, tags []string) (*graphrbac.ServicePrincipal, error)
	CreateApplication(ctx context.Context, displayName string) (*graphrbac.Application, error)
	DeleteServicePrincipal(ctx context.Context, objectID string) (autorest.Response, error)
	DeleteApplication(ctx context.Context, objectID string) (autorest.Response, error)
	GetServicePrincipal(ctx context.Context, displayName string) (*graphrbac.ServicePrincipal, error)
	GetApplication(ctx context.Context, displayName string) (*graphrbac.Application, error)

	// Role assignment methods
	CreateRoleAssignment(ctx context.Context, scope, roleName, principalID string) (authorization.RoleAssignment, error)
	DeleteRoleAssignment(ctx context.Context, roleAssignmentID string) (authorization.RoleAssignment, error)

	// Federation methods
	AddFederatedCredential(ctx context.Context, objectID string, fc FederatedCredential) error
	GetFederatedCredential(ctx context.Context, objectID, issuer, subject string) (FederatedCredential, error)
	DeleteFederatedCredential(ctx context.Context, objectID, federatedCredentialID string) error
}

type AzureClient struct {
	environment    azure.Environment
	subscriptionID string

	authorizationClient authorization.RoleAssignmentsClient

	applicationsClient      graphrbac.ApplicationsClient
	servicePrincipalsClient graphrbac.ServicePrincipalsClient

	federatedCredentialsClient FederatedCredentialsClient
}

// NewAzureClientWithCLI creates an AzureClient configured from Azure CLI 2.0 for local development scenarios.
func NewAzureClientWithCLI(env azure.Environment, subscriptionID, tenantID string) (*AzureClient, error) {
	_, tenantID, err := getOAuthConfig(env, subscriptionID, tenantID)
	if err != nil {
		return nil, err
	}

	token, err := cli.GetTokenFromCLI(env.ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}

	adalToken, err := token.ToADALToken()
	if err != nil {
		return nil, err
	}

	aadGraphToken, err := cli.GetTokenFromCLI(env.GraphEndpoint)
	if err != nil {
		return nil, err
	}

	adalAADGraphToken, err := aadGraphToken.ToADALToken()
	if err != nil {
		return nil, err
	}

	msGraphToken, err := cli.GetTokenFromCLI(msGraphEndpoint[env])
	if err != nil {
		return nil, err
	}

	adalMSGraphToken, err := msGraphToken.ToADALToken()
	if err != nil {
		return nil, err
	}

	return getClient(env, subscriptionID, tenantID, autorest.NewBearerAuthorizer(&adalToken), autorest.NewBearerAuthorizer(&adalAADGraphToken), autorest.NewBearerAuthorizer(&adalMSGraphToken)), nil
}

// NewAzureClientWithClientSecret returns an AzureClient via client_id and client_secret
func NewAzureClientWithClientSecret(env azure.Environment, subscriptionID, clientID, clientSecret, tenantID string) (*AzureClient, error) {
	oauthConfig, tenantID, err := getOAuthConfig(env, subscriptionID, tenantID)
	if err != nil {
		return nil, err
	}

	armSpt, err := adal.NewServicePrincipalToken(*oauthConfig, clientID, clientSecret, env.ServiceManagementEndpoint)
	if err != nil {
		return nil, err
	}
	aadGraphSpt, err := adal.NewServicePrincipalToken(*oauthConfig, clientID, clientSecret, env.GraphEndpoint)
	if err != nil {
		return nil, err
	}
	if err = aadGraphSpt.Refresh(); err != nil {
		log.Error(err)
	}
	msGraphSpt, err := adal.NewServicePrincipalToken(*oauthConfig, clientID, clientSecret, msGraphEndpoint[env])
	if err != nil {
		return nil, err
	}
	if err = msGraphSpt.Refresh(); err != nil {
		log.Error(err)
	}

	return getClient(env, subscriptionID, tenantID, autorest.NewBearerAuthorizer(armSpt), autorest.NewBearerAuthorizer(aadGraphSpt), autorest.NewBearerAuthorizer(msGraphSpt)), nil
}

// NewAzureClientWithClientCertificateFile returns an AzureClient via client_id and jwt certificate assertion
func NewAzureClientWithClientCertificateFile(env azure.Environment, subscriptionID, clientID, tenantID, certificatePath, privateKeyPath string) (*AzureClient, error) {
	certificateData, err := ioutil.ReadFile(certificatePath)
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

	return NewAzureClientWithClientCertificate(env, subscriptionID, clientID, tenantID, certificate, privateKey)
}

// NewAzureClientWithClientCertificate returns an AzureClient via client_id and jwt certificate assertion
func NewAzureClientWithClientCertificate(env azure.Environment, subscriptionID, clientID, tenantID string, certificate *x509.Certificate, privateKey *rsa.PrivateKey) (*AzureClient, error) {
	oauthConfig, tenantID, err := getOAuthConfig(env, subscriptionID, tenantID)
	if err != nil {
		return nil, err
	}

	return newAzureClientWithCertificate(env, oauthConfig, subscriptionID, clientID, tenantID, certificate, privateKey)
}

// NewAzureClientWithClientCertificateExternalTenant returns an AzureClient via client_id and jwt certificate assertion against a 3rd party tenant
func NewAzureClientWithClientCertificateExternalTenant(env azure.Environment, subscriptionID, tenantID, clientID string, certificate *x509.Certificate, privateKey *rsa.PrivateKey) (*AzureClient, error) {
	oauthConfig, err := adal.NewOAuthConfig(env.ActiveDirectoryEndpoint, tenantID)
	if err != nil {
		return nil, err
	}

	return newAzureClientWithCertificate(env, oauthConfig, subscriptionID, clientID, tenantID, certificate, privateKey)
}

func getOAuthConfig(env azure.Environment, subscriptionID, tenantID string) (*adal.OAuthConfig, string, error) {
	oauthConfig, err := adal.NewOAuthConfig(env.ActiveDirectoryEndpoint, tenantID)
	if err != nil {
		return nil, "", err
	}

	return oauthConfig, tenantID, nil
}

func newAzureClientWithCertificate(env azure.Environment, oauthConfig *adal.OAuthConfig, subscriptionID, clientID, tenantID string, certificate *x509.Certificate, privateKey *rsa.PrivateKey) (*AzureClient, error) {
	if certificate == nil {
		return nil, errors.New("certificate should not be nil")
	}

	if privateKey == nil {
		return nil, errors.New("privateKey should not be nil")
	}

	armSpt, err := adal.NewServicePrincipalTokenFromCertificate(*oauthConfig, clientID, certificate, privateKey, env.ServiceManagementEndpoint)
	if err != nil {
		return nil, err
	}
	aadGraphSpt, err := adal.NewServicePrincipalTokenFromCertificate(*oauthConfig, clientID, certificate, privateKey, env.GraphEndpoint)
	if err != nil {
		return nil, err
	}
	if err = aadGraphSpt.Refresh(); err != nil {
		log.Error(err)
	}
	msGraphSpt, err := adal.NewServicePrincipalTokenFromCertificate(*oauthConfig, clientID, certificate, privateKey, msGraphEndpoint[env])
	if err != nil {
		return nil, err
	}
	if err = msGraphSpt.Refresh(); err != nil {
		log.Error(err)
	}

	return getClient(env, subscriptionID, tenantID, autorest.NewBearerAuthorizer(armSpt), autorest.NewBearerAuthorizer(aadGraphSpt), autorest.NewBearerAuthorizer(msGraphSpt)), nil
}

func getClient(env azure.Environment, subscriptionID, tenantID string, armAuthorizer, aadGraphAuthorizer, msGraphAuthorizer autorest.Authorizer) *AzureClient {
	azClient := &AzureClient{
		environment:    env,
		subscriptionID: subscriptionID,

		authorizationClient: authorization.NewRoleAssignmentsClientWithBaseURI(env.ResourceManagerEndpoint, subscriptionID),

		applicationsClient:      graphrbac.NewApplicationsClientWithBaseURI(env.GraphEndpoint, tenantID),
		servicePrincipalsClient: graphrbac.NewServicePrincipalsClientWithBaseURI(env.GraphEndpoint, tenantID),

		federatedCredentialsClient: NewFederatedCredentialsClient(msGraphEndpoint[env]),
	}

	azClient.authorizationClient.Authorizer = armAuthorizer

	azClient.applicationsClient.Authorizer = aadGraphAuthorizer
	azClient.servicePrincipalsClient.Authorizer = aadGraphAuthorizer

	azClient.federatedCredentialsClient.Authorizer = msGraphAuthorizer
	return azClient
}

// GetTenantID figures out the AAD tenant ID of the subscription by making an
// unauthenticated request to the Get Subscription Details endpoint and parses
// the value from WWW-Authenticate header.
// TODO this should probably to to the armhelpers library
func GetTenantID(resourceManagerEndpoint string, subscriptionID string) (string, error) {
	const hdrKey = "WWW-Authenticate"
	c := subscriptions.NewClientWithBaseURI(resourceManagerEndpoint)

	log.Debugf("Resolving tenantID for subscriptionID: %s", subscriptionID)

	// we expect this request to fail (err != nil), but we are only interested
	// in headers, so surface the error if the Response is not present (i.e.
	// network error etc)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*150)
	defer cancel()
	subs, err := c.Get(ctx, subscriptionID)
	if subs.Response.Response == nil {
		return "", errors.Wrap(err, "Request failed")
	}

	// Expecting 401 StatusUnauthorized here, just read the header
	if subs.StatusCode != http.StatusUnauthorized {
		return "", errors.Errorf("Unexpected response from Get Subscription: %v", subs.StatusCode)
	}
	hdr := subs.Header.Get(hdrKey)
	if hdr == "" {
		return "", errors.Errorf("Header %v not found in Get Subscription response", hdrKey)
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
	privateKeyData, err := ioutil.ReadFile(path)
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
