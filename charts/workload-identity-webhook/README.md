# Azure AD Workload Identity Helm Chart

## Get Repo

```console
helm repo add azure-workload-identity https://azure.github.io/azure-workload-identity/charts
helm repo update
```

## Install Chart

```console
# Helm install with azure-workload-identity-system namespace already created
helm install -n azure-workload-identity-system [RELEASE_NAME] azure-workload-identity/workload-identity-webhook

# Helm install and create namespace
helm install -n azure-workload-identity-system [RELEASE_NAME] azure-workload-identity/workload-identity-webhook --create-namespace
```

_See [parameters](#parameters) below._

_See [helm install](https://helm.sh/docs/helm/helm_install/) for command documentation._

## Upgrade Chart

```console
helm upgrade -n azure-workload-identity-system [RELEASE_NAME] azure-workload-identity/workload-identity-webhook
```

## Parameters

| Parameter                     | Description                                                                                                                                                          | Default                                                 |
| :---------------------------- | :------------------------------------------------------------------------------------------------------------------------------------------------------------------- | :------------------------------------------------------ |
| labels                        | The labels to add to the azure-workload-identity webhook pods                                                                                                        | `azure-workload-identity.io/system: "true"`             |
| replicaCount                  | The number of azure-workload-identity replicas to deploy for the webhook                                                                                             | `2`                                                     |
| image.repository              | Image repository                                                                                                                                                     | `mcr.microsoft.com/oss/azure/workload-identity/webhook` |
| image.pullPolicy              | Image pullPolicy                                                                                                                                                     | `IfNotPresent`                                          |
| image.release                 | The image release tag to use                                                                                                                                         | Current release version: `v0.14.0`                      |
| imagePullSecrets              | Image pull secrets to use for retrieving images from private registries                                                                                              | `[]`                                                    |
| nodeSelector                  | The node selector to use for pod scheduling                                                                                                                          | `kubernetes.io/os: linux`                               |
| arcCluster                    | Specify if it runs on Arc cluster                                                                                                                                    | `false`                                                 |
| resources                     | The resource request/limits for the container image                                                                                                                  | limits: 100m CPU, 30Mi, requests: 100m CPU, 20Mi        |
| affinity                      | The node affinity to use for pod scheduling                                                                                                                          | `{}`                                                    |
| tolerations                   | The tolerations to use for pod scheduling                                                                                                                            | `[]`                                                    |
| service.type                  | Service type                                                                                                                                                         | `ClusterIP`                                             |
| service.port                  | Service port                                                                                                                                                         | `443`                                                   |
| service.targetPort            | Service target port                                                                                                                                                  | `9443`                                                  |
| azureTenantID                 | [**REQUIRED**] Azure tenant ID                                                                                                                                       | ``                                                      |
| azureEnvironment              | Azure Environment                                                                                                                                                    | `AzurePublicCloud`                                      |
| logEncoder                    | The log encoder to use for the webhook manager (`json`, `console`)                                                                                                   | `console`                                               |
| metricsAddr                   | The address to bind the metrics server to                                                                                                                            | `:8095`                                                 |
| metricsBackend                | The metrics backend to use (`prometheus`)                                                                                                                            | `prometheus`                                            |
| mutatingWebhookFailurePolicy  | The failurePolicy for the mutating webhook. Default is `Ignore` and it's safe. Setting this to fail closed could cause cluster outage when webhook is not available. | `Ignore`                                                |
| priorityClassName             | The priority class name for webhook manager                                                                                                                          | `system-cluster-critical`                               |
| mutatingWebhookObjectSelector | The label selector to further refine which namespaced resources will be selected by the webhook.                                                                     | ``                                                      |
| mutatingWebhookAnnotations    | The annotations to add to the MutatingWebhookConfiguration                                                                                                           | `{}`                                                    |

## Contributing Changes

This Helm chart is autogenerated from the Azure AD Workload Identity static manifest. The generator code lives under `third_party/open-policy-agent/gatekeeper/helmify`. To make modifications to this template, please edit `kustomization.yaml`, `kustomize-for-helm.yaml` and `replacements.go` under that directory and then run `make manifests`. Your changes will show up in the `manifest_staging` directory and will be promoted to the root `charts` directory the next time an azure-workload-identity release is cut.
