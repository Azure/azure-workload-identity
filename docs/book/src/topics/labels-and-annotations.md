# Labels and Annotations

The following is a list of available labels and annotations that can be used to configure the behavior when obtaining an AAD token via MSAL.

## Service Account

### Labels

| Label                    | Description                                                    | Recommened value | Required? |
|--------------------------|----------------------------------------------------------------|------------------|-----------|
| `azure.pod.identity/use` | Represents the service account is to be used for pod identity. | `true`           | ✓         |

### Annotations

| Annotation                                            | Description                                                                                                                                                                                                                                                                                                                                                                   | Default                                                                                      |
|-------------------------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------------------------------------------------------------------------------------------|
| `azure.pod.identity/client-id`                        | Represents the identity client ID to be used with the pod.                                                                                                                                                                                                                                                                                                                    |                                                                                              |
| `azure.pod.identity/tenant-id`                        | Represents the Azure tenant ID to be used with the pod.                                                                                                                                                                                                                                                                                                                       | `AZURE_TENANT_ID` environment variable extracted from [`aad-pi-webhook-config`][1] ConfigMap |
| `azure.pod.identity/service-account-token-expiration` | Represents the `expirationSeconds` field for the projected service account token. It is an optional field that the user might want to configure this to prevent any downtime caused by errors during service account token refresh. Kubernetes service account token expiry will not be correlated with AAD tokens. AAD tokens will expire in 24 hours after they are issued. | `86400` (minimum `3600`)                                                                     |
| `azure.pod.identity/skip-containers`                  | Represents a semi-colon-separated list of containers (e.g. `container1;container2`) to skip adding projected service account token volume. By default, the projected service account token volume will be added to all containers if the service account is labeled with `azure.pod.identity/use: true`.                                                                      |                                                                                              |


## Pod

### Annotations

| Annotation                           | Description                                                                                                                                                                                                                                                                                              | Default |
|--------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------|
| `azure.pod.identity/skip-containers` | Represents a semi-colon-separated list of containers (e.g. `container1;container2`) to skip adding projected service account token volume. By default, the projected service account token volume will be added to all containers if the service account is labeled with `azure.pod.identity/use: true`. |         |

[1]: https://github.com/Azure/aad-pod-managed-identity/blob/1f4c734cfad7f0653601aa375daf4d32ef0cb5d2/manifest_staging/deploy/aad-pi-webhook.yaml#L43-L52
