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

var commentReplyCmd = &cobra.Command{
	Use:     "comment-reply [repo-slug] [pr-id] [thread-id] [message]",
	Aliases: []string{"reply"},
	Short:   "Reply to a comment thread on a pull request",
	Long:    `Reply to a specific comment thread (inline or top-level) on a pull request. Avoids adding top-level noise.`,
	Args:    cobra.ExactArgs(4),
	Run: func(cmd *cobra.Command, args []string) {
		jsonOutput, _ := cmd.Flags().GetBool("json")
		repoSlug := args[0]
		prIDStr := args[1]
		threadIDStr := args[2]
		message := args[3]

		prID, err := strconv.Atoi(prIDStr)
		if err != nil {
			log.Fatalf("Invalid pull request ID: %s", prIDStr)
		}

		threadID, err := strconv.Atoi(threadIDStr)
		if err != nil {
			log.Fatalf("Invalid thread ID: %s", threadIDStr)
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

		// Reply to comment
		comment, err := client.ReplyToPullRequestComment(repoSlug, prID, threadID, message)
		if err != nil {
			log.Fatalf("Error replying to comment: %v", err)
		}

		if jsonOutput {
			output := struct {
				Workspace     string             `json:"workspace"`
				Repository    string             `json:"repository"`
				PullRequestID int                `json:"pull_request_id"`
				ThreadID      int                `json:"thread_id"`
				Comment       *bitbucket.Comment `json:"comment"`
			}{
				Workspace:     cfg.Bitbucket.Workspace,
				Repository:    repoSlug,
				PullRequestID: prID,
				ThreadID:      threadID,
				Comment:       comment,
			}

			jsonBytes, err := json.MarshalIndent(output, "", "  ")
			if err != nil {
				log.Fatalf("Error marshaling JSON: %v", err)
			}
			fmt.Println(string(jsonBytes))
			return
		}

		fmt.Printf("✅ Successfully replied to thread #%d\n", threadID)
		fmt.Printf("Comment ID: %d\n", comment.ID)
		fmt.Printf("Created: %s\n", comment.CreatedOn)
	},
}

func init() {
	commentReplyCmd.Flags().Bool("json", false, "Output in JSON format")
}
