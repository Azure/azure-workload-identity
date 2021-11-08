package main

import (
	"github.com/Azure/azure-workload-identity/pkg/cloud"

	"k8s.io/klog/v2"
)

var (
	testTenantID     string
	testClientID     string
	testClientSecret string
)

func init() {
	klog.InitFlags(nil)
}

func main() {
	app, err := cloud.GraphClient(testTenantID, testClientID, testClientSecret)
	if err != nil {
		klog.Fatalf("failed to create application: %v", err)
	}
	klog.InfoS("application name", "name", app.GetDisplayName())
	klog.InfoS("application app id", "app id", *app.GetAppId())
}
