package serviceaccount

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// NewServiceAccountCmd returns a new serviceaccount command
func NewServiceAccountCmd() *cobra.Command {
	serviceAccountCmd := &cobra.Command{
		Use:   "serviceaccount",
		Short: "Manage the workload identity",
		Long:  "Manage the workload identity",
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
}

type authArgs struct {
	azureTenantID       string
	azureClientID       string
	azureClientSecret   string
	azureSubscriptionID string
}

func addAuthFlags(authArgs *authArgs, f *pflag.FlagSet) {
	f.StringVarP(&authArgs.azureTenantID, "azure-tenant-id", "", "", "Azure Tenant ID")
	f.StringVarP(&authArgs.azureClientID, "azure-client-id", "", "", "Azure Client ID")
	f.StringVarP(&authArgs.azureClientSecret, "azure-client-secret", "", "", "Azure Client Secret")
	f.StringVarP(&authArgs.azureSubscriptionID, "azure-subscription-id", "", "", "Azure Subscription ID")
}

func (a *authArgs) getAuthArgs() *authArgs {
	return a
}

func (a *authArgs) validate() error {
	if a.azureTenantID == "" {
		return fmt.Errorf("azure-tenant-id is required")
	}
	if a.azureClientID == "" {
		return fmt.Errorf("azure-client-id is required")
	}
	if a.azureClientSecret == "" {
		return fmt.Errorf("azure-client-secret is required")
	}
	if a.azureSubscriptionID == "" {
		return fmt.Errorf("azure-subscription-id is required")
	}
	return nil
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
