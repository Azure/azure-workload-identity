package proxy

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	lru "github.com/hashicorp/golang-lru/v2"
)

const defaultCacheSize = 128

type wiCredCacheKey struct {
	clientID string
	tenantID string

	// NOTE:
	// - authorityHost is inferred from the environment variable in the Azure Go SDK, so omitting it from cache key
	// - token file path is omitted as the proxy is using the same environment variable as the Azure Go SDK
}

// CredCache is an LRU cache for azidentity.WorkloadIdentityCredential
type CredCache = lru.Cache[wiCredCacheKey, *azidentity.WorkloadIdentityCredential]

// CreateWICredCache creates a new LRU cache for azidentity.WorkloadIdentityCredential with a default size of 128.
func CreateWICredCache() (*CredCache, error) {
	return lru.New[wiCredCacheKey, *azidentity.WorkloadIdentityCredential](defaultCacheSize)
}
