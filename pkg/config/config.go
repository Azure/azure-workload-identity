package config

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Config holds configuration from the env variables
type Config struct {
	Cloud          string `envconfig:"AZURE_ENVIRONMENT" default:"AzurePublicCloud"`
	TenantID       string `envconfig:"AZURE_TENANT_ID" required:"true"`
	ProxyImage     string `envconfig:"PROXY_IMAGE"`
	ProxyInitImage string `envconfig:"PROXY_INIT_IMAGE"`

	AzureKubernetesTokenProxy            string `envconfig:"AZURE_KUBERNETES_TOKEN_PROXY"`
	AzureKubernetesTokenEndpointFallback string `envconfig:"AZURE_KUBERNETES_TOKEN_ENDPOINT"` // this should be removed after AKS side deployment has been done

	AzureKubernetesSNIName string `envconfig:"AZURE_KUBERNETES_SNI_NAME"`
	AzureKubernetesCAData  string `envconfig:"AZURE_KUBERNETES_CA_DATA"`
	// AzureKubernetesCAConfigMapName is the name of the ConfigMap that contains the CA data
	// The key in the ConfigMap must be "ca.crt".
	AzureKubernetesCAConfigMapName    string           `envconfig:"AZURE_KUBERNETES_CA_CONFIGMAP_NAME"`
	AzureKubernetesCACTBSignerName    string           `envconfig:"AZURE_KUBERNETES_CA_CTB_SIGNER_NAME"`
	AzureKubernetesCACTBLabelSelector LabelSelectorPtr `envconfig:"AZURE_KUBERNETES_CA_CTB_LABEL_SELECTOR"`
}

// ParseConfig parses the configuration from env variables
func ParseConfig() (*Config, error) {
	c := new(Config)
	if err := envconfig.Process("config", c); err != nil {
		return c, err
	}

	if c.AzureKubernetesTokenProxy == "" && c.AzureKubernetesTokenEndpointFallback != "" {
		// for backward compatibility, if AZURE_KUBERNETES_TOKEN_PROXY is not set, use AZURE_KUBERNETES_TOKEN_ENDPOINT if set
		c.AzureKubernetesTokenProxy = c.AzureKubernetesTokenEndpointFallback
	}

	// validate parsed config
	if err := validateConfig(c); err != nil {
		return nil, err
	}
	return c, nil
}

// validateConfig validates the configuration
func validateConfig(c *Config) error {
	if len(c.TenantID) == 0 {
		return errors.New("AZURE_TENANT_ID is required")
	}

	// ca data, configmap name and signer name are mutually exclusive
	values := []string{
		c.AzureKubernetesCAData,
		c.AzureKubernetesCAConfigMapName,
		c.AzureKubernetesCACTBSignerName,
	}

	setCount := 0
	for _, v := range values {
		if len(v) > 0 {
			setCount++
		}
	}

	if setCount > 1 {
		return errors.New("only one of AZURE_KUBERNETES_CA_DATA, AZURE_KUBERNETES_CA_CONFIGMAP_NAME or AZURE_KUBERNETES_CA_CTB_SIGNER_NAME can be set")
	}

	if c.AzureKubernetesCACTBLabelSelector.Value != nil && len(c.AzureKubernetesCACTBSignerName) == 0 {
		return errors.New("AZURE_KUBERNETES_CA_CTB_SIGNER_NAME needs to be set when AZURE_KUBERNETES_CA_CTB_LABEL_SELECTOR is set")
	}

	return nil
}

type LabelSelectorPtr struct {
	Value *metav1.LabelSelector
}

// Decode implements envconfig.Decoder
func (l *LabelSelectorPtr) Decode(value string) error {
	if strings.TrimSpace(value) == "" {
		// env var not set or empty
		l.Value = nil
		return nil
	}

	var selector metav1.LabelSelector
	if err := json.Unmarshal([]byte(value), &selector); err != nil {
		return fmt.Errorf("failed to decode LabelSelector: %w", err)
	}
	l.Value = &selector
	return nil
}
