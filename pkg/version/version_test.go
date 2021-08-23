package version

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
)

func TestGetUserAgent(t *testing.T) {
	BuildTime = "Now"
	BuildVersion = "version"
	Vcs = "hash"

	expected := fmt.Sprintf("azure-workload-identity/webhook/%s (%s/%s) %s/%s", BuildVersion, runtime.GOOS, runtime.GOARCH, Vcs, BuildTime)
	actual := GetUserAgent("webhook")
	if !strings.EqualFold(expected, actual) {
		t.Fatalf("expected: %s, got: %s", expected, actual)
	}
}
