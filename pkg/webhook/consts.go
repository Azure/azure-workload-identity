package webhook

// Annotations and labels defined in service account
const (
	// usePodIdentityLabel represents the service account is to be used for pod identity
	usePodIdentityLabel = "azure.pod.identity/use"
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

// Environment variables injected in the pod
const (
	azureClientIDEnvVar = "AZURE_CLIENT_ID"
	azureTenantIDEnvVar = "AZURE_TENANT_ID"
	tokenFilePathEnvVar = "TOKEN_FILE_PATH"
)
