# Installation

<!-- toc -->

## Prerequisites

*   [Azure CLI][1] (≥2.32.0)
    *   with [aks-preview][7] CLI extension installed (≥0.5.50)
*   [Helm 3][2]
*   A Kubernetes cluster with version ≥ v1.20
    *   **Follow the cluster-specific setup guide below before deploying Azure AD Workload Identity:**

| Cluster type         | Steps                                                                                                                                                                                                                    | Guide     |
| -------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | --------- |
| Managed cluster      | 1. Enable any OIDC-specific feature flags<br>2. Extract the OIDC issuer URL                                                                                                                                              | [Link][3] |
| Self-managed cluster | 1. Generate service account key pair or bring your own keys<br>2. Setup the public OIDC issuer URL<br>3. Generate OIDC discovery and JWKS documents<br>4. Configure `kube-apiserver` and `kube-controller-manager` flags | [Link][4] |

## Azure AD Workload Identity Components

| Component                               | Description                                                                                                                                                                                                                  | Guide     |
| --------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------- |
| Mutating Admission Webhook              | Projects a signed service account token to a well-known path (`/var/run/secrets/azure/tokens/azure-identity-token`) and inject authentication-related environment variables to your pods based on annotated service account. | [Link][5] |
| Azure AD Workload Identity CLI (`azwi`) | A utility CLI that helps manage Azure AD Workload Identity and automate error-prone operations.                                                                                                                              | [Link][6] |

[1]: https://docs.microsoft.com/en-us/cli/azure/install-azure-cli

[2]: https://helm.sh/docs/intro/install/

[3]: ./installation/managed-clusters.md

[4]: ./installation/self-managed-clusters.md

[5]: ./installation/mutating-admission-webhook.md

[6]: ./installation/azwi.md

[7]: https://github.com/Azure/azure-cli-extensions/tree/main/src/aks-preview
