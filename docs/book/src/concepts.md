# Concepts

![Flow Diagram][1]

## Service Account

> "A service account provides an identity for processes that run in a Pod." - [source][2]

Azure AD Workload Identity supports the following mappings:

*   one-to-one (a service account referencing an AAD object)
*   many-to-one (multiple service accounts referencing the same AAD object).
*   one-to-many (a service account referencing multiple AAD objects by changing the [client ID annotation][6]).

> Note: if the service account annotations are updated, you need to restart the pod for the changes to take effect.

Users who used [aad-pod-identity][3] can think of a service account as an [AzureIdentity][4], except service account is part of the core Kubernetes API, rather than a CRD. This [doc][5] describes a list of available labels and annotations to configure.

## Workload Identity Federation

Using workload identity federation allows you to access Azure Active Directory (Azure AD) protected resources without needing to manage secrets. This [doc][7] describes in detail on workload identity federation works and steps to create, delete, get or update federated identity credentials.

[1]: ./images/flow-diagram.png

[2]: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/

[3]: https://github.com/Azure/aad-pod-identity

[4]: https://azure.github.io/aad-pod-identity/docs/concepts/azureidentity/

[5]: ./topics/service-account-labels-and-annotations.md

[6]: ./topics/service-account-labels-and-annotations.md#annotations

[7]: https://docs.microsoft.com/en-us/azure/active-directory/develop/workload-identity-federation
