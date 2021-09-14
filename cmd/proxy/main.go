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
)

func init() {
	klog.InitFlags(nil)
}

func main() {
	logger := logger.New()
	logger.AddFlags()

	flag.IntVar(&proxyPort, "proxy-port", 8000, "Port for the proxy to listen on")
	flag.Parse()

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
