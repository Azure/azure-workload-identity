# OpenID Connect Issuer

Azure AD Workload Identity and Azure Active Directory (AAD) leverages an authentication protocol called OpenID Connect (OIDC) to securely exchange a cryptographically signed Kubernetes service account token for an AAD token. Your workload can then consume the AAD token to access Azure cloud resources via the Azure SDK or MSAL.

In the case of self-managed clusters, cluster administrator will have to manually set up an OIDC-compliant issuer URL and upload various documents to a public endpoint. The following table describes the documents that are required for the Azure AD Workload Identity webhook to work:

| Endpoint                                            | Description                                                                                                      |
| --------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| [`{IssuerURL}/.well-known/openid-configuration`][1] | Also known as the OIDC discovery document. This contains the metadata about the issuer's configurations.         |
| [`{IssuerURL}/openid/v1/jwks`][2]                   | This contains the public signing key(s) that AAD uses to validate the authenticity of the service account token. |

## Sequence Diagram

![Sequence Diagram][3]

[1]: ./oidc-issuer/discovery-document.md

[2]: ./oidc-issuer/json-web-key-sets-jwks.md

[3]: ../../images/oidc-issuer-sequence-diagram.png
