package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
	"github.com/Azure/go-autorest/autorest"
	"k8s.io/klog/v2"
)

func main() {
	keyvaultURL := os.Getenv("KEYVAULT_URL")
        if keyvaultURL == "" {
                keyvaultName := os.Getenv("KEYVAULT_NAME")
		// fallback to use global cloud
                keyvaultURL = fmt.Sprintf("https://%s.vault.azure.net/", keyvaultName)
        }
        secretName := os.Getenv("SECRET_NAME")

	// initialize keyvault client with custom authorizer
	kvClient := keyvault.New()
	kvClient.Authorizer = autorest.NewBearerAuthorizerCallback(nil, clientAssertionBearerAuthorizerCallback)

	for {
		secretBundle, err := kvClient.GetSecret(context.Background(), keyvaultURL, secretName, "")
		if err != nil {
			klog.ErrorS(err, "failed to get secret from keyvault", "keyvault", keyvaultName, "secretName", secretName)
			os.Exit(1)
		}
		klog.InfoS("successfully got secret", "secret", *secretBundle.Value)

		// wait for 60 seconds before polling again
		time.Sleep(60 * time.Second)
	}
}
