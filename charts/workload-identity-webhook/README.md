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

## Air-gapped Environments

For air-gapped environments where you need to use a custom registry for proxy sidecar containers, you can configure the proxy image settings:

```console
# Install with custom registry for proxy images
helm install -n azure-workload-identity-system workload-identity-webhook azure-workload-identity/workload-identity-webhook \
  --set azureTenantID="your-tenant-id" \
  --set proxy.image.registry="your-registry.example.com" \
  --set proxy.initImage.registry="your-registry.example.com" \
  --create-namespace

# Or with completely custom image references
helm install -n azure-workload-identity-system workload-identity-webhook azure-workload-identity/workload-identity-webhook \
  --set azureTenantID="your-tenant-id" \
  --set proxy.image.registry="your-registry.example.com" \
  --set proxy.image.repository="custom-proxy" \
  --set proxy.image.tag="v1.0.0" \
  --set proxy.initImage.registry="your-registry.example.com" \
  --set proxy.initImage.repository="custom-proxy-init" \
  --set proxy.initImage.tag="v1.0.0" \
  --create-namespace
```

## Upgrade Chart

```console
helm upgrade -n azure-workload-identity-system [RELEASE_NAME] azure-workload-identity/workload-identity-webhook
```

## Parameters

| Parameter                          | Description                                                                                                                       | Default                                                 |
| :--------------------------------- | :-------------------------------------------------------------------------------------------------------------------------------- | :------------------------------------------------------ |
| replicaCount                       | The number of azure-workload-identity replicas to deploy for the webhook                                                          | `2`                                                     |
| image.repository                   | Image repository                                                                                                                  | `mcr.microsoft.com/oss/azure/workload-identity/webhook` |
| image.pullPolicy                   | Image pullPolicy                                                                                                                  | `IfNotPresent`                                          |
| image.release                      | The image release tag to use                                                                                                      | Current release version: `v1.5.1`                       |
| imagePullSecrets                   | Image pull secrets to use for retrieving images from private registries                                                           | `[]`                                                    |
| nodeSelector                       | The node selector to use for pod scheduling                                                                                       | `kubernetes.io/os: linux`                               |
| resources                          | The resource request/limits for the container image                                                                               | limits: 100m CPU, 30Mi, requests: 100m CPU, 20Mi        |
| affinity                           | The node affinity to use for pod scheduling                                                                                       | `{}`                                                    |
| topologySpreadConstraints          | The topology spread constraints to use for pod scheduling                                                                         | `[]`                                                    |
| tolerations                        | The tolerations to use for pod scheduling                                                                                         | `[]`                                                    |
| service.type                       | Service type                                                                                                                      | `ClusterIP`                                             |
| service.port                       | Service port                                                                                                                      | `443`                                                   |
| service.targetPort                 | Service target port                                                                                                               | `9443`                                                  |
| azureTenantID                      | [**REQUIRED**] Azure tenant ID                                                                                                    | ``                                                      |
| azureEnvironment                   | Azure Environment                                                                                                                 | `AzurePublicCloud`                                      |
| logLevel                           | The log level to use for the webhook manager. In order of increasing verbosity: unset (empty string), info, debug, trace and all. | `info`                                                  |
| metricsAddr                        | The address to bind the metrics server to                                                                                         | `:8095`                                                 |
| metricsBackend                     | The metrics backend to use (`prometheus`)                                                                                         | `prometheus`                                            |
| priorityClassName                  | The priority class name for webhook manager                                                                                       | `system-cluster-critical`                               |
| mutatingWebhookAnnotations         | The annotations to add to the MutatingWebhookConfiguration                                                                        | `{}`                                                    |
| podLabels                          | The labels to add to the azure-workload-identity webhook pods                                                                     | `{}`                                                    |
| podAnnotations                     | The annotations to add to the azure-workload-identity webhook pods                                                                | `{}`                                                    |
| mutatingWebhookNamespaceSelector   | The namespace selector to further refine which namespaces will be selected by the webhook.                                        | `{}`                                                    |
| podDisruptionBudget.minAvailable   | The minimum number of pods that must be available for the webhook to be considered available                                      | `1`                                                     |
| podDisruptionBudget.maxUnavailable | The maximum number of pods that may be unavailable for the webhook to be considered available                                     | `nil`                                                   |
| proxy.image.registry               | The registry for the proxy sidecar image                                                                                         | `mcr.microsoft.com/oss/azure/workload-identity`        |
| proxy.image.repository             | The repository for the proxy sidecar image                                                                                       | `proxy`                                                 |
| proxy.image.tag                    | The tag for the proxy sidecar image (defaults to chart appVersion)                                                               | ``                                                      |
| proxy.initImage.registry           | The registry for the proxy init image                                                                                            | `mcr.microsoft.com/oss/azure/workload-identity`        |
| proxy.initImage.repository         | The repository for the proxy init image                                                                                          | `proxy-init`                                            |
| proxy.initImage.tag                | The tag for the proxy init image (defaults to chart appVersion)                                                                  | ``                                                      |
| revisionHistoryLimit               | The number of old ReplicaSets to retain for the webhook deployment                                                                | `10`                                                    |

## Contributing Changes

This Helm chart is autogenerated from the Azure AD Workload Identity static manifest. The generator code lives under `third_party/open-policy-agent/gatekeeper/helmify`. To make modifications to this template, please edit `kustomization.yaml`, `kustomize-for-helm.yaml` and `replacements.go` under that directory and then run `make manifests`. Your changes will show up in the `manifest_staging` directory and will be promoted to the root `charts` directory the next time an azure-workload-identity release is cut.
