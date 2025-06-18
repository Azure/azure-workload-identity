package version

import (
	"bytes"
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

func TestPrintVersion(t *testing.T) {
	BuildTime = "Now"
	BuildVersion = "version"
	Vcs = "hash"

	var buf bytes.Buffer
	err := printVersion(&buf)
	if err != nil {
		t.Fatalf("PrintVersion() failed: %v", err)
	}

	out := strings.TrimSpace(buf.String())
	expected := fmt.Sprintf(`{"buildVersion":"version","gitCommit":"hash","buildDate":"Now","goVersion":"%s","platform":"%s/%s"}`, runtime.Version(), runtime.GOOS, runtime.GOARCH)
	if out != expected {
		t.Fatalf("expected %q, got %q", expected, out)
	}
}
