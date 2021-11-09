# Known Issues

## Permission denied when reading the projected service account token file

In Kubernetes 1.18, the default mode for the projected service account token file is `0600`. This causes containers running as non-root to fail while trying to read the token file:

```bash
F0826 20:03:20.113998 1 main.go:27] failed to get secret from keyvault, err: autorest/Client#Do: Preparing request failed: StatusCode=0 -- Original Error: failed to read service account token: open /var/run/secrets/tokens/azure-identity-token: permission denied
```

The default mode was changed to `0644` in Kubernetes v1.19, which allows containers running as non-root to read the projected service account token.

If you ran into this issue, you can either:

1. Upgrade your cluster to v1.20+ or

2. Apply the following `securityContext` field to your pod spec:

```yaml
spec:
  securityContext:
    fsGroup: 65534
```

## User tried to log in to a device from a platform (Unknown) that's currently not supported through Conditional Access policy

When creating a federated identity credential, your request might be blocked by Azure Active Directory [Conditional Access: Require compliant devices](https://docs.microsoft.com/en-us/azure/active-directory/conditional-access/howto-conditional-access-policy-compliant-device) policy:

```bash
az rest --method POST --uri "https://graph.microsoft.com/beta/applications/${APPLICATION_OBJECT_ID}/federatedIdentityCredentials" --body @body.json
AADSTS50005: User tried to log in to a device from a platform (Unknown) that's currently not supported through Conditional Access policy. Supported device platforms are: iOS, Android, Mac, and Windows flavors.
...
To re-authenticate, please run:
az login --scope https://graph.microsoft.com//.default
```

Another quick way to verify if your tenant has a conditional access policy in place:

```bash
az account get-access-token --resource-type=ms-graph
```

To bypass this policy:

- `az login` with a user account on a supported system - Windows or MacOS, and make the device compliant.
- `az login --service-principal` with a service principal which does not have the above compilance check.

In the case of service principal, you will have to grant the `Application.ReadWrite.All` API permission:

```bash
# get the app role ID of `Application.ReadWrite.All`
APPLICATION_OBJECT_ID="$(az ad app show --id ${APPLICATION_CLIENT_ID} --query objectId -otsv)"
GRAPH_RESOURCE_ID="$(az ad sp list --display-name "Microsoft Graph" --query '[0].objectId' -otsv)"
APPLICATION_READWRITE_ALL_ID="$(az ad sp list --display-name "Microsoft Graph" --query "[0].appRoles[?value=='Application.ReadWrite.All' && contains(allowedMemberTypes, 'Application')].id" --output tsv)"

URI="https://graph.microsoft.com/v1.0/servicePrincipals/${APPLICATION_OBJECT_ID}/appRoleAssignments"
BODY="{'principalId':'${APPLICATION_OBJECT_ID}','resourceId':'${GRAPH_RESOURCE_ID}','appRoleId':'${APPLICATION_READWRITE_ALL_ID}'}"
az rest --method post --uri "${URI}" --body "${BODY}" --headers "Content-Type=application/json"
```
