package serviceaccount

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/Azure/azure-workload-identity/pkg/cloud"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	ini "gopkg.in/ini.v1"
)

const (
	clientSecretAuthMethod      = "client_secret"
	clientCertificateAuthMethod = "client_certificate"
	cliAuthMethod               = "cli"
)

// NewServiceAccountCmd returns a new serviceaccount command
func NewServiceAccountCmd() *cobra.Command {
	serviceAccountCmd := &cobra.Command{
		Use:     "serviceaccount",
		Short:   "Manage the workload identity",
		Long:    "Manage the workload identity",
		Aliases: []string{"sa"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Usage()
		},
	}
	serviceAccountCmd.AddCommand(newCreateCmd())
	serviceAccountCmd.AddCommand(newDeleteCmd())

	return serviceAccountCmd
}

type authProvider interface {
	getAuthArgs() *authArgs
	getClient() (cloud.Interface, error)
}

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
}

func addAuthFlags(authArgs *authArgs, f *pflag.FlagSet) {
	f.StringVar(&authArgs.rawAzureEnvironment, "azure-env", "AzurePublicCloud", "the target Azure cloud")
	f.StringVarP(&authArgs.rawSubscriptionID, "subscription-id", "s", "", "azure subscription id (required)")
	f.StringVar(&authArgs.authMethod, "auth-method", "cli", "auth method (default:`client_secret`, `cli`, `client_certificate`)")
	f.StringVar(&authArgs.rawClientID, "client-id", "", "client id (used with --auth-method=[client_secret|client_certificate])")
	f.StringVar(&authArgs.clientSecret, "client-secret", "", "client secret (used with --auth-method=client_secret)")
	f.StringVar(&authArgs.certificatePath, "certificate-path", "", "path to client certificate (used with --auth-method=client_certificate)")
	f.StringVar(&authArgs.privateKeyPath, "private-key-path", "", "path to private key (used with --auth-method=client_certificate)")
}

func (a *authArgs) getAuthArgs() *authArgs {
	return a
}

func (a *authArgs) getClient() (cloud.Interface, error) {
	var client *cloud.AzureClient
	env, err := azure.EnvironmentFromName(a.rawAzureEnvironment)
	if err != nil {
		return nil, err
	}
	if a.tenantID, err = cloud.GetTenantID(env.ResourceManagerEndpoint, a.subscriptionID.String()); err != nil {
		return nil, err
	}
	switch a.authMethod {
	case cliAuthMethod:
		client, err = cloud.NewAzureClientWithCLI(env, a.subscriptionID.String(), a.tenantID)
	case clientSecretAuthMethod:
		client, err = cloud.NewAzureClientWithClientSecret(env, a.subscriptionID.String(), a.clientID.String(), a.clientSecret, a.tenantID)
	case clientCertificateAuthMethod:
		client, err = cloud.NewAzureClientWithClientCertificateFile(env, a.subscriptionID.String(), a.clientID.String(), a.tenantID, a.certificatePath, a.privateKeyPath)
	default:
		return nil, errors.Errorf("--auth-method: ERROR: method unsupported. method=%q", a.authMethod)
	}
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (a *authArgs) validate() error {
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
		log.Infoln("No subscription provided, using selected subscription from azure CLI:", subID.String())
		a.subscriptionID = subID
	}

	if _, err = azure.EnvironmentFromName(a.rawAzureEnvironment); err != nil {
		return errors.New("failed to parse --azure-env as a valid target Azure cloud environment")
	}

	return nil
}

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

func getCloudSubFromAzConfig(cloud string, f *ini.File) (uuid.UUID, error) {
	cfg, err := f.GetSection(cloud)
	if err != nil {
		return uuid.UUID{}, errors.New("could not find user defined subscription id")
	}
	sub, err := cfg.GetKey("subscription")
	if err != nil {
		return uuid.UUID{}, errors.Wrap(err, "error reading subscription id from cloud config")
	}
	return uuid.Parse(sub.String())
}

// getIssuerHash returns a hash of the issuer URL
func getIssuerHash(issuer string) string {
	h := sha256.New()
	h.Write([]byte(issuer))
	return base64.URLEncoding.EncodeToString(h.Sum(nil))
}

// getSubject returns the subject of the federated credential
func getSubject(namespace, name string) string {
	return fmt.Sprintf("system:serviceaccount:%s:%s", namespace, name)
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
