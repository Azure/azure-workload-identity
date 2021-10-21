# Kubernetes in Docker (Kind)

<!-- toc -->

This document shows you how to create a kind cluster and customize the required [configuration][1] for the kube-apiserver.

## 1. Create and upload OIDC discovery document and JWKS

### Generating service account signing key

Follow this [guide][2] to generate a service account signing key using openssl. If you're planning to bring your own keys, you can skip this step.

### Upload the OIDC discovery document and JWKS

Follow the walkthrough in [Discovery Document][3] and [JSON Web Key Sets][4] to create and upload the OIDC discovery document and JWKS.

## 2. Create a kind cluster

Export the following environment variables:

```bash
export SERVICE_ACCOUNT_ISSUER="https://${AZURE_STORAGE_ACCOUNT}.blob.core.windows.net/${AZURE_STORAGE_CONTAINER}/"
export SERVICE_ACCOUNT_KEY_FILE="$(pwd)/sa.pub"
export SERVICE_ACCOUNT_SIGNING_KEY_FILE="$(pwd)/sa.key"
```

Create a kind cluster with one control plane node and customize various service account related flags for the kube-apiserver:

> The minimum supported Kubernetes version for the webhook is v1.18.0, however, we recommend using Kubernetes version v1.20.0+.

```yaml
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

Now that we have confirmed the cluster is up and running with the required configuration, you can follow the tutorial in [Quick Start][5] to learn the basics of how to use the Azure AD Workload Identity webhook to acquire a token to access a secret in an [Azure Key Vault][1].

[1]: ../configurations.md

[2]: ../service-account-key-generation.md

[3]: ../oidc-issuer/discovery-document.md

[4]: ../oidc-issuer/json-web-key-sets-jwks.md

[5]: ../../../quick-start.md
