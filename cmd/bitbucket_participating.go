package cmd

import (
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
		var total, printed int
		for w := range watched {
			prs, err := client.GetParticipatingPullRequests(w, username)
			if err != nil {
				fmt.Printf("Warning: failed to fetch '%s': %v\n", w, err)
				continue
			}
			if len(prs) == 0 {
				continue
			}
			fmt.Printf("Repository: %s (%d PRs)\n", w, len(prs))
			for _, pr := range prs {
				statusIcon := getPRStatusIcon(pr.State)
				fmt.Printf("  %s #%d - %s 🔗 https://bitbucket.org/%s/%s/pull-requests/%d\n", statusIcon, pr.ID, pr.Title, cfg.Bitbucket.Workspace, w, pr.ID)
			}
			fmt.Println()
			printed++
			total += len(prs)
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
}
