package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"devflow/internal/bitbucket"
	"devflow/internal/config"
	"github.com/spf13/cobra"
)

var showPRCmd = &cobra.Command{
	Use:     "show [repo-slug] [pr-id]",
	Aliases: []string{"show-pr"},
	Short:   "Show detailed information about a pull request",
	Long:    `Display comprehensive details about a specific pull request including description, author, branches, reviewers, and status. Use --diff to show file changes.`,
	Args:    cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		showDiff, _ := cmd.Flags().GetBool("diff")
		jsonOutput, _ := cmd.Flags().GetBool("json")
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

		// Get pull request details
		pr, err := client.GetPullRequestDetails(repoSlug, prID)
		if err != nil {
			log.Fatalf("Error fetching pull request details: %v", err)
		}

		if jsonOutput {
			output := struct {
				Workspace  string                        `json:"workspace"`
				Repository string                        `json:"repository"`
				Details    *bitbucket.PullRequestDetails `json:"details"`
				Diff       string                        `json:"diff,omitempty"`
				URL        string                        `json:"url"`
			}{
				Workspace:  cfg.Bitbucket.Workspace,
				Repository: repoSlug,
				Details:    pr,
				URL:        fmt.Sprintf("https://bitbucket.org/%s/%s/pull-requests/%d", cfg.Bitbucket.Workspace, repoSlug, pr.ID),
			}

			if showDiff {
				diff, err := client.GetPullRequestDiff(repoSlug, prID)
				if err != nil {
					log.Fatalf("Error fetching diff: %v", err)
				}
				output.Diff = diff
			}

			jsonBytes, err := json.MarshalIndent(output, "", "  ")
			if err != nil {
				log.Fatalf("Error marshaling JSON: %v", err)
			}
			fmt.Println(string(jsonBytes))
			return
		}

		// Display pull request details
		displayPRDetails(pr, cfg.Bitbucket.Workspace, repoSlug)

		if showDiff {
			fmt.Println(strings.Repeat("-", 80))
			diff, err := client.GetPullRequestDiff(repoSlug, prID)
			if err != nil {
				fmt.Printf("Error fetching diff: %v\n", err)
			} else {
				fmt.Println("🔀 File Diff:")
				fmt.Println(diff)
			}
		}
	},
}

func init() {
	showPRCmd.Flags().Bool("diff", false, "Show file diff for the pull request")
	showPRCmd.Flags().Bool("json", false, "Output in JSON format")
}

func displayPRDetails(pr *bitbucket.PullRequestDetails, workspace, repoSlug string) {
	fmt.Printf("🔹 #%d: %s\n", pr.ID, pr.Title)

	// URL
	fmt.Printf("🔗 https://bitbucket.org/%s/%s/pull-requests/%d\n", workspace, repoSlug, pr.ID)

	fmt.Println(strings.Repeat("=", 80))

	// Status and State
	statusIcon := getPRStatusIcon(pr.State)
	fmt.Printf("📊 Status: %s %s\n", statusIcon, pr.State)

	// Author
	fmt.Printf("👤 Author: %s\n", pr.Author.DisplayName)

	// Branches
	fmt.Printf("📂 Branches: %s → %s\n", pr.Source.Branch.Name, pr.Destination.Branch.Name)

	// Created and Updated dates
	if pr.CreatedOn != "" {
		fmt.Printf("📅 Created: %s\n", pr.CreatedOn)
	}
	if pr.UpdatedOn != "" {
		fmt.Printf("🔄 Updated: %s\n", pr.UpdatedOn)
	}

	fmt.Println()

	// Description
	if pr.Description != "" {
		fmt.Println("📄 Description:")
		fmt.Println("─────────────")
		fmt.Println(pr.Description)
		fmt.Println()
	}

	// Reviewers
	if len(pr.Reviewers) > 0 {
		fmt.Printf("👥 Reviewers (%d):\n", len(pr.Reviewers))
		fmt.Println("────────────")
		for _, reviewer := range pr.Reviewers {
			fmt.Printf("• %s\n", reviewer.DisplayName)
		}
		fmt.Println()
	}
}
