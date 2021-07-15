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

	expected := fmt.Sprintf("aad-pod-managed-identity/%s (%s/%s) %s/%s", BuildVersion, runtime.GOOS, runtime.GOARCH, Vcs, BuildTime)
	actual := GetUserAgent()
	if !strings.EqualFold(expected, actual) {
		t.Fatalf("expected: %s, got: %s", expected, actual)
	}
}
