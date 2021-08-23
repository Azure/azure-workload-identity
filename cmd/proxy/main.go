package main

import (
	"flag"

	"github.com/Azure/azure-workload-identity/pkg/proxy"

	"k8s.io/klog/v2"
)

var (
	proxyPort int
)

func main() {
	klog.InitFlags(nil)
	defer klog.Flush()

	flag.IntVar(&proxyPort, "proxy-port", 8000, "Port for the proxy to listen on")
	flag.Parse()

	p, err := proxy.NewProxy(proxyPort)
	if err != nil {
		klog.Fatalf("failed to get proxy, error: %+v", err)
	}
	if err = p.Run(); err != nil {
		klog.Fatalf("failed to run proxy, error: %+v", err)
	}
}
