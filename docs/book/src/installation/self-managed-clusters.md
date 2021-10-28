# Self-Managed Clusters

When compared to using managed Kubernetes services like AKS, managing your own Kubernetes cluster provides the most freedom in customizing Kubernetes and your workload. However, there are additional setup required before deploying Azure AD Workload Identity to a self-managed cluster. If you are a cluster administrator, make sure you can perform the following actions:

1.  [Generate your own service account signing key pair][1] and [rotate it regularly][2] (at least quarterly)
2.  Manually set up your [OIDC issuer URL][3], and upload your [discovery document][4] and [JWKS][5] to a public endpoint
3.  Ability to [configure flags][6] for system-critical pods such as `kube-apiserver` and `kube-controller-manager`

[1]: ./self-managed-clusters/service-account-key-generation.md

[2]: ../topics/self-managed-clusters/service-account-key-rotation.md

[3]: ./self-managed-clusters/oidc-issuer.md

[4]: ./self-managed-clusters/oidc-issuer/discovery-document.md

[5]: ./self-managed-clusters/oidc-issuer/jwks.md

[6]: ./self-managed-clusters/configurations.md
