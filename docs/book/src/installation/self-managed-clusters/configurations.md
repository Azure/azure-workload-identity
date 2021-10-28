# Configurations

The following configurations are required to be enabled/set in the cluster configuration for Azure AD Workload Identity to function properly.

<!-- toc -->

## kube-apiserver flags

This a list of required flags that need to be configured in the `kube-apiserver`. Refer to [kube-apiserver][1] for more available flags.

### `--service-account-issuer`

Identifier of the service account token issuer. The issuer will assert this identifier in "iss" claim of issued tokens. This value is a string or URI. If this option is not a valid URI per the OpenID Discovery 1.0 spec, the ServiceAccountIssuerDiscovery feature will remain disabled, even if the feature gate is set to true. It is highly recommended that this value comply with the OpenID spec: https://openid.net/specs/openid-connect-discovery-1_0.html. In practice, this means that service-account-issuer must be an https URL. It is also highly recommended that this URL be capable of serving OpenID discovery documents at {service-account-issuer}/.well-known/openid-configuration. When this flag is specified multiple times, the first is used to generate tokens and all are used to determine which issuers are accepted.

### `--service-account-signing-key-file`

Path to the file that contains the current private key of the service account token issuer. The issuer will sign issued ID tokens with this private key.

### `--service-account-key-file`

File containing PEM-encoded x509 RSA or ECDSA private or public keys, used to verify ServiceAccount tokens. The specified file can contain multiple keys, and the flag can be specified multiple times with different files. If unspecified, --tls-private-key-file is used. Must be specified when --service-account-signing-key is provided

## kube-controller-manager flags

This is a list of required flags that need to be configured in the `kube-controller-manager`. Refer to [kube-controller-manager][2] for more available flags.

### `--service-account-private-key-file`

Filename containing a PEM-encoded private RSA or ECDSA key used to sign service account tokens.

## Feature Flags

### Service Account Token Volume Projection

This feature is stable in Kubernetes v1.20 and is enabled by default. Refer to [Service Account Token Volume Projection][3] for more information.

[1]: https://kubernetes.io/docs/reference/command-line-tools-reference/kube-apiserver/

[2]: https://kubernetes.io/docs/reference/command-line-tools-reference/kube-controller-manager/

[3]: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#service-account-token-volume-projection
