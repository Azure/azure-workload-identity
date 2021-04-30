package webhook

const (
	// usePodIdentityAnnotation represents the service account is to be used for pod identity
	usePodIdentityAnnotation = "azure.pod.identity/use"
	// clientIDAnnotation represents the clientID to be used with pod
	clientIDAnnotation = "azure.pod.identity/client-id"
	// tenantIDAnnotation represent the tenantID to be used with pod
	tenantIDAnnotation = "azure.pod.identity/tenant-id"
	// serviceAccountTokenExpiryAnnotation represents the expirationSeconds for projected service account token
	// [OPTIONAL] field. User might want to configure this to prevent any downtime caused by errors during service account token refresh.
	// Kubernetes service account token expiry will not be correlated with AAD tokens. AAD tokens expiry will be 24h.
	serviceAccountTokenExpiryAnnotation = "azure.pod.identity/service-account-token-expiration"
	// skipContainersAnnotation represents list of containers to skip added projected service account token volume
	// By default, the projected service account token volume will be added to all containers if the service account is annotated with clientID
	skipContainersAnnotation = "azure.pod.identity/skip-containers"

	// defaultServiceAccountTokenExpiration is the default service account token expiration in seconds
	defaultServiceAccountTokenExpiration = int64(86400)
	// minServiceAccountTokenExpiration is the minimum service account token expiration in seconds
	minServiceAccountTokenExpiration = int64(3600)
)
