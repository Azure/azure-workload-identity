package main

import (
	"context"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	"k8s.io/klog/v2"
)

func main() {
	keyvaultURL := os.Getenv("KEYVAULT_URL")
	secretName := os.Getenv("SECRET_NAME")

	// create a secret client with the default credential
	// DefaultAzureCredential will use the environment variables injected by the Azure Workload Identity
	// mutating webhook to authenticate with Azure Key Vault.

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		klog.Fatal(err)
	}
	client, err := azsecrets.NewClient(keyvaultURL, cred, nil)
	if err != nil {
		klog.Fatal(err)
	}

	secretBundle, err := client.GetSecret(context.Background(), secretName, "", nil)
	if err != nil {
		klog.ErrorS(err, "failed to get secret from keyvault", "keyvault", keyvaultURL, "secretName", secretName)
		os.Exit(1)
	}
	klog.InfoS("successfully got secret", "secret", *secretBundle.Value)
}
