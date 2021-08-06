package webhook

// Annotations and labels defined in service account
const (
	// UsePodIdentityLabel represents the service account is to be used for pod identity
	UsePodIdentityLabel = "azure.pod.identity/use"
	// ClientIDAnnotation represents the clientID to be used with pod
	ClientIDAnnotation = "azure.pod.identity/client-id"
	// TenantIDAnnotation represent the tenantID to be used with pod
	TenantIDAnnotation = "azure.pod.identity/tenant-id"
	// ServiceAccountTokenExpiryAnnotation represents the expirationSeconds for projected service account token
	// [OPTIONAL] field. User might want to configure this to prevent any downtime caused by errors during service account token refresh.
	// Kubernetes service account token expiry will not be correlated with AAD tokens. AAD tokens expiry will be 24h.
	ServiceAccountTokenExpiryAnnotation = "azure.pod.identity/service-account-token-expiration" // #nosec
	// SkipContainersAnnotation represents list of containers to skip adding projected service account token volume.
	// By default, the projected service account token volume will be added to all containers if the service account is labeled with `azure.pod.identity/use: true`
	SkipContainersAnnotation = "azure.pod.identity/skip-containers"

	// DefaultServiceAccountTokenExpiration is the default service account token expiration in seconds
	DefaultServiceAccountTokenExpiration = int64(86400)
	// MinServiceAccountTokenExpiration is the minimum service account token expiration in seconds
	MinServiceAccountTokenExpiration = int64(3600)
)

// Environment variables injected in the pod
const (
	AzureClientIDEnvVar           = "AZURE_CLIENT_ID"
	AzureTenantIDEnvVar           = "AZURE_TENANT_ID"
	AzureFederatedTokenFileEnvVar = "AZURE_FEDERATED_TOKEN_FILE" // #nosec
	AzureAuthorityHostEnvVar      = "AZURE_AUTHORITY_HOST"
	TokenFilePathName             = "azure-identity-token"
	TokenFileMountPath            = "/var/run/secrets/tokens" // #nosec
	// DefaultAudience is the audience added to the service account token audience
	// This value is to be consistent with other token exchange flows in AAD and has
	// no impact on the actual token exchange flow.
	DefaultAudience = "api://AzureADTokenExchange"
)
