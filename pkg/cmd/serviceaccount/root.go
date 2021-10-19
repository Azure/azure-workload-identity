package serviceaccount

import (
	"github.com/spf13/cobra"
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
