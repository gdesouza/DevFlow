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
	repoSlug string
)

var listPRsCmd = &cobra.Command{
	Use:     "list [repo-slug]",
	Aliases: []string{"list-prs"},
	Short:   "List pull requests (watched repos)",
	Long: `List pull requests for watched repositories.

Behavior:
- With a repo slug argument or --repo flag: lists PRs only if repository is watched.
- Without a slug: aggregates PRs across all watched repositories.
- A repository must be added via 'devflow bitbucket repo watch add <repo>' to be included.`,
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

		watched := make(map[string]struct{})
		for _, w := range cfg.Bitbucket.WatchedRepos {
			watched[strings.ToLower(w)] = struct{}{}
		}
		if len(watched) == 0 {
			log.Fatal("No watched repositories. Add some with: devflow bitbucket repo watch add <repo>")
		}

		slug := repoSlug
		if slug == "" && len(args) > 0 {
			slug = args[0]
		}

		client := bitbucket.NewClient(&cfg.Bitbucket)

		if slug != "" {
			key := strings.ToLower(slug)
			if _, ok := watched[key]; !ok {
				log.Fatalf("Repository '%s' is not in watched list. Add it first with: devflow bitbucket repo watch add %s", slug, slug)
			}
			prs, err := client.GetPullRequests(slug)
			if err != nil {
				log.Fatalf("Error fetching pull requests: %v", err)
			}
			printRepoPRs(cfg.Bitbucket.Workspace, slug, prs)
			return
		}

		// Aggregate across all watched repos
		var total int
		for w := range watched {
			prs, err := client.GetPullRequests(w)
			if err != nil {
				fmt.Printf("Warning: failed to fetch '%s': %v\n", w, err)
				continue
			}
			if len(prs) == 0 {
				continue
			}
			printRepoPRs(cfg.Bitbucket.Workspace, w, prs)
			total += len(prs)
		}
		if total == 0 {
			fmt.Println("No pull requests in watched repositories.")
		}
	},
}

func init() {
	listPRsCmd.Flags().StringVarP(&repoSlug, "repo", "r", "", "Repository slug (optional; defaults to all watched)")
}

func printRepoPRs(workspace, slug string, prs []bitbucket.PullRequest) {
	if len(prs) == 0 {
		return
	}
	fmt.Printf("Repository: %s (%d PRs)\n", slug, len(prs))
	for _, pr := range prs {
		statusIcon := getPRStatusIcon(pr.State)
		fmt.Printf("  %s #%d - %s 🔗 https://bitbucket.org/%s/%s/pull-requests/%d\n", statusIcon, pr.ID, pr.Title, workspace, slug, pr.ID)
	}
	fmt.Println()
}

// getPRStatusIcon returns an appropriate emoji/icon for the given PR state
func getPRStatusIcon(state string) string {
	switch state {
	case "OPEN":
		return "🔄"
	case "MERGED":
		return "✅"
	case "DECLINED", "REJECTED":
		return "❌"
	case "SUPERSEDED":
		return "🔄"
	default:
		return "📝"
	}
}
