package cmd

import (
	"fmt"
	"strings"

	"devflow/internal/config"
	"github.com/spf13/cobra"
)

var setConfigCmd = &cobra.Command{
	Use:   "set [key] [value]",
	Short: "Set configuration value",
	Long:  `Set a configuration value (e.g., API tokens, URLs)`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		key := args[0]
		value := args[1]

		// Load existing config
		cfg, err := config.Load()
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			return
		}

		// Set the value based on the key path
		if err := setConfigValue(cfg, key, value); err != nil {
			fmt.Printf("Error setting config: %v\n", err)
			return
		}

		// Save the updated config
		if err := config.Save(cfg); err != nil {
			fmt.Printf("Error saving config: %v\n", err)
			return
		}

		fmt.Printf("Setting config %s = %s\n", key, value)
	},
}

func setConfigValue(cfg *config.Config, key, value string) error {
	parts := strings.Split(key, ".")
	if len(parts) != 2 {
		return fmt.Errorf("invalid key format, expected 'section.key'")
	}

	section := parts[0]
	field := parts[1]

	switch section {
	case "jira":
		switch field {
		case "url":
			cfg.Jira.URL = value
		case "username":
			cfg.Jira.Username = value
		case "token":
			cfg.Jira.Token = value
		default:
			return fmt.Errorf("unknown jira field: %s", field)
		}
	case "bitbucket":
		switch field {
		case "workspace":
			cfg.Bitbucket.Workspace = value
		case "username":
			cfg.Bitbucket.Username = value
		case "bitbucket_user":
			cfg.Bitbucket.BitbucketUser = value
		case "token":
			cfg.Bitbucket.Token = value
		default:
			return fmt.Errorf("unknown bitbucket field: %s", field)
		}
	default:
		return fmt.Errorf("unknown section: %s", section)
	}

	return nil
}
