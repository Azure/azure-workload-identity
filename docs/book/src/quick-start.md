# Quick Start

<!-- toc -->

In this tutorial, we will cover the basics of how to use the AAD Pod Identity webhook to acquire a token to access a secret in an [Azure Key Vault](https://azure.microsoft.com/en-us/services/key-vault/).

## Prerequisites

- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [kind](https://kind.sigs.k8s.io/docs/user/quick-start/#installation) and [Docker](https://www.docker.com/)
- [Microsoft Azure](https://azure.microsoft.com/en-us/) account
- [Azure CLI](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli)

## 1. Create and upload OIDC discovery document and JWKs

Generate a public/private key pair:

> Skip this step if you are planning to bring your own keys.

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

TODO (steps to upload files to a storage account)

## 2. Create a kind cluster

Export the following environment variables:

```bash
export SERVICE_ACCOUNT_ISSUER=TODO
export SERVICE_ACCOUNT_KEY_FILE=$(pwd)/sa.pub
export SERVICE_ACCOUNT_SIGNING_KEY_FILE=$(pwd)/sa.key
```

Create a kind cluster with one control plane node and customize various service account-related flags for the API server:

> The minimum supported Kubernetes version for the webhook is v1.18.0, however, we recommend using Kubernetes version v1.20.0+.

```bash
kind create cluster --name aad-pod-managed-identity --image kindest/node:v1.21.1 --config=-
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
```
<details>
<summary>Output</summary>

```bash
Creating cluster "aad-pod-managed-identity" ...
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
Set kubectl context to "kind-aad-pod-managed-identity"
You can now use your cluster with:

kubectl cluster-info --context kind-aad-pod-managed-identity

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
aad-pod-managed-identity-control-plane   Ready    control-plane,master   2m28s   v1.21.1   172.18.0.2    <none>        Ubuntu 21.04   5.4.0-1047-azure   containerd://1.5.2
```

</details>

## 3. Install cert-manager

[cert-manager](https://github.com/jetstack/cert-manager) is used for provisioning the certificates for the webhook server. Cert manager also has a component called CA injector, which is responsible for injecting the CA bundle into the MutatingWebhookConfiguration.

```bash
kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v1.2.0/cert-manager.yaml
```

<details>
<summary>Output</summary>

```bash
customresourcedefinition.apiextensions.k8s.io/issuers.cert-manager.io created
customresourcedefinition.apiextensions.k8s.io/orders.acme.cert-manager.io created
namespace/cert-manager created
serviceaccount/cert-manager-cainjector created
serviceaccount/cert-manager created
serviceaccount/cert-manager-webhook created
clusterrole.rbac.authorization.k8s.io/cert-manager-cainjector created
clusterrole.rbac.authorization.k8s.io/cert-manager-controller-issuers created
clusterrole.rbac.authorization.k8s.io/cert-manager-controller-clusterissuers created
clusterrole.rbac.authorization.k8s.io/cert-manager-controller-certificates created
clusterrole.rbac.authorization.k8s.io/cert-manager-controller-orders created
clusterrole.rbac.authorization.k8s.io/cert-manager-controller-challenges created
clusterrole.rbac.authorization.k8s.io/cert-manager-controller-ingress-shim created
clusterrole.rbac.authorization.k8s.io/cert-manager-view created
clusterrole.rbac.authorization.k8s.io/cert-manager-edit created
clusterrolebinding.rbac.authorization.k8s.io/cert-manager-cainjector created
clusterrolebinding.rbac.authorization.k8s.io/cert-manager-controller-issuers created
clusterrolebinding.rbac.authorization.k8s.io/cert-manager-controller-clusterissuers created
clusterrolebinding.rbac.authorization.k8s.io/cert-manager-controller-certificates created
clusterrolebinding.rbac.authorization.k8s.io/cert-manager-controller-orders created
clusterrolebinding.rbac.authorization.k8s.io/cert-manager-controller-challenges created
clusterrolebinding.rbac.authorization.k8s.io/cert-manager-controller-ingress-shim created
role.rbac.authorization.k8s.io/cert-manager-cainjector:leaderelection created
role.rbac.authorization.k8s.io/cert-manager:leaderelection created
role.rbac.authorization.k8s.io/cert-manager-webhook:dynamic-serving created
rolebinding.rbac.authorization.k8s.io/cert-manager-cainjector:leaderelection created
rolebinding.rbac.authorization.k8s.io/cert-manager:leaderelection created
rolebinding.rbac.authorization.k8s.io/cert-manager-webhook:dynamic-serving created
service/cert-manager created
service/cert-manager-webhook created
deployment.apps/cert-manager-cainjector created
deployment.apps/cert-manager created
deployment.apps/cert-manager-webhook created
mutatingwebhookconfiguration.admissionregistration.k8s.io/cert-manager-webhook created
validatingwebhookconfiguration.admissionregistration.k8s.io/cert-manager-webhook created
```

</details>

## 4. Install the AAD Pod Identity webhook

Obtain your Azure tenant ID by running the following command:

```bash
export AZURE_TENANT_ID="$(az account show -s <AzureSubscriptionID> --query tenantId -otsv)"
# TODO: account for different environments
export AZURE_ENVIRONMENT="AzurePublicCloud"
```

To install the webhook, choose one of the following options below:

   1. Deployment YAML

      > Replace the Azure tenant ID and cloud environment name in [here](https://github.com/Azure/aad-pod-managed-identity/blob/c6b92d50910091441a71c1cb32517d53649d74e7/manifest_staging/deploy/aad-pi-webhook.yaml#L45-L46) before executing

      ```bash
      sed -i "s/AZURE_TENANT_ID: .*/AZURE_TENANT_ID: ${AZURE_TENANT_ID}/" deploy/aad-pi-webhook.yaml
      sed -i "s/AZURE_ENVIRONMENT: .*/AZURE_ENVIRONMENT: ${AZURE_ENVIRONMENT}/" deploy/aad-pi-webhook.yaml
      kubectl apply -f deploy/aad-pi-webhook.yaml
      ```

      <details>
      <summary>Output</summary>

      ```bash
      namespace/aad-pi-webhook-system created
      clusterrole.rbac.authorization.k8s.io/aad-pi-webhook-manager-role created
      clusterrolebinding.rbac.authorization.k8s.io/aad-pi-webhook-manager-rolebinding created
      configmap/aad-pi-webhook-config created
      service/aad-pi-webhook-webhook-service created
      deployment.apps/aad-pi-webhook-controller-manager created
      certificate.cert-manager.io/aad-pi-webhook-serving-cert created
      issuer.cert-manager.io/aad-pi-webhook-selfsigned-issuer created
      mutatingwebhookconfiguration.admissionregistration.k8s.io/aad-pi-webhook-mutating-webhook-configuration created
      ```

      </details>


   1. Helm

      ```bash
      helm install pod-identity-webhook manifest_staging/charts/pod-identity-webhook \
         --namespace aad-pi-webhook-system \
         --set azureTenantID=<AzureTenantID>
      ```

      <details>
      <summary>Output</summary>

      ```bash
      TODO
      ```

      </details>

## 5. Create an Azure Key Vault and secret

Export the following environment variables:

```bash
export RESOURCE_GROUP="aad-pi-webhook-test"
export LOCATION="westus2"
export KEYVAULT_NAME="aad-pi-webhook-test-$(openssl rand -hex 2)"
export KEYVAULT_SECRET_NAME="my-secret"
```

Create a resource group:

```bash
az group create --name ${RESOURCE_GROUP} --location ${LOCATION}
```

Create an Azure Key Vault:

```bash
az keyvault create --resource-group ${RESOURCE_GROUP} \
   --location ${LOCATION} \
   --name ${KEYVAULT_NAME}
```

Create a secret:

```bash
az keyvault secret set --vault-name ${KEYVAULT_NAME} \
   --name ${KEYVAULT_SECRET_NAME} \
   --value "Hello!"
```

## 6. Create a service principal and grant permissions to access the secret

```bash
az ad sp create-for-rbac --skip-assignment --name https://test-sp
export SERVICE_PRINCIPAL_CLIENT_ID="$(az ad sp show --id https://test-sp --query appId -otsv)"
```

Set access policy for the service principal to access the keyvault secret:

```bash
az keyvault set-policy --name ${KEYVAULT_NAME} \
  --secret-permissions get \
  --spn ${SERVICE_PRINCIPAL_CLIENT_ID}
```

</details>

## 7. Setup trust between service principal and cluster OIDC issue

TODO

## 8. Create a Kubernetes service account

Create a Kubernetes service account with the required labels and annotations.

```bash
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  annotations:
    azure.pod.identity/client-id: ${SERVICE_PRINCIPAL_CLIENT_ID}
  labels:
    azure.pod.identity/use: "true"
  name: pod-identity-sa
EOF
```

<details>
<summary>Output</summary>

```bash
serviceaccount/pod-identity-sa created
```

</details>

## 9. Deploy workload

Deploy a pod referencing the service account created in the last step:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: quick-start
spec:
  serviceAccountName: pod-identity-sa
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

You can verifiy the following injected properties in the output:

| Environment variable   | Description                                           |
|------------------------|-------------------------------------------------------|
| `AZURE_AUTHORITY_HOST` | The Azure Active Directory (AAD) endpoint.            |
| `AZURE_CLIENT_ID`      | The client ID of the identity.                        |
| `AZURE_TENANT_ID`      | The tenant ID of the Azure account.                   |
| `TOKEN_FILE_PATH`      | The path of the projected service account token file.  |

<br/>

| Volume mount                                   | Description                                             |
|------------------------------------------------|---------------------------------------------------------|
| `/var/run/secrets/tokens/azure-identity-token` | The path of the projected service account token file.    |

<br/>

| Volume                 | Description                           |
|------------------------|---------------------------------------|
| `azure-identity-token` | The projected service account volume. |

```log
Name:         quick-start
Namespace:    default
Priority:     0
Node:         aad-pod-managed-identity-control-plane/172.18.0.2
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
      KEYVAULT_NAME:        ${KEYVAULT_NAME}
      SECRET_NAME:          ${KEYVAULT_SECRET_NAME}
      AZURE_AUTHORITY_HOST: (Injected by the webhook)
      AZURE_CLIENT_ID:      (Injected by the webhook)
      AZURE_TENANT_ID:      (Injected by the webhook)
      TOKEN_FILE_PATH:      (Injected by the webhook)
    Mounts:
      /var/run/secrets/kubernetes.io/serviceaccount from pod-identity-sa-token-mlgn8 (ro)
      /var/run/secrets/tokens from azure-identity-token (ro) (Injected by the webhook)
Conditions:
  Type              Status
  Initialized       True
  Ready             True
  ContainersReady   True
  PodScheduled      True
Volumes:
  pod-identity-sa-token-mlgn8:
    Type:        Secret (a volume populated by a Secret)
    SecretName:  pod-identity-sa-token-mlgn8
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
  Normal  Scheduled  3m27s  default-scheduler  Successfully assigned default/quick-start to aad-pod-managed-identity-control-plane
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


## 10. Cleanup

```bash
kubectl delete pod quick-start
kubectl delete sa pod-identity-sa

az keyvault delete --name ${KEYVAULT_NAME} --resource-group ${RESOURCE_GROUP}
az ad sp delete --id ${SERVICE_PRINCIPAL_CLIENT_ID}
```
