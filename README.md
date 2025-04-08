# Azure AD Workload Identity

[![Build Status][14]][13]
[![OpenSSF Scorecard][23]][24]

Azure AD Workload Identity is the next iteration of [Azure AD Pod Identity][1] that enables Kubernetes applications to access Azure cloud resources securely with [Azure Active Directory][2] based on annotated [service accounts][3].

| Kubernetes Version | Supported |
| ------------------ | --------- |
| 1.32               | ✅        |
| 1.31               | ✅        |
| 1.30               | ✅        |

## Installation

Check out the [installation guide][12] on how to deploy the Azure AD Workload Identity webhook.

## Quick Start

Check out the Azure AD Workload Identity [Quick Start][4] on how to securely access Azure cloud resources from your Kubernetes workload using the Microsoft Authentication Library (MSAL).

## Code of Conduct

This project has adopted the [Microsoft Open Source Code of Conduct][17]. For more information, see the [Code of Conduct FAQ][18] or contact [opencode@microsoft.com][19] with any additional questions or comments.

## Release

Currently, Azure Workload Identity releases on a monthly basis, targeting the last week of the month.

## Support

Azure AD Workload Identity is an open source project that is [**not** covered by the Microsoft Azure support policy][20]. [Please search open issues here][21], and if your issue isn't already represented please [open a new one][22]. The project maintainers will respond to the best of their abilities.

<!-- - Ensure backward compatibility when upgrading from [AAD Pod Identity](https://github.com/Azure/aad-pod-identity). -->

[1]: https://github.com/Azure/aad-pod-identity
<!-- markdown-link-check-disable-next-line -->
[2]: https://azure.microsoft.com/products/active-directory/
[3]: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/
[4]: https://azure.github.io/azure-workload-identity/docs/quick-start.html
[5]: https://azure.github.io/azure-workload-identity/docs/installation/mutating-admission-webhook.html
[8]: https://azure.github.io/aad-pod-identity/docs/getting-started/role-assignment/
[9]: https://docs.microsoft.com/en-us/azure/virtual-machines/windows/instance-metadata-service?tabs=windows
[10]: https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/#customresourcedefinitions
[11]: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#service-account-token-volume-projection
[12]: https://azure.github.io/azure-workload-identity/docs/installation.html
[13]: https://dev.azure.com/AzureContainerUpstream/Azure%20Workload%20Identity/_build/latest?definitionId=365&branchName=main
[14]: https://dev.azure.com/AzureContainerUpstream/Azure%20Workload%20Identity/_apis/build/status/Azure%20Workload%20Identity%20Nightly?branchName=main
[15]: https://azure.github.io/azure-workload-identity/docs/known-issues.html#permission-denied-when-reading-the-projected-service-account-token-file
[17]: https://opensource.microsoft.com/codeofconduct/
[18]: https://opensource.microsoft.com/codeofconduct/faq
[19]: mailto:opencode@microsoft.com
[20]: https://support.microsoft.com/en-us/help/2941892/support-for-linux-and-open-source-technology-in-azure
[21]: https://github.com/Azure/azure-workload-identity/issues
[22]: https://github.com/Azure/azure-workload-identity/issues/new/choose
[23]: https://api.securityscorecards.dev/projects/github.com/Azure/azure-workload-identity/badge
[24]: https://api.securityscorecards.dev/projects/github.com/Azure/azure-workload-identity
