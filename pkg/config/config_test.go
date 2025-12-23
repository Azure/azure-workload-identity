package config

import (
	"os"
	"testing"
)

func TestParseConfig(t *testing.T) {
	tests := []struct {
		name      string
		cloud     string
		tenantID  string
		wantErr   bool
		wantCloud string
	}{
		{
			name:      "cloud name defaulting to AzurePublicCloud",
			cloud:     "",
			tenantID:  "tenant-id",
			wantCloud: "AzurePublicCloud",
			wantErr:   false,
		},
		{
			name:      "cloud name set to AzureChinaCloud",
			cloud:     "AzureChinaCloud",
			tenantID:  "tenant-id",
			wantCloud: "AzureChinaCloud",
			wantErr:   false,
		},
		{
			name:      "missing tenant id should return error",
			cloud:     "AzureChinaCloud",
			tenantID:  "",
			wantCloud: "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("AZURE_TENANT_ID", tt.tenantID)
			os.Setenv("AZURE_ENVIRONMENT", tt.cloud)
			defer func() {
				os.Unsetenv("AZURE_TENANT_ID")
				os.Unsetenv("AZURE_ENVIRONMENT")
			}()

			c, err := ParseConfig()
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if c.Cloud != tt.cloud {
					t.Errorf("ParseConfig() got = %v, want %v", c.Cloud, tt.cloud)
				}
				if c.TenantID != tt.tenantID {
					t.Errorf("ParseConfig() got = %v, want %v", c.TenantID, tt.tenantID)
				}
			}
		})
	}
}

func TestParseConfigProxyImages(t *testing.T) {
	tests := []struct {
		name               string
		tenantID           string
		proxyImage         string
		proxyInitImage     string
		wantProxyImage     string
		wantProxyInitImage string
	}{
		{
			name:               "default empty proxy images",
			tenantID:           "tenant-id",
			proxyImage:         "",
			proxyInitImage:     "",
			wantProxyImage:     "",
			wantProxyInitImage: "",
		},
		{
			name:               "custom proxy images",
			tenantID:           "tenant-id",
			proxyImage:         "my-registry.com/proxy:v2.0.0",
			proxyInitImage:     "my-registry.com/proxy-init:v2.0.0",
			wantProxyImage:     "my-registry.com/proxy:v2.0.0",
			wantProxyInitImage: "my-registry.com/proxy-init:v2.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("AZURE_TENANT_ID", tt.tenantID)
			os.Setenv("PROXY_IMAGE", tt.proxyImage)
			os.Setenv("PROXY_INIT_IMAGE", tt.proxyInitImage)
			defer func() {
				os.Unsetenv("AZURE_TENANT_ID")
				os.Unsetenv("PROXY_IMAGE")
				os.Unsetenv("PROXY_INIT_IMAGE")
			}()

			c, err := ParseConfig()
			if err != nil {
				t.Fatalf("ParseConfig() error = %v", err)
			}
			if c.ProxyImage != tt.wantProxyImage {
				t.Errorf("ParseConfig() ProxyImage = %v, want %v", c.ProxyImage, tt.wantProxyImage)
			}
			if c.ProxyInitImage != tt.wantProxyInitImage {
				t.Errorf("ParseConfig() ProxyInitImage = %v, want %v", c.ProxyInitImage, tt.wantProxyInitImage)
			}
		})
	}
}
