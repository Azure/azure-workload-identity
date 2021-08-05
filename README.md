# AAD Pod Managed Identity

AAD Pod Managed Identity is the next iteration of [AAD Pod Identity][1] that enables Kubernetes applications to access Azure cloud resources securely with [Azure Active Directory][2] based on annotated [service accounts][3].

## Quick Start

Check out the AAD Pod Managed Identity [Quick Start][4] to create your first application with .

## Overview

The repository contains the following components:

1.  [Mutating Webhook][5]
    > The webhook is for mutating pods that reference an annotated service account. The webhook will inject the environment variables and the projected service account token volume.

2.  [Proxy Init][6] and [Proxy][7]
    > The proxy init container and proxy sidecar container will be used for applications that are still using [AAD Pod Identity][1].

## Motivation

*   Industry-standard and Kubernetes-friendly authentication based on OpenID Connect (OIDC).
*   Remove convoluted steps to set up [cluster role assignments][8].
*   Remove the following dependencies:
    *   [Instance Metadata Service][9] (IMDS)
    *   [CustomResourceDefinitions][10] (CRDs)

## Goals

*   A secure way for cloud-native applications to obtain AAD tokens and access Azure cloud resources in a Kubernetes cluster.

<!-- - Ensure backward compatibility when upgrading from [AAD Pod Identity](https://github.com/Azure/aad-pod-identity). -->

[1]: https://github.com/Azure/aad-pod-identity

[2]: https://azure.microsoft.com/en-us/services/active-directory/

[3]: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/

[4]: https://azure.github.io/aad-pod-managed-identity/quick-start.html

[5]: https://azure.github.io/aad-pod-managed-identity/concepts.html#mutating-webhook

[6]: https://azure.github.io/aad-pod-managed-identity/concepts.html#proxy-init

[7]: https://azure.github.io/aad-pod-managed-identity/concepts.html#proxy

[8]: https://azure.github.io/aad-pod-identity/docs/getting-started/role-assignment/

[9]: https://docs.microsoft.com/en-us/azure/virtual-machines/windows/instance-metadata-service?tabs=windows

[10]: https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/#customresourcedefinitions
