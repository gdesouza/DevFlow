package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	Jira      JiraConfig      `json:"jira"`
	Bitbucket BitbucketConfig `json:"bitbucket"`
}

type JiraConfig struct {
	URL      string `json:"url"`
	Username string `json:"username"`
	Token    string `json:"token"`
}

type BitbucketConfig struct {
	Workspace     string `json:"workspace"`
	Username      string `json:"username"`       // Email address for authentication
	BitbucketUser string `json:"bitbucket_user"` // Username for API calls
	Token         string `json:"token"`
}

var configPath = filepath.Join(os.Getenv("HOME"), ".devflow", "config.json")

// Load reads the configuration from disk
func Load() (*Config, error) {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &Config{}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// Save writes the configuration to disk
func Save(config *Config) error {
	// Create config directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
