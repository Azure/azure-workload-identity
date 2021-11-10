# `azwi serviceaccount delete`

Delete a workload identity.

## Synopsis

The "delete" command executes the following phases in order:

    role-assignment     Delete the role assignment between the AAD application and the Azure cloud resource
    federated-identity  Delete federated identity credential for the AAD application and the Kubernetes service account
    service-account     Delete the Kubernetes service account in the current KUBECONFIG context
    aad-application     Delete the Azure Active Directory (AAD) application and its underlying service principal

<!---->

    azwi serviceaccount delete [flags]

## Options

          --aad-application-name string         Name of the AAD application. If not specified, the namespace, the name of the service account and the hash of the issuer URL will be used
          --aad-application-object-id string    Object ID of the AAD application. If not specified, it will be fetched using the AAD application name
          --auth-method string                  auth method to use. Supported values: cli, client_secret, client_certificate (default "cli")
          --azure-env string                    the target Azure cloud (default "AzurePublicCloud")
          --certificate-path string             path to client certificate (used with --auth-method=client_certificate)
          --client-id string                    client id (used with --auth-method=[client_secret|client_certificate])
          --client-secret string                client secret (used with --auth-method=client_secret)
      -h, --help                                help for delete
          --private-key-path string             path to private key (used with --auth-method=client_certificate)
          --role-assignment-id string           Azure role assignment ID
          --service-account-issuer-url string   URL of the issuer
          --service-account-name string         Name of the service account
          --service-account-namespace string    Namespace of the service account (default "default")
          --skip-phases strings                 List of phases to skip
      -s, --subscription-id string              azure subscription id (required)

## Example

```bash
az login && az account set -s <SubscriptionID>
azwi sa delete \
  --service-account-name azwi-sa \
  --service-account-issuer-url https://chuwon.blob.core.windows.net/oidc-test/ \
  --skip-phases role-assignment
```

<details>
<summary>Output</summary>

    INFO[0000] No subscription provided, using selected subscription from Azure CLI: <SubscriptionID>
    INFO[0001] skipping phase                                phase=role-assignment
    INFO[0001] [federated-identity] deleted federated identity credential  issuerURL="https://chuwon.blob.core.windows.net/oidc-test/" subject="system:serviceaccount:default:azwi-sa"
    INFO[0001] [service-account] deleted service account     name=azwi-sa namespace=default
    INFO[0001] [aad-application] deleted aad application     objectID=19888f97-e0d3-4f61-8eb9-b87bf161e27d

</details>

## Invokes a single phase of the delete workflow

To Invokes a single phase of the delete workflow:

```
azwi sa create phase <phase name>
```
