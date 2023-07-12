# Troubleshooting

<!-- toc -->

An overview of a list of components to assist in troubleshooting.

## Logging

Below is a list of commands you can use to view relevant logs of azure-workload-identity components.

### Mutating Admission Webhook

To get the logs of the mutating admission webhook, run the following command:

```bash
kubectl logs -n azure-workload-identity-system -l app=workload-identity-webhook
```

#### Isolate errors from logs

You can use `grep ^E` and `--since` flag from kubectl to isolate any errors occurred after a given duration.

```bash
kubectl logs -n azure-workload-identity-system -l app=workload-identity-webhook --since=1h | grep ^E
```

> It is always a good idea to include relevant logs from the webhook when opening a new [issue][1]

## AADSTS70021: No matching federated identity record found for presented assertion.

```
token_credential.go:70] "failed to acquire token" err="FromAssertion(): http call(https://login.microsoftonline.com//{tenant-id}//oauth2/v2.0/token)(POST) error: reply status code was 400:\n{\"error\":\"invalid_request\",\"error_description\":\"AADSTS70021: No matching federated identity record found for presented assertion. Assertion Issuer: 'https://oidc.prod-aks.azure.com/XXXXXX/'. Assertion Subject: 'system:serviceaccount:default:workload-identity-sa'. Assertion Audience: 'api://AzureADTokenExchange'.\\r\\nTrace ID: b0f62116-10b6-4a73-bdb2-281524404e00\\r\\nCorrelation ID: 4a42e576-85bc-46ae-b7e3-b52cb8958917\\r\\nTimestamp: 2022-01-20 22:54:42Z\",\"error_codes\":[70021],\"timestamp\":\"2022-01-20 22:54:42Z\",\"trace_id\":\"b0f62116-10b6-4a73-bdb2-281524404e00\",\"correlation_id\":\"4a42e576-85bc-46ae-b7e3-b52cb8958917\",\"error_uri\":\"https://login.microsoftonline.com/error?code=70021\"}"
E0120 22:55:12.472912       1 token_credential.go:70] "failed to acquire token" err="FromAssertion(): http call(https://login.microsoftonline.com/{tenant-id}/oauth2/v2.0/token)(POST) error: reply status code was 400:\n{\"error\":\"invalid_request\",\"error_description\":\"AADSTS70021: No matching federated identity record found for presented assertion. Assertion Issuer: 'https://oidc.prod-aks.azure.com/XXXXXX/'. Assertion Subject: 'system:serviceaccount:default:workload-identity-sa'. Assertion Audience: 'api://AzureADTokenExchange'.\\r\\nTrace ID: 8f29172d-bf0d-4165-9d86-816665612d00\\r\\nCorrelation ID: 472f3de0-666f-411e-8d4c-cd46b6d6db26\\r\\nTimestamp: 2022-01-20 22:55:12Z\",\"error_codes\":[70021],\"timestamp\":\"2022-01-20 22:55:12Z\",\"trace_id\":\"8f29172d-bf0d-4165-9d86-816665612d00\",\"correlation_id\":\"472f3de0-666f-411e-8d4c-cd46b6d6db26\",\"error_uri\":\"https://login.microsoftonline.com/error?code=70021\"}"
```

If you encounter the error above, it means that the issuer of the service account token does not match the issuer you defined in the federated identity credential. In the case of an AKS cluster with OIDC issuer enabled, the most common cause is when the user is missing the trailing `/` when creating the federated identity credential (e.g. `https://oidc.prod-aks.azure.com/XXXXXX` vs `https://oidc.prod-aks.azure.com/XXXXXX/`).

You can follow [this guide](./installation/managed-clusters.md#steps-to-get-the-oidc-issuer-url-from-a-generic-managed-cluster) on how to get the token issuer of your cluster.

[1]: https://github.com/Azure/azure-workload-identity/issues/new

## AADSTS90061: Request to External OIDC endpoint failed.

```
 ---> MSAL.NetCore.4.39.0.0.MsalServiceException:
        ErrorCode: invalid_request
Microsoft.Identity.Client.MsalServiceException: AADSTS90061: Request to External OIDC endpoint failed.
Trace ID: 86c81de3-efbd-4c0f-854d-cb8f329e5b00
Correlation ID: a92568a5-cc37-4136-ad4d-1ba82988abc2
Timestamp: 2022-10-21 04:36:48Z
   at Microsoft.Identity.Client.Internal.Requests.RequestBase.HandleTokenRefreshErrorAsync(MsalServiceException e, MsalAccessTokenCacheItem cachedAccessTokenItem)
   at Microsoft.Identity.Client.Internal.Requests.ClientCredentialRequest.ExecuteAsync(CancellationToken cancellationToken)
   at Microsoft.Identity.Client.Internal.Requests.RequestBase.RunAsync(CancellationToken cancellationToken)
   at Microsoft.Identity.Client.ApiConfig.Executors.ConfidentialClientExecutor.ExecuteAsync(AcquireTokenCommonParameters commonParameters, AcquireTokenForClientParameters clientParameters, CancellationToken cancellationToken)
   at Azure.Identity.AbstractAcquireTokenParameterBuilderExtensions.ExecuteAsync[T](AbstractAcquireTokenParameterBuilder`1 builder, Boolean async, CancellationToken cancellationToken)
   at Azure.Identity.MsalConfidentialClient.AcquireTokenForClientCoreAsync(String[] scopes, String tenantId, Boolean async, CancellationToken cancellationToken)
   at Azure.Identity.MsalConfidentialClient.AcquireTokenForClientAsync(String[] scopes, String tenantId, Boolean async, CancellationToken cancellationToken)
   at Azure.Identity.ClientAssertionCredential.GetTokenAsync(TokenRequestContext requestContext, CancellationToken cancellationToken)
        StatusCode: 400
        ResponseBody: {"error":"invalid_request","error_description":"AADSTS90061: Request to External OIDC endpoint failed.\r\nTrace ID: 86c81de3-efbd-4c0f-854d-cb8f329e5b00\r\nCorrelation ID: a92568a5-cc37-4136-ad4d-1ba82988abc2\r\nTimestamp: 2022-10-21 04:36:48Z","error_codes":[90061],"timestamp":"2022-10-21 04:36:48Z","trace_id":"86c81de3-efbd-4c0f-854d-cb8f329e5b00","correlation_id":"a92568a5-cc37-4136-ad4d-1ba82988abc2"}
...
```

If you encounter the error above, it means the OIDC issuer endpoint is not exposed to the internet or is inaccessible. You can verify this by running the following commands to check if the endpoints are accessible:

```bash
export SERVICE_ACCOUNT_ISSUER="<your service account issuer url>" # see section 1.1 on how to get the service account issuer url

# check if the OIDC issuer endpoint is accessible
curl ${SERVICE_ACCOUNT_ISSUER}/.well-known/openid-configuration
curl ${SERVICE_ACCOUNT_ISSUER}/openid/v1/jwks
```
<!-- markdown-link-check-disable-next-line -->
If you're seeing this issue with an AKS cluster, to resolve the issue try to reconcile the cluster by running [`az aks update`](https://learn.microsoft.com/cli/azure/aks?view=azure-cli-latest#az-aks-update). If the issue persists after reconciliation, create an [Azure support ticket](https://azure.microsoft.com/support/create-ticket).

## Workload pod doesn't have the Azure specific environment variables and projected service account token volume after upgrading to v1.0.0

As of v1.0.0 release, the azure-workload-identity mutating admission webhook is defaulting to using `failurePolicy: Fail` instead of `Ignore`. With this change, we have added an [object selector](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#matching-requests-objectselector) in the configuration to only intercept and mutate pods that have the `azure.workload.identity/use: "true"` label. This change reduces the latency impact of the webhook and prevents workload pods that require the injected environment variables and projected service account token volume from starting in an unexpected state. Refer to [issue](https://github.com/Azure/azure-workload-identity/issues/601) for more details.

If you are upgrading from a previous version of the azure-workload-identity, you will need to add the `azure.workload.identity/use: "true"` label to your workload pods to ensure that the mutating admission webhook is able to inject the required environment variables and projected service account token volume.

Run the following command to check if the workload pod is labeled:

```bash
kubectl get pod quick-start -n oidc -o jsonpath='{.metadata.labels.azure\.workload\.identity/use}'
```

<details>
<summary>Output</summary>

```bash
kubectl get pod quick-start -n oidc -o jsonpath='{.metadata.labels.azure\.workload\.identity/use}'
true
```

</details>
