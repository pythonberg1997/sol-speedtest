package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadConfigFromFile loads configuration from a YAML file
func LoadConfigFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := validateConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// validateConfig validates the configuration
func validateConfig(cfg *Config) error {
	if cfg.TestCount <= 0 {
		cfg.TestCount = 1000
	}

	if cfg.Timeout <= 0 {
		cfg.Timeout = 30
	}

	if cfg.OutputPath == "" {
		cfg.OutputPath = "results"
	}

	if len(cfg.Providers) == 0 {
		return fmt.Errorf("no providers configured")
	}

	for i, provider := range cfg.Providers {
		if provider.Name == "" {
			return fmt.Errorf("provider at index %d has no name", i)
		}

		if len(provider.Endpoints) == 0 {
			return fmt.Errorf("provider '%s' has no endpoints", provider.Name)
		}

		for j, endpoint := range provider.Endpoints {
			if endpoint.URL == "" {
				return fmt.Errorf("provider '%s' endpoint at index %d has no URL", provider.Name, j)
			}

			if endpoint.TipAccount == "" {
				return fmt.Errorf("provider '%s' endpoint at index %d has no tip account", provider.Name, j)
			}
		}
	}

	return nil
}
