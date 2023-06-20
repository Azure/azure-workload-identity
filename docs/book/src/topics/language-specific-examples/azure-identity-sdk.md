# Azure Identity SDK

In the Azure Identity client libraries, choose one of the following approaches:

- Use `DefaultAzureCredential`, which will attempt to use the `WorkloadIdentityCredential`.
- Create a `ChainedTokenCredential` instance that includes `WorkloadIdentityCredential`.
- Use `WorkloadIdentityCredential` directly.

The following client libraries are the **minimum** version required and recommended:

* `Minimum Recommended Version` - The version of the client library that is recommended for use with Workload Identity.
  * This version is the minimum version that includes the `WorkloadIdentityCredential` in addition to the `DefaultAzureCredential`.
* `Minimum Version Required for Workload Identity` - The minimum version of the client library that will work with Workload Identity (i.e. the version that supports workload identity with the `DefaultAzureCredential`).

| Language              | Library                                                               | Minimum Recommended Version                                                                               | Minimum Version Required for Workload Identity                                                          |
| --------------------- | --------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------- |
| Go                    | [azure-sdk-for-go](https://github.com/Azure/azure-sdk-for-go)         | [sdk/azidentity/v1.3.0](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azidentity@v1.3.0)       | [sdk/azidentity/v1.3.0](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azidentity@v1.3.0)     |
| C#                    | [azure-sdk-for-net](https://github.com/Azure/azure-sdk-for-net)       | [Azure.Identity_1.9.0](https://github.com/Azure/azure-sdk-for-net/releases/tag/Azure.Identity_1.9.0)      | [Azure.Identity_1.5.0](https://github.com/Azure/azure-sdk-for-net/releases/tag/Azure.Identity_1.5.0)    |
| JavaScript/TypeScript | [azure-sdk-for-js](https://github.com/Azure/azure-sdk-for-js)         | [@azure/identity_3.2.0](https://github.com/Azure/azure-sdk-for-js/releases/tag/@azure/identity_3.2.0)     | [@azure/identity_2.0.0](https://github.com/Azure/azure-sdk-for-js/releases/tag/@azure/identity_2.0.0)   |
| Python                | [azure-sdk-for-python](https://github.com/Azure/azure-sdk-for-python) | [azure-identity_1.13.0](https://github.com/Azure/azure-sdk-for-python/releases/tag/azure-identity_1.13.0) | [azure-identity_1.7.0](https://github.com/Azure/azure-sdk-for-python/releases/tag/azure-identity_1.7.0) |
| Java                  | [azure-sdk-for-java](https://github.com/Azure/azure-sdk-for-java)     | [azure-identity_1.9.0](https://github.com/Azure/azure-sdk-for-java/releases/tag/azure-identity_1.9.0)     | [azure-identity_1.4.0](https://github.com/Azure/azure-sdk-for-java/releases/tag/azure-identity_1.4.0)   |

## Examples

### Using `DefaultAzureCredential`

| Language              | Library                                                               | Example                                                                                           |
| --------------------- | --------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------- |
| Go                    | [azure-sdk-for-go](https://github.com/Azure/azure-sdk-for-go)         | [Link](https://github.com/Azure/azure-workload-identity/tree/main/examples/azure-identity/go)     |
| Python                | [azure-sdk-for-python](https://github.com/Azure/azure-sdk-for-python) | [Link](https://github.com/Azure/azure-workload-identity/tree/main/examples/azure-identity/python) |
| JavaScript/TypeScript | [azure-sdk-for-js](https://github.com/Azure/azure-sdk-for-js)         | [Link](https://github.com/Azure/azure-workload-identity/tree/main/examples/azure-identity/node)   |
| C#                    | [azure-sdk-for-net](https://github.com/Azure/azure-sdk-for-net)       | [Link](https://github.com/Azure/azure-workload-identity/tree/main/examples/azure-identity/dotnet) |
| Java                  | [azure-sdk-for-java](https://github.com/Azure/azure-sdk-for-java)     | [Link](https://github.com/Azure/azure-workload-identity/tree/main/examples/azure-identity/java)   |
