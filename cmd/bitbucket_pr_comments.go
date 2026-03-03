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

		// Organize comments into threads
		threads := organizeThreads(comments)

		// Display thread summary
		fmt.Printf("💬 Comments on PR #%d (%d total, %d threads)\n", prID, len(comments), len(threads))
		fmt.Println(strings.Repeat("=", 80))

		for i, thread := range threads {
			displayThread(&thread, i+1)
			if i < len(threads)-1 {
				fmt.Println(strings.Repeat("-", 80))
			}
		}
	},
}

type CommentThread struct {
	RootComment *bitbucket.Comment
	Replies     []*bitbucket.Comment
	Resolved    bool
}

func organizeThreads(comments []bitbucket.Comment) []CommentThread {
	// Map to store threads by root comment ID
	threadMap := make(map[int]*CommentThread)
	var rootComments []*bitbucket.Comment

	// First pass: identify root comments
	for i := range comments {
		comment := &comments[i]
		if comment.Parent == nil {
			rootComments = append(rootComments, comment)
			threadMap[comment.ID] = &CommentThread{
				RootComment: comment,
				Replies:     []*bitbucket.Comment{},
				Resolved:    comment.Resolved,
			}
		}
	}

	// Second pass: organize replies
	for i := range comments {
		comment := &comments[i]
		if comment.Parent != nil {
			parentID := comment.Parent.ID
			if thread, exists := threadMap[parentID]; exists {
				thread.Replies = append(thread.Replies, comment)
			}
		}
	}

	// Convert map to slice in order of root comments
	var threads []CommentThread
	for _, rootComment := range rootComments {
		if thread, exists := threadMap[rootComment.ID]; exists {
			threads = append(threads, *thread)
		}
	}

	return threads
}

func displayThread(thread *CommentThread, index int) {
	comment := thread.RootComment

	// Display thread header with resolution status
	resolvedStatus := ""
	if thread.Resolved {
		resolvedStatus = " ✅ RESOLVED"
	} else {
		resolvedStatus = " ⚠️  UNRESOLVED"
	}

	fmt.Printf("[Thread %d] ID: %d%s\n", index, comment.ID, resolvedStatus)
	fmt.Printf("👤 %s\n", comment.User.DisplayName)
	fmt.Printf("📅 Created: %s\n", comment.CreatedOn)

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

	// Display replies
	if len(thread.Replies) > 0 {
		fmt.Printf("  💬 %d replies:\n", len(thread.Replies))
		for i, reply := range thread.Replies {
			fmt.Printf("  [%d.%d] %s (ID: %d)\n", index, i+1, reply.User.DisplayName, reply.ID)
			fmt.Printf("       📅 %s\n", reply.CreatedOn)
			fmt.Printf("       %s\n", reply.Content.Raw)
			if i < len(thread.Replies)-1 {
				fmt.Println()
			}
		}
		fmt.Println()
	}
}
