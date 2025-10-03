package cmd

import (
	"fmt"
	"log"

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
	Short:   "List pull requests",
	Long:    `List all pull requests for a repository. If no repo-slug is provided, uses the default repository.`,
	Run: func(cmd *cobra.Command, args []string) {
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

		// Determine repository slug
		slug := repoSlug
		if slug == "" && len(args) > 0 {
			slug = args[0]
		}
		if slug == "" {
			log.Fatal("Repository slug is required. Use --repo or provide as argument")
		}

		// Create Bitbucket client
		client := bitbucket.NewClient(&cfg.Bitbucket)

		// Get pull requests
		prs, err := client.GetPullRequests(slug)
		if err != nil {
			log.Fatalf("Error fetching pull requests: %v", err)
		}

		// Display results
		if len(prs) == 0 {
			fmt.Printf("No pull requests found for repository '%s'.\n", slug)
			return
		}

		fmt.Printf("Found %d pull requests for repository '%s':\n\n", len(prs), slug)
		for _, pr := range prs {
			statusIcon := getPRStatusIcon(pr.State)
			fmt.Printf("%s #%d - %s 🔗 https://bitbucket.org/%s/%s/pull-requests/%d\n", statusIcon, pr.ID, pr.Title, cfg.Bitbucket.Workspace, slug, pr.ID)
		}
	},
}

func init() {
	listPRsCmd.Flags().StringVarP(&repoSlug, "repo", "r", "", "Repository slug (required)")
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
