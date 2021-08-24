package version

import (
	"fmt"
	"runtime"
)

var (
	// Vcs is the commit hash for the binary build
	Vcs string
	// BuildTime is the date for the binary build
	BuildTime string
	// BuildVersion is the azure-workload-identity version. Will be overwritten from build.
	BuildVersion string
)

// GetUserAgent returns a user agent of the format: azure-workload-identity/<version> (<goos>/<goarch>) <vcs>/<timestamp>
func GetUserAgent(component string) string {
	return fmt.Sprintf("azure-workload-identity/%s/%s (%s/%s) %s/%s", component, BuildVersion, runtime.GOOS, runtime.GOARCH, Vcs, BuildTime)
}
