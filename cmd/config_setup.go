package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"devflow/internal/config"
	"github.com/spf13/cobra"
)

var setupConfigCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive configuration setup",
	Long:  "Set up your devflow configuration interactively",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Welcome to devflow configuration setup!")
		fmt.Println("This will guide you through setting up Jira and Bitbucket integration.")
		fmt.Println("")

		// Load existing config
		cfg, err := config.Load()
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			return
		}

		reader := bufio.NewReader(os.Stdin)

		// Configure Jira
		fmt.Println("=== Jira Configuration ===")
		cfg.Jira.URL = promptWithDefault(reader, "Jira URL (e.g., https://company.atlassian.net)", cfg.Jira.URL)
		cfg.Jira.Username = promptWithDefault(reader, "Jira username/email", cfg.Jira.Username)
		cfg.Jira.Token = promptWithDefault(reader, "Jira API token", cfg.Jira.Token)

		fmt.Println("")

		// Configure Bitbucket
		fmt.Println("=== Bitbucket Configuration ===")
		cfg.Bitbucket.Workspace = promptWithDefault(reader, "Bitbucket workspace", cfg.Bitbucket.Workspace)
		cfg.Bitbucket.Username = promptWithDefault(reader, "Bitbucket email (for authentication)", cfg.Bitbucket.Username)
		cfg.Bitbucket.BitbucketUser = promptWithDefault(reader, "Bitbucket username (for API calls)", cfg.Bitbucket.BitbucketUser)
		cfg.Bitbucket.Token = promptWithDefault(reader, "Bitbucket app password", cfg.Bitbucket.Token)

		// Save configuration
		if err := config.Save(cfg); err != nil {
			fmt.Printf("Error saving config: %v\n", err)
			return
		}

		fmt.Println("")
		fmt.Println("✅ Configuration saved successfully!")
		fmt.Println("You can now use 'devflow jira' and 'devflow bitbucket' commands.")
		fmt.Println("")
		fmt.Println("Next steps:")
		fmt.Println("- Test Jira connection: devflow jira list")
		fmt.Println("- Test Bitbucket connection: devflow bitbucket list")
	},
}

func promptWithDefault(reader *bufio.Reader, prompt, defaultValue string) string {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", prompt, defaultValue)
	} else {
		fmt.Printf("%s: ", prompt)
	}

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return defaultValue
	}
	return input
}
