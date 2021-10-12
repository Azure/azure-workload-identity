# Federated Identity

<!-- toc -->

Not all service account tokens can be exchanged for a valid AAD token. A federated identity between an existing Kubernetes service account and an AAD application has to be established in advance.

Currently, there are several ways to create a federated identity:

## Azure Workload Identity CLI

TODO

## Azure CLI

To create a federated identity, login to [Azure Cloud Shell][1] and export the following environment variables:

```bash
# Get the client and object ID of the AAD application
export APPLICATION_CLIENT_ID="..."
export APPLICATION_OBJECT_ID="$(az ad app show --id ${APPLICATION_CLIENT_ID} --query objectId -otsv)"
export SERVICE_ACCOUNT_ISSUER="..."
export SERVICE_ACCOUNT_NAMESPACE="..."
export SERVICE_ACCOUNT_NAME="..."
```

Add the federated identity:

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
