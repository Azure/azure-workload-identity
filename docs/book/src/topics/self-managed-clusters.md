# Self-Managed Clusters

When compared to using managed Kubernetes services like AKS, managing your own Kubernetes cluster provides the most freedom in customizing Kubernetes and your workload. Make sure the cluster administrator can perform the following actions before deploying the Azure AD Workload Identity webhook to a self-managed cluster:

*   Access to the cluster's control plane node(s)
*   Ability to modify arguments for system-critical pods such as kube-apiserver and kube-controller-manager
*   Bring your own service account signing key pair and [rotate it regularly][6] (at least quarterly)
*   Manually set up your [OIDC issuer URL][3], and upload your [discovery document][4] and [JWKS][5] to a public endpoint

## Examples

| Tool                                   | Description                                                                                                                                          |
| -------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| [Cluster API Provider Azure][1] (CAPZ) | Cluster API implementation for Microsoft Azure.                                                                                                      |
| [Kubernetes in Docker][2] (kind)       | Run local Kubernetes clusters using Docker container. A fast way to create a conformant Kubernetes cluster. Great for local testing and development. |

[1]: https://capz.sigs.k8s.io/

[2]: https://kind.sigs.k8s.io/

[3]: ./self-managed-clusters/oidc-issuer.md

[4]: self-managed-clusters/oidc-issuer/discovery-document.md

[5]: self-managed-clusters/oidc-issuer/json-web-key-sets-jwks.md

[6]: ./self-managed-clusters/service-account-key-rotation.md
