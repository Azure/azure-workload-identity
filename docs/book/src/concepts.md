# Concepts

![Flow Diagram][1]

## Service Account

> "A service account provides an identity for processes that run in a Pod." - [source][2]

Azure AD Workload Identity supports the following mappings:

*   one-to-one (a service account referencing an AAD object)
*   many-to-one (multiple service accounts referencing the same AAD object).
*   one-to-many (a service account referencing multiple AAD objects by changing the [client ID annotation][12]).

> Note: if the service account annotations are updated, you need to restart the pod for the changes to take effect.

Users who used [aad-pod-identity][3] can think of a service account as an [AzureIdentity][4], except service account is part of the core Kubernetes API, rather than a CRD. This [doc][5] describes a list of available labels and annotations to configure.

## Proxy Init

Proxy Init is an [init container][6] that establishes an iptables rule to redirect all IMDS requests from `169.254.169.254` to the [proxy][7] server by running the following command:

```sh
{{#include ../../../init/init-iptables.sh:3:8}}
```

## Proxy

![Proxy][13]

Proxy is a [sidecar container][8] that obtains an AAD token using MSAL on behalf of applications that are still relying on the AAD Authentication Library (ADAL), for example, [AAD Pod Identity][3].

> "Starting June 30th, 2020, we will no longer add new features to ADAL. We'll continue adding critical security fixes to ADAL until June 30th, 2022. After this date, your apps using ADAL will continue to work, but we recommend upgrading to MSAL to take advantage of the latest features and to stay secure." - [source][9]

All IMDS requests from the container are routed to this proxy server due to an existing iptables rule created by [Proxy Init][10].

[1]: ./images/flow-diagram.png

[2]: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/

[3]: https://github.com/Azure/aad-pod-identity

[4]: https://azure.github.io/aad-pod-identity/docs/concepts/azureidentity/

[5]: ../topics/service-account-labels-and-annotations.html

[6]: https://kubernetes.io/docs/concepts/workloads/pods/init-containers/

[7]: #proxy

[8]: https://docs.microsoft.com/en-us/azure/architecture/patterns/sidecar

[9]: https://docs.microsoft.com/en-us/azure/active-directory/develop/msal-migration#frequently-asked-questions-faq

[10]: #proxy-init

[11]: https://docs.microsoft.com/en-us/azure/active-directory/managed-identities-azure-resources/overview

[12]: ../topics/service-account-labels-and-annotations.html#annotations

[13]: ./images/proxy-diagram.png
