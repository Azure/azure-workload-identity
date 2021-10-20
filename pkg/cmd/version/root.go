package version

import (
	"fmt"

	"github.com/Azure/azure-workload-identity/pkg/version"

	"github.com/spf13/cobra"
)

// NewVersionCmd returns a new version command
func NewVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version of azwi",
		Long:  "Print the version of azwi",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(getVersion())
		},
	}

	return cmd
}

func getVersion() string {
	return fmt.Sprintf("Version: %s\nGitCommit: %s", version.BuildVersion, version.Vcs)
}
