//go:build e2e

package e2e

import (
	"flag"
	"os"
	"testing"

	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/e2e/framework/config"
)

func init() {
	flag.BoolVar(&arcCluster, "e2e.arc-cluster", false, "Running on an arc-enabled cluster")
	flag.StringVar(&tokenExchangeE2EImage, "e2e.token-exchange-image", "aramase/msal-go:v0.6.0", "The image to use for token exchange tests")
	flag.StringVar(&proxyInitImage, "e2e.proxy-init-image", "mcr.microsoft.com/oss/azure/workload-identity/proxy-init:v0.7.0", "The proxy-init image")
	flag.StringVar(&proxyImage, "e2e.proxy-image", "mcr.microsoft.com/oss/azure/workload-identity/proxy:v0.7.0", "The proxy image")
}

// handleFlags sets up all flags and parses the command line.
func handleFlags() {
	config.CopyFlags(config.Flags, flag.CommandLine)
	framework.RegisterCommonFlags(flag.CommandLine)
	framework.RegisterClusterFlags(flag.CommandLine)
	flag.Parse()
}

func TestMain(m *testing.M) {
	// Register test flags, then parse flags.
	handleFlags()
	framework.AfterReadingAllFlags(&framework.TestContext)

	os.Exit(m.Run())
}

func TestE2E(t *testing.T) {
	RunE2ETests(t)
}
