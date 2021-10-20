package version

import (
	"testing"

	"github.com/Azure/azure-workload-identity/pkg/version"
)

func TestGetVersion(t *testing.T) {
	version.BuildVersion = "v0.6.0"
	version.Vcs = "1ebf89c"

	expectedVersion := "Version: v0.6.0\nGitCommit: 1ebf89c"
	if getVersion() != expectedVersion {
		t.Errorf("getVersion() = %s, want %s", getVersion(), expectedVersion)
	}
}
