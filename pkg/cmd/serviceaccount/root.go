package serviceaccount

import (
	"github.com/spf13/cobra"

	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/auth"
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
			// run root command pre-run to register the debug flag
			if cmd.Root() != nil && cmd.Root().PersistentPreRunE != nil {
				if err := cmd.Root().PersistentPreRunE(cmd.Root(), args); err != nil {
					return err
				}
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
