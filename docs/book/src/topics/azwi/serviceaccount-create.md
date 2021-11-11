# `azwi serviceaccount create`

Create a workload identity.

## Synopsis

The "create" command executes the following phases in order:

    aad-application     Create Azure Active Directory (AAD) application and its underlying service principal
    service-account     Create Kubernetes service account in the current KUBECONFIG context and add azure-workload-identity labels and annotations to it
    federated-identity  Create federated identity credential between the AAD application and the Kubernetes service account
    role-assignment     Create role assignment between the AAD application and the Azure cloud resource

<!---->

    azwi serviceaccount create [flags]

## Options

          --aad-application-client-id string            Client ID of the AAD application. If not specified, it will be fetched using the AAD application name
          --aad-application-name string                 Name of the AAD application, If not specified, the namespace, the name of the service account and the hash of the issuer URL will be used
          --aad-application-object-id string            Object ID of the AAD application. If not specified, it will be fetched using the AAD application name
          --auth-method string                          auth method to use. Supported values: cli, client_secret, client_certificate (default "cli")
          --azure-env string                            the target Azure cloud (default "AzurePublicCloud")
          --azure-role string                           Role of the AAD application (see all available roles at https://docs.microsoft.com/en-us/azure/role-based-access-control/built-in-roles)
          --azure-scope string                          Scope at which the role assignment or definition applies to
          --certificate-path string                     path to client certificate (used with --auth-method=client_certificate)
          --client-id string                            client id (used with --auth-method=[client_secret|client_certificate])
          --client-secret string                        client secret (used with --auth-method=client_secret)
      -h, --help                                        help for create
          --private-key-path string                     path to private key (used with --auth-method=client_certificate)
          --service-account-issuer-url string           URL of the issuer
          --service-account-name string                 Name of the service account
          --service-account-namespace string            Namespace of the service account (default "default")
          --service-account-token-expiration duration   Expiration time of the service account token. Must be between 1 hour and 24 hours (default 1h0m0s)
          --service-principal-name string               Name of the service principal that backs the AAD application. If this is not specified, the name of the AAD application will be used
          --service-principal-object-id string          Object ID of the service principal that backs the AAD application. If not specified, it will be fetched using the service principal name
          --skip-phases strings                         List of phases to skip
      -s, --subscription-id string                      azure subscription id (required)

## Example

```bash
az login && az account set -s <SubscriptionID>
azwi serviceaccount create \
  --service-account-name azwi-sa \
  --service-account-issuer-url https://chuwon.blob.core.windows.net/oidc-test/ \
  --skip-phases role-assignment
```

<details>
<summary>Output</summary>

    INFO[0000] No subscription provided, using selected subscription from Azure CLI: <SubscriptionID>
    INFO[0003] skipping phase                                phase=role-assignment
    INFO[0003] [aad-application] created an AAD application  clientID=936ed007-52c2-4785-8c09-04eeca2e5970 name="default-azwi-sa-1g7d7NgSw9Q2EsSeafgx8uQKqR4q6zTrsPjDdrvN79Y=" objectID=19888f97-e0d3-4f61-8eb9-b87bf161e27d
    INFO[0003] [aad-application] created service principal   clientID=936ed007-52c2-4785-8c09-04eeca2e5970 name="default-azwi-sa-1g7d7NgSw9Q2EsSeafgx8uQKqR4q6zTrsPjDdrvN79Y=" objectID=4e3c51e5-ec74-40e2-8e28-2606803a048e
    INFO[0003] [service-account] created Kubernetes service account  name=azwi-sa namespace=default
    INFO[0004] [federated-identity] added federated credential  objectID=19888f97-e0d3-4f61-8eb9-b87bf161e27d subject="system:serviceaccount:default:azwi-sa"

</details>

## Invoke a single phase of the create workflow

To invoke a single phase of the create workflow:

```
azwi sa create phase <phase name>
```
