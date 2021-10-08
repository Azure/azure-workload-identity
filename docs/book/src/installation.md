# Installation

Obtain your Azure tenant ID by running the following command:

```bash
export AZURE_TENANT_ID="$(az account show -s <AzureSubscriptionID> --query tenantId -otsv)"
# TODO: account for different environments
export AZURE_ENVIRONMENT="AzurePublicCloud"
```

The tenant ID above will be the default tenant ID that the webhook uses when configuring the `AZURE_TENANT_ID` environment variable in the pod. In the case of a multi-tenant cluster, you can override the tenant ID by adding the `azure.workload.identity/tenant-id` annotation to your service account.

## Helm

```bash
# TODO(chewong): use https://azure.github.io/azure-workload-identity/charts
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

## Deployment YAML

> Replace the Azure tenant ID and cloud environment name in [here][1] before executing

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

[1]: https://github.com/Azure/azure-workload-identity/blob/1cb9d78159458b0c820c9c08fadf967833c8cdb4/deploy/azure-wi-webhook.yaml#L103-L104
