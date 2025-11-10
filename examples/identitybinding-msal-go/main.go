package main

import (
	"context"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azsecrets"
	"k8s.io/klog/v2"
)

func createCredential() (azcore.TokenCredential, error) {
	return azidentity.NewWorkloadIdentityCredential(&azidentity.WorkloadIdentityCredentialOptions{
		EnableAzureTokenProxy: true,
	})
}

func main() {
	keyvaultURL := os.Getenv("KEYVAULT_URL")
	if keyvaultURL == "" {
		klog.Fatal("KEYVAULT_URL environment variable is not set")
	}
	secretName := os.Getenv("SECRET_NAME")
	if secretName == "" {
		klog.Fatal("SECRET_NAME environment variable is not set")
	}

	var cred azcore.TokenCredential
	cred, err := createCredential()
	if err != nil {
		klog.Fatal(err)
	}

	// initialize keyvault client
	client, err := azsecrets.NewClient(keyvaultURL, cred, &azsecrets.ClientOptions{})
	if err != nil {
		klog.Fatal(err)
	}

	for {
		secretBundle, err := client.GetSecret(context.Background(), secretName, "", nil)
		if err != nil {
			klog.ErrorS(err, "failed to get secret from keyvault", "keyvault", keyvaultURL, "secretName", secretName)
			os.Exit(1)
		}
		klog.InfoS("successfully got secret", "secret", *secretBundle.Value)

		// wait for 60 seconds before polling again
		time.Sleep(60 * time.Second)
	}
}
