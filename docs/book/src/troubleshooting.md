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

[1]: https://github.com/Azure/azure-workload-identity/issues/new
