# Azure AD Workload CLI (azwi)

`azwi` is a utility CLI that helps manage Azure AD Workload Identity and automate error-prone operations:

*   Generate the JWKS document from a list of public keys
*   Streamline the creation and deletion of the following resources:
    *   AAD applications
    *   Kubernetes service accounts
    *   Federated identities
    *   Azure role assignments

### `go install`

```bash
go install github.com/Azure/azure-workload-identity/cmd/azwi@v0.7.0
```

### Homebrew (MacOS only)

```bash
brew install Azure/azure-workload-identity/azwi
```
