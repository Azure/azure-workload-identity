# Azure AD Workload CLI (azwi)

`azwi` is a utility CLI that helps manage Azure AD Workload Identity and automate error-prone operations:

*   Generate the JWKS document from a list of public keys
*   Streamline the creation and deletion of the following resources:
    *   AAD applications
    *   Kubernetes service accounts
    *   Federated identities
    *   Azure role assignments

### GitHub Releases

You can download `azwi` from our [latest GitHub releases][1].

### Homebrew (MacOS only)

```bash
brew install Azure/azure-workload-identity/azwi
```

[1]: https://github.com/Azure/azure-workload-identity/releases
