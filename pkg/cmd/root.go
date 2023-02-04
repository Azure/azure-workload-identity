package cmd

import (
	"context"

	"github.com/spf13/cobra"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // import auth plugins. See https://github.com/Azure/azure-workload-identity/issues/362.
	"monis.app/mlog"

	"github.com/Azure/azure-workload-identity/pkg/cmd/jwks"
	"github.com/Azure/azure-workload-identity/pkg/cmd/podidentity"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount"
	"github.com/Azure/azure-workload-identity/pkg/cmd/version"
)

const (
	rootName             = "azwi"
	rootShortDescription = "azwi helps to manage workload identity"
	rootLongDescription  = rootShortDescription + " in Azure."
)

var (
	debug bool
)

// NewRootCmd returns the root command for Azure Workload Identity.
func NewRootCmd() *cobra.Command {
	flushLogs := mlog.Setup()
	cmd := &cobra.Command{
		Use:   rootName,
		Short: rootShortDescription,
		Long:  rootLongDescription,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// default to info instead of warning because existing info logs expect to always be printed
			logLevel := mlog.LevelInfo
			if debug {
				logLevel = mlog.LevelAll
			}

			// inputs are essentially static so this should never error
			return mlog.ValidateAndSetLogLevelAndFormatGlobally(
				context.Background(), // context is unused with mlog.FormatCLI
				mlog.LogSpec{
					Level:  logLevel,
					Format: mlog.FormatCLI,
				},
			)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Usage()
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			flushLogs()
		},
	}

	p := cmd.PersistentFlags()
	p.BoolVar(&debug, "debug", false, "Enable debug logging")

	cmd.AddCommand(version.NewVersionCmd())
	cmd.AddCommand(serviceaccount.NewServiceAccountCmd())
	cmd.AddCommand(jwks.NewJWKSCmd())
	cmd.AddCommand(podidentity.NewPodIdentityCmd())

	return cmd
}
