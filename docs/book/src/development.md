# Development

<!-- toc -->

## Setting up

### Base requirements

1.  Prerequisites from [Installation][1]
2.  Install [go][2]
    *   Get the latest patch version for go 1.20.
3.  Install [jq][3]
    *   `brew install jq` on macOS.
    *   `chocolatey install jq` on Windows.
    *   `sudo apt install jq` on Ubuntu Linux.
4.  Install make.

### Clone the repository

```bash
git clone https://github.com/Azure/azure-workload-identity.git $(go env GOPATH)/src/github.com/Azure/azure-workload-identity
```

## Create a test cluster

### Generate a custom public/private key pair

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

### Setup the OIDC discovery document and JWKS

> Skip this step if you already set up the OIDC discovery document and JWKS.

Azure blob storage will be used to host the OIDC discovery document and JWKS. However, you can host them in anywhere, as long as they are publicly available.

```bash
export AZURE_STORAGE_ACCOUNT="azwi$(openssl rand -hex 4)"
# This $web container is a special container that serves static web content without requiring public access enablement.
# See https://learn.microsoft.com/en-us/azure/storage/blobs/storage-blob-static-website
AZURE_STORAGE_CONTAINER="\$web"
az storage account create --resource-group "${RESOURCE_GROUP}" --name "${AZURE_STORAGE_ACCOUNT}"
az storage container create --name "${AZURE_STORAGE_CONTAINER}"
```

Generate and upload the OIDC discovery document:

```bash
cat <<EOF > openid-configuration.json
{
  "issuer": "https://${AZURE_STORAGE_ACCOUNT}.blob.core.windows.net/${AZURE_STORAGE_CONTAINER}/",
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

Download `azwi` from our [latest GitHub releases][4], which is a CLI tool that helps generate the JWKS document in JSON.

Generate and upload the JWKS:

```bash
azwi jwks --public-keys sa.pub --output-file jwks.json
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

### Create a kind cluster

Export the following environment variables:

```bash
export SERVICE_ACCOUNT_ISSUER="https://${AZURE_STORAGE_ACCOUNT}.blob.core.windows.net/${AZURE_STORAGE_CONTAINER}/"
export SERVICE_ACCOUNT_KEY_FILE="$(pwd)/sa.pub"
export SERVICE_ACCOUNT_SIGNING_KEY_FILE="$(pwd)/sa.key"
```

Create a kind cluster with one control plane node and customize various service account-related flags for the API server:

> The minimum supported Kubernetes version for the webhook is v1.18.0, however, we recommend using Kubernetes version v1.20.0+.

```bash
cat <<EOF | kind create cluster --name azure-workload-identity --image kindest/node:v1.22.4 --config=-
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
 • Ensuring node image (kindest/node:v1.22.4) 🖼  ...
 ✓ Ensuring node image (kindest/node:v1.22.4) 🖼
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
azure-workload-identity-control-plane   Ready    control-plane,master   2m28s   v1.22.4   172.18.0.2    <none>        Ubuntu 21.04   5.4.0-1047-azure   containerd://1.5.2
```

</details>

## Build and deploy the webhook

```bash
export REGISTRY=<YourPublicRegistry>
export IMAGE_VERSION="$(git describe --tags --always)"
export AZURE_TENANT_ID="..."
ALL_IMAGES=webhook make clean docker-build docker-push-manifest deploy
```

## Unit Test

```bash
make test
```

## E2E Test

```bash
make test-e2e-run
```

Optional settings are:

| Environment variables | Description                                                        | Default                |
| --------------------- | ------------------------------------------------------------------ | ---------------------- |
| `GINKGO_FOCUS`        | Allow you to focus on a subset of specs using regex.               |                        |
| `GINKGO_SKIP`         | Allow you to skip a subset of specs using regex.                   |                        |
| `GINKGO_NODES`        | The number of ginkgo workers to run the specs.                     | `3`                    |
| `GINKGO_NO_COLOR`     | True if you want colorized output.                                 | `false`                |
| `GINKGO_TIMEOUT`      | The test suite timeout duration.                                   | `5m`                   |
| `KUBECONFIG`          | The cluster KUBECONFIG you want to run the e2e test against.       | `${HOME}/.kube/config` |
| `E2E_EXTRA_ARGS`      | Allow you to insert extra arguments when executing the test suite. |                        |

[1]: ./installation.md#prerequisites

[2]: https://golang.org/dl/

[3]: https://stedolan.github.io/jq/

[4]: https://github.com/Azure/azure-workload-identity/releases
