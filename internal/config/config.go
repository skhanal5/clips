// Package config loads and validates application configuration from a YAML file.
package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds the application settings loaded from config.yaml.
type Config struct {
	ClientID string   `yaml:"client_id"`
	Channels []string `yaml:"channels"`
}

// Load reads and parses a YAML config file at the given path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
