package proxy

import (
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"k8s.io/apimachinery/pkg/util/cache"
)

const (
	defaultCacheSize = 128
	defaultCacheTTL  = 25 * time.Hour // WI token is valid for 24 hours, so set TTL to 25 hours to allow extended usage
)

type wiCredCacheKey struct {
	clientID string
	tenantID string

	// NOTE:
	// - authorityHost is inferred from the environment variable in the Azure Go SDK, so omitting it from cache key
	// - token file path is omitted as the proxy is using the same environment variable as the Azure Go SDK
}

// CredCache is an LRU cache for azidentity.WorkloadIdentityCredential
type CredCache struct {
	cache *cache.LRUExpireCache
}

func (c *CredCache) Get(key wiCredCacheKey) (*azidentity.WorkloadIdentityCredential, bool) {
	value, ok := c.cache.Get(key)
	if !ok {
		return nil, false
	}
	cred, ok := value.(*azidentity.WorkloadIdentityCredential)
	return cred, ok
}

func (c *CredCache) Add(key wiCredCacheKey, cred *azidentity.WorkloadIdentityCredential) {
	c.cache.Add(key, cred, defaultCacheTTL)
}

// CreateWICredCache creates a new LRU cache for azidentity.WorkloadIdentityCredential with a default size of 128.
func CreateWICredCache() *CredCache {
	return &CredCache{
		cache: cache.NewLRUExpireCache(defaultCacheSize),
	}
}
