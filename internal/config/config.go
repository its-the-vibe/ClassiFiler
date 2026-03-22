// Package config loads and validates the ClassiFiler configuration file.
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the top-level configuration structure.
type Config struct {
	Redis       RedisConfig        `yaml:"redis"`
	Classifiers []ClassifierConfig `yaml:"classifiers"`
}

// RedisConfig holds the Redis connection and queue settings.
type RedisConfig struct {
	Host          string `yaml:"host"`
	Port          int    `yaml:"port"`
	InputQueue    string `yaml:"input_queue"`
	OutputChannel string `yaml:"output_channel"`
}

// ClassifierConfig defines a single classifier entry in the config file.
type ClassifierConfig struct {
	Name      string `yaml:"name"`
	Type      string `yaml:"type"`
	Pattern   string `yaml:"pattern,omitempty"`
	TargetDir string `yaml:"target_dir"`
}

// Load reads and parses the YAML configuration file at the given path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file %q: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file %q: %w", path, err)
	}

	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

func validate(cfg *Config) error {
	if cfg.Redis.Host == "" {
		return fmt.Errorf("redis.host is required")
	}
	if cfg.Redis.Port == 0 {
		return fmt.Errorf("redis.port is required")
	}
	if cfg.Redis.InputQueue == "" {
		return fmt.Errorf("redis.input_queue is required")
	}
	if cfg.Redis.OutputChannel == "" {
		return fmt.Errorf("redis.output_channel is required")
	}
	if len(cfg.Classifiers) == 0 {
		return fmt.Errorf("at least one classifier is required")
	}
	for i, c := range cfg.Classifiers {
		if c.Name == "" {
			return fmt.Errorf("classifiers[%d].name is required", i)
		}
		if c.Type == "" {
			return fmt.Errorf("classifiers[%d].type is required", i)
		}
		if c.TargetDir == "" {
			return fmt.Errorf("classifiers[%d].target_dir is required", i)
		}
	}
	return nil
}
