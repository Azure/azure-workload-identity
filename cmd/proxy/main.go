package main

import (
	"flag"
	"fmt"

	"monis.app/mlog"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"github.com/Azure/azure-workload-identity/pkg/proxy"
)

var (
	proxyPort int
	probe     bool
	logLevel  string
)

func main() {
	if err := mainErr(); err != nil {
		mlog.Fatal(err)
	}
}

func mainErr() error {
	defer mlog.Setup()()

	flag.IntVar(&proxyPort, "proxy-port", 8000, "Port for the proxy to listen on")
	flag.BoolVar(&probe, "probe", false, "Run a readyz probe on the proxy")
	flag.StringVar(&logLevel, "log-level", "",
		"In order of increasing verbosity: unset (empty string), info, debug, trace and all.")
	flag.Parse()

	if err := mlog.ValidateAndSetLogLevelAndFormatGlobally(signals.SetupSignalHandler(), mlog.LogSpec{
		Level:  mlog.LogLevel(logLevel),
		Format: mlog.FormatJSON,
	}); err != nil {
		return fmt.Errorf("invalid --log-level set: %w", err)
	}

	// when proxy is run with --probe, it will run a readyz probe on the proxy
	// this is used in the postStart lifecycle hook to verify the proxy is ready
	// to serve requests
	if probe {
		if err := proxy.Probe(proxyPort); err != nil {
			return fmt.Errorf("failed to probe: %w", err)
		}
		return nil
	}

	p, err := proxy.NewProxy(proxyPort, mlog.New().WithName("proxy"))
	if err != nil {
		return fmt.Errorf("setup: failed to create proxy: %w", err)
	}
	if err := p.Run(); err != nil {
		return fmt.Errorf("setup: failed to run proxy: %w", err)
	}

	return nil
}
