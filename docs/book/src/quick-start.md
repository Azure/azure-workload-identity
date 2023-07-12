# Quick Start

<!-- toc -->

In this tutorial, we will cover the basics of how to use the webhook to acquire an Azure AD token to access a secret in an [Azure Key Vault][1].

> While this tutorial shows a 1:1 mapping between a Kubernetes service account and an Azure AD identity, it is possible to map:
> 1. Multiple Kubernetes service accounts to a single Azure AD identity. Refer to [FAQ][15] for more details.
> 2. Multiple Azure AD identities to a single Kubernetes service account. Refer to [FAQ][16] for more details.

Before we get started, ensure the following:

* Azure CLI version 2.40.0 or higher. Run `az --version` to verify.
*  You are logged in with the Azure CLI as a user.
   *  If you are logged in with a Service Principal, ensure that it has the correct [API permissions][14] enabled.
*  Your logged in account must have sufficient permissions to create applications and service principals or user-assigned managed identities in Azure AD.

## 1. Complete the installation guide

[Installation guide][13]. At this point, you should have already:
- installed the mutating admission webhook
- obtained your cluster's OIDC issuer URL
- [optional] installed the Azure AD Workload Identity CLI

## 2. Export environment variables

```bash
# environment variables for the Azure Key Vault resource
export KEYVAULT_NAME="azwi-kv-$(openssl rand -hex 2)"
export KEYVAULT_SECRET_NAME="my-secret"
export RESOURCE_GROUP="azwi-quickstart-$(openssl rand -hex 2)"
export LOCATION="westus2"

# environment variables for the AAD application
# [OPTIONAL] Only set this if you're using a Azure AD Application as part of this tutorial
export APPLICATION_NAME="<your application name>"

# environment variables for the user-assigned managed identity
# [OPTIONAL] Only set this if you're using a user-assigned managed identity as part of this tutorial
export USER_ASSIGNED_IDENTITY_NAME="<your user-assigned managed identity name>"

# environment variables for the Kubernetes service account & federated identity credential
export SERVICE_ACCOUNT_NAMESPACE="default"
export SERVICE_ACCOUNT_NAME="workload-identity-sa"
export SERVICE_ACCOUNT_ISSUER="<your service account issuer url>" # see section 1.1 on how to get the service account issuer url
```

## 3. Create an Azure Key Vault and secret

Create an Azure resource group:

```bash
az group create --name "${RESOURCE_GROUP}" --location "${LOCATION}"
```

Create an Azure Key Vault:

```bash
az keyvault create --resource-group "${RESOURCE_GROUP}" \
   --location "${LOCATION}" \
   --name "${KEYVAULT_NAME}"
```

Create a secret:

```bash
az keyvault secret set --vault-name "${KEYVAULT_NAME}" \
   --name "${KEYVAULT_SECRET_NAME}" \
   --value "Hello\!"
```

## 4. Create an AAD application or user-assigned managed identity and grant permissions to access the secret

<details>
<summary>Azure Workload Identity CLI</summary>

> NOTE: `azwi` currently only supports Azure AD Applications. If you want to use a user-assigned managed identity, skip this section and follow the steps in the Azure CLI section.

```bash
azwi serviceaccount create phase app --aad-application-name "${APPLICATION_NAME}"
```

<details>
<summary>Output</summary>

```
INFO[0000] No subscription provided, using selected subscription from Azure CLI: REDACTED
INFO[0005] [aad-application] created an AAD application  clientID=REDACTED name=azwi-test objectID=REDACTED
WARN[0005] --service-principal-name not specified, falling back to AAD application name
INFO[0005] [aad-application] created service principal   clientID=REDACTED name=azwi-test objectID=REDACTED
```

</details>

</details>

<br>

<details>
<summary>Azure CLI</summary>

```bash
# create an AAD application if using Azure AD Application for this tutorial
az ad sp create-for-rbac --name "${APPLICATION_NAME}"
```

```bash
# create a user-assigned managed identity if using user-assigned managed identity for this tutorial
az identity create --name "${USER_ASSIGNED_IDENTITY_NAME}" --resource-group "${RESOURCE_GROUP}"
```

</details>

Set access policy for the AAD application or user-assigned managed identity to access the keyvault secret:

If using Azure AD Application:

```bash
export APPLICATION_CLIENT_ID="$(az ad sp list --display-name "${APPLICATION_NAME}" --query '[0].appId' -otsv)"
az keyvault set-policy --name "${KEYVAULT_NAME}" \
  --secret-permissions get \
  --spn "${APPLICATION_CLIENT_ID}"
```

if using user-assigned managed identity:

```bash
export USER_ASSIGNED_IDENTITY_CLIENT_ID="$(az identity show --name "${USER_ASSIGNED_IDENTITY_NAME}" --resource-group "${RESOURCE_GROUP}" --query 'clientId' -otsv)"
export USER_ASSIGNED_IDENTITY_OBJECT_ID="$(az identity show --name "${USER_ASSIGNED_IDENTITY_NAME}" --resource-group "${RESOURCE_GROUP}" --query 'principalId' -otsv)"
az keyvault set-policy --name "${KEYVAULT_NAME}" \
  --secret-permissions get \
  --object-id "${USER_ASSIGNED_IDENTITY_OBJECT_ID}"
```

## 5. Create a Kubernetes service account

Create a Kubernetes service account and annotate it with the client ID of the AAD application we created in step 4:

<details>
<summary>Azure Workload Identity CLI</summary>

> NOTE: `azwi` currently only supports Azure AD Applications. If you want to use a user-assigned managed identity, skip this section and follow the steps in the `kubectl` section.

```bash
azwi serviceaccount create phase sa \
  --aad-application-name "${APPLICATION_NAME}" \
  --service-account-namespace "${SERVICE_ACCOUNT_NAMESPACE}" \
  --service-account-name "${SERVICE_ACCOUNT_NAME}"
```

<details>
<summary>Output</summary>

```
INFO[0000] No subscription provided, using selected subscription from Azure CLI: REDACTED
INFO[0002] [service-account] created Kubernetes service account  name=workload-identity-sa namespace=default
```

</details>

</details>

<br>

<details>
<summary>kubectl</summary>

```bash
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  annotations:
    azure.workload.identity/client-id: ${APPLICATION_CLIENT_ID:-$USER_ASSIGNED_IDENTITY_CLIENT_ID}
  name: ${SERVICE_ACCOUNT_NAME}
  namespace: ${SERVICE_ACCOUNT_NAMESPACE}
EOF
```

<details>
<summary>Output</summary>

```bash
serviceaccount/workload-identity-sa created
```

</details>

</details>

If the AAD application or user-assigned managed identity is not in the same tenant as the default tenant defined during installation, then annotate the service account with the application or user-assigned managed identity tenant ID:

```bash
kubectl annotate sa ${SERVICE_ACCOUNT_NAME} -n ${SERVICE_ACCOUNT_NAMESPACE} azure.workload.identity/tenant-id="${APPLICATION_OR_USER_ASSIGNED_IDENTITY_TENANT_ID}" --overwrite
```

## 6. Establish federated identity credential between the identity and the service account issuer & subject

<details>
<summary>Azure Workload Identity CLI</summary>

> NOTE: `azwi` currently only supports Azure AD Applications. If you want to use a user-assigned managed identity, skip this section and follow the steps in the `Azure CLI` section.

```bash
azwi serviceaccount create phase federated-identity \
  --aad-application-name "${APPLICATION_NAME}" \
  --service-account-namespace "${SERVICE_ACCOUNT_NAMESPACE}" \
  --service-account-name "${SERVICE_ACCOUNT_NAME}" \
  --service-account-issuer-url "${SERVICE_ACCOUNT_ISSUER}"
```

<details>
<summary>Output</summary>

```
INFO[0000] No subscription provided, using selected subscription from Azure CLI: REDACTED
INFO[0032] [federated-identity] added federated credential  objectID=REDACTED subject="system:serviceaccount:default:workload-identity-sa"
```

</details>

</details>

<br>

<details>
<summary>Azure CLI</summary>

If using Azure AD Application:

```bash
# Get the object ID of the AAD application
export APPLICATION_OBJECT_ID="$(az ad app show --id ${APPLICATION_CLIENT_ID} --query id -otsv)"
```

Add the federated identity credential:

```bash
cat <<EOF > params.json
{
  "name": "kubernetes-federated-credential",
  "issuer": "${SERVICE_ACCOUNT_ISSUER}",
  "subject": "system:serviceaccount:${SERVICE_ACCOUNT_NAMESPACE}:${SERVICE_ACCOUNT_NAME}",
  "description": "Kubernetes service account federated credential",
  "audiences": [
    "api://AzureADTokenExchange"
  ]
}
EOF

az ad app federated-credential create --id ${APPLICATION_OBJECT_ID} --parameters @params.json
```

If using user-assigned managed identity:

```bash
az identity federated-credential create \
  --name "kubernetes-federated-credential" \
  --identity-name "${USER_ASSIGNED_IDENTITY_NAME}" \
  --resource-group "${RESOURCE_GROUP}" \
  --issuer "${SERVICE_ACCOUNT_ISSUER}" \
  --subject "system:serviceaccount:${SERVICE_ACCOUNT_NAMESPACE}:${SERVICE_ACCOUNT_NAME}"
```

</details>

## 7. Deploy workload

Deploy a pod that references the service account created in the last step:

```bash
export KEYVAULT_URL="$(az keyvault show -g ${RESOURCE_GROUP} -n ${KEYVAULT_NAME} --query properties.vaultUri -o tsv)"
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: quick-start
  namespace: ${SERVICE_ACCOUNT_NAMESPACE}
  labels:
    azure.workload.identity/use: "true"
spec:
  serviceAccountName: ${SERVICE_ACCOUNT_NAME}
  containers:
    - image: ghcr.io/azure/azure-workload-identity/msal-go
      name: oidc
      env:
      - name: KEYVAULT_URL
        value: ${KEYVAULT_URL}
      - name: SECRET_NAME
        value: ${KEYVAULT_SECRET_NAME}
  nodeSelector:
    kubernetes.io/os: linux
EOF
```
Note: Newer version of the sample image will only need KEYVAULT_URL variable.

> Feel free to swap the msal-go example image above with a list of [language-specific examples](./topics/language-specific-examples/msal.md) we provide.

To check whether all properties are injected properly by the webhook:

```bash
kubectl describe pod quick-start
```

<details>
<summary>Output</summary>

You can verify the following injected properties in the output:

| Environment variable         | Description                                                                        |
| ---------------------------- | ---------------------------------------------------------------------------------- |
| `AZURE_AUTHORITY_HOST`       | The Azure Active Directory (AAD) endpoint.                                         |
| `AZURE_CLIENT_ID`            | The client ID of the AAD application or user-assigned managed identity.            |
| `AZURE_TENANT_ID`            | The tenant ID of the registered AAD application or user-assigned managed identity. |
| `AZURE_FEDERATED_TOKEN_FILE` | The path of the projected service account token file.                              |

<br/>

| Volume mount                                         | Description                                           |
| ---------------------------------------------------- | ----------------------------------------------------- |
| `/var/run/secrets/azure/tokens/azure-identity-token` | The path of the projected service account token file. |

<br/>

| Volume                 | Description                           |
| ---------------------- | ------------------------------------- |
| `azure-identity-token` | The projected service account volume. |

```log
Name:         quick-start
Namespace:    default
Priority:     0
Node:         k8s-agentpool1-38097163-vmss000002/10.240.0.34
Start Time:   Wed, 13 Oct 2021 15:49:25 -0700
Labels:       azure.workload.identity/use=true
Annotations:  <none>
Status:       Running
IP:           10.240.0.55
IPs:
  IP:  10.240.0.55
Containers:
  oidc:
    Container ID:   containerd://f425e89eef9aa3a62eb51a3daa5af8c06d8a59baa79c4e4dbb1887aea2647048
    Image:          ghcr.io/azure/azure-workload-identity/msal-go:latest
    Image ID:       ghcr.io/azure/azure-workload-identity/msal-go@sha256:84421aeea707ce66ade0891d9fcd3bb3f7bbd5dd3f810caced0acd315dcf8751
    Port:           <none>
    Host Port:      <none>
    State:          Running
      Started:      Wed, 13 Oct 2021 15:49:29 -0700
    Ready:          True
    Restart Count:  0
    Environment:
      KEYVAULT_URL:               ${KEYVAULT_URL}
      SECRET_NAME:                ${KEYVAULT_SECRET_NAME}
      AZURE_AUTHORITY_HOST:       (Injected by the webhook)
      AZURE_CLIENT_ID:            (Injected by the webhook)
      AZURE_TENANT_ID:            (Injected by the webhook)
      AZURE_FEDERATED_TOKEN_FILE: (Injected by the webhook)
    Mounts:
      /var/run/secrets/kubernetes.io/serviceaccount from kube-api-access-844ns (ro)
      /var/run/secrets/azure/tokens from azure-identity-token (ro) (Injected by the webhook)
Conditions:
  Type              Status
  Initialized       True
  Ready             True
  ContainersReady   True
  PodScheduled      True
Volumes:
  kube-api-access-844ns:
    Type:                    Projected (a volume that contains injected data from multiple sources)
    TokenExpirationSeconds:  3607
    ConfigMapName:           kube-root-ca.crt
    ConfigMapOptional:       <nil>
    DownwardAPI:             true
  azure-identity-token: (Injected by the webhook)
    Type:                    Projected (a volume that contains injected data from multiple sources)
    TokenExpirationSeconds:  3600
QoS Class:                   BestEffort
Node-Selectors:              kubernetes.io/os=linux
Tolerations:                 node.kubernetes.io/not-ready:NoExecute op=Exists for 300s
                             node.kubernetes.io/unreachable:NoExecute op=Exists for 300s
Events:
  Type    Reason     Age   From               Message
  ----    ------     ----  ----               -------
  Normal  Scheduled  19s   default-scheduler  Successfully assigned oidc/quick-start to k8s-agentpool1-38097163-vmss000002
  Normal  Pulling    18s   kubelet            Pulling image "ghcr.io/azure/azure-workload-identity/msal-go:latest"
  Normal  Pulled     16s   kubelet            Successfully pulled image "ghcr.io/azure/azure-workload-identity/msal-go:latest" in 1.987165801s
  Normal  Created    15s   kubelet            Created container oidc
  Normal  Started    15s   kubelet            Started container oidc
```

</details>

To verify that pod is able to get a token and access the secret from the Key Vault:

```bash
kubectl logs quick-start
```

<details>
<summary>Output</summary>

If successful, the log output would be similar to the following output:

```bash
I1013 22:49:29.872708       1 main.go:30] "successfully got secret" secret="Hello!"
```

</details>

## 8. Cleanup

```bash
kubectl delete pod quick-start
kubectl delete sa "${SERVICE_ACCOUNT_NAME}" --namespace "${SERVICE_ACCOUNT_NAMESPACE}"

az group delete --name "${RESOURCE_GROUP}"
# if you used Azure AD Application for tutorial, delete it by running the following command
az ad sp delete --id "${APPLICATION_CLIENT_ID}"
```

<!-- markdown-link-check-disable-next-line -->
[1]: https://azure.microsoft.com/services/key-vault/

[2]: https://kubernetes.io/docs/tasks/tools/

[3]: https://kind.sigs.k8s.io/docs/user/quick-start/#installation

[4]: https://www.docker.com/

<!-- markdown-link-check-disable-next-line -->
[5]: https://azure.microsoft.com/

[6]: https://docs.microsoft.com/en-us/cli/azure/install-azure-cli

[7]: https://github.com/Azure/azure-workload-identity/blob/1cb9d78159458b0c820c9c08fadf967833c8cdb4/deploy/azure-wi-webhook.yaml#L103-L104

[8]: https://portal.azure.com/#cloudshell/

[9]: ./topics/managed-clusters.md

[10]: ./topics/self-managed-clusters.md

[11]: ../installation.md#helm

[12]: ../installation.md#deployment-yaml

[13]: ./installation.md

[14]: ./known-issues.md#user-tried-to-log-in-to-a-device-from-a-platform-unknown-thats-currently-not-supported-through-conditional-access-policy

[15]: ./faq.md#how-to-federate-multiple-kubernetes-service-accounts-with-a-single-identity

[16]: ./faq.md#how-to-federate-multiple-identities-with-a-kubernetes-service-account
