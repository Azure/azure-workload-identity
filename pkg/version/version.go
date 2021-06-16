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
	// BuildVersion is the aad-pod-managed-identity version. Will be overwritten from build.
	BuildVersion string
)

// GetUserAgent returns a user agent of the format: aad-pod-managed-identity/<version> (<goos>/<goarch>) <vcs>/<timestamp>
func GetUserAgent(component string) string {
	return fmt.Sprintf("aad-pod-managed-identity/%s/%s (%s/%s) %s/%s", component, BuildVersion, runtime.GOOS, runtime.GOARCH, Vcs, BuildTime)
}
