package config

import (
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	// Create a temporary config file for testing
	tempDir := t.TempDir()
	tempConfigPath := filepath.Join(tempDir, "config.json")

	// Override the config path for testing
	originalPath := configPath
	configPath = tempConfigPath
	defer func() { configPath = originalPath }()

	// Test loading non-existent config
	config, err := Load()
	if err != nil {
		t.Fatalf("Expected no error loading non-existent config, got: %v", err)
	}
	if config == nil {
		t.Fatal("Expected config to be initialized, got nil")
	}

	// Test saving and loading config
	testConfig := &Config{
		Jira: JiraConfig{
			URL:      "https://test.atlassian.net",
			Username: "test@example.com",
			Token:    "test-token",
		},
		Bitbucket: BitbucketConfig{
			Workspace: "test-workspace",
			Username:  "test-user",
			Token:     "test-token",
		},
	}

	err = Save(testConfig)
	if err != nil {
		t.Fatalf("Expected no error saving config, got: %v", err)
	}

	loadedConfig, err := Load()
	if err != nil {
		t.Fatalf("Expected no error loading config, got: %v", err)
	}

	if loadedConfig.Jira.URL != testConfig.Jira.URL {
		t.Errorf("Expected Jira URL %s, got %s", testConfig.Jira.URL, loadedConfig.Jira.URL)
	}
	if loadedConfig.Bitbucket.Workspace != testConfig.Bitbucket.Workspace {
		t.Errorf("Expected Bitbucket workspace %s, got %s", testConfig.Bitbucket.Workspace, loadedConfig.Bitbucket.Workspace)
	}
}
