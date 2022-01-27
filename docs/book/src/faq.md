# Frequently Asked Questions

## Why is managed identity not supported by Azure Workload Identity?

This is due to the limitation of Azure AD, rather than the Azure Workload Identity project. Azure AD is currently working on enabling workload identity federation for managed identity and Azure Workload Identity will support the feature as soon as it is available.

## How does the azwi-cli differ from the azure-cli?

The azwi-cli tool is specific to the Azure Workload Identity support in Kubernetes to group several manual steps (e.g. the creation of federated identity credential, annotated service accounts, etc) and automate them. Comparing with the azure-cli, it does not have an official command to add/delete federated identity (configuring federated identity credential with `az rest` is available [here](https://docs.microsoft.com/en-us/azure/active-directory/develop/workload-identity-federation-create-trust))

Azure CLI and AKS are currently working on the above requirements, as well as an Azure CLI extension that natively integrate this project with AKS clusters.

## How is azure-workload-identity different from aad-pod-identity v2?

Azure Workload Identity is v2 of the AAD Pod Identity. AAD Pod Identity v2 was a placeholder name and is now rebranded as Azure Workload Identity.
