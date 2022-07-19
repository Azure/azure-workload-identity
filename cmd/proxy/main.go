package main

import (
	"flag"
	"os"

	"github.com/Azure/azure-workload-identity/pkg/logger"
	"github.com/Azure/azure-workload-identity/pkg/proxy"

	"k8s.io/klog/v2"
)

var (
	proxyPort int
	probe     bool
)

func init() {
	klog.InitFlags(nil)
}

func main() {
	logger := logger.New()
	logger.AddFlags()

	flag.IntVar(&proxyPort, "proxy-port", 8000, "Port for the proxy to listen on")
	flag.BoolVar(&probe, "probe", false, "Run a readyz probe on the proxy")
	flag.Parse()

	// when proxy is run with --probe, it will run a readyz probe on the proxy
	// this is used in the postStart lifecycle hook to verify the proxy is ready
	// to serve requests
	if probe {
		setupLog := logger.Get().WithName("probe")
		if err := proxy.Probe(proxyPort); err != nil {
			setupLog.Error(err, "failed to probe")
			os.Exit(1)
		}
		os.Exit(0)
	}

	setupLog := logger.Get().WithName("setup")
	p, err := proxy.NewProxy(proxyPort, logger.Get().WithName("proxy"))
	if err != nil {
		setupLog.Error(err, "failed to create proxy")
		os.Exit(1)
	}
	if err := p.Run(); err != nil {
		setupLog.Error(err, "failed to run proxy")
		os.Exit(1)
	}
}
