package cmd

import (
	"encoding/json"
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
	Long:    `Create a new comment on a specific pull request. Use --file and --line to post an inline comment anchored to a specific location in the diff.`,
	Args:    cobra.MinimumNArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		jsonOutput, _ := cmd.Flags().GetBool("json")
		filePath, _ := cmd.Flags().GetString("file")
		line, _ := cmd.Flags().GetInt("line")

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

		// Validate inline flags: both must be set, or neither
		if (filePath == "") != (line == 0) {
			log.Fatal("--file and --line must be used together")
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

		// Create comment (inline or top-level)
		var comment *bitbucket.Comment
		if filePath != "" {
			comment, err = client.CreatePullRequestInlineComment(repoSlug, prID, content, filePath, line)
		} else {
			comment, err = client.CreatePullRequestComment(repoSlug, prID, content)
		}
		if err != nil {
			log.Fatalf("Error creating pull request comment: %v", err)
		}

		if jsonOutput {
			output := struct {
				Workspace     string             `json:"workspace"`
				Repository    string             `json:"repository"`
				PullRequestID int                `json:"pull_request_id"`
				Comment       *bitbucket.Comment `json:"comment"`
			}{
				Workspace:     cfg.Bitbucket.Workspace,
				Repository:    repoSlug,
				PullRequestID: prID,
				Comment:       comment,
			}

			jsonBytes, err := json.MarshalIndent(output, "", "  ")
			if err != nil {
				log.Fatalf("Error marshaling JSON: %v", err)
			}
			fmt.Println(string(jsonBytes))
			return
		}

		// Display success message
		if filePath != "" {
			fmt.Printf("✅ Inline comment added successfully to PR #%d (%s:%d)\n", prID, filePath, line)
		} else {
			fmt.Printf("✅ Comment added successfully to PR #%d\n", prID)
		}
		fmt.Printf("💬 Comment ID: %d\n", comment.ID)
		fmt.Printf("📅 Created: %s\n", comment.CreatedOn)
		fmt.Println()
		fmt.Println("Comment content:")
		fmt.Println(comment.Content.Raw)
	},
}

func init() {
	addCommentCmd.Flags().Bool("json", false, "Output in JSON format")
	addCommentCmd.Flags().String("file", "", "File path for inline comment (requires --line)")
	addCommentCmd.Flags().Int("line", 0, "New-file line number for inline comment (requires --file)")
}
