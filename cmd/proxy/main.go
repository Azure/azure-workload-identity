package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"monis.app/mlog"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"github.com/Azure/azure-workload-identity/pkg/proxy"
	"github.com/Azure/azure-workload-identity/pkg/version"
)

var (
	proxyPort   int
	probe       bool
	logLevel    string
	versionInfo bool
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
	flag.BoolVar(&versionInfo, "version", false, "Print version information and exit")
	flag.Parse()

	if versionInfo {
		return version.PrintVersionToStdout()
	}

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

	ctx := withShutdownSignal(context.Background())

	p, err := proxy.NewProxy(proxyPort, mlog.New().WithName("proxy"))
	if err != nil {
		return fmt.Errorf("setup: failed to create proxy: %w", err)
	}
	if err := p.Run(ctx); err != nil {
		return fmt.Errorf("setup: failed to run proxy: %w", err)
	}

	return nil
}

// withShutdownSignal returns a copy of the parent context that will close if
// the process receives termination signals.
func withShutdownSignal(ctx context.Context) context.Context {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM, syscall.SIGINT, os.Interrupt)

	nctx, cancel := context.WithCancel(ctx)

	go func() {
		<-signalChan
		mlog.Info("received shutdown signal")
		cancel()
	}()
	return nctx
}
