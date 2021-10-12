package cmd

import (
	"github.com/Azure/azure-workload-identity/pkg/cmd/jwks"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount"
	"github.com/Azure/azure-workload-identity/pkg/cmd/version"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	rootName             = "azwi"
	rootShortDescription = "azwi helps to manage workload identity"
	rootLongDescription  = rootShortDescription + " in Azure."
)

var (
	debug bool
)

// NewRootCmd returns the root command for Azure Pod Identity.
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   rootName,
		Short: rootShortDescription,
		Long:  rootLongDescription,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if debug {
				log.SetLevel(log.DebugLevel)
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Usage()
		},
	}

	p := cmd.PersistentFlags()
	p.BoolVar(&debug, "debug", false, "Enable debug logging")

	cmd.AddCommand(version.NewVersionCmd())
	cmd.AddCommand(serviceaccount.NewServiceAccountCmd())
	cmd.AddCommand(jwks.NewJWKSCmd())

	return cmd
}
