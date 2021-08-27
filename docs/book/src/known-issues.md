# Known Issues

## Permission denied when reading the projected service account token file

In Kubernetes 1.18, the default mode for the projected service account token file is `0600`. This causes containers running as non-root to fail while trying to read the token file:

```bash
F0826 20:03:20.113998 1 main.go:27] failed to get secret from keyvault, err: autorest/Client#Do: Preparing request failed: StatusCode=0 -- Original Error: failed to read service account token: open /var/run/secrets/tokens/azure-identity-token: permission denied
```

The default mode was changed to `0644` in Kubernetes v1.19, which allows containers running as non-root to read the projected service account token.

If you ran into this issue, you can either:

1. Upgrade your cluster to v1.19+ or

2. Apply the following `securityContext` field to your pod spec:

```yaml
spec:
  securityContext:
    fsGroup: 65534
```
