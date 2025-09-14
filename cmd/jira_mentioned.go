package cmd

import (
	"fmt"
	"log"

	"devflow/internal/config"
	"devflow/internal/jira"
	"github.com/spf13/cobra"
)

var mentionedCmd = &cobra.Command{
	Use:   "mentioned",
	Short: "Find Jira issues where you are mentioned",
	Long:  `Search for Jira issues where you are mentioned in comments, descriptions, or other text fields`,
	Run: func(cmd *cobra.Command, args []string) {
		// Load configuration
		cfg, err := config.Load()
		if err != nil {
			log.Fatalf("Error loading config: %v", err)
		}

		// Validate required config
		if cfg.Jira.URL == "" {
			log.Fatal("Jira URL not configured. Run: devflow config set jira.url <url>")
		}
		if cfg.Jira.Username == "" {
			log.Fatal("Jira username not configured. Run: devflow config set jira.username <username>")
		}
		if cfg.Jira.Token == "" {
			log.Fatal("Jira token not configured. Run: devflow config set jira.token <token>")
		}

		// Create Jira client
		client := jira.NewClient(&cfg.Jira)

		// Search for mentions
		issues, err := client.FindMentions()
		if err != nil {
			log.Fatalf("Error searching for mentions: %v", err)
		}

		// Display results
		if len(issues) == 0 {
			fmt.Println("No Jira issues found where you are mentioned.")
			return
		}

		fmt.Printf("Found %d Jira issues where you are mentioned:\n\n", len(issues))
		for _, issue := range issues {
			statusIcon := getStatusIcon(issue.Fields.Status.Name)
			priorityIcon := getPriorityIcon(issue.Fields.Priority.Name)
			fmt.Printf("%s %s - %s %s 🔗 %s/browse/%s\n", statusIcon, issue.Key, priorityIcon, issue.Fields.Summary, cfg.Jira.URL, issue.Key)
			fmt.Printf("   📅 Updated: %s\n", issue.Fields.Updated)
			fmt.Printf("   👤 Assignee: %s\n", issue.Fields.Assignee.DisplayName)
			fmt.Println()
		}
	},
}

func init() {
	// This will be called when the jira command is initialized
}
