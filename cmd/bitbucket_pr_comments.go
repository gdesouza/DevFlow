package cmd

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"devflow/internal/bitbucket"
	"devflow/internal/config"
	"github.com/spf13/cobra"
)

var prCommentsCmd = &cobra.Command{
	Use:     "comments [repo-slug] [pr-id]",
	Aliases: []string{"comment", "list-comments"},
	Short:   "List all comments on a pull request",
	Long:    `Display all comments on a specific pull request, including inline code comments.`,
	Args:    cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		repoSlug := args[0]
		prIDStr := args[1]

		prID, err := strconv.Atoi(prIDStr)
		if err != nil {
			log.Fatalf("Invalid pull request ID: %s", prIDStr)
		}

		// Load configuration
		cfg, err := config.Load()
		if err != nil {
			log.Fatalf("Error loading config: %v", err)
		}

		// Validate required config
		if cfg.Bitbucket.Workspace == "" {
			log.Fatal("Bitbucket workspace not configured. Run: devflow config set bitbucket.workspace <workspace>")
		}
		if cfg.Bitbucket.Username == "" {
			log.Fatal("Bitbucket username not configured. Run: devflow config set bitbucket.username <username>")
		}
		if cfg.Bitbucket.Token == "" {
			log.Fatal("Bitbucket token not configured. Run: devflow config set bitbucket.token <token>")
		}

		// Create Bitbucket client
		client := bitbucket.NewClient(&cfg.Bitbucket)

		// Get comments
		comments, err := client.GetPullRequestComments(repoSlug, prID)
		if err != nil {
			log.Fatalf("Error fetching pull request comments: %v", err)
		}

		if len(comments) == 0 {
			fmt.Printf("No comments found on pull request #%d\n", prID)
			return
		}

		// Display comments
		fmt.Printf("💬 Comments on PR #%d (%d total)\n", prID, len(comments))
		fmt.Println(strings.Repeat("=", 80))

		for i, comment := range comments {
			displayComment(&comment, i+1)
			if i < len(comments)-1 {
				fmt.Println(strings.Repeat("-", 80))
			}
		}
	},
}

func displayComment(comment *bitbucket.Comment, index int) {
	fmt.Printf("[%d] 👤 %s (ID: %d)\n", index, comment.User.DisplayName, comment.ID)
	fmt.Printf("📅 Created: %s\n", comment.CreatedOn)

	if comment.UpdatedOn != comment.CreatedOn {
		fmt.Printf("🔄 Updated: %s\n", comment.UpdatedOn)
	}

	// Check if it's an inline comment
	if comment.Inline != nil {
		fmt.Printf("📄 File: %s", comment.Inline.Path)
		if comment.Inline.To > 0 {
			if comment.Inline.From > 0 && comment.Inline.From != comment.Inline.To {
				fmt.Printf(" (lines %d-%d)", comment.Inline.From, comment.Inline.To)
			} else {
				fmt.Printf(" (line %d)", comment.Inline.To)
			}
		}
		fmt.Println()
	}

	fmt.Println()
	fmt.Println(comment.Content.Raw)
	fmt.Println()
}
