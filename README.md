# AAD Pod Managed Identity

AAD Pod Managed Identity enables Kubernetes applications to access cloud resources securely with Azure Active Directory based on annotated service accounts.

## Overview

This repo contains the following:

1. Mutating Webhook
   1. The webhook is for mutating pods that reference an annotated service account. The webhook will inject the environment variables and the projected service account token volume.
2. Proxy init and sidecar container
   1. The init and sidecar container will be used for applications that are still using the older versions of the library.

## Installation

### Install Webhook

1. Install [cert-manager]((https://github.com/jetstack/cert-manager))

   cert-manager is used for provisioning the certificates for the webhook server. Cert manager also has a component called CA injector, which is responsible for injecting the CA bundle into the MutatingWebhookConfiguration.

   ```bash
   kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v1.2.0/cert-manager.yaml
   ```

1. Deploy the webhook

   Replace the tenant ID and environment name in [here](https://github.com/Azure/aad-pod-managed-identity/blob/master/deploy/aad-pi-webhook.yaml#L41-L42) before executing

   ```bash
   kubectl apply -f deploy/aad-pi-webhook.yaml
   ```

1. Validate the webhook has been installed and is running

   ```bash
   kubectl get all -n aad-pi-webhook-system
   NAME                                                     READY   STATUS    RESTARTS   AGE
   pod/aad-pi-webhook-controller-manager-5fc5559ddd-rgj46   1/1     Running   0          8d

   NAME                                                        TYPE        CLUSTER-IP    EXTERNAL-IP   PORT(S)    AGE
   service/aad-pi-webhook-controller-manager-metrics-service   ClusterIP   10.0.123.94   <none>        8443/TCP   8d
   service/aad-pi-webhook-webhook-service                      ClusterIP   10.0.2.106    <none>        443/TCP    8d

   NAME                                                READY   UP-TO-DATE   AVAILABLE   AGE
   deployment.apps/aad-pi-webhook-controller-manager   1/1     1            1           8d

   NAME                                                           DESIRED   CURRENT   READY   AGE
   replicaset.apps/aad-pi-webhook-controller-manager-5fc5559ddd   1         1         1       8d
   ```

## Uninstall

### Uninstall Webhook

1. Delete webhook

   ```bash
   kubectl delete -f deploy/aad-pi-webhook.yaml
   ```

1. Delete cert-manager

   If you installed cert-manager for use with the aad-pod-managed-identity webhook, then delete the cert-manager components

   ```bash
   kubectl delete -f https://github.com/jetstack/cert-manager/releases/download/v1.2.0/cert-manager.yaml
   ```

## Contributing

This project welcomes contributions and suggestions.  Most contributions require you to agree to a
Contributor License Agreement (CLA) declaring that you have the right to, and actually do, grant us
the rights to use your contribution. For details, visit https://cla.opensource.microsoft.com.

When you submit a pull request, a CLA bot will automatically determine whether you need to provide
a CLA and decorate the PR appropriately (e.g., status check, comment). Simply follow the instructions
provided by the bot. You will only need to do this once across all repos using our CLA.

This project has adopted the [Microsoft Open Source Code of Conduct](https://opensource.microsoft.com/codeofconduct/).
For more information see the [Code of Conduct FAQ](https://opensource.microsoft.com/codeofconduct/faq/) or
contact [opencode@microsoft.com](mailto:opencode@microsoft.com) with any additional questions or comments.

## Trademarks

This project may contain trademarks or logos for projects, products, or services. Authorized use of Microsoft
trademarks or logos is subject to and must follow
[Microsoft's Trademark & Brand Guidelines](https://www.microsoft.com/en-us/legal/intellectualproperty/trademarks/usage/general).
Use of Microsoft trademarks or logos in modified versions of this project must not cause confusion or imply Microsoft sponsorship.
Any use of third-party trademarks or logos are subject to those third-party's policies.
