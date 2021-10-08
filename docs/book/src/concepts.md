# Concepts

![Flow Diagram][1]

## Service Account

> "A service account provides an identity for processes that run in a Pod." - [source][2]

Azure AD Workload Identity supports the following mappings:

*   one-to-one (a service account referencing an AAD object)
*   many-to-one (multiple service accounts referencing the same AAD object).
*   one-to-many (a service account referencing multiple AAD objects by changing the [client ID annotation][15]).

> Note: if the service account annotations are updated, you need to restart the pod for the changes to take effect.

Users who used [aad-pod-identity][3] can think of a service account as an [AzureIdentity][4], except service account is part of the core Kubernetes API, rather than a CRD. This [doc][5] describes a list of available labels and annotations to configure.

## Mutating Webhook

Azure AD Workload Identity uses a [mutating admission webhook][6] to inject the following properties to pods with a service account that is configured to use Azure AD Workload Identity:

### Environment Variables

| Environment variable         | Description                                           |
| ---------------------------- | ----------------------------------------------------- |
| `AZURE_AUTHORITY_HOST`       | The Azure Active Directory (AAD) endpoint.            |
| `AZURE_CLIENT_ID`            | The client ID of the identity.                        |
| `AZURE_TENANT_ID`            | The tenant ID of the Azure account.                   |
| `AZURE_FEDERATED_TOKEN_FILE` | The path of the projected service account token file. |

### Volumes

| Volume                 | Description                           |
| ---------------------- | ------------------------------------- |
| `azure-identity-token` | The projected service account volume. |

### Volume Mounts

| Volume mount                                   | Description                                           |
| ---------------------------------------------- | ----------------------------------------------------- |
| `/var/run/secrets/tokens/azure-identity-token` | The path of the projected service account token file. |

With these properties injected, the webhook allows pods to use a [service account token][7] projected to its volume to exchange for a valid AAD token using the [Microsoft Authentication Library][8] (MSAL).

## Proxy Init

Proxy Init is an [init container][9] that establishes an iptables rule to redirect all IMDS requests from `169.254.169.254` to the [proxy][10] server by running the following command:

```sh
{{#include ../../../init/init-iptables.sh:3:8}}
```

## Proxy

![Proxy][17]

Proxy is a [sidecar container][11] that obtains an AAD token using MSAL on behalf of applications that are still relying on the AAD Authentication Library (ADAL), for example, [AAD Pod Identity][3].

> "Starting June 30th, 2020, we will no longer add new features to ADAL. We'll continue adding critical security fixes to ADAL until June 30th, 2022. After this date, your apps using ADAL will continue to work, but we recommend upgrading to MSAL to take advantage of the latest features and to stay secure." - [source][12]

All IMDS requests from the container are routed to this proxy server due to an existing iptables rule created by [Proxy Init][13].

## Trust

Not all service account tokens can be exchanged for a valid AAD token. Trust between an existing service account and an AAD application has to be established in advance.

To establish trust, login to [Azure Cloud Shell][16] and export the following environment variables:

```bash
# Get the client and object ID of the AAD application
export APPLICATION_CLIENT_ID="..."
export APPLICATION_OBJECT_ID="$(az ad app show --id ${APPLICATION_CLIENT_ID} --query objectId -otsv)"
export SERVICE_ACCOUNT_ISSUER="..."
export SERVICE_ACCOUNT_NAMESPACE="..."
export SERVICE_ACCOUNT_NAME="..."
```

Add the federated identity credential:

```bash
cat <<EOF > body.json
{
  "name": "kubernetes-federated-credential",
  "issuer": "${SERVICE_ACCOUNT_ISSUER}",
  "subject": "system:serviceaccount:${SERVICE_ACCOUNT_NAMESPACE}:${SERVICE_ACCOUNT_NAME}",
  "description": "Kubernetes service account federated credential",
  "audiences": [
    "api://AzureADTokenExchange"
  ]
}
EOF

az rest --method POST --uri "https://graph.microsoft.com/beta/applications/${APPLICATION_OBJECT_ID}/federatedIdentityCredentials" --body @body.json
```

[1]: ./images/flow-diagram.png

[2]: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/

[3]: https://github.com/Azure/aad-pod-identity

[4]: https://azure.github.io/aad-pod-identity/docs/concepts/azureidentity/

[5]: ../topics/service-account-labels-and-annotations.html

[6]: https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/

[7]: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#service-account-token-volume-projection

[8]: https://docs.microsoft.com/en-us/azure/active-directory/develop/msal-overview

[9]: https://kubernetes.io/docs/concepts/workloads/pods/init-containers/

[10]: #proxy

[11]: https://docs.microsoft.com/en-us/azure/architecture/patterns/sidecar

[12]: https://docs.microsoft.com/en-us/azure/active-directory/develop/msal-migration#frequently-asked-questions-faq

[13]: #proxy-init

[14]: https://docs.microsoft.com/en-us/azure/active-directory/managed-identities-azure-resources/overview

[15]: ../topics/service-account-labels-and-annotations.html#annotations

[16]: https://portal.azure.com/#cloudshell/

[17]: ./images/proxy-diagram.png
