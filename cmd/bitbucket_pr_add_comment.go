package cmd

import (
	"fmt"
	"log"
	"strconv"

	"devflow/internal/bitbucket"
	"devflow/internal/config"
	"github.com/spf13/cobra"
)

var addCommentCmd = &cobra.Command{
	Use:     "add-comment [repo-slug] [pr-id] [comment]",
	Aliases: []string{"comment-add", "create-comment"},
	Short:   "Add a comment to a pull request",
	Long:    `Create a new comment on a specific pull request.`,
	Args:    cobra.MinimumNArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		repoSlug := args[0]
		prIDStr := args[1]
		// Join all remaining args as the comment content (in case it has spaces)
		content := ""
		for i := 2; i < len(args); i++ {
			if i > 2 {
				content += " "
			}
			content += args[i]
		}

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

		// Create comment
		comment, err := client.CreatePullRequestComment(repoSlug, prID, content)
		if err != nil {
			log.Fatalf("Error creating pull request comment: %v", err)
		}

		// Display success message
		fmt.Printf("✅ Comment added successfully to PR #%d\n", prID)
		fmt.Printf("💬 Comment ID: %d\n", comment.ID)
		fmt.Printf("📅 Created: %s\n", comment.CreatedOn)
		fmt.Println()
		fmt.Println("Comment content:")
		fmt.Println(comment.Content.Raw)
	},
}
