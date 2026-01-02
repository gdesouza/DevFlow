package cmd

import (
	"fmt"
	"log"

	"devflow/internal/config"
	"devflow/internal/jira"

	"github.com/spf13/cobra"
)

var spacesCmd = &cobra.Command{
	Use:   "spaces",
	Short: "List Jira projects (spaces)",
	Long:  `List all Jira projects (spaces) accessible to the current user`,
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

		// Get projects
		projects, err := client.ListProjects()
		if err != nil {
			log.Fatalf("Error listing Jira projects: %v", err)
		}

		// Display results
		if len(projects) == 0 {
			fmt.Println("No Jira projects found.")
			return
		}

		fmt.Printf("Found %d Jira projects:\n\n", len(projects))
		for _, project := range projects {
			fmt.Printf("  %s - %s", project.Key, project.Name)
			if project.Lead.DisplayName != "" {
				fmt.Printf(" (Lead: %s)", project.Lead.DisplayName)
			}
			fmt.Println()
		}
	},
}

func init() {
	// No flags needed for now
}
