# Quick Start

<!-- toc -->

In this tutorial, we will cover the basics of how to use the Azure AD Workload Identity webhook to acquire a token to access a secret in an [Azure Key Vault][1].

## Prerequisites

*   [kubectl][2]
*   [Microsoft Azure][5] account
*   [Azure CLI][6]
*   A Kubernetes cluster, with service account issuer URL and signing key pair set up
    *   Check out [this section][9] if you are planning to use a managed Kubernetes cluster
    *   Check out [this section][10] if you are planning to use a self-managed Kubernetes cluster

## 1. Install the Azure AD Workload Identity webhook

*   [Helm][11]
*   [Deployment YAML][12]

## 2. Create an Azure Key Vault and secret

Create an Azure resource group:

```bash
export RESOURCE_GROUP="azure-wi-webhook-test"
export LOCATION="westus2"
az group create --name "${RESOURCE_GROUP}" --location "${LOCATION}"
```

Create an Azure Key Vault:

```bash
export KEYVAULT_NAME="azure-wi-webhook-test-$(openssl rand -hex 1)"
export KEYVAULT_SECRET_NAME="my-secret"
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

## 3. Create an AAD application and grant permissions to access the secret

```bash
export APPLICATION_CLIENT_ID="$(az ad sp create-for-rbac --skip-assignment --name https://test-sp --query appId -otsv)"
```

Set access policy for the AAD application to access the keyvault secret:

```bash
az keyvault set-policy --name "${KEYVAULT_NAME}" \
  --secret-permissions get \
  --spn "${APPLICATION_CLIENT_ID}"
```

</details>

## 4. Create a Kubernetes service account

Create a Kubernetes service account and associate it with the AAD application we created in step 3:

```bash
export SERVICE_ACCOUNT_NAMESPACE="default"
export SERVICE_ACCOUNT_NAME="workload-identity-sa"

cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  annotations:
    azure.workload.identity/client-id: ${APPLICATION_CLIENT_ID}
  labels:
    azure.workload.identity/use: "true"
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

If the AAD application is not in the same tenant as the Kubernetes cluster, then annotate the service account with the application tenant ID.

```bash
kubectl annotate sa workload-identity-sa azure.workload.identity/tenant-id="${APPLICATION_TENANT_ID}" --overwrite
```

## 5. Establish trust between the AAD application and the service account issuer & subject

Login to [Azure Cloud Shell][8] and run the following commands:

```bash
# Get the object ID of the AAD application
export APPLICATION_CLIENT_ID=<Azure AD Application ID from previous steps>
export APPLICATION_OBJECT_ID="$(az ad app show --id ${APPLICATION_CLIENT_ID} --query objectId -otsv)"
export SERVICE_ACCOUNT_ISSUER="<Your Service Account Issuer URL>"
export SERVICE_ACCOUNT_NAME="workload-identity-sa"
export SERVICE_ACCOUNT_NAMESPACE="default"
```

Add the federated identity credential:

```bash
cat <<EOF > body.json
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

az rest --method POST --uri "https://graph.microsoft.com/beta/applications/${APPLICATION_OBJECT_ID}/federatedIdentityCredentials" --body @body.json
```

## 6. Deploy workload

Deploy a pod that references the service account created in the last step:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: quick-start
  namespace: ${SERVICE_ACCOUNT_NAMESPACE}
spec:
  serviceAccountName: ${SERVICE_ACCOUNT_NAME}
  containers:
    - image: aramase/msal-go:v0.6.0
      imagePullPolicy: IfNotPresent
      name: oidc
      env:
      - name: KEYVAULT_NAME
        value: ${KEYVAULT_NAME}
      - name: SECRET_NAME
        value: ${KEYVAULT_SECRET_NAME}
  nodeSelector:
    kubernetes.io/os: linux
EOF
```

<details>
<summary>Output</summary>

```bash
pod/quick-start created
```

</details>

To check whether all properties are injected properly by the webhook:

```bash
kubectl describe pod quick-start
```

<details>
<summary>Output</summary>

You can verify the following injected properties in the output:

| Environment variable         | Description                                           |
| ---------------------------- | ----------------------------------------------------- |
| `AZURE_AUTHORITY_HOST`       | The Azure Active Directory (AAD) endpoint.            |
| `AZURE_CLIENT_ID`            | The client ID of the AAD application.                 |
| `AZURE_TENANT_ID`            | The tenant ID of the registered AAD application.      |
| `AZURE_FEDERATED_TOKEN_FILE` | The path of the projected service account token file. |

<br/>

| Volume mount                                   | Description                                           |
| ---------------------------------------------- | ----------------------------------------------------- |
| `/var/run/secrets/tokens/azure-identity-token` | The path of the projected service account token file. |

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
Labels:       <none>
Annotations:  <none>
Status:       Running
IP:           10.240.0.55
IPs:
  IP:  10.240.0.55
Containers:
  oidc:
    Container ID:   containerd://f425e89eef9aa3a62eb51a3daa5af8c06d8a59baa79c4e4dbb1887aea2647048
    Image:          aramase/msal-go:v0.6.0
    Image ID:       docker.io/aramase/msal-go@sha256:864edcc9baacb6a14fa714af2fc0327cd4ef67d1c5ff28f38e7dc8a479ac17a1
    Port:           <none>
    Host Port:      <none>
    State:          Running
      Started:      Wed, 13 Oct 2021 15:49:29 -0700
    Ready:          True
    Restart Count:  0
    Environment:
      KEYVAULT_NAME:              ${KEYVAULT_NAME}
      SECRET_NAME:                ${KEYVAULT_SECRET_NAME}
      AZURE_AUTHORITY_HOST:       (Injected by the webhook)
      AZURE_CLIENT_ID:            (Injected by the webhook)
      AZURE_TENANT_ID:            (Injected by the webhook)
      AZURE_FEDERATED_TOKEN_FILE: (Injected by the webhook)
    Mounts:
      /var/run/secrets/kubernetes.io/serviceaccount from kube-api-access-844ns (ro)
      /var/run/secrets/tokens from azure-identity-token (ro) (Injected by the webhook)
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
    TokenExpirationSeconds:  86400
QoS Class:                   BestEffort
Node-Selectors:              kubernetes.io/os=linux
Tolerations:                 node.kubernetes.io/not-ready:NoExecute op=Exists for 300s
                             node.kubernetes.io/unreachable:NoExecute op=Exists for 300s
Events:
  Type    Reason     Age   From               Message
  ----    ------     ----  ----               -------
  Normal  Scheduled  19s   default-scheduler  Successfully assigned oidc/quick-start to k8s-agentpool1-38097163-vmss000002
  Normal  Pulling    18s   kubelet            Pulling image "aramase/msal-go:v0.6.0"
  Normal  Pulled     16s   kubelet            Successfully pulled image "aramase/msal-go:v0.6.0" in 1.987165801s
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

## 7. Cleanup

```bash
kubectl delete pod quick-start
kubectl delete sa workload-identity-sa

az group delete --name "${RESOURCE_GROUP}"
az ad sp delete --id "${APPLICATION_CLIENT_ID}"
```

[1]: https://azure.microsoft.com/en-us/services/key-vault/

[2]: https://kubernetes.io/docs/tasks/tools/

[3]: https://kind.sigs.k8s.io/docs/user/quick-start/#installation

[4]: https://www.docker.com/

[5]: https://azure.microsoft.com/en-us/

[6]: https://docs.microsoft.com/en-us/cli/azure/install-azure-cli

[7]: https://github.com/Azure/azure-workload-identity/blob/1cb9d78159458b0c820c9c08fadf967833c8cdb4/deploy/azure-wi-webhook.yaml#L103-L104

[8]: https://portal.azure.com/#cloudshell/

[9]: ./topics/managed-clusters.md

[10]: ./topics/self-managed-clusters.md

[11]: ../installation.md#helm

[12]: ../installation.md#deployment-yaml
