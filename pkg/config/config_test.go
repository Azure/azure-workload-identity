package config

import (
	"os"
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestParseConfig(t *testing.T) {
	tests := []struct {
		name             string
		cloud            string
		tenantID         string
		caData           string
		caConfigMapName  string
		caCTBSignerName  string
		ctbLabelSelector string
		wantErr          string
		wantConfig       *Config
	}{
		{
			name:     "cloud name defaulting to AzurePublicCloud",
			cloud:    "",
			tenantID: "tenant-id",
			wantConfig: &Config{
				Cloud:                             "AzurePublicCloud",
				TenantID:                          "tenant-id",
				AzureKubernetesCACTBLabelSelector: LabelSelectorPtr{},
			},
		},
		{
			name:     "cloud name set to AzureChinaCloud",
			cloud:    "AzureChinaCloud",
			tenantID: "tenant-id",
			wantConfig: &Config{
				Cloud:                             "AzureChinaCloud",
				TenantID:                          "tenant-id",
				AzureKubernetesCACTBLabelSelector: LabelSelectorPtr{},
			},
		},
		{
			name:     "missing tenant id should return error",
			cloud:    "AzureChinaCloud",
			tenantID: "",
			wantErr:  `required key AZURE_TENANT_ID missing value`,
		},
		{
			name:             "valid matchLabels only",
			tenantID:         "tenant-id",
			caCTBSignerName:  "ctb-signer",
			ctbLabelSelector: `{"matchLabels":{"app":"nginx","tier":"frontend"}}`,
			wantConfig: &Config{
				Cloud:                          "AzurePublicCloud",
				TenantID:                       "tenant-id",
				AzureKubernetesCACTBSignerName: "ctb-signer",
				AzureKubernetesCACTBLabelSelector: LabelSelectorPtr{
					Value: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app":  "nginx",
							"tier": "frontend",
						},
					},
				},
			},
		},
		{
			name:             "valid matchExpressions only",
			tenantID:         "tenant-id",
			caCTBSignerName:  "ctb-signer",
			ctbLabelSelector: `{"matchExpressions":[{"key":"env","operator":"In","values":["prod","staging"]}]}`,
			wantConfig: &Config{
				Cloud:                          "AzurePublicCloud",
				TenantID:                       "tenant-id",
				AzureKubernetesCACTBSignerName: "ctb-signer",
				AzureKubernetesCACTBLabelSelector: LabelSelectorPtr{
					Value: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      "env",
								Operator: metav1.LabelSelectorOpIn,
								Values:   []string{"prod", "staging"},
							},
						},
					},
				},
			},
		},
		{
			name:             "valid both matchLabels and matchExpressions",
			tenantID:         "tenant-id",
			caCTBSignerName:  "ctb-signer",
			ctbLabelSelector: `{"matchLabels":{"role":"api"},"matchExpressions":[{"key":"env","operator":"NotIn","values":["dev"]}]}`,
			wantConfig: &Config{
				Cloud:                          "AzurePublicCloud",
				TenantID:                       "tenant-id",
				AzureKubernetesCACTBSignerName: "ctb-signer",
				AzureKubernetesCACTBLabelSelector: LabelSelectorPtr{
					Value: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"role": "api",
						},
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      "env",
								Operator: metav1.LabelSelectorOpNotIn,
								Values:   []string{"dev"},
							},
						},
					},
				},
			},
		},
		{
			name:             "invalid JSON in label selector",
			tenantID:         "tenant-id",
			caCTBSignerName:  "ctb-signer",
			ctbLabelSelector: `not-a-json`,
			wantConfig:       nil,
			wantErr:          `envconfig.Process: assigning CONFIG_AZURE_KUBERNETES_CA_CTB_LABEL_SELECTOR to AzureKubernetesCACTBLabelSelector: converting 'not-a-json' to type config.LabelSelectorPtr. details: failed to decode LabelSelector: invalid character 'o' in literal null (expecting 'u')`,
		},
		{
			name:             "missing values with In operator (still valid structurally)",
			tenantID:         "tenant-id",
			caCTBSignerName:  "ctb-signer",
			ctbLabelSelector: `{"matchExpressions":[{"key":"env","operator":"In"}]}`,
			wantConfig: &Config{
				Cloud:                          "AzurePublicCloud",
				TenantID:                       "tenant-id",
				AzureKubernetesCACTBSignerName: "ctb-signer",
				AzureKubernetesCACTBLabelSelector: LabelSelectorPtr{
					Value: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      "env",
								Operator: metav1.LabelSelectorOpIn,
							},
						},
					},
				},
			},
		},
		{
			name:            "ca data and configmap mutually exclusive",
			cloud:           "AzurePublicCloud",
			tenantID:        "tenant-id",
			caData:          "some-ca-data",
			caConfigMapName: "ca-configmap",
			wantErr:         "only one of AZURE_KUBERNETES_CA_DATA, AZURE_KUBERNETES_CA_CONFIGMAP_NAME or AZURE_KUBERNETES_CA_CTB_SIGNER_NAME can be set",
		},
		{
			name:            "ca data and signer name mutually exclusive",
			cloud:           "AzurePublicCloud",
			tenantID:        "tenant-id",
			caData:          "some-ca-data",
			caCTBSignerName: "ctb-signer",
			wantErr:         "only one of AZURE_KUBERNETES_CA_DATA, AZURE_KUBERNETES_CA_CONFIGMAP_NAME or AZURE_KUBERNETES_CA_CTB_SIGNER_NAME can be set",
		},
		{
			name:            "configmap name and signer name mutually exclusive",
			cloud:           "AzurePublicCloud",
			tenantID:        "tenant-id",
			caConfigMapName: "ca-configmap",
			caCTBSignerName: "ctb-signer",
			wantErr:         "only one of AZURE_KUBERNETES_CA_DATA, AZURE_KUBERNETES_CA_CONFIGMAP_NAME or AZURE_KUBERNETES_CA_CTB_SIGNER_NAME can be set",
		},
		{
			name:             "signer name required with label selector",
			cloud:            "AzurePublicCloud",
			tenantID:         "tenant-id",
			ctbLabelSelector: `{"matchLabels":{"app":"nginx"}}`,
			wantErr:          "AZURE_KUBERNETES_CA_CTB_SIGNER_NAME needs to be set when AZURE_KUBERNETES_CA_CTB_LABEL_SELECTOR is set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setEnvIfNotEmpty(t, "AZURE_ENVIRONMENT", tt.cloud)
			setEnvIfNotEmpty(t, "AZURE_TENANT_ID", tt.tenantID)
			setEnvIfNotEmpty(t, "AZURE_KUBERNETES_CA_DATA", tt.caData)
			setEnvIfNotEmpty(t, "AZURE_KUBERNETES_CA_CONFIGMAP_NAME", tt.caConfigMapName)
			setEnvIfNotEmpty(t, "AZURE_KUBERNETES_CA_CTB_SIGNER_NAME", tt.caCTBSignerName)
			setEnvIfNotEmpty(t, "AZURE_KUBERNETES_CA_CTB_LABEL_SELECTOR", tt.ctbLabelSelector)

			c, err := ParseConfig()
			if len(tt.wantErr) > 0 {
				if err == nil || tt.wantErr != err.Error() {
					t.Fatalf("ParseConfig() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Fatalf("ParseConfig() unexpected error = %v", err)
				}
				if !reflect.DeepEqual(c, tt.wantConfig) {
					t.Errorf("ParseConfig() got = %v, want %v", c, tt.wantConfig)
				}
			}
		})
	}
}

func TestParseConfig_TokenEndpointFallback(t *testing.T) {
	const (
		tokenEndpointValue = "https://token-endpoint"
		tokenProxyValue    = "https://token-proxy"
	)

	t.Run("use token endpoint if token proxy is not set", func(t *testing.T) {
		setEnvIfNotEmpty(t, "AZURE_ENVIRONMENT", "AzurePublicCloud")
		setEnvIfNotEmpty(t, "AZURE_TENANT_ID", "tenant-id")
		setEnvIfNotEmpty(t, "AZURE_KUBERNETES_TOKEN_ENDPOINT", tokenEndpointValue)
		c, err := ParseConfig()
		if err != nil {
			t.Fatalf("ParseConfig() unexpected error = %v", err)
		}
		if c.AzureKubernetesTokenProxy != tokenEndpointValue {
			t.Errorf("ParseConfig() got = %v, want %v", c.AzureKubernetesTokenProxy, tokenEndpointValue)
		}
	})

	t.Run("use token proxy if both token proxy and token endpoint are set", func(t *testing.T) {
		setEnvIfNotEmpty(t, "AZURE_ENVIRONMENT", "AzurePublicCloud")
		setEnvIfNotEmpty(t, "AZURE_TENANT_ID", "tenant-id")
		setEnvIfNotEmpty(t, "AZURE_KUBERNETES_TOKEN_ENDPOINT", tokenEndpointValue)
		setEnvIfNotEmpty(t, "AZURE_KUBERNETES_TOKEN_PROXY", tokenProxyValue)
		c, err := ParseConfig()
		if err != nil {
			t.Fatalf("ParseConfig() unexpected error = %v", err)
		}
		if c.AzureKubernetesTokenProxy != tokenProxyValue {
			t.Errorf("ParseConfig() got = %v, want %v", c.AzureKubernetesTokenProxy, tokenProxyValue)
		}
	})

	t.Run("empty if neither token proxy nor token endpoint are set", func(t *testing.T) {
		setEnvIfNotEmpty(t, "AZURE_ENVIRONMENT", "AzurePublicCloud")
		setEnvIfNotEmpty(t, "AZURE_TENANT_ID", "tenant-id")
		// ensure both are unset
		_ = os.Unsetenv("AZURE_KUBERNETES_TOKEN_ENDPOINT")
		_ = os.Unsetenv("AZURE_KUBERNETES_TOKEN_PROXY")

		c, err := ParseConfig()
		if err != nil {
			t.Fatalf("ParseConfig() unexpected error = %v", err)
		}
		if c.AzureKubernetesTokenProxy != "" {
			t.Errorf("ParseConfig() got = %v, want empty string", c.AzureKubernetesTokenProxy)
		}
	})
}

func setEnvIfNotEmpty(t *testing.T, key, value string) {
	t.Helper()

	_ = os.Unsetenv(key) // Clear any existing value
	if len(value) > 0 {
		_ = os.Setenv(key, value)
	}
	t.Cleanup(func() {
		_ = os.Unsetenv(key) // Ensure cleanup after test
	})
}
