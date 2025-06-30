module github.com/Azure/azure-workload-identity/examples/identitybinding-msal-go

go 1.23.10

require (
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.18.2
	github.com/Azure/azure-sdk-for-go/sdk/keyvault/azsecrets v0.12.0
	k8s.io/klog/v2 v2.130.1
)

require (
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.11.1-0.20250811211210-f7ac5a70412a // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.11.2 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/keyvault/internal v0.7.1 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	golang.org/x/net v0.42.0 // indirect
	golang.org/x/text v0.27.0 // indirect
)

replace github.com/Azure/azure-sdk-for-go/sdk/azidentity => github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.11.1-0.20250811231238-0ec7258063bb
