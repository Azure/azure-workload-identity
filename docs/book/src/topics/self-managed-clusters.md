# Self-Managed Clusters

When compared to using managed Kubernetes services like AKS, managing your own Kubernetes cluster provides the most freedom in customizing Kubernetes and your workload. Make sure the cluster administrator can perform the following actions before deploying the Azure AD Workload Identity webhook to a self-managed cluster:

*   Access to the cluster's control plane node(s)
*   Ability to modify arguments for system-critical pods such as kube-apiserver and kube-controller-manager
*   Bring your own service account signing key pair and rotate it regularly (at least quarterly)
*   Upload your OIDC discovery document and JWK to a public storage endpoint

## Examples

| Tool                                                           | Description                                                                                                                                          |
| -------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| [Cluster API Provider Azure][1] (CAPZ) | Cluster API implementation for Microsoft Azure.                                                                                                      |
| [Amazon EKS Anywhere][2]     | Run Amazon EKS on your own infrastructure.                                                                                                           |
| [Kubernetes in Docker][3] (kind)       | Run local Kubernetes clusters using Docker container. A fast way to create a conformant Kubernetes cluster. Great for local testing and development. |

[1]: https://capz.sigs.k8s.io/

[2]: https://anywhere.eks.amazonaws.com/

[3]: https://kind.sigs.k8s.io/
