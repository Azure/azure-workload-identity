# Mutating Admission Webhook

<!-- toc -->

Azure AD Workload Identity uses a [mutating admission webhook][1] to inject the following properties to pods with a service account that is configured to use Azure AD Workload Identity:

## Environment Variables

| Environment variable         | Description                                           |
| ---------------------------- | ----------------------------------------------------- |
| `AZURE_AUTHORITY_HOST`       | The Azure Active Directory (AAD) endpoint.            |
| `AZURE_CLIENT_ID`            | The client ID of the identity.                        |
| `AZURE_TENANT_ID`            | The tenant ID of the Azure account.                   |
| `AZURE_FEDERATED_TOKEN_FILE` | The path of the projected service account token file. |

## Volumes

| Volume                 | Description                           |
| ---------------------- | ------------------------------------- |
| `azure-identity-token` | The projected service account volume. |

## Volume Mounts

| Volume mount                                   | Description                                           |
| ---------------------------------------------- | ----------------------------------------------------- |
| `/var/run/secrets/tokens/azure-identity-token` | The path of the projected service account token file. |

With these properties injected, the webhook allows pods to use a [service account token][2] projected to its volume to exchange for a valid AAD token using the Azure SDK or [Microsoft Authentication Library][3] (MSAL).

[1]: https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#mutatingadmissionwebhook

[2]: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#service-account-token-volume-projection

[3]: https://docs.microsoft.com/en-us/azure/active-directory/develop/msal-overview
