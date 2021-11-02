# Federated Identity Credential

<!-- toc -->

Not all service account tokens can be exchanged for a valid AAD token. A federated identity credential between an existing Kubernetes service account and an AAD application has to be created in advance. Refer to [doc][2] for an overview of federated identity credentials in Azure Active Directory.

> NOTE: Federated identity credentials are supported on applications only. A maximum of 20 federated identity credentials can be added per application object. The federated identity credentials API is not available in [national cloud deployments][3] - [source][2]

Currently, there are several ways to create a federated identity credential:

## Azure Workload Identity CLI

TODO

## Azure CLI

To create a federated identity credential, login to [Azure Cloud Shell][1] and export the following environment variables:

```bash
# Get the client and object ID of the AAD application
export APPLICATION_CLIENT_ID="..."
export APPLICATION_OBJECT_ID="$(az ad app show --id ${APPLICATION_CLIENT_ID} --query objectId -otsv)"
export SERVICE_ACCOUNT_ISSUER="..."
export SERVICE_ACCOUNT_NAMESPACE="..."
export SERVICE_ACCOUNT_NAME="..."
```

Add the federated identity credential:

```bash
cat <<EOF > body.json
{
  "name": "kubernetes-federated-identity",
  "issuer": "${SERVICE_ACCOUNT_ISSUER}",
  "subject": "system:serviceaccount:${SERVICE_ACCOUNT_NAMESPACE}:${SERVICE_ACCOUNT_NAME}",
  "description": "Kubernetes service account federated identity",
  "audiences": [
    "api://AzureADTokenExchange"
  ]
}
EOF

az rest --method POST --uri "https://graph.microsoft.com/beta/applications/${APPLICATION_OBJECT_ID}/federatedIdentityCredentials" --body @body.json
```

## Azure Portal UI

Coming soon.

[1]: https://portal.azure.com/#cloudshell/

[2]: https://docs.microsoft.com/en-us/graph/api/resources/federatedidentitycredentials-overview?view=graph-rest-beta&preserve-view=true

[3]: https://docs.microsoft.com/en-us/graph/deployments
