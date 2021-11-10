package main

import (
	"context"

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
	c, err := cloud.GraphClient(testTenantID, testClientID, testClientSecret)
	if err != nil {
		klog.Fatalf("failed to create client: %w", err)
	}
	client := &cloud.AzureClient{
		GraphServiceClient: c,
	}
	app, err := client.GetApplication(context.Background(), "azwi-e2e-app-b886")
	if err != nil {
		klog.ErrorS(err, "failed to get application")
		return
	}
	klog.InfoS("application", "name", *app.GetDisplayName(), "id", *app.GetId(), "appID", *app.GetAppId())
}
