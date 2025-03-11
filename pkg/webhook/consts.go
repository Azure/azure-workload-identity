package webhook

// Annotations and labels defined in service account
const (
	// UseWorkloadIdentityLabel represents the service account is to be used for workload identity
	UseWorkloadIdentityLabel = "azure.workload.identity/use"
	// ClientIDAnnotation represents the clientID to be used with pod
	ClientIDAnnotation = "azure.workload.identity/client-id"
	// TenantIDAnnotation represent the tenantID to be used with pod
	TenantIDAnnotation = "azure.workload.identity/tenant-id"
	// ServiceAccountTokenExpiryAnnotation represents the expirationSeconds for projected service account token
	// [OPTIONAL] field. User might want to configure this to prevent any downtime caused by errors during service account token refresh.
	// Kubernetes service account token expiry will not be correlated with AAD tokens. AAD tokens expiry will be 24h.
	ServiceAccountTokenExpiryAnnotation = "azure.workload.identity/service-account-token-expiration" // #nosec
	// SkipContainersAnnotation represents list of containers to skip adding projected service account token volume.
	// By default, the projected service account token volume will be added to all containers if the service account is labeled with `azure.workload.identity/use: true`
	SkipContainersAnnotation = "azure.workload.identity/skip-containers"
	// InjectProxySidecarAnnotation represents the annotation to be used to inject proxy sidecar into the pod
	InjectProxySidecarAnnotation = "azure.workload.identity/inject-proxy-sidecar"
	// ProxySidecarPortAnnotation represents the annotation to be used to specify the port for proxy sidecar
	ProxySidecarPortAnnotation = "azure.workload.identity/proxy-sidecar-port"

	// MinServiceAccountTokenExpiration is the minimum service account token expiration in seconds
	MinServiceAccountTokenExpiration = int64(3600)
	// MaxServiceAccountTokenExpiration is the maximum service account token expiration in seconds
	MaxServiceAccountTokenExpiration = int64(86400)
	// DefaultServiceAccountTokenExpiration is the default service account token expiration in seconds
	// This is the Kubernetes default value for projected service account token
	DefaultServiceAccountTokenExpiration = int64(3600)
	// DefaultProxySidecarPort is the default port for proxy sidecar
	DefaultProxySidecarPort = 8000
)

const (
	// ProxyInitContainerName is the name of the init container that will be used to inject proxy sidecar
	ProxyInitContainerName = "azwi-proxy-init"
	// ProxySidecarContainerName is the name of the container that will be used to inject proxy sidecar
	ProxySidecarContainerName = "azwi-proxy"
	// ProxyInitImageName is the name of the image that will be used to inject proxy init container
	ProxyInitImageName = "proxy-init"
	// ProxySidecarImageName is the name of the image that will be used to inject proxy sidecar
	ProxySidecarImageName = "proxy"
	// ProxyPortEnvVar is the environment variable name for the proxy port
	ProxyPortEnvVar = "PROXY_PORT"
)

// Environment variables injected in the pod
const (
	AzureClientIDEnvVar           = "AZURE_CLIENT_ID"
	AzureTenantIDEnvVar           = "AZURE_TENANT_ID"
	AzureFederatedTokenFileEnvVar = "AZURE_FEDERATED_TOKEN_FILE" // #nosec
	AzureAuthorityHostEnvVar      = "AZURE_AUTHORITY_HOST"
	TokenFilePathName             = "azure-identity-token"
	TokenFileMountPath            = "/var/run/secrets/azure/tokens" // #nosec
	// DefaultAudience is the audience added to the service account token audience
	// This value is to be consistent with other token exchange flows in AAD and has
	// no impact on the actual token exchange flow.
	DefaultAudience = "api://AzureADTokenExchange"
)
