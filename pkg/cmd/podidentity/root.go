package podidentity

import "github.com/spf13/cobra"

func NewPodIdentityCmd() *cobra.Command {
	podIdentityCmd := &cobra.Command{
		Use:     "podidentity",
		Short:   "Configuration created for aad-pod-identity",
		Long:    "Configuration created for aad-pod-identity",
		Aliases: []string{"pi"},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// run root command pre-run to register the debug flag
			if cmd.Root() != nil && cmd.Root().PersistentPreRunE != nil {
				if err := cmd.Root().PersistentPreRunE(cmd.Root(), args); err != nil {
					return err
				}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Usage()
		},
	}

	podIdentityCmd.AddCommand(newDetectCmd())

	return podIdentityCmd
}
