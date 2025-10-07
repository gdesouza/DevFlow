package cmd

import (
	"fmt"
	"log"
	"net/url"

	"devflow/internal/config"
	"devflow/internal/jira"
	"github.com/spf13/cobra"
)

var (
	linkTitle   string
	linkSummary string
)

var linkCmd = &cobra.Command{
	Use:   "link [issue-key] [url]",
	Short: "Add an external document/link to a Jira issue",
	Long:  "Create a remote link (Jira remote object) pointing to an external document, dashboard, or resource.",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		issueKey := args[0]
		linkURL := args[1]

		if _, err := url.ParseRequestURI(linkURL); err != nil {
			log.Fatalf("Invalid URL: %v", err)
		}

		cfg, err := config.Load()
		if err != nil {
			log.Fatalf("Error loading config: %v", err)
		}
		if cfg.Jira.URL == "" || cfg.Jira.Username == "" || cfg.Jira.Token == "" {
			log.Fatal("Jira configuration incomplete. Run: devflow config set jira.url|jira.username|jira.token ...")
		}

		client := jira.NewClient(&cfg.Jira)
		if err := client.AddRemoteLink(issueKey, linkURL, linkTitle, linkSummary); err != nil {
			log.Fatalf("Failed to add link: %v", err)
		}
		fmt.Printf("🔗 Added link to %s: %s\n", issueKey, linkURL)
	},
}

func init() {
	linkCmd.Flags().StringVar(&linkTitle, "title", "", "Optional link title (defaults to URL if omitted)")
	linkCmd.Flags().StringVar(&linkSummary, "summary", "", "Optional short description of the link")
}
