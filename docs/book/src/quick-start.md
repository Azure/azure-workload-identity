# Quick Start

<!-- toc -->

In this tutorial, we will cover the basics of how to use the Azure AD Workload Identity webhook to acquire a token to access a secret in an [Azure Key Vault][1]. If you are using an AKS cluster with OIDC enabled, you may skip step 0 to step 4.

## Prerequisites

* [kubectl][2]
* [kind][3] and [Docker][4]
* [Microsoft Azure][5] account
* [Azure CLI][6]

## 0. Export environment variables and create resource group

Export the following environment variables:

```bash
export RESOURCE_GROUP="azure-wi-webhook-test"
export LOCATION="westus2"
az group create --name "${RESOURCE_GROUP}" --location "${LOCATION}"
```

## 1. Create and upload OIDC discovery document and JWKS

A private key is used to cryptographically sign all the projected service account tokens. The relying party (Azure AD in this case) can utilize our OpenID Connect (OIDC) provider to obtain the public key information to ensure the integrity of the projected service account tokens. In this step, we will create and publish two JSON files to a public storage account:

1. [OpenID Provider Configuration Information](https://openid.net/specs/openid-connect-discovery-1_0.html#ProviderConfig) - `.well-known/openid-configuration`
2. [JSON Web Key Set (JWKS)](https://openid.net/specs/openid-connect-discovery-1_0-21.html#ProviderMetadata) - `openid/v1/jwks`

> Skip the following command if you are planning to bring your own keys.

Generate a public/private key pair:

```bash
openssl genrsa -out sa.key 2048
openssl rsa -in sa.key -pubout -out sa.pub
```

<details>
<summary>Output</summary>

```bash
Generating RSA private key, 2048 bit long modulus
..............+++
......+++
e is 65537 (0x10001)
writing RSA key
```

</details>

>  Skip this step if you already set up the OIDC discovery document and JWKS.

Azure blob storage will be used to host the OIDC discovery document and JWKS. However, you can host them in anywhere, as long as they are publicly available.

```bash
export AZURE_STORAGE_ACCOUNT="pmi$(openssl rand -hex 4)"
export AZURE_STORAGE_CONTAINER="oidc-test"
az storage account create --resource-group "${RESOURCE_GROUP}" --name "${AZURE_STORAGE_ACCOUNT}"
az storage container create --name "${AZURE_STORAGE_CONTAINER}" --public-access container
```

Generate and upload the OIDC discovery document:

```bash
cat <<EOF > openid-configuration.json
{
  "issuer": "https://${AZURE_STORAGE_ACCOUNT}.blob.core.windows.net/${AZURE_STORAGE_CONTAINER}/",
  "authorization_endpoint": "https://${AZURE_STORAGE_ACCOUNT}.blob.core.windows.net/${AZURE_STORAGE_CONTAINER}/connect/authorize",
  "jwks_uri": "https://${AZURE_STORAGE_ACCOUNT}.blob.core.windows.net/${AZURE_STORAGE_CONTAINER}/openid/v1/jwks",
  "response_types_supported": [
    "id_token"
  ],
  "subject_types_supported": [
    "public"
  ],
  "id_token_signing_alg_values_supported": [
    "RS256"
  ]
}
EOF
az storage blob upload \
  --container-name "${AZURE_STORAGE_CONTAINER}" \
  --file openid-configuration.json \
  --name .well-known/openid-configuration
```

Install the `generate-jwks` tool:

> Make sure the environment variable `GOBIN` is defined and it is part of your `PATH`.

```bash
pushd hack/generate-jwks
go install .
popd
```

Generate and upload the JWKS:

```bash
generate-jwks --public-keys sa.pub > jwks.json
az storage blob upload \
  --container-name "${AZURE_STORAGE_CONTAINER}" \
  --file jwks.json \
  --name openid/v1/jwks
```

Verify that the OIDC discovery document is publicly accessible:

```bash
curl -s "https://${AZURE_STORAGE_ACCOUNT}.blob.core.windows.net/${AZURE_STORAGE_CONTAINER}/.well-known/openid-configuration"
```

<details>
<summary>Output</summary>

```json
{
  "issuer": "https://<REDACTED>.blob.core.windows.net/oidc-test/",
  "authorization_endpoint": "https://<REDACTED>.blob.core.windows.net/oidc-test/connect/authorize",
  "jwks_uri": "https://<REDACTED>.blob.core.windows.net/oidc-test/openid/v1/jwks",
  "response_types_supported": [
    "id_token"
  ],
  "subject_types_supported": [
    "public"
  ],
  "id_token_signing_alg_values_supported": [
    "RS256"
  ]
}
```

</details>

Verify that the JWKS is publicly accessible:

```bash
curl -s "https://${AZURE_STORAGE_ACCOUNT}.blob.core.windows.net/${AZURE_STORAGE_CONTAINER}/openid/v1/jwks"
```

<details>
<summary>Output</summary>

```json
{
  "keys": [
    {
      "use": "sig",
      "kty": "RSA",
      "kid": "Me5VC6i4_4mymFj7T5rcUftFjYX70YoCfSnZB6-nBY4",
      "alg": "RS256",
      "n": "ywg7HeKIFX3vleVKZHeYoNpuLHIDisnczYXrUdIGCNilCJFA1ymjG2UAADnt_FpYUsCVyKYJTqcxNbK4boNg_P3uK39OAqXabwYrilEZvsVJQKhzn8dXLeqAnM98L8eBpySU208KTsfMkS3Q6lqwurUP7c_a3g_1XRJukz_EmQxg9jLD_fQd5VwPTEo8HJQIFqIxFWzjTkkK5hbcL9Cclkf6RpeRyjh7Vem57Fu-jAlxDUiYiqyieM4OBNm4CQjiqDE8_xOC8viNpHNw542MYVDKSRnYui31lCOj32wBDphczR8BbnrZgbqN3K_zzB3gIjcGbWbbGA5xKJYqSu5uRwN89_CWrT3vGw5RN3XQPSbhGC4smgZkOCw3N9i1b-x-rrd-mRse6F95ONaoslCJUbJvxvDdb5X0P4_CVZRwJvUyP3OJ44ZvwzshA-zilG-QC9E1j2R9DTSMqOJzUuOxS0JIvoboteI1FAByV9KyU948zQRM7r7MMZYBKWIsu6h7",
      "e": "AQAB"
    }
  ]
}
```

</details>

## 2. Create a kind cluster

Export the following environment variables:

```bash
export SERVICE_ACCOUNT_ISSUER="https://${AZURE_STORAGE_ACCOUNT}.blob.core.windows.net/${AZURE_STORAGE_CONTAINER}/"
export SERVICE_ACCOUNT_KEY_FILE="$(pwd)/sa.pub"
export SERVICE_ACCOUNT_SIGNING_KEY_FILE="$(pwd)/sa.key"
```

Create a kind cluster with one control plane node and customize various service account-related flags for the API server:

> The minimum supported Kubernetes version for the webhook is v1.18.0, however, we recommend using Kubernetes version v1.20.0+.

```bash
cat <<EOF | kind create cluster --name azure-workload-identity --image kindest/node:v1.21.1 --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  extraMounts:
    - hostPath: ${SERVICE_ACCOUNT_KEY_FILE}
      containerPath: /etc/kubernetes/pki/sa.pub
    - hostPath: ${SERVICE_ACCOUNT_SIGNING_KEY_FILE}
      containerPath: /etc/kubernetes/pki/sa.key
  kubeadmConfigPatches:
  - |
    kind: ClusterConfiguration
    apiServer:
      extraArgs:
        service-account-issuer: ${SERVICE_ACCOUNT_ISSUER}
        service-account-key-file: /etc/kubernetes/pki/sa.pub
        service-account-signing-key-file: /etc/kubernetes/pki/sa.key
EOF
```

<details>
<summary>Output</summary>

```bash
Creating cluster "azure-workload-identity" ...
 • Ensuring node image (kindest/node:v1.21.1) 🖼  ...
 ✓ Ensuring node image (kindest/node:v1.21.1) 🖼
 • Preparing nodes 📦   ...
 ✓ Preparing nodes 📦
 • Writing configuration 📜  ...
 ✓ Writing configuration 📜
 • Starting control-plane 🕹️  ...
 ✓ Starting control-plane 🕹️
 • Installing CNI 🔌  ...
 ✓ Installing CNI 🔌
 • Installing StorageClass 💾  ...
 ✓ Installing StorageClass 💾
Set kubectl context to "kind-azure-workload-identity"
You can now use your cluster with:

kubectl cluster-info --context kind-azure-workload-identity

Have a question, bug, or feature request? Let us know! https://kind.sigs.k8s.io/#community 🙂
```

</details>

Run the following command to verify that the kind cluster is online:

```bash
kubectl get nodes
```

<details>
<summary>Output</summary>

```bash
NAME                                     STATUS   ROLES                  AGE     VERSION   INTERNAL-IP   EXTERNAL-IP   OS-IMAGE       KERNEL-VERSION     CONTAINER-RUNTIME
azure-workload-identity-control-plane   Ready    control-plane,master   2m28s   v1.21.1   172.18.0.2    <none>        Ubuntu 21.04   5.4.0-1047-azure   containerd://1.5.2
```

</details>

## 3. Install the Azure AD Workload Identity webhook

Obtain your Azure tenant ID by running the following command:

```bash
export AZURE_TENANT_ID="$(az account show -s <AzureSubscriptionID> --query tenantId -otsv)"
# TODO: account for different environments
export AZURE_ENVIRONMENT="AzurePublicCloud"
```

To install the webhook, choose one of the following options below:

1.  Deployment YAML

    > Replace the Azure tenant ID and cloud environment name in [here][7] before executing

    ```bash
    sed -i "s/AZURE_TENANT_ID: .*/AZURE_TENANT_ID: ${AZURE_TENANT_ID}/" deploy/azure-wi-webhook.yaml
    sed -i "s/AZURE_ENVIRONMENT: .*/AZURE_ENVIRONMENT: ${AZURE_ENVIRONMENT}/" deploy/azure-wi-webhook.yaml
    kubectl apply -f deploy/azure-wi-webhook.yaml
    ```

    <details>
    <summary>Output</summary>

    ```bash
    namespace/azure-workload-identity-system created
    serviceaccount/azure-wi-webhook-admin created
    role.rbac.authorization.k8s.io/azure-wi-webhook-manager-role created
    clusterrole.rbac.authorization.k8s.io/azure-wi-webhook-manager-role created
    rolebinding.rbac.authorization.k8s.io/azure-wi-webhook-manager-rolebinding created
    clusterrolebinding.rbac.authorization.k8s.io/azure-wi-webhook-manager-rolebinding created
    configmap/azure-wi-webhook-config created
    secret/azure-wi-webhook-server-cert created
    service/azure-wi-webhook-webhook-service created
    deployment.apps/azure-wi-webhook-controller-manager created
    mutatingwebhookconfiguration.admissionregistration.k8s.io/azure-wi-webhook-mutating-webhook-configuration created
    ```

    </details>

2.  Helm

    ```bash
    kubectl create namespace azure-workload-identity-system
    helm install workload-identity-webhook manifest_staging/charts/workload-identity-webhook \
       --namespace azure-workload-identity-system \
       --set azureTenantID="${AZURE_TENANT_ID}"
    ```

    <details>
    <summary>Output</summary>

    ```bash
    namespace/azure-workload-identity-system created
    NAME: workload-identity-webhook
    LAST DEPLOYED: Wed Aug  4 10:49:20 2021
    NAMESPACE: azure-workload-identity-system
    STATUS: deployed
    REVISION: 1
    TEST SUITE: None
    ```

    </details>

## 4. Create an Azure Key Vault and secret

Export the following environment variables:

```bash
export KEYVAULT_NAME="azure-wi-webhook-test-$(openssl rand -hex 2)"
export KEYVAULT_SECRET_NAME="my-secret"
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
   --value "Hello!"
```

## 5. Create an AAD application and grant permissions to access the secret

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

## 6. Create a Kubernetes service account

Create a Kubernetes service account with the required labels and annotations.

```bash
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  annotations:
    azure.workload.identity/client-id: ${APPLICATION_CLIENT_ID}
  labels:
    azure.workload.identity/use: "true"
  name: workload-identity-sa
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

## 7. Establish trust between the AAD application and the service account issuer & subject

Login to [Azure Cloud Shell][8] and run the following commands:

```bash
# Get the object ID of the AAD application
export APPLICATION_OBJECT_ID="$(az ad app show --id ${APPLICATION_CLIENT_ID} --query objectId -otsv)"
# If you skip step 2
export SERVICE_ACCOUNT_ISSUER="..."
```

Add the federated identity credential:

```bash
cat <<EOF > body.json
{
  "name": "kubernetes-federated-credential",
  "issuer": "${SERVICE_ACCOUNT_ISSUER}",
  "subject": "system:serviceaccount:default:workload-identity-sa",
  "description": "Kubernetes service account federated credential",
  "audiences": [
    "api://AzureADTokenExchange"
  ]
}
EOF

az rest --method POST --uri "https://graph.microsoft.com/beta/applications/${APPLICATION_OBJECT_ID}/federatedIdentityCredentials" --body @body.json
```

## 8. Deploy workload

Deploy a pod referencing the service account created in the last step:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: quick-start
spec:
  serviceAccountName: workload-identity-sa
  containers:
    - image: aramase/dotnet:v0.4
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
Node:         azure-workload-identity-control-plane/172.18.0.2
Start Time:   Wed, 07 Jul 2021 14:45:38 -0700
Labels:       <none>
Annotations:  <none>
Status:       Running
IP:           10.244.0.9
IPs:
  IP:  10.244.0.9
Containers:
  oidc:
    Container ID:   containerd://efa8d09f78dc88dd17518ce5430ea820cef5743b33d77ae2b31e1082cc439218
    Image:          aramase/dotnet:v0.4
    Image ID:       docker.io/aramase/dotnet@sha256:821dbaa070bf7e26dd9172092658f6e6f910a2db198723e10b3ebb4e35a99eb5
    Port:           <none>
    Host Port:      <none>
    State:          Running
      Started:      Wed, 07 Jul 2021 14:45:45 -0700
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
      /var/run/secrets/kubernetes.io/serviceaccount from workload-identity-sa-token-mlgn8 (ro)
      /var/run/secrets/tokens from azure-identity-token (ro) (Injected by the webhook)
Conditions:
  Type              Status
  Initialized       True
  Ready             True
  ContainersReady   True
  PodScheduled      True
Volumes:
  workload-identity-sa-token-mlgn8:
    Type:        Secret (a volume populated by a Secret)
    SecretName:  workload-identity-sa-token-mlgn8
    Optional:    false
  azure-identity-token: (Injected by the webhook)
    Type:                    Projected (a volume that contains injected data from multiple sources)
    TokenExpirationSeconds:  86400
QoS Class:                   BestEffort
Node-Selectors:              kubernetes.io/os=linux
Tolerations:                 node.kubernetes.io/not-ready:NoExecute op=Exists for 300s
                             node.kubernetes.io/unreachable:NoExecute op=Exists for 300s
Events:
  Type    Reason     Age    From               Message
  ----    ------     ----   ----               -------
  Normal  Scheduled  3m27s  default-scheduler  Successfully assigned default/quick-start to azure-workload-identity-control-plane
  Normal  Pulling    3m26s  kubelet            Pulling image "aramase/dotnet:v0.4"
  Normal  Pulled     3m21s  kubelet            Successfully pulled image "aramase/dotnet:v0.4" in 5.824712366s
  Normal  Created    3m20s  kubelet            Created container oidc
  Normal  Started    3m20s  kubelet            Started container oidc
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
START 07/07/2021 21:45:45 (quick-start)
Your secret is Hello!
```

</details>

## 9. Cleanup

```bash
kubectl delete pod quick-start
kubectl delete sa workload-identity-sa

az keyvault delete --name "${KEYVAULT_NAME}" --resource-group "${RESOURCE_GROUP}"
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
