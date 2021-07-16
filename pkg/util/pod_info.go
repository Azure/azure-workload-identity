package util

import "os"

// GetNamespace returns the namespace for aad-pi-webhook
func GetNamespace() string {
	ns, found := os.LookupEnv("POD_NAMESPACE")
	if !found {
		return "aad-pi-webhook-system"
	}
	return ns
}
