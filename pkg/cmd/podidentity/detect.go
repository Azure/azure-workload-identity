package podidentity

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type detectCmd struct {
	namespace string
	outputDir string
}

func newDetectCmd() *cobra.Command {
	detectCmd := &detectCmd{}

	cmd := &cobra.Command{
		Use:   "detect",
		Short: "Detect the existing aad-pod-identity configuration",
		Long:  "This command will detect the existing aad-pod-identity configuration and generate a sample configuration file for migration to workload identity",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return detectCmd.validate()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return detectCmd.run()
		},
	}

	f := cmd.Flags()
	f.StringVar(&detectCmd.namespace, "namespace", "default", "Namespace to detect the configuration")
	f.StringVar(&detectCmd.outputDir, "output-dir", "", "Output directory to write the configuration files")

	_ = cmd.MarkFlagRequired("output-dir")

	return cmd
}

func (dc *detectCmd) validate() error {
	return nil
}

func (dc *detectCmd) run() error {
	log.Debugf("detecting aad-pod-identity configuration in namespace: %s", dc.namespace)

	// Implementing force namespaced mode for now
	// 1. Get AzureIdentityBinding in the namespace
	// 2. Get AzureIdentity referenced by AzureIdentityBinding and store in map with aadpodidbinding label value as key and AzureIdentity as value
	// 3. Get all pods in the namespace that have aadpodidbinding label
	// 4. For each pod, check if there is an owner reference (deployment, statefulset, cronjob, job, daemonset, replicaset, replicationcontroller)
	// 5. If there is an owner reference, get the owner reference object and add to map with aadpodidbinding label value as key and owner reference as value
	// 6. If no owner reference, then assume it's a static pod and add to map with aadpodidbinding label value as key and pod as value
	// 7. Loop through the first map and generate new config file for each owner reference and service account
	//    1. If owner using service account, get service account and generate config file with it
	//    2. If owner doesn't use service account, generate a new service account yaml file with owner name as service account name

	return nil
}
