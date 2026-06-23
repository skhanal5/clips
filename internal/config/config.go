// Package config loads and validates application configuration from a YAML file.
package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Thresholds configures the detection engine's spike detection.
type Thresholds struct {
	EvaluationSeconds int     `yaml:"evaluation_seconds"`
	BaselineSeconds   int     `yaml:"baseline_seconds"`
	TriggerRatio      float64 `yaml:"trigger_ratio"`
	CooldownSeconds   int     `yaml:"cooldown_seconds"`
}

// Config holds the application settings loaded from config.yaml.
type Config struct {
	ClientID   string     `yaml:"client_id"`
	Channels   []string   `yaml:"channels"`
	Verbose    bool       `yaml:"verbose"`
	Thresholds Thresholds `yaml:"thresholds"`
}

// Load reads and parses a YAML config file.
// If a file named <path>.local exists (e.g. config.local.yaml), it takes priority.
func Load(path string) (*Config, error) {
	if _, err := os.Stat(path + ".local"); err == nil {
		path = path + ".local"
	}
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
