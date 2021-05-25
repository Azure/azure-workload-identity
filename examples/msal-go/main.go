package main

import (
	"context"
	"fmt"
	"os"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/confidential"
	"k8s.io/klog/v2"
)

func main() {
	ctx := context.Background()

	// AAD Pod Identity webhook will inject the following env vars
	// 	AZURE_CLIENT_ID with the clientID set in the service account annotation
	// 	AZURE_TENANT_ID with the tenantID set in the service account annotation. If not defined, then
	// 		the tenantID provided via aad-pi-webhook-config for the webhook will be used.
	// 	TOKEN_FILE_PATH is the service account token path

	tokenFilePath := os.Getenv("TOKEN_FILE_PATH")
	tenantID := os.Getenv("AZURE_TENANT_ID")
	// if the service account wasn't annotated with the clientID, then this will be empty
	// ensure to set your clientID if not provided through annotation
	clientID := os.Getenv("AZURE_CLIENT_ID")

	// read the service account token from the filesystem
	signedAssertion, err := readJWTFromFS(tokenFilePath)
	if err != nil {
		klog.Fatalf("failed to read service account token: %v", err)
	}

	cred, err := confidential.NewCredFromAssertion(signedAssertion)
	if err != nil {
		klog.Fatalf("failed to create confidential creds: %v", err)
	}

	// create the confidential client to request an AAD token
	confidentialClientApp, err := confidential.New(clientID, cred,
		// TODO (aramase) remove query params after available in prod
		confidential.WithAuthority(fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/token?dc=ESTS-PUB-WUS2-AZ1-FD000-TEST1&fiextoidc=true", tenantID)))
	if err != nil {
		klog.Fatalf("failed to create confidential client app: %v", err)
	}

	scopes := []string{"https://graph.microsoft.com/.default"}
	result, err := confidentialClientApp.AcquireTokenByCredential(ctx, scopes)
	if err != nil {
		klog.Fatalf("failed to get token: %v", err)
	}
	klog.InfoS("successfully obtained the token", "token", result.AccessToken, "expiry", result.ExpiresOn)
}

func readJWTFromFS(tokenFilePath string) (string, error) {
	token, err := os.ReadFile(tokenFilePath)
	if err != nil {
		return "", err
	}
	return string(token), nil
}
