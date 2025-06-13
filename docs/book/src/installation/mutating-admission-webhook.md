# Mutating Admission Webhook

<!-- toc -->

Azure AD Workload Identity uses a [mutating admission webhook][1] to project a signed service account token to your workload's volume and inject the following properties to pods with a service account that is configured to use the webhook:

<details>
<summary>Properties</summary>

| Environment variable         | Description                                                                              |
| ---------------------------- | ---------------------------------------------------------------------------------------- |
| `AZURE_AUTHORITY_HOST`       | The Azure Active Directory (AAD) endpoint.                                               |
| `AZURE_CLIENT_ID`            | The application/client ID of the Azure AD application or user-assigned managed identity. |
| `AZURE_TENANT_ID`            | The tenant ID of the Azure subscription.                                                 |
| `AZURE_FEDERATED_TOKEN_FILE` | The path of the projected service account token file.                                    |

| Volume                 | Description                           |
| ---------------------- | ------------------------------------- |
| `azure-identity-token` | The projected service account volume. |

| Volume mount                                         | Description                                           |
| ---------------------------------------------------- | ----------------------------------------------------- |
| `/var/run/secrets/azure/tokens/azure-identity-token` | The path of the projected service account token file. |

</details>

The webhook allows pods to use a [service account token][2] projected to a well-known volume path to exchange for an Azure AD access token by leveraging the above properties with the Azure Identity SDKs or the [Microsoft Authentication Library][3] (MSAL).

## Prerequisites

Obtain your Azure tenant ID by running the following command:

```bash
export AZURE_TENANT_ID="$(az account show -s <AzureSubscriptionID> --query tenantId -otsv)"
```

The tenant ID above will be the default tenant ID that the webhook uses when configuring the `AZURE_TENANT_ID` environment variable in the pod. In the case of a multi-tenant cluster, you can override the tenant ID by adding the `azure.workload.identity/tenant-id` annotation to your service account.

You can install the mutating admission webhook with one of the following methods:

## Helm 3 (Recommended)

```bash
helm repo add azure-workload-identity https://azure.github.io/azure-workload-identity/charts
helm repo update
helm install workload-identity-webhook azure-workload-identity/workload-identity-webhook \
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

### Install `envsubst`

The deployment YAML contains the environment variables we defined above and we rely on the `envsubst` binary to substitute them for their respective values before deploying. See [the `envsubst`'s installation guide][4] on how to install it.

Install the webhook using the deployment YAML via `kubectl apply -f` and `envsubst`:

```bash
curl -sL https://github.com/Azure/azure-workload-identity/releases/download/v1.5.1/azure-wi-webhook.yaml | envsubst | kubectl apply -f -
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

[1]: https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#mutatingadmissionwebhook

[2]: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#service-account-token-volume-projection

[3]: https://docs.microsoft.com/en-us/azure/active-directory/develop/msal-overview

[4]: https://github.com/a8m/envsubst#installation
