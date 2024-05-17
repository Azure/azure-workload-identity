# Frequently Asked Questions

<!-- toc -->

## How does the azwi-cli differ from the azure-cli?

The azwi-cli tool is specific to the Azure Workload Identity support in Kubernetes to group several manual steps (e.g. the creation of federated identity credential, annotated service accounts, etc) and automate them. Comparing with the azure-cli, it does not have an official command to add/delete federated identity (configuring federated identity credential with `az rest` is available [here](https://docs.microsoft.com/en-us/azure/active-directory/develop/workload-identity-federation-create-trust))

Azure CLI and AKS are currently working on the above requirements, as well as an Azure CLI extension that natively integrate this project with AKS clusters.

## How is azure-workload-identity different from aad-pod-identity v2?

Azure Workload Identity is v2 of the AAD Pod Identity. AAD Pod Identity v2 was a placeholder name and is now rebranded as Azure Workload Identity.

## How to federate multiple identities with a Kubernetes service account?

It's possible to have a many-to-one relationship between multiple identities and a Kubernetes service account. For example, you can create multiple federated identity credentials that reference the same service account in your Kubernetes cluster.

The `azure.workload.identity/client-id` annotation in your service account represents the default identity's client ID used by an Azure Identity library during authentication. If you'd like to use a different identity, specify the client ID when creating the Azure Identity library's credential object.

For example, if you're using the Azure Identity for Python library's [`DefaultAzureCredential`](https://learn.microsoft.com/python/api/azure-identity/azure.identity.defaultazurecredential?view=azure-python) to authenticate your application, you can specify which identity to use by adding the [`workload_identity_client_id`](https://learn.microsoft.com/python/api/azure-identity/azure.identity.defaultazurecredential?view=azure-python#keyword-only-parameters) parameter to the `DefaultAzureCredential` constructor.

## How to federate multiple Kubernetes service accounts with a single identity?

It is possible to have a many-to-one relationship between a single identity and multiple Kubernetes service accounts, i.e. you can create a federated identity credential for every <service account namespace, service account name, OIDC issuer> tuple that you want to federate with the same identity.

## Is there a propagation delay after creating a federated identity credential?

It takes a few seconds for the federated identity credential to be propagated after being initially added. If a token request is made immediately after adding the federated identity credential, it **might** lead to failure for a couple of minutes as the cache is populated in the directory with old data. To avoid this issue, you can add a slight delay after adding the federated identity credential.

## What is the Azure Workload Identity release schedule?

Currently, we release on a monthly basis, targeting the last week of the month.

## What permissions are required to create a federated identity credential for Azure AD Application?

One of the following roles is required:

- [Application Administrator](https://learn.microsoft.com/en-us/azure/active-directory/roles/permissions-reference#application-administrator)
- [Application Developer](https://learn.microsoft.com/en-us/azure/active-directory/roles/permissions-reference#application-developer)
- [Cloud Application Administrator](https://docs.microsoft.com/en-us/azure/role-based-access-control/built-in-roles#cloud-application-administrator)
- [Application Owner](https://docs.microsoft.com/en-us/azure/role-based-access-control/built-in-roles#application-owner)

Required permissions to create/update/delete federated identity credential:

- [`microsoft.directory/applications/credentials/update`](https://learn.microsoft.com/en-us/azure/active-directory/roles/custom-available-permissions#microsoftdirectoryapplicationscredentialsupdate)

## What permissions are required to create a federated identity credential for user-assigned managed identity?

One of the following roles is required:

- [Owner](https://learn.microsoft.com/en-us/azure/role-based-access-control/built-in-roles#owner)
- [Contributor](https://learn.microsoft.com/en-us/azure/role-based-access-control/built-in-roles#contributor)

Required permissions to create/update/delete federated identity credential:

- `Microsoft.ManagedIdentity/userAssignedIdentities/federatedIdentityCredentials/write`
- `Microsoft.ManagedIdentity/userAssignedIdentities/federatedIdentityCredentials/delete`

## How to migrate from `aad-pod-identity`?

You can refer to [this link](https://learn.microsoft.com/azure/aks/workload-identity-migrate-from-pod-identity) for more information.

tl;dr:

1. If you use `DefaultAzureCredential` in your workload, you can update the Azure Identity client library version to the latest that supports Workload Identity (ref: [link](https://azure.github.io/azure-workload-identity/docs/topics/language-specific-examples/azure-identity-sdk.html)).
2. If you still have workloads that get managed identity token from IMDS, you can annotate the pod to get a proxy sidecar injected, that'll do the token exchange with the new flow.

## Does Workload Identity work in disconnected environments?

No, Workload Identity doesn't work in completely disconnected environments. The AAD token is valid for 24 hours, so the workload needs to be connected to the network at least once every 24 hours to refresh the token.
