package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"devflow/internal/bitbucket"
	"devflow/internal/config"
	"github.com/spf13/cobra"
)

var (
	participatingRepoSlug string
)

var participatingCmd = &cobra.Command{
	Use:   "participating [repo-slug]",
	Short: "PRs you participate in (watched repos)",
	Long: `List pull requests where you participate (author/reviewer/etc.).

Behavior:
- With a repo slug: requires it be watched.
- Without slug: aggregates across all watched repositories.`,
	Run: func(cmd *cobra.Command, args []string) {
		jsonOutput, _ := cmd.Flags().GetBool("json")
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

		// Determine username (prefer bitbucket_user override)
		username := cfg.Bitbucket.BitbucketUser
		if username == "" {
			username = cfg.Bitbucket.Username
			if username == "" {
				log.Fatal("Bitbucket username or bitbucket_user not configured. Set one with: devflow config set bitbucket.username <email> or bitbucket.bitbucket_user <username>")
			}
		}

		// Build watched repo set
		watched := map[string]struct{}{}
		for _, w := range cfg.Bitbucket.WatchedRepos {
			watched[strings.ToLower(w)] = struct{}{}
		}
		if len(watched) == 0 {
			log.Fatal("No watched repositories. Add some with: devflow bitbucket repo watch add <repo>")
		}

		// Determine single slug if provided via flag or arg
		slug := participatingRepoSlug
		if slug == "" && len(args) > 0 {
			slug = args[0]
		}

		client := bitbucket.NewClient(&cfg.Bitbucket)

		if slug != "" { // Validate and show for single repo
			key := strings.ToLower(slug)
			if _, ok := watched[key]; !ok {
				log.Fatalf("Repository '%s' is not watched. Add with: devflow bitbucket repo watch add %s", slug, slug)
			}
			prs, err := client.GetParticipatingPullRequests(slug, username)
			if err != nil {
				log.Fatalf("Error fetching participating pull requests: %v", err)
			}

			if jsonOutput {
				output := struct {
					Workspace    string                  `json:"workspace"`
					Repository   string                  `json:"repository"`
					Username     string                  `json:"username"`
					TotalPRs     int                     `json:"total_prs"`
					PullRequests []bitbucket.PullRequest `json:"pull_requests"`
				}{
					Workspace:    cfg.Bitbucket.Workspace,
					Repository:   slug,
					Username:     username,
					TotalPRs:     len(prs),
					PullRequests: prs,
				}

				jsonBytes, err := json.MarshalIndent(output, "", "  ")
				if err != nil {
					log.Fatalf("Error marshaling JSON: %v", err)
				}
				fmt.Println(string(jsonBytes))
				return
			}

			if len(prs) == 0 {
				fmt.Printf("No pull requests found in '%s' where you participate.\n", slug)
				return
			}
			fmt.Printf("Found %d pull requests in '%s' where you participate (user: %s):\n\n", len(prs), slug, username)
			for _, pr := range prs {
				statusIcon := getPRStatusIcon(pr.State)
				fmt.Printf("%s #%d - %s 🔗 https://bitbucket.org/%s/%s/pull-requests/%d\n", statusIcon, pr.ID, pr.Title, cfg.Bitbucket.Workspace, slug, pr.ID)
			}
			return
		}

		// Aggregate across all watched repos
		type RepoWithPRs struct {
			Repository   string                  `json:"repository"`
			PullRequests []bitbucket.PullRequest `json:"pull_requests"`
		}
		var reposWithPRs []RepoWithPRs
		var total, printed int

		for w := range watched {
			prs, err := client.GetParticipatingPullRequests(w, username)
			if err != nil {
				if !jsonOutput {
					fmt.Printf("Warning: failed to fetch '%s': %v\n", w, err)
				}
				continue
			}
			if len(prs) == 0 {
				continue
			}

			if jsonOutput {
				reposWithPRs = append(reposWithPRs, RepoWithPRs{
					Repository:   w,
					PullRequests: prs,
				})
			} else {
				fmt.Printf("Repository: %s (%d PRs)\n", w, len(prs))
				for _, pr := range prs {
					statusIcon := getPRStatusIcon(pr.State)
					fmt.Printf("  %s #%d - %s 🔗 https://bitbucket.org/%s/%s/pull-requests/%d\n", statusIcon, pr.ID, pr.Title, cfg.Bitbucket.Workspace, w, pr.ID)
				}
				fmt.Println()
				printed++
			}
			total += len(prs)
		}

		if jsonOutput {
			output := struct {
				Workspace    string        `json:"workspace"`
				Username     string        `json:"username"`
				TotalPRs     int           `json:"total_prs"`
				Repositories []RepoWithPRs `json:"repositories"`
			}{
				Workspace:    cfg.Bitbucket.Workspace,
				Username:     username,
				TotalPRs:     total,
				Repositories: reposWithPRs,
			}

			jsonBytes, err := json.MarshalIndent(output, "", "  ")
			if err != nil {
				log.Fatalf("Error marshaling JSON: %v", err)
			}
			fmt.Println(string(jsonBytes))
			return
		}

		if total == 0 {
			fmt.Println("No participating pull requests found in watched repositories.")
		} else {
			fmt.Printf("Total participating pull requests across watched repos: %d\n", total)
		}
	},
}

func init() {
	participatingCmd.Flags().StringVarP(&participatingRepoSlug, "repo", "r", "", "Repository slug (optional; defaults to all watched)")
	participatingCmd.Flags().Bool("json", false, "Output in JSON format")
}
