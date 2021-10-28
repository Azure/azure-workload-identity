# Installation

<!-- toc -->

## Webhook

Export your Azure tenant ID as an environment variable by running the following command:

```bash
export AZURE_TENANT_ID="$(az account show -s <AzureSubscriptionID> --query tenantId -otsv)"
```

The tenant ID above will be the default tenant ID that the webhook uses when configuring the `AZURE_TENANT_ID` environment variable in the pod. In the case of a multi-tenant cluster, you can override the tenant ID by adding the `azure.workload.identity/tenant-id` annotation to your service account.

You can install the mutating webhook with one of the following methods:

### Helm

```bash
helm install workload-identity-webhook manifest_staging/charts/workload-identity-webhook \
   --namespace azure-workload-identity-system \
   --create-namespace \
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

### Deployment YAML

#### Install `envsubst`

The deployment YAML contains the environment variables we defined above and we rely on the `envsubst` binary to substitute them for their respective values before deploying. See [the `envsubst`'s installation guide][1] on how to install it.

Install the webhook using the deployment YAML via `kubectl apply -f` and `envsubst`:

```bash
curl -s https://github.com/Azure/azure-workload-identity/releases/download/v0.6.0/azure-wi-webhook.yaml | envsubst | kubectl apply -f -
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

## [Azure Workload Identity CLI (`azwi`)][2]

### `go install`

```bash
go install github.com/Azure/azure-workload-identity/cmd/azwi@v0.6.0
```

### Homebrew (MacOS only)

```bash
brew install Azure/azure-workload-identity/azwi
```

[1]: https://github.com/a8m/envsubst#installation

[2]: ./topics/azwi.md
