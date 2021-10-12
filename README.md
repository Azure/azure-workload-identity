# Azure AD Workload Identity

Azure AD Workload Identity is the next iteration of [AAD Pod Identity][1] that enables Kubernetes applications to access Azure cloud resources securely with [Azure Active Directory][2] based on annotated [service accounts][3].

## Installation

Check out the [installation guide][12] on how to deploy the Azure AD Workload Identity webhook.

## Quick Start

Check out the Azure AD Workload Identity [Quick Start][4] on how to securely access Azure cloud resources from your application using the webhook and MSAL.

## Overview

The repository contains the following components:

1.  [Mutating Webhook][5]
    > The webhook is for mutating pods that reference an annotated service account. The webhook will inject the environment variables and the [projected service account token volume][11]. Your application/SDK will consume them to authenticate itself to Azure resources.

2.  [Proxy Init][6] and [Proxy][7]
    > The proxy init container and proxy sidecar container will be used for applications that are still using [AAD Pod Identity][1].

## Motivation

*   Cloud-agnostic.
*   Support Linux and Windows workload.
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

[4]: https://azure.github.io/azure-workload-identity/quick-start.html

[5]: https://azure.github.io/azure-workload-identity/topics/mutating-admission-webhook.html

[6]: https://azure.github.io/azure-workload-identity/concepts.html#proxy-init

[7]: https://azure.github.io/azure-workload-identity/concepts.html#proxy

[8]: https://azure.github.io/aad-pod-identity/docs/getting-started/role-assignment/

[9]: https://docs.microsoft.com/en-us/azure/virtual-machines/windows/instance-metadata-service?tabs=windows

[10]: https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/#customresourcedefinitions

[11]: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#service-account-token-volume-projection

[12]: https://azure.github.io/azure-workload-identity/installation.html
