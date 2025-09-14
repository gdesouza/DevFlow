package cmd

import (
	"fmt"
	"strings"

	"devflow/internal/config"
	"github.com/spf13/cobra"
)

var getConfigCmd = &cobra.Command{
	Use:   "get [key]",
	Short: "Get configuration value",
	Long:  `Get a configuration value`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		key := args[0]

		// Load existing config
		cfg, err := config.Load()
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			return
		}

		// Get the value based on the key path
		value, err := getConfigValue(cfg, key)
		if err != nil {
			fmt.Printf("Error getting config: %v\n", err)
			return
		}

		if value == "" {
			fmt.Printf("No value set for %s\n", key)
		} else {
			fmt.Printf("%s = %s\n", key, value)
		}
	},
}

func getConfigValue(cfg *config.Config, key string) (string, error) {
	parts := strings.Split(key, ".")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid key format, expected 'section.key'")
	}

	section := parts[0]
	field := parts[1]

	switch section {
	case "jira":
		switch field {
		case "url":
			return cfg.Jira.URL, nil
		case "username":
			return cfg.Jira.Username, nil
		case "token":
			return cfg.Jira.Token, nil
		default:
			return "", fmt.Errorf("unknown jira field: %s", field)
		}
	case "bitbucket":
		switch field {
		case "workspace":
			return cfg.Bitbucket.Workspace, nil
		case "username":
			return cfg.Bitbucket.Username, nil
		case "bitbucket_user":
			return cfg.Bitbucket.BitbucketUser, nil
		case "token":
			return cfg.Bitbucket.Token, nil
		default:
			return "", fmt.Errorf("unknown bitbucket field: %s", field)
		}
	default:
		return "", fmt.Errorf("unknown section: %s", section)
	}
}
