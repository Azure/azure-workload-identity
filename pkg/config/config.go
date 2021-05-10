package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

// Config holds configuration from azure.json
type Config struct {
	Cloud          string `json:"cloud" yaml:"cloud"`
	TenantID       string `json:"tenantId" yaml:"tenantId"`
	SubscriptionID string `json:"subscriptionId" yaml:"subscriptionId"`
}

// ParseConfig parses the configuration from azure.json or env variables
func ParseConfig(configFile string) (*Config, error) {
	c := new(Config)
	if configFile != "" {
		bytes, err := os.ReadFile(configFile)
		if err != nil {
			return nil, fmt.Errorf("unable to read config file %s, error: %w", configFile, err)
		}
		if err = yaml.Unmarshal(bytes, &c); err != nil {
			return nil, fmt.Errorf("unable to unmarshal JSON, error: %w", err)
		}
	} else {
		c.Cloud = os.Getenv("AZURE_ENVIRONMENT")
		c.TenantID = os.Getenv("AZURE_TENANT_ID")
		c.SubscriptionID = os.Getenv("AZURE_SUBSCRIPTION_ID")
	}
	// validate parsed config
	if err := validateConfig(c); err != nil {
		return nil, err
	}
	return c, nil
}

// validateConfig validates the configuration
func validateConfig(c *Config) error {
	if c.TenantID == "" {
		return fmt.Errorf("tenant ID is required")
	}
	return nil
}
