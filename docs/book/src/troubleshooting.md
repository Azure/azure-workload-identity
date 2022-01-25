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

## Ensure the service account is labeled with `azure.workload.identity/use=true`

`azure.workload.identity/use=true` label on the service account represents the service account is to be used for workload identity. If the service account is not labeled, the mutating admission webhook will not inject the required environment variables or volume mounts into the workload pod.

Run the following command to check if the service account is labeled:

```bash
kubectl get sa workload-identity-sa -n oidc -o jsonpath='{.metadata.labels.azure\.workload\.identity/use}'
```

<details>
<summary>Output</summary>

```bash
kubectl get sa workload-identity-sa -n oidc -o jsonpath='{.metadata.labels.azure\.workload\.identity/use}'
true
```

</details>

## AADSTS70021: No matching federated identity record found for presented assertion.

```
token_credential.go:70] "failed to acquire token" err="FromAssertion(): http call(https://login.microsoftonline.com//{tenant-id}//oauth2/v2.0/token)(POST) error: reply status code was 400:\n{\"error\":\"invalid_request\",\"error_description\":\"AADSTS70021: No matching federated identity record found for presented assertion. Assertion Issuer: 'https://oidc.prod-aks.azure.com/XXXXXX/'. Assertion Subject: 'system:serviceaccount:default:workload-identity-sa'. Assertion Audience: 'api://AzureADTokenExchange'.\\r\\nTrace ID: b0f62116-10b6-4a73-bdb2-281524404e00\\r\\nCorrelation ID: 4a42e576-85bc-46ae-b7e3-b52cb8958917\\r\\nTimestamp: 2022-01-20 22:54:42Z\",\"error_codes\":[70021],\"timestamp\":\"2022-01-20 22:54:42Z\",\"trace_id\":\"b0f62116-10b6-4a73-bdb2-281524404e00\",\"correlation_id\":\"4a42e576-85bc-46ae-b7e3-b52cb8958917\",\"error_uri\":\"https://login.microsoftonline.com/error?code=70021\"}"
E0120 22:55:12.472912       1 token_credential.go:70] "failed to acquire token" err="FromAssertion(): http call(https://login.microsoftonline.com/{tenant-id}/oauth2/v2.0/token)(POST) error: reply status code was 400:\n{\"error\":\"invalid_request\",\"error_description\":\"AADSTS70021: No matching federated identity record found for presented assertion. Assertion Issuer: 'https://oidc.prod-aks.azure.com/XXXXXX/'. Assertion Subject: 'system:serviceaccount:default:workload-identity-sa'. Assertion Audience: 'api://AzureADTokenExchange'.\\r\\nTrace ID: 8f29172d-bf0d-4165-9d86-816665612d00\\r\\nCorrelation ID: 472f3de0-666f-411e-8d4c-cd46b6d6db26\\r\\nTimestamp: 2022-01-20 22:55:12Z\",\"error_codes\":[70021],\"timestamp\":\"2022-01-20 22:55:12Z\",\"trace_id\":\"8f29172d-bf0d-4165-9d86-816665612d00\",\"correlation_id\":\"472f3de0-666f-411e-8d4c-cd46b6d6db26\",\"error_uri\":\"https://login.microsoftonline.com/error?code=70021\"}"
```

If you encounter the error above, it means that the issuer of the service account token does not match the issuer you defined in the federated identity credential. In the case of an AKS cluster with OIDC issuer enabled, the most common cause is when the user is missing the trailing `/` when creating the federated identity credential (e.g. `https://oidc.prod-aks.azure.com/XXXXXX` vs `https://oidc.prod-aks.azure.com/XXXXXX/`).

You can follow [this guide](./installation/managed-clusters.md#steps-to-get-the-oidc-issuer-url-from-a-generic-managed-cluster) on how to get the token issuer of your cluster.

[1]: https://github.com/Azure/azure-workload-identity/issues/new
