package version

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
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

// PrintVersionToStdout prints the current driver version to stdout
func PrintVersionToStdout() error {
	return printVersion(os.Stdout)
}

func printVersion(w io.Writer) error {
	pv := struct {
		BuildVersion string `json:"buildVersion"`
		GitCommit    string `json:"gitCommit"`
		BuildDate    string `json:"buildDate"`
		GoVersion    string `json:"goVersion"`
		Platform     string `json:"platform"`
	}{
		BuildDate:    BuildTime,
		BuildVersion: BuildVersion,
		GitCommit:    Vcs,
		GoVersion:    runtime.Version(),
		Platform:     fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}

	res, err := json.Marshal(pv)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(w, "%s\n", res)
	return err
}
