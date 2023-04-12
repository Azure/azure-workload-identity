package auth

import (
	"crypto/tls"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/google/uuid"
	nethttplibrary "github.com/microsoft/kiota-http-go"
	msgrapsdkgo "github.com/microsoftgraph/msgraph-sdk-go"
	msgraphgocore "github.com/microsoftgraph/msgraph-sdk-go-core"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	ini "gopkg.in/ini.v1"
	utilnet "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/transport"
	"monis.app/mlog"

	"github.com/Azure/azure-workload-identity/pkg/cloud"
)

const (
	clientSecretAuthMethod      = "client_secret"
	clientCertificateAuthMethod = "client_certificate"
	cliAuthMethod               = "cli"
)

// Provider is an interface for getting an Azure client
type Provider interface {
	AddFlags(f *pflag.FlagSet)
	GetAzureClient() cloud.Interface
	GetAzureTenantID() string
	Validate() error
}

// authArgs is an implementation of the Provider interface
type authArgs struct {
	rawAzureEnvironment string
	rawSubscriptionID   string
	subscriptionID      uuid.UUID
	authMethod          string
	rawClientID         string

	tenantID        string
	clientID        uuid.UUID
	clientSecret    string
	certificatePath string
	privateKeyPath  string
	azureClient     cloud.Interface

	client *http.Client
}

// NewProvider returns a new authArgs
func NewProvider() Provider {
	return &authArgs{client: defaultClient()}
}

func defaultClient() *http.Client {
	return &http.Client{
		Transport: defaultWrap(defaultTransport()),
		Timeout:   3 * time.Hour, // make it impossible for requests to hang indefinitely
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse // copied from MS SDK
		},
	}
}

func defaultTransport() *http.Transport {
	baseRT := http.DefaultTransport.(*http.Transport).Clone()
	baseRT.MaxIdleConnsPerHost = 25 // copied from client-go
	baseRT.TLSClientConfig = &tls.Config{
		MinVersion: tls.VersionTLS12, // same as client-go and MS SDK
		// enable HTTP2
		// setting this explicitly is only required in very specific circumstances
		// it is simpler to just set it here than to try and determine if we need to
		NextProtos: []string{"h2", "http/1.1"},
	}
	utilnet.SetTransportDefaults(baseRT)
	return baseRT
}

func defaultWrap(rt http.RoundTripper) http.RoundTripper {
	opts := msgrapsdkgo.GetDefaultClientOptions()
	rt = newMiddlewarePipeline(msgraphgocore.GetDefaultMiddlewaresWithOptions(&opts), rt)
	rt = transport.NewUserAgentRoundTripper(rest.DefaultKubernetesUserAgent(), rt)
	rt = newDelayDebugWrappers(rt)
	return rt
}

type delayDebugWrappers struct {
	transport http.RoundTripper
}

func newDelayDebugWrappers(rt http.RoundTripper) http.RoundTripper {
	return &delayDebugWrappers{transport: rt}
}

func (d *delayDebugWrappers) RoundTrip(req *http.Request) (*http.Response, error) {
	rt := d.transport
	if mlog.Enabled(mlog.LevelTrace) {
		rt = transport.DebugWrappers(rt) // delay wrapping because DebugWrappers makes static checks about log level
	}
	return rt.RoundTrip(req)
}

// copied from MS SDK so we can inject custom base round tripper
type middlewarePipeline struct {
	transport   http.RoundTripper
	middlewares []nethttplibrary.Middleware
}

func newMiddlewarePipeline(middlewares []nethttplibrary.Middleware, rt http.RoundTripper) http.RoundTripper {
	return &middlewarePipeline{
		transport:   rt,
		middlewares: middlewares,
	}
}

func (p *middlewarePipeline) Next(req *http.Request, middlewareIndex int) (*http.Response, error) {
	if middlewareIndex < len(p.middlewares) {
		middleware := p.middlewares[middlewareIndex]
		return middleware.Intercept(p, middlewareIndex+1, req)
	}

	return p.transport.RoundTrip(req)
}

func (p *middlewarePipeline) RoundTrip(req *http.Request) (*http.Response, error) {
	return p.Next(req, 0)
}

// AddFlags adds the flags for this package to the specified FlagSet
func (a *authArgs) AddFlags(f *pflag.FlagSet) {
	f.StringVar(&a.rawAzureEnvironment, "azure-env", "AzurePublicCloud", "the target Azure cloud")
	f.StringVarP(&a.rawSubscriptionID, "subscription-id", "s", "", "azure subscription id (required)")
	f.StringVar(&a.authMethod, "auth-method", cliAuthMethod, "auth method to use. Supported values: cli, client_secret, client_certificate")
	f.StringVar(&a.rawClientID, "client-id", "", "client id (used with --auth-method=[client_secret|client_certificate])")
	f.StringVar(&a.clientSecret, "client-secret", "", "client secret (used with --auth-method=client_secret)")
	f.StringVar(&a.certificatePath, "certificate-path", "", "path to client certificate (used with --auth-method=client_certificate)")
	f.StringVar(&a.privateKeyPath, "private-key-path", "", "path to private key (used with --auth-method=client_certificate)")
}

// GetAzureClient returns an Azure client
func (a *authArgs) GetAzureClient() cloud.Interface {
	return a.azureClient
}

// GetAzureTenantID returns the Azure tenant ID
func (a *authArgs) GetAzureTenantID() string {
	return a.tenantID
}

// Validate validates the authArgs
func (a *authArgs) Validate() error {
	var err error

	if a.authMethod == "" {
		return errors.New("--auth-method is a required parameter")
	}
	if a.authMethod == cliAuthMethod && a.rawClientID != "" && a.clientSecret != "" {
		a.authMethod = clientSecretAuthMethod
	}
	if a.authMethod == clientSecretAuthMethod || a.authMethod == clientCertificateAuthMethod {
		if a.clientID, err = uuid.Parse(a.rawClientID); err != nil {
			return errors.Wrap(err, "parsing --client-id")
		}
		if a.authMethod == clientSecretAuthMethod {
			if a.clientSecret == "" {
				return errors.New(`--client-secret must be specified when --auth-method="client_secret"`)
			}
		} else if a.authMethod == clientCertificateAuthMethod {
			if a.certificatePath == "" || a.privateKeyPath == "" {
				return errors.New(`--certificate-path and --private-key-path must be specified when --auth-method="client_certificate"`)
			}
		}
	}

	a.subscriptionID, _ = uuid.Parse(a.rawSubscriptionID)
	if a.subscriptionID.String() == "00000000-0000-0000-0000-000000000000" {
		var subID uuid.UUID
		subID, err = getSubFromAzDir(filepath.Join(getHomeDir(), ".azure"))
		if err != nil || subID.String() == "00000000-0000-0000-0000-000000000000" {
			return errors.New("--subscription-id is required (and must be a valid UUID)")
		}
		mlog.Info("No subscription provided, using selected subscription from Azure CLI", "subscriptionID", subID.String())
		a.subscriptionID = subID
	}

	env, err := azure.EnvironmentFromName(a.rawAzureEnvironment)
	if err != nil {
		return errors.Wrap(err, "failed to parse --azure-env as a valid target Azure cloud environment")
	}

	if a.tenantID, err = cloud.GetTenantID(a.subscriptionID.String(), a.client); err != nil {
		return err
	}

	switch a.authMethod {
	case cliAuthMethod:
		a.azureClient, err = cloud.NewAzureClientWithCLI(env, a.subscriptionID.String(), a.client)
	case clientSecretAuthMethod:
		a.azureClient, err = cloud.NewAzureClientWithClientSecret(env, a.subscriptionID.String(), a.clientID.String(), a.clientSecret, a.tenantID, a.client)
	case clientCertificateAuthMethod:
		a.azureClient, err = cloud.NewAzureClientWithClientCertificateFile(env, a.subscriptionID.String(), a.clientID.String(), a.tenantID, a.certificatePath, a.privateKeyPath, a.client)
	default:
		err = errors.Errorf("--auth-method: ERROR: method unsupported. method=%q", a.authMethod)
	}

	return err
}

// getSubFromAzDir returns the subscription ID from the Azure CLI directory
func getSubFromAzDir(root string) (uuid.UUID, error) {
	subConfig, err := ini.Load(filepath.Join(root, "clouds.config"))
	if err != nil {
		return uuid.UUID{}, errors.Wrap(err, "error decoding cloud subscription config")
	}

	cloudConfig, err := ini.Load(filepath.Join(root, "config"))
	if err != nil {
		return uuid.UUID{}, errors.Wrap(err, "error decoding cloud config")
	}

	cloud := getSelectedCloudFromAzConfig(cloudConfig)
	return getCloudSubFromAzConfig(cloud, subConfig)
}

// getSelectedCloudFromAzConfig returns the selected cloud from the Azure CLI config
func getSelectedCloudFromAzConfig(f *ini.File) string {
	selectedCloud := "AzureCloud"
	if cloud, err := f.GetSection("cloud"); err == nil {
		if name, err := cloud.GetKey("name"); err == nil {
			if s := name.String(); s != "" {
				selectedCloud = s
			}
		}
	}
	return selectedCloud
}

// getCloudSubFromAzConfig returns the subscription ID from the Azure CLI config
func getCloudSubFromAzConfig(cloud string, f *ini.File) (uuid.UUID, error) {
	cfg, err := f.GetSection(cloud)
	if err != nil {
		return uuid.UUID{}, errors.Wrap(err, "could not find user defined subscription id")
	}
	sub, err := cfg.GetKey("subscription")
	if err != nil {
		return uuid.UUID{}, errors.Wrap(err, "error reading subscription id from cloud config")
	}
	return uuid.Parse(sub.String())
}

// getHomeDir attempts to get the home dir from env
func getHomeDir() string {
	if runtime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	}
	return os.Getenv("HOME")
}
