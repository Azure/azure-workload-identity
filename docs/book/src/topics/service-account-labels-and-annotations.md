# Service Account Labels and Annotations

<!-- toc -->

The following is a list of available labels and annotations that can be used to configure the behavior when exchanging the service account token for an AAD access token:

## Pod

### Labels

| Label                         | Description                                                                                                                                                                                                                                                 | Recommended value | Required? |
| ----------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------- | --------- |
| `azure.workload.identity/use` | This label is **required** in the pod template spec. Only pods with this label will be mutated by the azure-workload-identity mutating admission webhook to inject the Azure specific environment variables and the projected service account token volume. | `true`            | ✓         |

### Annotations

| Annotation                                                 | Description                                                                                                                                                                                                                                                                                                                                                                                                                                   | Default                                   |
| ---------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------- |
| `azure.workload.identity/service-account-token-expiration` | **(Takes precedence if the service account is also annotated)** Represents the `expirationSeconds` field for the projected service account token. It is an optional field that the user might want to configure this to prevent any downtime caused by errors during service account token refresh. Kubernetes service account token expiry will not be correlated with AAD tokens. AAD tokens will expire in 24 hours after they are issued. | `3600` (acceptable range: `3600 - 86400`) |
| `azure.workload.identity/skip-containers`                  | Represents a semi-colon-separated list of containers (e.g. `container1;container2`) to skip adding projected service account token volume. By default, the projected service account token volume will be added to all containers.                                                                                                                                                                                                            |                                           |
| `azure.workload.identity/inject-proxy-sidecar`             | Injects a proxy init container and proxy sidecar into the pod. The proxy sidecar is used to intercept token requests to IMDS and acquire an AAD token on behalf of the user with federated identity credential.                                                                                                                                                                                                                               | `true`                                    |
| `azure.workload.identity/proxy-sidecar-port`               | Represents the port of the proxy sidecar.                                                                                                                                                                                                                                                                                                                                                                                                     | `8000`                                    |


## Service Account

### Labels

| Label                         | Description                                                         | Recommended value | Required?                                                                         |
| ----------------------------- | ------------------------------------------------------------------- | ----------------- | --------------------------------------------------------------------------------- |
| `azure.workload.identity/use` | Represents the service account is to be used for workload identity. | `true`            | **This label is only required if using azure-workload-identity version < v1.0.0** |

### Annotations

| Annotation                                                 | Description                                                                                                                                                                                                                                                                                                                                                                   | Default                                                                                        |
| ---------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| `azure.workload.identity/client-id`                        | Represents the AAD application client ID to be used with the pod.                                                                                                                                                                                                                                                                                                             |                                                                                                |
| `azure.workload.identity/tenant-id`                        | Represents the Azure tenant ID where the AAD application is registered.                                                                                                                                                                                                                                                                                                       | `AZURE_TENANT_ID` environment variable extracted from [`azure-wi-webhook-config`][1] ConfigMap |
| `azure.workload.identity/service-account-token-expiration` | Represents the `expirationSeconds` field for the projected service account token. It is an optional field that the user might want to configure this to prevent any downtime caused by errors during service account token refresh. Kubernetes service account token expiry will not be correlated with AAD tokens. AAD tokens will expire in 24 hours after they are issued. | `3600` (acceptable range: `3600 - 86400`)                                                      |

[1]: https://github.com/Azure/azure-workload-identity/blob/40b3842dc49784bb014ad5d8b02cf6c959244196/deploy/azure-wi-webhook.yaml#L101-L110
