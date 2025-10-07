package cmd

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"devflow/internal/config"
	"devflow/internal/jira"
	"github.com/spf13/cobra"
)

var (
	commentBody     string
	commentBodyFile string
)

var commentCmd = &cobra.Command{
	Use:   "comment [issue-key]",
	Short: "Add a comment to a Jira issue",
	Long:  "Add a textual comment to a Jira issue (supports inline text or file input)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		issueKey := args[0]

		if commentBody == "" && commentBodyFile == "" {
			log.Fatal("Provide a comment with --body or --body-file")
		}
		if commentBody != "" && commentBodyFile != "" {
			log.Fatal("Specify either --body or --body-file, not both")
		}

		// Load config
		cfg, err := config.Load()
		if err != nil {
			log.Fatalf("Error loading config: %v", err)
		}
		if cfg.Jira.URL == "" || cfg.Jira.Username == "" || cfg.Jira.Token == "" {
			log.Fatal("Jira configuration incomplete. Run: devflow config set jira.url|jira.username|jira.token ...")
		}

		// Resolve comment body
		body, err := resolveCommentBody(commentBody, commentBodyFile)
		if err != nil {
			log.Fatalf("Failed to read comment body: %v", err)
		}

		client := jira.NewClient(&cfg.Jira)
		if err := client.AddComment(issueKey, body); err != nil {
			log.Fatalf("Failed to add comment: %v", err)
		}

		fmt.Printf("✅ Added comment to %s\n", issueKey)
	},
}

func init() {
	commentCmd.Flags().StringVarP(&commentBody, "body", "b", "", "Inline comment body text")
	commentCmd.Flags().StringVar(&commentBodyFile, "body-file", "", "Path to file containing comment body")
}

func resolveCommentBody(inline, filePath string) (string, error) {
	if inline != "" && filePath != "" {
		return "", errors.New("specify either --body or --body-file, not both")
	}
	if filePath != "" {
		data, err := os.ReadFile(filepath.Clean(filePath))
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
	return inline, nil
}
