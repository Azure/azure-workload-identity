# Federated Identity Credential

<!-- toc -->

Not all service account tokens can be exchanged for a valid AAD token. A federated identity credential between an existing Kubernetes service account and an AAD application has to be created in advance. Refer to [doc][2] for an overview of federated identity credentials in Azure Active Directory.

> NOTE: Federated identity credentials are supported on **AAD applications** only. A maximum of **20** federated identity credentials can be added per AAD application object. The federated identity credentials API is not available in [national cloud deployments][3] - [source][2]

Export the following environment variables:

```bash
export APPLICATION_NAME="<your application name>"
export SERVICE_ACCOUNT_NAMESPACE="..."
export SERVICE_ACCOUNT_NAME="..."
export SERVICE_ACCOUNT_ISSUER="..." # see section 1.1 on how to get the service account issuer url
```

Currently, there are several ways to create and delete a federated identity credential:

## Azure Workload Identity CLI

To create a federated identity credential:

```bash
azwi serviceaccount create phase federated-identity \
  --aad-application-name "${APPLICATION_NAME}" \
  --service-account-namespace "${SERVICE_ACCOUNT_NAMESPACE}" \
  --aad-application-name "${SERVICE_ACCOUNT_NAME}" \
  --service-account-issuer-url "${SERVICE_ACCOUNT_ISSUER}"
```

To delete a federated identity credential:

```bash
azwi serviceaccount delete phase federated-identity \
  --aad-application-name "${APPLICATION_NAME}" \
  --service-account-namespace "${SERVICE_ACCOUNT_NAMESPACE}" \
  --aad-application-name "${SERVICE_ACCOUNT_NAME}" \
  --service-account-issuer-url "${SERVICE_ACCOUNT_ISSUER}"
```

## Azure CLI

To create a federated identity credential, login to [Azure Cloud Shell][1] and run the following commands:

```bash
# Get the client and object ID of the AAD application
export APPLICATION_CLIENT_ID="$(az ad sp list --display-name "${APPLICATION_NAME}" --query '[0].appId' -otsv)"
export APPLICATION_OBJECT_ID="$(az ad app show --id "${APPLICATION_CLIENT_ID}" --query id -otsv)"

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

To delete a federated identity credential, the federated identity credential ID needs to be obtained with the following command:

```bash
az rest --method GET --uri "https://graph.microsoft.com/beta/applications/${APPLICATION_OBJECT_ID}/federatedIdentityCredentials"
```

Select the desired ID of the federated identity credential and run the following command:

```bash
az rest --method DELETE --uri "https://graph.microsoft.com/beta/applications/${APPLICATION_OBJECT_ID}/federatedIdentityCredentials/${FIC_ID}"
```

## Azure Portal UI

1. Sign in to the [Azure portal](https://portal.azure.com). 
1. Go to **App registrations** and open the app you want to configure.
1. Go to **Certificates and secrets**. 
1. In the **Federated credentials** tab, select **Add credential**. The **Add a credential** blade opens.
1. In the **Federated credential scenario** drop-down box select **Kubernetes accessing Azure resources**.
1. Specify the **Cluster issuer URL**.
1. Specify the **Namespace**.
1. Specify the **Service account name**.
1. The **Subject identifier** field autopopulate based on the values you entered.
1. Add a **Name** for the federated credential.
1. Click **Add** to configure the federated credential.

![Screenshot showing Azure Portal app registration federated credential screen for Kubernetes scenario](../images/azure-portal-federated-credential-kubernetes.png)

[1]: https://portal.azure.com/#cloudshell/

[2]: https://docs.microsoft.com/en-us/graph/api/resources/federatedidentitycredentials-overview?view=graph-rest-beta&preserve-view=true

[3]: https://docs.microsoft.com/en-us/graph/deployments
