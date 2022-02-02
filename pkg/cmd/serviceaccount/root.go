package serviceaccount

import (
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/auth"

	"github.com/spf13/cobra"
)

// NewServiceAccountCmd returns a new serviceaccount command
func NewServiceAccountCmd() *cobra.Command {
	authProvider := auth.NewProvider()
	serviceAccountCmd := &cobra.Command{
		Use:     "serviceaccount",
		Short:   "Manage the workload identity",
		Long:    "Manage the workload identity",
		Aliases: []string{"sa"},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// run root command pre-run to registry the debug flag
			if cmd.Root() != nil && cmd.Root().PersistentPreRun != nil {
				cmd.Root().PersistentPreRun(cmd.Root(), args)
			}
			return authProvider.Validate()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Usage()
		},
	}

	// auth flags should be available for all subcommands
	authProvider.AddFlags(serviceAccountCmd.PersistentFlags())

	serviceAccountCmd.AddCommand(newCreateCmd(authProvider))
	serviceAccountCmd.AddCommand(newDeleteCmd(authProvider))

	return serviceAccountCmd
}
