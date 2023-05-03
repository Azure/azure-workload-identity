package config

import (
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
)

// Config holds configuration from the env variables
type Config struct {
	Cloud               string `envconfig:"AZURE_ENVIRONMENT"`
	TenantID            string `envconfig:"AZURE_TENANT_ID"`
	IsArcEnabledCluster bool   `envconfig:"IS_ARC_ENABLED_CLUSTER" default:"false"`
}

// ParseConfig parses the configuration from env variables
func ParseConfig() (*Config, error) {
	c := new(Config)
	if err := envconfig.Process("config", c); err != nil {
		return c, err
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
		return errors.New("tenant ID is required")
	}
	return nil
}
