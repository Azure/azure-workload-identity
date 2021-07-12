# Introduction

AAD Pod Managed Identity is the next iteration of [AAD Pod Identity](https://github.com/Azure/aad-pod-identity) that enables Kubernetes applications to access Azure cloud resources securely with [Azure Active Directory](https://azure.microsoft.com/en-us/services/active-directory/) based on annotated [service accounts](https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/).


## Overview

The repository contains the following components:

1. Mutating Webhook
   > The webhook is for mutating pods that reference an annotated service account. The webhook will inject the environment variables and the projected service account token volume.

2. Proxy init and sidecar container
   > The init and sidecar container will be used for applications that are still using [AAD Pod Identity](https://github.com/Azure/aad-pod-identity).
