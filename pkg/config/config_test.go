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
