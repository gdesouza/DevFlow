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

var prDiffCmd = &cobra.Command{
	Use:     "diff [repo-slug] [pr-id]",
	Aliases: []string{"unified-diff"},
	Short:   "Display unified diff for a pull request",
	Long:    `Retrieve and display the unified diff for a specific pull request. Optimized for AI consumption.`,
	Args:    cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
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

		// Get diff
		diff, err := client.GetPullRequestDiff(repoSlug, prID)
		if err != nil {
			log.Fatalf("Error fetching pull request diff: %v", err)
		}

		if jsonOutput {
			output := struct {
				Workspace     string `json:"workspace"`
				Repository    string `json:"repository"`
				PullRequestID int    `json:"pull_request_id"`
				Diff          string `json:"diff"`
			}{
				Workspace:     cfg.Bitbucket.Workspace,
				Repository:    repoSlug,
				PullRequestID: prID,
				Diff:          diff,
			}

			jsonBytes, err := json.MarshalIndent(output, "", "  ")
			if err != nil {
				log.Fatalf("Error marshaling JSON: %v", err)
			}
			fmt.Println(string(jsonBytes))
			return
		}

		// Output the diff directly
		fmt.Print(diff)
	},
}

func init() {
	prDiffCmd.Flags().Bool("json", false, "Output in JSON format")
}
