package cmd

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"devflow/internal/bitbucket"
	"devflow/internal/config"
	"github.com/spf13/cobra"
)

var buildsCmd = &cobra.Command{
	Use:     "builds [repo-slug] [pr-id]",
	Aliases: []string{"statuses", "build-status"},
	Short:   "List build/status checks for commits in a pull request",
	Long: `Show all commit statuses (builds, deployments, checks) associated with each commit in a pull request.

Displays per-commit status including state, key/name, and URL.`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		repoSlug := args[0]
		prIDStr := args[1]

		prID, err := strconv.Atoi(prIDStr)
		if err != nil {
			log.Fatalf("Invalid pull request ID: %s", prIDStr)
		}

		cfg, err := config.Load()
		if err != nil {
			log.Fatalf("Error loading config: %v", err)
		}
		if cfg.Bitbucket.Workspace == "" {
			log.Fatal("Bitbucket workspace not configured. Run: devflow config set bitbucket.workspace <workspace>")
		}
		if cfg.Bitbucket.Token == "" {
			log.Fatal("Bitbucket token not configured. Run: devflow config set bitbucket.token <token>")
		}

		client := bitbucket.NewClient(&cfg.Bitbucket)

		fmt.Printf("Fetching commits for PR #%d in %s...\n", prID, repoSlug)
		commits, err := client.GetPullRequestCommits(repoSlug, prID)
		if err != nil {
			log.Fatalf("Error fetching pull request commits: %v", err)
		}
		if len(commits) == 0 {
			fmt.Println("No commits found for this pull request.")
			return
		}

		fmt.Printf("Found %d commits. Fetching statuses...\n\n", len(commits))

		for i, commit := range commits {
			shortHash := commit.Hash
			if len(shortHash) > 12 {
				shortHash = shortHash[:12]
			}
			fmt.Printf("Commit %d/%d: %s - %s\n", i+1, len(commits), shortHash, firstLine(commit.Message))
			fmt.Printf("Author: %s  Date: %s\n", commit.Author.Raw, commit.Date)

			statuses, err := client.GetCommitStatuses(repoSlug, commit.Hash)
			if err != nil {
				fmt.Printf("  Warning: Failed to fetch statuses: %v\n\n", err)
				continue
			}

			if len(statuses) == 0 {
				fmt.Println("  No statuses")
				fmt.Println()
				continue
			}

			for _, st := range statuses {
				icon := statusStateIcon(st.State)
				age := relativeTime(st.UpdatedOn)
				name := st.Name
				if name == "" {
					name = st.Key
				}
				display := name
				if st.Name != "" && st.Key != "" && st.Name != st.Key {
					display = fmt.Sprintf("%s (%s)", st.Name, st.Key)
				}
				fmt.Printf("  %s %s - %s\n", icon, st.State, display)
				if st.Description != "" {
					fmt.Printf("    %s\n", firstLine(st.Description))
				}
				if st.URL != "" {
					fmt.Printf("    🔗 %s\n", st.URL)
				}
				fmt.Printf("    Updated: %s\n", age)
			}
			fmt.Println()
		}
	},
}

func init() {
	// wired in bitbucket.go
}

func firstLine(s string) string {
	for i, r := range s {
		if r == '\n' || r == '\r' {
			return s[:i]
		}
	}
	return s
}

func statusStateIcon(state string) string {
	switch state {
	case "SUCCESSFUL":
		return "✅"
	case "FAILED", "ERROR":
		return "❌"
	case "INPROGRESS", "PENDING":
		return "🔄"
	case "STOPPED", "CANCELLED":
		return "🚫"
	default:
		return "📝"
	}
}

// relativeTime converts unix millis to human readable age
func relativeTime(ts string) string {
	if ts == "" {
		return "unknown"
	}
	// Bitbucket returns ISO8601 timestamps like 2025-10-15T14:52:21+00:00
	then, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		// Fallback: attempt to parse millis encoded as string
		if ms, convErr := strconv.ParseInt(ts, 10, 64); convErr == nil {
			sec := ms / 1000
			then = time.Unix(sec, 0)
		} else {
			return "unknown"
		}
	}
	dur := time.Since(then)
	if dur < time.Minute {
		return "just now"
	}
	if dur < time.Hour {
		return fmt.Sprintf("%dm ago", int(dur.Minutes()))
	}
	if dur < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(dur.Hours()))
	}
	if dur < 7*24*time.Hour {
		return fmt.Sprintf("%dd ago", int(dur.Hours()/24))
	}
	return then.Format("2006-01-02")
}
