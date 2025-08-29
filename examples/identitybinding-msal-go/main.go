package main

import (
	"context"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azsecrets"
	"k8s.io/klog/v2"
)

func createCredentialFromEnv() (azcore.TokenCredential, error) {
	// Azure AD Workload Identity webhook will inject the following env vars
	// 	AZURE_CLIENT_ID with the clientID set in the service account annotation
	// 	AZURE_FEDERATED_TOKEN_FILE is the service account token path
	// 	AZURE_KUBERNETES_TOKEN_PROXY is the identity binding token endpoint
	// 	AZURE_KUBERNETES_SNI_NAME is the SNI name for token endpoint
	// 	AZURE_KUBERNETES_CA_FILE is the CA file for the token endpoint
	clientID := os.Getenv("AZURE_CLIENT_ID")
	tokenFilePath := os.Getenv("AZURE_FEDERATED_TOKEN_FILE")
	tokenEndpoint := os.Getenv("AZURE_KUBERNETES_TOKEN_PROXY")
	sni := os.Getenv("AZURE_KUBERNETES_SNI_NAME")
	caFile := os.Getenv("AZURE_KUBERNETES_CA_FILE")

	if clientID == "" {
		klog.Fatal("AZURE_CLIENT_ID environment variable is not set")
	}
	if tokenFilePath == "" {
		klog.Fatal("AZURE_FEDERATED_TOKEN_FILE environment variable is not set")
	}
	if tokenEndpoint == "" {
		klog.Fatal("AZURE_KUBERNETES_TOKEN_PROXY environment variable is not set")
	}
	if sni == "" {
		klog.Fatal("AZURE_KUBERNETES_SNI_NAME environment variable is not set")
	}
	if caFile == "" {
		klog.Fatal("AZURE_KUBERNETES_CA_FILE environment variable is not set")
	}

	cred, err := newClientAssertionCredential(clientID, tokenEndpoint, sni, caFile, tokenFilePath, nil)
	if err != nil {
		return nil, err
	}
	return cred, nil
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
	cred, err := createCredentialFromEnv()
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
