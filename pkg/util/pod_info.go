package util

import "os"

// GetNamespace returns the namespace for azure-wi-webhook
func GetNamespace() string {
	ns, found := os.LookupEnv("POD_NAMESPACE")
	if !found {
		return "azure-workload-identity-system"
	}
	return ns
}
